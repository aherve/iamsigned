# golang-iam-requests

Provides helpers to send IAM-signed requests to AWS AppSync and AWS API Gateway services

- Generates a v4 sign using IAM credentials
- Properly sends the request to the aws service
- Checks for http status error code
- Parses graphQL response and checks for graphql-formatted errors (Appsync only)

## Example Usage (Appsync GraphQL)

```go
package main

import (
	"log"

	signedReq "github.com/aherve/golang-iam-requests"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var endpoint = "https://xxx.appsync-api.eu-west-1.amazonaws.com/graphql"
var region = "eu-west-1"

var myMutation = `
{
	"Query": "mutation(...)",
	"Variables": {...},
}
`

func main() {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := signedReq.AppSyncDeliver([]byte(myMutation), endpoint, region, sess.Config.Credentials)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("SUCCESS : %s", string(resp))
}

```
