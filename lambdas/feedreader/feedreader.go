package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/mmcdole/gofeed"
	"gjhr.me/newsletter/data/list"
	"gjhr.me/newsletter/data/mail"
	"gjhr.me/newsletter/data/subscription"
	naws "gjhr.me/newsletter/providers/aws"
	"gjhr.me/newsletter/providers/config"
	"golang.org/x/exp/slices"
)

func main() {
	loglevel := os.Getenv("NEWSLETTER_LOG_LEVEL")
	if loglevel != "" {
		log.SetLevelFromString(loglevel)
	}
	lambda.Start(Handle)
}

func Handle(ctx context.Context, event events.CloudWatchEvent) error {
	reqJson, _ := json.Marshal(event)
	log.Debug(string(reqJson))

	lists, err := list.GetAll()
	if err != nil {
		return err
	}

	hasErrored := false

	now := time.Now()

	for _, l := range *lists {
		for _, feed := range l.Feeds {
			logger := log.WithFields(log.Fields{
				"list": l.Name,
				"feed": feed.Url,
			})
			logger.Info("Processing feed")
			parser := gofeed.NewParser()
			parsed, err := parser.ParseURL(feed.Url)
			if err != nil {
				hasErrored = true
				logger.WithError(err).Error("Failed to parse feed")
				continue
			}

			if parsed.UpdatedParsed != nil && parsed.UpdatedParsed.Equal(feed.LastUpdated) {
				logger.Info("Not updated since last checked, skipping")
				continue
			}

			for _, item := range parsed.Items {
				// Only process new items published in the last day or later
				if !slices.Contains(feed.ProcessedGuids, item.GUID) && item.PublishedParsed != nil && item.PublishedParsed.After(now.Add(-24*time.Hour)) {
					// Mark guid as processed first to avoid bugs causing multiple sends.
					// todo mark guid as processed

					// Queue up a message here
					err := QueueMails(item, l, logger)
					if err != nil {
						hasErrored = true
						continue
					}
				}
			}

			// todo Update last updated
		}
	}

	if hasErrored {
		return fmt.Errorf("Failed while processing one or more feeds.")
	}

	return nil
}

func QueueMails(item *gofeed.Item, l *list.List, logger *log.Entry) error {
	logger = logger.WithFields(log.Fields{
		"item": item.GUID,
	})
	logger.Info("Found new item, queueing mail")
	// Save body to S3
	// Get hash of content
	logger.Info("Saving content to S3")
	hasher := sha1.New()
	hasher.Write([]byte(item.Content))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	_, err := naws.S3().PutObject(&s3.PutObjectInput{
		Body:   strings.NewReader(item.Content),
		Bucket: aws.String(config.Get().TemplateBucket),
		Key:    &sha,
	})

	if err != nil {
		logger.WithError(err).Error("Failed to save item to S3")
		return err
	}

	// Retrieve list of subscribers
	logger.Info("Getting all subscribers for list.")
	subs, err := subscription.GetAllVerifiedFromList(l.Name)
	if err != nil {
		logger.WithError(err).Error("Failed to get subscribers for list")
		return err
	}

	// For each subscriber queue a mail
	for _, sub := range *subs {
		subLogger := logger.WithField("subscription", sub.Email)
		subLogger.Info("Queuing email")

		msg := mail.New(sub, l, item.Title, config.Get().TemplateBucket, sha)
		body, err := json.Marshal(msg)
		if err != nil {
			subLogger.WithError(err).Error("Failed to format message")
			return err
		}

		_, err = naws.SQS().SendMessage(&sqs.SendMessageInput{
			MessageDeduplicationId: &sub.Email,
			MessageGroupId:         aws.String(l.Name + item.GUID),
			QueueUrl:               aws.String(config.Get().SenderQueueUrl),
			MessageBody:            aws.String(string(body)),
		})

		if err != nil {
			subLogger.WithError(err).Error("Failed to queue email")
			//todo return err here?
		}
	}

	return nil
}
