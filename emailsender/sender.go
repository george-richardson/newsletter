package emailsender

import (
	"html/template"
	"strings"

	"github.com/apex/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sesv2"
)

var ses *sesv2.SESV2

func init() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ses = sesv2.New(sess)
}

func SendMail(email string, sender string, replyTo string, subject string, template *template.Template, data interface{}) error {
	log.Infof("Sending email with subject '%v' to '%v'...", subject, email)
	// Format the body
	sb := &strings.Builder{}
	err := template.Execute(sb, data)
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
		FromEmailAddress: &sender,
		ReplyToAddresses: aws.StringSlice([]string{replyTo}),
		Destination: &sesv2.Destination{
			ToAddresses: aws.StringSlice([]string{email}),
		},
	})
	if err != nil {
		return err
	}

	return nil
}
