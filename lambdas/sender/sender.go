package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"gjhr.me/newsletter/data/mail"
	"gjhr.me/newsletter/emailsender"
	"gjhr.me/newsletter/providers/aws"
)

func main() {
	loglevel := os.Getenv("NEWSLETTER_LOG_LEVEL")
	if loglevel != "" {
		log.SetLevelFromString(loglevel)
	}
	lambda.Start(Handle)
}

func Handle(ctx context.Context, event events.SQSEvent) error {
	logger := log.WithFields(log.Fields{})
	if logger.Level == log.DebugLevel {
		reqJson, _ := json.Marshal(event)
		log.Debug(string(reqJson))
	}

	for _, record := range event.Records {
		// Unmarshal message
		mail := mail.Mail{}
		err := json.Unmarshal([]byte(record.Body), &mail)
		if err != nil {
			return err
		}

		// Download template
		res, err := aws.S3().GetObject(&s3.GetObjectInput{
			Bucket: &mail.TemplateBucket,
			Key:    &mail.TemplateKey,
		})
		if err != nil {
			return err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, res.Body)
		if err != nil {
			return err
		}
		t, err := template.New("body").Parse(buf.String())
		if err != nil {
			return err
		}

		// Prepare for message deletion
		// MUST be done before sending mail to avoid possiblity of bugs causing multiple sends
		queueArn, err := arn.Parse(record.EventSourceARN)
		queueUrl := fmt.Sprintf("https://sqs.%v.amazonaws.com/%v/%v", queueArn.Region, queueArn.AccountID, queueArn.Resource)

		// Send mail
		err = emailsender.SendMail(mail.Subscription.Email, mail.List.FromAddress, mail.List.ReplyToAddress, mail.Subject, t, mail) //todo use real message content here
		if err != nil {
			return err
		}

		// Delete message from queue
		aws.SQS().DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      &queueUrl,
			ReceiptHandle: &record.ReceiptHandle,
		})
	}

	return nil
}
