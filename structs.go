package main

import "github.com/bwmarrin/discordgo"

type DMessageCreateEvent struct {
	Event   *discordgo.MessageCreate
	BotUser *discordgo.User
}
