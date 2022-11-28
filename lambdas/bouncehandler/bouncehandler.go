package main

import (
	"context"
	"encoding/json"
	"github.com/apex/log"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gjhr.me/newsletter/data/subscription"
)

func main() {
	lambda.Start(Handle)
}

func Handle(ctx context.Context, event events.SNSEvent) {
	hasErrored := false
	for _, record := range event.Records {
		log.Info(record.SNS.Message)
		var message Message
		err := json.Unmarshal([]byte(record.SNS.Message), &message)
		if err != nil {
			hasErrored = true
		}
		// If permanent hard bounce or abuse complaint
		if (message.Bounce != nil && message.Bounce.BounceType == BounceTypePermanent) ||
			(message.Complaint != nil && message.Complaint.ComplaintFeedbackType != "not-spam") {
			// Permanently unsubscribe email from all lists
			for _, recepient := range message.Mail.Destination {
				err := subscription.DeleteAllForEmail(recepient)
				if err != nil {
					log.Warnf("Failed to delete all subscriptions for %v", recepient)
				}
				log.Infof("Deleted all subscriptions for %v", recepient)
			}
		}
	}
	if hasErrored {
		panic("Failure while handling one or more SNS records")
	}
}
