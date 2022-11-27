package subscription

import (
	"time"

	"github.com/guregu/dynamo"
	"gjhr.me/newsletter/providers/aws"
	"gjhr.me/newsletter/providers/config"
	"gjhr.me/newsletter/utils/consterror"
)

const (
	ERR_SUBSCRIPTION_NOT_FOUND       = consterror.ConstError("Subscription not found")
	ERR_MULTIPLE_SUBSCRIPTIONS_FOUND = consterror.ConstError("Multiple subscriptions have been found")
)

var table dynamo.Table

func init() {
	table = aws.Dynamo().Table(config.Get().SubscriptionsTable)
}

type Subscription struct {
	Email                string    `dynamo:"email"`
	List                 string    `dynamo:"list"`
	VerificationToken    string    `dynamo:"verification_token"`
	Verified             string    `dynamo:"verified,omitempty"`
	LastSentVerification time.Time `dynamo:"last_sent_verification,unixtime"`
}

func Get(list, email string) (*Subscription, error) {
	var sub *Subscription
	err := table.Get("list", list).Range("email", dynamo.Equal, email).One(sub)
	if err != nil {
		return nil, err // todo return sub not found error
	}
	return sub, nil
}

func GetFromToken(token string) (*Subscription, error) {
	var subs []*Subscription
	err := table.Scan().Index("verification-token").Filter("'verification_token' = ?", token).All(subs)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, ERR_SUBSCRIPTION_NOT_FOUND
	}
	if len(subs) > 1 {
		return nil, ERR_MULTIPLE_SUBSCRIPTIONS_FOUND
	}
	return subs[0], nil
}

func Put(sub *Subscription) error {
	return table.Put(sub).Run()
}

func (sub *Subscription) update() *dynamo.Update {
	return table.Update("list", sub.List).Range("email", sub.Email)
}

func (sub *Subscription) UpdateLastSentVerification() error {
	return sub.update().Set("last_sent_verification", time.Now()).Run()
}

func (sub *Subscription) Verify() error {
	return sub.update().Set("verified", true).Run()
}

func (sub *Subscription) Delete() error {
	return table.Delete("list", sub.List).Range("email", sub.Email).Run()
}
