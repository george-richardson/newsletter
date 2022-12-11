package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/guregu/dynamo"
)

var awsSession *session.Session
var dynamoClient *dynamo.DB
var sesClient *sesv2.SESV2
var s3Client *s3.S3
var sqsClient *sqs.SQS

func init() {
	awsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dynamoClient = dynamo.New(awsSession)
	sesClient = sesv2.New(awsSession)
	s3Client = s3.New(awsSession)
	sqsClient = sqs.New(awsSession)
}

func Dynamo() *dynamo.DB {
	return dynamoClient
}

func SES() *sesv2.SESV2 {
	return sesClient
}

func S3() *s3.S3 {
	return s3Client
}

func SQS() *sqs.SQS {
	return sqsClient
}
