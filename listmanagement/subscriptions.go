package listmanagement

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type Subscription struct {
	Email                string    `dynamodbav:"email"`
	List                 string    `dynamodbav:"list"`
	VerificationToken    string    `dynamodbav:"verification_token"`
	Verified             string    `dynamodbav:"verified,omitempty"`
	LastSentVerification time.Time `dynamodbav:"last_sent_verification,unixtime"`
}

func (s Subscription) FormatVerificationLink() string {
	return fmt.Sprintf("FORMATTED_VERIFICATION_LINK?token=%v", s.VerificationToken)
}

func Subscribe(list, email string) error {
	log.Infof("Subscribing '%v' to list '%v'...", email, list)
	// Validate email
	validAddress, err := mail.ParseAddress(email)
	if err != nil {
		return ERR_INVALID_EMAIL
	}
	email = validAddress.Address

	// TODO check if list exists

	subscription, err := getSubscription(list, email)
	if err != nil {
		return err
	}

	if subscription != nil {
		return resendVerificationEmail(*subscription)
	}

	// Generate verification token
	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	// Save row to Dynamodb table
	subscription = &Subscription{
		Email:                email,
		List:                 list,
		VerificationToken:    uuid.String(),
		LastSentVerification: time.Now(),
	}
	av, err := dynamodbattribute.MarshalMap(subscription)
	if err != nil {
		return err
	}

	_, err = dynamo.PutItem(&dynamodb.PutItemInput{
		Item:      av,
		TableName: &subscriptionsTableName,
	})
	if err != nil {
		return err
	}

	// Send verification email
	return sendVerificationEmail(*subscription)
}

func resendVerificationEmail(subscription Subscription) error {
	log.Infof("Resending verification email to '%v' for list '%v'...", subscription.Email, subscription.List)
	if subscription.LastSentVerification.After(time.Now().Add(time.Minute * -15)) {
		return ERR_RECENTLY_SENT_VERIFICATION
	}
	err := sendVerificationEmail(subscription)
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
				S: &subscription.Email,
			},
			"list": {
				S: &subscription.List,
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

func sendVerificationEmail(subscription Subscription) error {
	log.Infof("Sending verification email to '%v' for list '%v'...", subscription.Email, subscription.List)
	if subscription.Verified != "" {
		return ERR_ALREADY_VERIFIED
	}

	// TODO actually send the email
	return nil
}

func Verify(token string) error {
	log.Infof("Verifiying token '%v'...", token)
	// Set email as verified
	response, err := querySubscriptions(&dynamodb.QueryInput{
		TableName: &subscriptionsTableName,
		IndexName: aws.String("verification-token"),
	})

	subscriptions := *response

	if err != nil {
		return err
	}
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
