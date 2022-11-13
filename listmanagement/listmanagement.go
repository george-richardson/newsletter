package listmanagement

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var dynamo *dynamodb.DynamoDB
var subscriptionsTableName string

var ERR_INVALID_EMAIL = errors.New("Invalid email address.")
var ERR_RECENTLY_SENT_VERIFICATION = errors.New("A verification email for this subscription has recently been sent.")
var ERR_ALREADY_VERIFIED = errors.New("Subscription already verified.")
var ERR_UNKNOWN_TOKEN = errors.New("Unknown verification token.")
var ERR_UNEXPECTED = errors.New("An unexpected error has occurred.")
var ERR_SUBSCRIPTION_NOT_FOUND = errors.New("Subscription does not exist.")

func init() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dynamo = dynamodb.New(sess)
	subscriptionsTableName = "subscriptions"
}
