package main

import (
	"time"

	"gitlab.com/Cacophony/go-kit/logging"
)

type config struct {
	Port                  int                 `envconfig:"PORT" default:"8000"`
	Environment           logging.Environment `envconfig:"ENVIRONMENT" default:"development"`
	DiscordTokens         map[string]string   `envconfig:"DISCORD_TOKENS"`
	AMQPDSN               string              `envconfig:"AMQP_DSN" default:"amqp://guest:guest@localhost:5672/"`
	LoggingDiscordWebhook string              `envconfig:"LOGGING_DISCORD_WEBHOOK"`
	EventTTL              time.Duration       `envconfig:"EVENT_TTL" default:"10m"`
	RedisAddress          string              `envconfig:"REDIS_ADDRESS" default:"localhost:6379"`
	RedisPassword         string              `envconfig:"REDIS_PASSWORD"`
}
