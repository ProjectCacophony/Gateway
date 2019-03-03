package main

import (
	"time"

	"gitlab.com/Cacophony/go-kit/errortracking"

	"gitlab.com/Cacophony/go-kit/logging"
)

type config struct {
	Port                  int                  `envconfig:"PORT" default:"8000"`
	Hash                  string               `envconfig:"HASH"`
	Environment           logging.Environment  `envconfig:"ENVIRONMENT" default:"development"`
	ClusterEnvironment    string               `envconfig:"CLUSTER_ENVIRONMENT" default:"development"`
	DiscordTokens         map[string]string    `envconfig:"DISCORD_TOKENS"`
	AMQPDSN               string               `envconfig:"AMQP_DSN" default:"amqp://guest:guest@localhost:5672/"`
	LoggingDiscordWebhook string               `envconfig:"LOGGING_DISCORD_WEBHOOK"`
	EventTTL              time.Duration        `envconfig:"EVENT_TTL" default:"10m"`
	RedisAddress          string               `envconfig:"REDIS_ADDRESS" default:"localhost:6379"`
	RedisPassword         string               `envconfig:"REDIS_PASSWORD"`
	EnableWhitelist       bool                 `envconfig:"ENABLE_WHITELIST" default:"false"`
	ErrorTracking         errortracking.Config `envconfig:"ERRORTRACKING"`
}
