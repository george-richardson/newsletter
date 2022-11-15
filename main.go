package main

import (
	"fmt"
	"regexp"

	"github.com/apex/log"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gjhr.me/newsletter/listmanagement"
)

// Route regex
var subscribeRoute = regexp.MustCompile(`/subscribe`)
var unsubscribeRoute = regexp.MustCompile(`/unsubscribe`)
var verifyRoute = regexp.MustCompile(`/verify`)

func main() {
	lambda.Start(HandleLambdaEvent)
}

func HandleLambdaEvent(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Infof("Handling request '%v' for path '%v'...", event.RequestContext.RequestID, event.Path)
	var err error
	var message string
	switch {
	case subscribeRoute.MatchString(event.Path):
		err = listmanagement.Subscribe(event.QueryStringParameters["list"], event.QueryStringParameters["email"])
		message = "subscribed!"
	case verifyRoute.MatchString(event.Path):
		err = listmanagement.Verify(event.QueryStringParameters["token"])
		message = "verified!"
	case unsubscribeRoute.MatchString(event.Path):
		err = listmanagement.Unsubscribe(event.QueryStringParameters["list"], event.QueryStringParameters["email"])
		message = "unsubscribed!"
	}
	result := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       fmt.Sprintf("{\"msg\": \"%v\"}", message),
	}
	return result, err
}
