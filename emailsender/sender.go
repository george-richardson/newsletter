package emailsender

import (
	"html/template"
	"strings"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sesv2"
	naws "gjhr.me/newsletter/providers/aws"
)

var ses *sesv2.SESV2

func init() {
	ses = naws.SES()
}

func SendMail(email string, sender string, replyTo string, subject string, template *template.Template, data interface{}) error {
	//todo add List-Unsubscribe header using raw email (enmime package?)
	//todo Get email sender name to match newsletter title
	//todo integration test with inbucket
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
