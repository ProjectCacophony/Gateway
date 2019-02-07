package main

type config struct {
	DiscordToken string `envconfig:"DISCORD_TOKEN"`
	AMQPDSN      string `envconfig:"AMQP_DSN" default:"amqp://guest:guest@localhost:5672/"`
}
