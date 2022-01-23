package iamsigned

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"golang.org/x/net/context/ctxhttp"
)

type (
	graphqlError struct {
		Locations []graphqlErrorLocation `json:"locations"`
		Message   string                 `json:"message"`
	}

	graphqlErrorLocation struct {
		Column int `json:"column"`
		Line   int `json:"line"`
	}

	graphqlResponse struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphqlError  `json:"errors"`
	}
)

type AWSService string

const (
	AppSyncService    AWSService = "appsync"
	APIGatewayService AWSService = "execute-api"
)

// AppSync signs and send a request to appsync. It also parse the response and looks for graphql errors
func AppSync(payload []byte, endpoint, region string, creds *credentials.Credentials) ([]byte, error) {
	return AppSyncWithContext(context.Background(), payload, endpoint, region, creds)
}

// AppSyncWithContext does the same as AppSyncDeliver, with a context.Context object
func AppSyncWithContext(ctx context.Context, payload []byte, endpoint, region string, creds *credentials.Credentials) ([]byte, error) {
	body, err := deliverWithContext(ctx, payload, AppSyncService, endpoint, region, http.MethodPost, creds)
	if err != nil {
		return nil, err
	}
	return ParseGraphQLResponse(body)
}

// APIGatewayWithContext does the same as APIGatewayDeliver, with a context.Context object
func APIGatewayWithContext(ctx context.Context, payload []byte, endpoint, region, method string, creds *credentials.Credentials) ([]byte, error) {
	body, err := deliverWithContext(ctx, payload, APIGatewayService, endpoint, region, method, creds)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// APIGateway signs and sends a request to API Gateway
func APIGateway(payload []byte, endpoint, region, method string, creds *credentials.Credentials) ([]byte, error) {
	return APIGatewayWithContext(context.Background(), payload, endpoint, region, method, creds)
}

// ParseGraphQLResponse attempts to read the response, and extract grpahql-formatted errors
func ParseGraphQLResponse(body io.ReadCloser) (json.RawMessage, error) {

	var parsed graphqlResponse
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(body); err != nil {
		return []byte{}, fmt.Errorf("could not read buffer: %w", err)
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		newStr := buf.String()
		return []byte{}, fmt.Errorf("could not parse response '%s': %w", newStr, err)
	}

	if len(parsed.Errors) > 0 {
		errStr := fmt.Sprintf("GraphQL returned %v error(s)", len(parsed.Errors))
		for _, err := range parsed.Errors {
			errStr += fmt.Sprintf("\n %+v: %s", err.Locations, err.Message)
		}
		return parsed.Data, fmt.Errorf(errStr)
	}
	return parsed.Data, nil
}

func deliver(ctx context.Context, payload []byte, service AWSService, endpoint, region, method string, creds *credentials.Credentials) (io.ReadCloser, error) {
	return deliverWithContext(context.Background(), payload, service, endpoint, region, method, creds)
}

func deliverWithContext(ctx context.Context, payload []byte, service AWSService, endpoint, region, method string, creds *credentials.Credentials) (io.ReadCloser, error) {

	// Create http request
	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign the request
	signer := v4.NewSigner(creds)
	_, err = signer.Sign(req, bytes.NewReader(payload), string(service), region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to sign the request: %w", err)
	}

	// Fire !
	response, err := ctxhttp.Do(ctx, nil, req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %v", response.StatusCode)
	}

	return response.Body, nil
}
