package listmanagement

import "fmt"

type List struct {
	name           string
	fromAddress    string
	replyToAddress string
}

func (l List) FormatUnsubscribeLink(subscription Subscription) string {
	return fmt.Sprintf("FORMATTED_UNSUBSCRIBE_LINK?list=%v;email=%v", l.name, subscription.Email)
}
