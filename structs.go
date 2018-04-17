package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type DMessageCreateEvent struct {
	Event             *discordgo.MessageCreate
	BotUser           *discordgo.User
	GatewayReceivedAt time.Time
}
