package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type DDiscordEvent struct {
	Type              EventType
	Prefix            string
	Event             interface{}
	BotUser           *discordgo.User
	SourceChannel     *discordgo.Channel
	SourceGuild       *discordgo.Guild
	GatewayReceivedAt time.Time
}
