package listmanagement

import (
	"fmt"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type List struct {
	Name           string `dynamodbav:"name"`
	Description    string `dynamodbav:"description"`
	Domain         string `dynamodbav:"domain"`
	FromAddress    string `dynamodbav:"from_address"`
	ReplyToAddress string `dynamodbav:"reply_to_address"`
}

func (l List) FormatUnsubscribeLink(subscription Subscription) string {
	return fmt.Sprintf("https://%v/unsubscribe?email=%v", l.Domain, subscription.Email)
}

func GetListByDomain(domain string) (*List, error) {
	log.Infof("Finding list for domain '%v'...", domain)
	response, err := queryLists(&dynamodb.QueryInput{
		TableName:              &listsTableName,
		IndexName:              aws.String("domain"),
		KeyConditionExpression: aws.String("#d = :domain"),
		ExpressionAttributeNames: map[string]*string{
			"#d": aws.String("domain"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":domain": {S: &domain},
		},
	})
	if err != nil {
		return nil, err
	}

	lists := *response
	if len(lists) == 0 {
		return nil, ERR_UNKNOWN_LIST
	}
	if len(lists) != 1 {
		return nil, ERR_UNEXPECTED
	}

	list := lists[0]
	log.Infof("Found list '%v' for domain '%v'!", list.Name, domain)
	return &list, nil
}

func getList(name string) (*List, error) {
	response, err := dynamo.GetItem(&dynamodb.GetItemInput{
		TableName: &listsTableName,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: &name,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	result, err := unmarshalList(response.Item)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func queryLists(input *dynamodb.QueryInput) (*[]List, error) {
	response, err := dynamo.Query(input)
	if err != nil {
		return nil, err
	}
	if response.Items == nil {
		return nil, nil
	}
	result, err := unmarshalLists(response.Items)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func unmarshalList(m map[string]*dynamodb.AttributeValue) (*List, error) {
	if m == nil {
		return nil, nil
	}
	list := List{}
	err := dynamodbattribute.UnmarshalMap(m, &list)
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func unmarshalLists(m []map[string]*dynamodb.AttributeValue) (*[]List, error) {
	if m == nil {
		return nil, nil
	}
	var lists []List
	err := dynamodbattribute.UnmarshalListOfMaps(m, &lists)
	if err != nil {
		return nil, err
	}
	return &lists, nil
}
