package mail

import (
	"gjhr.me/newsletter/data/list"
	"gjhr.me/newsletter/data/subscription"
)

type Mail struct {
	To             string             `json:"to"`
	From           string             `json:"from"`
	ReplyTo        string             `json:"reply_to"`
	Subject        string             `json:"subject"`
	TemplateBucket string             `json:"template_bucket"`
	TemplateKey    string             `json:"template_key"`
	TemplateValues MailTemplateValues `json:"template_values"`
}

func New(s *subscription.Subscription, l *list.List, subject string, templateBucket string, templateKey string) Mail {
	return Mail{
		To:             s.Email,
		From:           l.FromAddress,
		ReplyTo:        l.ReplyToAddress,
		TemplateBucket: templateBucket,
		TemplateKey:    templateKey,
		Subject:        subject,
		TemplateValues: MailTemplateValues{
			UnsubscribeLink: l.FormatUnsubscribeLink(*s),
			ListName:        l.Name,
			Email:           s.Email,
		},
	}
}

type MailTemplateValues struct {
	UnsubscribeLink string
	ListName        string
	Email           string
}
