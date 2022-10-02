package main

import (
	"gjhr.me/newsletter/listmanagement"

	"github.com/apex/log"
)

func main() {
	err := listmanagement.Unsubscribe("gjhr.me", "george@gjhr.me")
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
