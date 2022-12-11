package mail

import (
	"gjhr.me/newsletter/data/list"
	"gjhr.me/newsletter/data/subscription"
)

type Mail struct {
	Subscription   subscription.Subscription `json:"subscription"`
	List           list.List                 `json:"list"`
	Subject        string                    `json:"subject"`
	TemplateBucket string                    `json:"template_bucket"`
	TemplateKey    string                    `json:"template_key"`
}

func (m *Mail) GetMailTemplateValues() MailTemplateValues {
	return MailTemplateValues{
		UnsubscribeLink: m.List.FormatUnsubscribeLink(m.Subscription),
		ListName:        m.List.Name,
		Email:           m.Subscription.Email,
	}
}

type MailTemplateValues struct {
	UnsubscribeLink string
	ListName        string
	Email           string
}
