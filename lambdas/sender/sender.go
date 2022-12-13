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
	reqJson, _ := json.Marshal(event)
	log.Debug(string(reqJson))

	for _, record := range event.Records {
		logger := log.WithFields(log.Fields{
			"MessageId": record.MessageId,
		})
		// Unmarshal message
		mail := mail.Mail{}
		err := json.Unmarshal([]byte(record.Body), &mail)
		if err != nil {
			logger.WithError(err).Error("Error while unmarshaling message from queue")
			return err
		}

		logger.Infof("Processing mail for '%v' with subject '%v'...", mail.To, mail.Subject)

		// Download template
		logger.Debugf("Downloading template from bucket '%v' key '%v'", mail.TemplateBucket, mail.TemplateKey)
		res, err := aws.S3().GetObject(&s3.GetObjectInput{
			Bucket: &mail.TemplateBucket,
			Key:    &mail.TemplateKey,
		})
		if err != nil {
			logger.WithError(err).Error("Error downloading template")
			return err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, res.Body)
		if err != nil {
			logger.WithError(err).Error("Error reading template")
			return err
		}
		logger.Debugf("Parsing template:\n%v", buf.String())
		t, err := template.New("body").Parse(buf.String())
		if err != nil {
			logger.WithError(err).Error("Error parsing template")
			return err
		}

		// Prepare for message deletion
		// MUST be done before sending mail to avoid possiblity of bugs causing multiple sends
		queueArn, err := arn.Parse(record.EventSourceARN)
		queueUrl := fmt.Sprintf("https://sqs.%v.amazonaws.com/%v/%v", queueArn.Region, queueArn.AccountID, queueArn.Resource)

		// Send mail
		logger.Debugf("Sending mail to '%v'", mail.To)
		err = emailsender.SendMail(mail.To, mail.From, mail.ReplyTo, mail.Subject, t, mail.TemplateValues)
		if err != nil {
			logger.WithError(err).Error("Error sending mail")
			return err
		}

		// Delete message from queue
		logger.Debugf("Deleting message from queue")
		_, err = aws.SQS().DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      &queueUrl,
			ReceiptHandle: &record.ReceiptHandle,
		})
		if err != nil {
			logger.WithError(err).Error("Error deleting message from queue")
			return err
		}
	}

	return nil
}
