package listmanagement

import (
	"fmt"
	"html/template"
	"net/mail"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"gjhr.me/newsletter/emailsender"
)

type Subscription struct {
	Email                string    `dynamodbav:"email"`
	List                 string    `dynamodbav:"list"`
	VerificationToken    string    `dynamodbav:"verification_token"`
	Verified             string    `dynamodbav:"verified,omitempty"`
	LastSentVerification time.Time `dynamodbav:"last_sent_verification,unixtime"`
}

func Subscribe(list *List, email string) (*Subscription, error) {
	log.Infof("Subscribing '%v' to list '%v'...", email, list.Name)
	// Validate email
	validAddress, err := mail.ParseAddress(email)
	if err != nil {
		return nil, ERR_INVALID_EMAIL
	}
	email = validAddress.Address

	// TODO check if list exists

	subscription, err := getSubscription(list.Name, email)
	if err != nil {
		return nil, err
	}

	if subscription != nil {
		return subscription, resendVerificationEmail(*subscription, list)
	}

	// Generate verification token
	uuid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// Save row to Dynamodb table
	subscription = &Subscription{
		Email:                email,
		List:                 list.Name,
		VerificationToken:    uuid.String(),
		LastSentVerification: time.Now(),
	}
	av, err := dynamodbattribute.MarshalMap(subscription)
	if err != nil {
		return nil, err
	}

	_, err = dynamo.PutItem(&dynamodb.PutItemInput{
		Item:      av,
		TableName: &subscriptionsTableName,
	})
	if err != nil {
		return nil, err
	}

	// Send verification email
	return subscription, sendVerificationEmail(*subscription, list)
}

func resendVerificationEmail(s Subscription, l *List) error {
	log.Infof("Resending verification email to '%v' for list '%v'...", s.Email, s.List)
	if s.LastSentVerification.After(time.Now().Add(time.Minute * -15)) {
		return ERR_RECENTLY_SENT_VERIFICATION
	}
	err := sendVerificationEmail(s, l)
	if err != nil {
		return err
	}

	now := dynamodbattribute.UnixTime(time.Now())
	timeav := dynamodb.AttributeValue{}
	err = now.MarshalDynamoDBAttributeValue(&timeav)
	if err != nil {
		return err
	}
	_, err = dynamo.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: &subscriptionsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: &s.Email,
			},
			"list": {
				S: &s.List,
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": &timeav,
		},
		UpdateExpression: aws.String("set last_sent_verification = :v"),
	})

	if err != nil {
		return err
	}

	return nil
}

func sendVerificationEmail(s Subscription, l *List) error {
	log.Infof("Sending verification email to '%v' for list '%v'...", s.Email, s.List)
	if s.Verified != "" {
		return ERR_ALREADY_VERIFIED
	}

	t, err := template.New("verification-email").Parse(`
	<!DOCTYPE html>
	<html lang="en" xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:schemas-microsoft-com:office:office">
	<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width,initial-scale=1">
			<meta name="x-apple-disable-message-reformatting">
			<title></title>
			<style>
					body {font-family: Arial, sans-serif;}
			</style>
	</head>
	<body>
		<h3>Please verify your email</h3>
		<p>
			To complete your subscription to {{ .List }}, please click <a href="{{ .VerificationLink }}">this link</a> or browse to the URL below.
		</p>
		<p>
			{{ .VerificationLink }}
		</p>
	</body>
	</html>
	`)
	if err != nil {
		return err
	}

	err = emailsender.SendMail(s.Email, l.FromAddress, l.ReplyToAddress, fmt.Sprintf("Verify email for %v", l.Name), t, struct{ List, VerificationLink string }{List: l.Name, VerificationLink: l.FormatVerificationLink(s)})
	if err != nil {
		return err
	}

	return nil
}

func Verify(token string) error {
	log.Infof("Verifiying token '%v'...", token)
	// Set email as verified
	response, err := querySubscriptions(&dynamodb.QueryInput{
		TableName:              &subscriptionsTableName,
		IndexName:              aws.String("verification-token"),
		KeyConditionExpression: aws.String("verification_token = :token"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":token": {S: &token},
		},
	})
	if err != nil {
		return err
	}

	subscriptions := *response
	if len(subscriptions) == 0 {
		return ERR_UNKNOWN_TOKEN
	}

	if len(subscriptions) != 1 {
		return ERR_UNEXPECTED
	}

	subscription := subscriptions[0]
	log.Infof("Token '%v' mapped to email '%v' subscribed to list '%v'...", token, subscription.Email, subscription.List)

	if subscription.Verified != "" {
		return ERR_ALREADY_VERIFIED
	}

	_, err = dynamo.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: &subscriptionsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: &subscription.Email,
			},
			"list": {
				S: &subscription.List,
			},
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": {
				S: aws.String("true"),
			},
		},
		UpdateExpression: aws.String("set verified = :v"),
	})

	if err != nil {
		return err
	}

	return nil
}

func Unsubscribe(list, email string) error {
	// Delete row from table
	log.Infof("Removing subscription of email '%v' to list '%v'...", email, list)

	_, err := dynamo.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: &subscriptionsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: &email,
			},
			"list": {
				S: &list,
			},
		},
		ConditionExpression: aws.String("attribute_exists(email)"),
	})

	_, condErr := err.(*dynamodb.ConditionalCheckFailedException)
	if condErr {
		return ERR_SUBSCRIPTION_NOT_FOUND
	}

	return err
}

func getSubscription(list, email string) (*Subscription, error) {
	response, err := dynamo.GetItem(&dynamodb.GetItemInput{
		TableName: &subscriptionsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: &email,
			},
			"list": {
				S: &list,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result, err := unmarshalSubscription(response.Item)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func querySubscriptions(input *dynamodb.QueryInput) (*[]Subscription, error) {
	response, err := dynamo.Query(input)
	if err != nil {
		return nil, err
	}
	if response.Items == nil {
		return nil, nil
	}
	result, err := unmarshalSubscriptions(response.Items)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func unmarshalSubscription(m map[string]*dynamodb.AttributeValue) (*Subscription, error) {
	if m == nil {
		return nil, nil
	}
	subscription := Subscription{}
	err := dynamodbattribute.UnmarshalMap(m, &subscription)
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

func unmarshalSubscriptions(m []map[string]*dynamodb.AttributeValue) (*[]Subscription, error) {
	if m == nil {
		return nil, nil
	}
	var subscriptions []Subscription
	err := dynamodbattribute.UnmarshalListOfMaps(m, &subscriptions)
	if err != nil {
		return nil, err
	}
	return &subscriptions, nil
}
