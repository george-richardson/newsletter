package list

import (
	"fmt"
	"time"

	"github.com/guregu/dynamo"
	"gjhr.me/newsletter/data/subscription"
	"gjhr.me/newsletter/providers/aws"
	"gjhr.me/newsletter/providers/config"
	"gjhr.me/newsletter/utils/consterror"
)

const (
	ERR_LIST_NOT_FOUND         = consterror.ConstError("List not found")
	ERR_LIST_DOMAIN_DUPLICATED = consterror.ConstError("Multiple lists found for domain")
)

var table dynamo.Table

func init() {
	table = aws.Dynamo().Table(config.Get().ListsTable)
}

type List struct {
	Name           string `dynamo:"name" json:"name"`
	Description    string `dynamo:"description" json:"description"`
	Domain         string `dynamo:"domain" json:"domain"`
	FromAddress    string `dynamo:"from_address" json:"from_address"`
	ReplyToAddress string `dynamo:"reply_to_address" json:"reply_to_address"`
	Feeds          []Feed `dynamo:"feeds" json:"feeds"`
}

type Feed struct {
	Url            string    `dynamo:"url" json:"url"`
	LastUpdated    time.Time `dynamo:"last_updated,unixtime" json:"-"`
	ProcessedGuids []string  `dynamo:"processed_guids" json:"processed_guids"`
}

func (lst *List) FormatBaseURL() string {
	return fmt.Sprintf("https://%v", lst.Domain)
}

func (lst *List) FormatUnsubscribeLink(sub subscription.Subscription) string {
	return fmt.Sprintf("%v/unsubscribe?email=%v", lst.FormatBaseURL(), sub.Email)
}

func (lst *List) FormatVerificationLink(sub subscription.Subscription) string {
	return fmt.Sprintf("%v/verify?token=%v", lst.FormatBaseURL(), sub.VerificationToken)
}

func Get(name string) (*List, error) {
	var lst List
	err := table.Get("name", name).One(&lst)
	if err != nil {
		return nil, err
	}
	return &lst, nil
}

func GetFromDomain(domain string) (*List, error) {
	var lsts []*List
	err := table.Scan().Index("domain").Filter("'domain' = ?", domain).All(&lsts)
	if err != nil {
		return nil, err
	}
	if len(lsts) == 0 {
		return nil, ERR_LIST_NOT_FOUND
	}
	if len(lsts) > 1 {
		return nil, ERR_LIST_DOMAIN_DUPLICATED
	}
	return lsts[0], nil
}

func GetAll() (*[]*List, error) {
	var lsts []*List
	err := table.Scan().All(&lsts)
	if err != nil {
		return nil, err
	}
	if len(lsts) == 0 {
		return nil, ERR_LIST_NOT_FOUND
	}
	return &lsts, nil
}

func (lst *List) update() *dynamo.Update {
	return table.Update("name", lst.Name)
}

func (lst *List) UpdateFeedLastUpdated(feedIndex int) error {
	return lst.update().Set(fmt.Sprintf("feeds[%v].last_updated", feedIndex), time.Now().Unix()).Run()
}

func (lst *List) UpdateProcessedGuids(feedIndex int, guid string) error {
	return lst.update().Append(fmt.Sprintf("feeds[%v].processed_guids", feedIndex), []string{guid}).Run()
}
