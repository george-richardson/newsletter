package emailsender

import (
	"errors"
	"strings"

	"github.com/apex/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sesv2"
	"gjhr.me/newsletter/emailrenderer"
	"gjhr.me/newsletter/listmanagement"
)

var ses *sesv2.SESV2

var ERR_AGGREGATE_ERROR = errors.New("There was a problem sending email to at least one subscription.")

func init() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ses = sesv2.New(sess)
}

func SendMail(subscriptions *[]listmanagement.Subscription, list listmanagement.List, subject string, template string) error {
	renderer, err := emailrenderer.NewRenderer(strings.NewReader(template))
	if err != nil {
		return err
	}

	var aggregateErr error = nil

	for _, subscription := range *subscriptions {

		urlMappings := map[string]string{
			"maildata-href-unsub":  list.FormatUnsubscribeLink(subscription),
			"maildata-href-verify": subscription.FormatVerificationLink(),
		}

		renderer.ReplaceHrefByID(urlMappings)

		sb := &strings.Builder{}
		err := renderer.Render(sb)
		if err != nil {
			return err
		}

		_, err = ses.SendEmail(&sesv2.SendEmailInput{
			Content: &sesv2.EmailContent{
				Simple: &sesv2.Message{
					Body: &sesv2.Body{
						Html: &sesv2.Content{
							Charset: aws.String("UTF-8"),
							Data:    aws.String(sb.String()),
						},
					},
					Subject: &sesv2.Content{
						Charset: aws.String("UTF-8"),
						Data:    &subject,
					},
				},
			},
			Destination: &sesv2.Destination{
				ToAddresses: aws.StringSlice([]string{subscription.Email}),
			},
		})

		if err != nil {
			log.Warnf("Failed sending email to '%v'", subscription.Email)
			aggregateErr = ERR_AGGREGATE_ERROR
		}
	}

	return aggregateErr
}
