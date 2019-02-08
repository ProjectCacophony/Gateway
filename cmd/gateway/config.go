package main

import (
	"gitlab.com/Cacophony/Gateway/pkg/kit/logging"
)

type config struct {
	Environment  logging.Environment `envconfig:"ENVIRONMENT" default:"development"`
	DiscordToken string              `envconfig:"DISCORD_TOKEN"`
	AMQPDSN      string              `envconfig:"AMQP_DSN" default:"amqp://guest:guest@localhost:5672/"`
}
