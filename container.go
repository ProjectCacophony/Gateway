package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
	"gitlab.com/project-d-collab/dhelpers/state"
)

func createEventContainer(receivedAt time.Time, session *discordgo.Session, eventKey string, i interface{}) (container dhelpers.EventContainer) {
	// create enhanced Event
	dDEvent := dhelpers.EventContainer{
		ReceivedAt:     receivedAt,
		GatewayStarted: started,
		Key:            eventKey,
		BotUserID:      session.State.User.ID,
	}

	switch t := i.(type) {
	case *discordgo.GuildCreate:
		dDEvent.Type = dhelpers.GuildCreateEventType
		dDEvent.GuildCreate = t
	case *discordgo.GuildUpdate:
		dDEvent.Type = dhelpers.GuildUpdateEventType
		dDEvent.GuildUpdate = t
	case *discordgo.GuildDelete:
		dDEvent.Type = dhelpers.GuildDeleteEventType
		dDEvent.GuildDelete = t
	case *discordgo.GuildMemberAdd:
		dDEvent.Type = dhelpers.GuildMemberAddEventType
		dDEvent.GuildMemberAdd = t
	case *discordgo.GuildMemberUpdate:
		dDEvent.Type = dhelpers.GuildMemberUpdateEventType
		dDEvent.GuildMemberUpdate = t
	case *discordgo.GuildMemberRemove:
		dDEvent.Type = dhelpers.GuildMemberRemoveEventType
		dDEvent.GuildMemberRemove = t
	case *discordgo.GuildMembersChunk:
		dDEvent.Type = dhelpers.GuildMembersChunkEventType
		dDEvent.GuildMembersChunk = t
	case *discordgo.GuildRoleCreate:
		dDEvent.Type = dhelpers.GuildRoleCreateEventType
		dDEvent.GuildRoleCreate = t
	case *discordgo.GuildRoleUpdate:
		dDEvent.Type = dhelpers.GuildRoleUpdateEventType
		dDEvent.GuildRoleUpdate = t
	case *discordgo.GuildRoleDelete:
		dDEvent.Type = dhelpers.GuildRoleDeleteEventType
		dDEvent.GuildRoleDelete = t
	case *discordgo.GuildEmojisUpdate:
		dDEvent.Type = dhelpers.GuildEmojisUpdateEventType
		dDEvent.GuildEmojisUpdate = t
	case *discordgo.ChannelCreate:
		dDEvent.Type = dhelpers.ChannelCreateEventType
		dDEvent.ChannelCreate = t
	case *discordgo.ChannelUpdate:
		dDEvent.Type = dhelpers.ChannelUpdateEventType
		dDEvent.ChannelUpdate = t
	case *discordgo.ChannelDelete:
		dDEvent.Type = dhelpers.ChannelDeleteEventType
		dDEvent.ChannelDelete = t
	case *discordgo.MessageCreate:
		dDEvent.Type = dhelpers.MessageCreateEventType
		// args and prefix
		var guildID string
		channel, err := state.Channel(t.ChannelID)
		if err == nil {
			guildID = channel.GuildID
		} else {
			cache.GetLogger().Errorln("error getting channel #", t.ChannelID+":", err.Error())
		}
		prefixes := dhelpers.GetPrefix(prefixConfig, dDEvent.BotUserID, guildID)
		args, prefix := dhelpers.GetMessageArguments(t.Content, prefixes)
		dDEvent.Args = args
		dDEvent.Prefix = prefix
		dDEvent.MessageCreate = t
	case *discordgo.MessageUpdate:
		dDEvent.Type = dhelpers.MessageUpdateEventType
		// args and prefix
		var guildID string
		channel, err := state.Channel(t.ChannelID)
		if err == nil {
			guildID = channel.GuildID
		} else {
			cache.GetLogger().Errorln("error getting channel #", t.ChannelID+":", err.Error())
		}
		prefixes := dhelpers.GetPrefix(prefixConfig, dDEvent.BotUserID, guildID)
		args, prefix := dhelpers.GetMessageArguments(t.Content, prefixes)
		dDEvent.Args = args
		dDEvent.Prefix = prefix
		dDEvent.MessageUpdate = t
	case *discordgo.MessageDelete:
		dDEvent.Type = dhelpers.MessageDeleteEventType
		dDEvent.MessageDelete = t
	case *discordgo.ChannelPinsUpdate:
		dDEvent.ChannelPinsUpdate = t
	case *discordgo.GuildBanAdd:
		dDEvent.GuildBanAdd = t
	case *discordgo.GuildBanRemove:
		dDEvent.GuildBanRemove = t
	case *discordgo.MessageReactionAdd:
		dDEvent.MessageReactionAdd = t
	case *discordgo.MessageReactionRemove:
		dDEvent.MessageReactionRemove = t
	case *discordgo.MessageReactionRemoveAll:
		dDEvent.MessageReactionRemoveAll = t
	}

	return dDEvent
}
