package config

import (
	"github.com/spf13/viper"
)

var conf Config

type Config struct {
	ListsTable         string
	SubscriptionsTable string
	LogLevel           string
}

func init() {
	viper.BindEnv("ListsTable", "NEWSLETTER_LISTS_TABLE")
	viper.BindEnv("SubscriptionsTable", "NEWSLETTER_SUBSCRIPTIONS_TABLE")
	viper.BindEnv("LogLevel", "NEWSLETTER_LOG_LEVEL")
	err := viper.Unmarshal(&conf)
	if err != nil {
		panic(err)
	}
}

func Get() Config {
	return conf
}
