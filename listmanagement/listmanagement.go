package listmanagement

import (
	"errors"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var dynamo *dynamodb.DynamoDB
var subscriptionsTableName, listsTableName string

var ERR_UNEXPECTED = errors.New("An unexpected error has occurred.")

// Subscription errors
var ERR_INVALID_EMAIL = errors.New("Invalid email address.")
var ERR_RECENTLY_SENT_VERIFICATION = errors.New("A verification email for this subscription has recently been sent.")
var ERR_ALREADY_VERIFIED = errors.New("Subscription already verified.")
var ERR_UNKNOWN_TOKEN = errors.New("Unknown verification token.")
var ERR_SUBSCRIPTION_NOT_FOUND = errors.New("Subscription does not exist.")

// List errors
var ERR_UNKNOWN_LIST = errors.New("Mailing list does not exist.")

func init() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dynamo = dynamodb.New(sess)
	subscriptionsTableName = os.Getenv("NEWSLETTER_SUBSCRIPTIONS_TABLE")
	listsTableName = os.Getenv("NEWSLETTER_LISTS_TABLE")
}
