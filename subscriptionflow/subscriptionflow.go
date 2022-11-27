package listmanagement

import (
	"errors"
	"fmt"
	"html/template"
	"net/mail"
	"time"

	"github.com/apex/log"
	"github.com/google/uuid"
	"gjhr.me/newsletter/data/list"
	"gjhr.me/newsletter/data/subscription"
	"gjhr.me/newsletter/emailsender"
)

var ERR_UNEXPECTED = errors.New("An unexpected error has occurred.")

// Subscription errors
// todo constantise
var ERR_INVALID_EMAIL = errors.New("Invalid email address.")
var ERR_RECENTLY_SENT_VERIFICATION = errors.New("A verification email for this subscription has recently been sent.")
var ERR_ALREADY_VERIFIED = errors.New("Subscription already verified.")
var ERR_SUBSCRIPTION_NOT_FOUND = errors.New("Subscription does not exist.")

func Subscribe(list *list.List, email string) (*subscription.Subscription, error) {
	log.Infof("Subscribing '%v' to list '%v'...", email, list.Name)
	// Validate email
	validAddress, err := mail.ParseAddress(email)
	if err != nil {
		return nil, ERR_INVALID_EMAIL
	}
	email = validAddress.Address

	sub, err := subscription.Get(list.Name, email)
	if err != nil {
		return nil, err
	}

	if sub != nil {
		return sub, resendVerificationEmail(*sub, list)
	}

	// Generate verification token
	uuid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// Save row to Dynamodb table
	sub = &subscription.Subscription{
		Email:                email,
		List:                 list.Name,
		VerificationToken:    uuid.String(),
		LastSentVerification: time.Now(),
	}
	err = subscription.Put(sub)
	if err != nil {
		return nil, err
	}

	// Send verification email
	return sub, sendVerificationEmail(*sub, list)
}

func resendVerificationEmail(sub subscription.Subscription, l *list.List) error {
	log.Infof("Resending verification email to '%v' for list '%v'...", sub.Email, sub.List)
	if sub.LastSentVerification.After(time.Now().Add(time.Minute * -15)) {
		return ERR_RECENTLY_SENT_VERIFICATION
	}
	err := sendVerificationEmail(sub, l)
	if err != nil {
		return err
	}

	return sub.UpdateLastSentVerification()
}

func sendVerificationEmail(sub subscription.Subscription, l *list.List) error {
	log.Infof("Sending verification email to '%v' for list '%v'...", sub.Email, sub.List)
	if sub.Verified != "" {
		return ERR_ALREADY_VERIFIED
	}

	t, err := template.New("verification-email").Parse(`
	<!DOCTYPE html>
	<html lang="en" xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:schemas-microsoft-com:office:office">
	<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width,initial-scale=1">
			<meta name="x-apple-disable-message-reformatting">
			<title></title>
			<style>
					body {font-family: Arial, sans-serif;}
			</style>
	</head>
	<body>
		<h3>Please verify your email</h3>
		<p>
			To complete your subscription to {{ .List }}, please click <a href="{{ .VerificationLink }}">this link</a> or browse to the URL below.
		</p>
		<p>
			{{ .VerificationLink }}
		</p>
	</body>
	</html>
	`)
	if err != nil {
		return err
	}

	err = emailsender.SendMail(sub.Email, l.FromAddress, l.ReplyToAddress, fmt.Sprintf("Verify email for %v", l.Name), t, struct{ List, VerificationLink string }{List: l.Name, VerificationLink: l.FormatVerificationLink(sub)})
	if err != nil {
		return err
	}

	return nil
}

func Verify(token string) error {
	log.Infof("Verifiying token '%v'...", token)
	// Set email as verified
	sub, err := subscription.GetFromToken(token)
	if err != nil {
		return ERR_SUBSCRIPTION_NOT_FOUND
	}
	log.Infof("Token '%v' mapped to email '%v' subscribed to list '%v'...", token, sub.Email, sub.List)

	if sub.Verified != "" {
		return ERR_ALREADY_VERIFIED
	}

	err = sub.Verify()
	if err != nil {
		return err
	}

	return nil
}

func Unsubscribe(list, email string) error {
	// Delete row from table
	log.Infof("Removing subscription of email '%v' to list '%v'...", email, list)

	sub, err := subscription.Get(list, email)
	if err != nil {
		return ERR_SUBSCRIPTION_NOT_FOUND
	}

	return sub.Delete()
}
