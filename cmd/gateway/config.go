package main

import (
	"time"

	"gitlab.com/Cacophony/go-kit/logging"
)

type config struct {
	Environment           logging.Environment `envconfig:"ENVIRONMENT" default:"development"`
	DiscordToken          string              `envconfig:"DISCORD_TOKEN"`
	AMQPDSN               string              `envconfig:"AMQP_DSN" default:"amqp://guest:guest@localhost:5672/"`
	LoggingDiscordWebhook string              `envconfig:"LOGGING_DISCORD_WEBHOOK"`
	EventTTL              time.Duration       `envconfig:"EVENT_TTL" default:"10m"`
}
