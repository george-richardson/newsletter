package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/guregu/dynamo"
)

var awsSession *session.Session
var dynamoClient *dynamo.DB
var sesClient *sesv2.SESV2

func init() {
	awsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dynamoClient = dynamo.New(awsSession)
	sesClient = sesv2.New(awsSession)
}

func Dynamo() *dynamo.DB {
	return dynamoClient
}

func SES() *sesv2.SESV2 {
	return sesClient
}
