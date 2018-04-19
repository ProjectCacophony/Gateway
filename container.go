package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/project-d-collab/dhelpers"
)

func createEventContainer(receivedAt time.Time, session *discordgo.Session, eventKey string, i interface{}) (container dhelpers.EventContainer) {
	// create enhanced Event
	dDEvent := dhelpers.EventContainer{
		ReceivedAt:     receivedAt,
		GatewayStarted: started,
		Key:            eventKey,
	}

	switch t := i.(type) {
	case *discordgo.GuildCreate:
		dDEvent.Type = dhelpers.GuildCreateEventType
		dDEvent.GuildCreate = t
		// additional payload from state
		if t.Guild != nil {
			dDEvent.SourceGuild = t.Guild
		}
		// add additional state payload
		dDEvent.BotUser = session.State.User
	case *discordgo.GuildUpdate:
		dDEvent.Type = dhelpers.GuildUpdateEventType
		dDEvent.GuildUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
	case *discordgo.GuildDelete:
		dDEvent.Type = dhelpers.GuildDeleteEventType
		dDEvent.GuildDelete = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
	case *discordgo.GuildMemberAdd:
		dDEvent.Type = dhelpers.GuildMemberAddEventType
		dDEvent.GuildMemberAdd = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
	case *discordgo.GuildMemberUpdate:
		dDEvent.Type = dhelpers.GuildMemberUpdateEventType
		dDEvent.GuildMemberUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
	case *discordgo.GuildMemberRemove:
		dDEvent.Type = dhelpers.GuildMemberRemoveEventType
		dDEvent.GuildMemberRemove = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildMembersChunk:
		dDEvent.Type = dhelpers.GuildMembersChunkEventType
		dDEvent.GuildMembersChunk = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildRoleCreate:
		dDEvent.Type = dhelpers.GuildRoleCreateEventType
		dDEvent.GuildRoleCreate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildRoleUpdate:
		dDEvent.Type = dhelpers.GuildRoleUpdateEventType
		dDEvent.GuildRoleUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildRoleDelete:
		dDEvent.Type = dhelpers.GuildRoleDeleteEventType
		dDEvent.GuildRoleDelete = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildEmojisUpdate:
		dDEvent.Type = dhelpers.GuildEmojisUpdateEventType
		dDEvent.GuildEmojisUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.ChannelCreate:
		dDEvent.Type = dhelpers.ChannelCreateEventType
		dDEvent.ChannelCreate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
		if t.Channel != nil {
			dDEvent.SourceChannel = t.Channel
		}
	case *discordgo.ChannelUpdate:
		dDEvent.Type = dhelpers.ChannelUpdateEventType
		dDEvent.ChannelUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
		if t.Channel != nil {
			dDEvent.SourceChannel = t.Channel
		}
	case *discordgo.ChannelDelete:
		dDEvent.Type = dhelpers.ChannelDeleteEventType
		dDEvent.ChannelDelete = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
		if t.Channel != nil {
			dDEvent.SourceChannel = t.Channel
		}
	case *discordgo.MessageCreate:
		dDEvent.Type = dhelpers.MessageCreateEventType
		// args and prefix
		args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
		dDEvent.Args = args
		dDEvent.Prefix = prefix
		dDEvent.MessageCreate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.MessageUpdate:
		dDEvent.Type = dhelpers.MessageUpdateEventType
		// args and prefix
		args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
		dDEvent.Args = args
		dDEvent.Prefix = prefix
		dDEvent.MessageUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.MessageDelete:
		dDEvent.Type = dhelpers.MessageDeleteEventType
		// args and prefix
		args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
		dDEvent.Args = args
		dDEvent.Prefix = prefix
		dDEvent.MessageDelete = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.PresenceUpdate:
		dDEvent.Type = dhelpers.PresenceUpdateEventType
		dDEvent.PresenceUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.ChannelPinsUpdate:
		dDEvent.ChannelPinsUpdate = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.GuildBanAdd:
		dDEvent.GuildBanAdd = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.GuildBanRemove:
		dDEvent.GuildBanRemove = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.GuildID != "" {
			sourceGuild, err := session.State.Guild(t.GuildID)
			if err == nil {
				dDEvent.SourceGuild = sourceGuild
			}
		}
	case *discordgo.MessageReactionAdd:
		dDEvent.MessageReactionAdd = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.MessageReactionRemove:
		dDEvent.MessageReactionRemove = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	case *discordgo.MessageReactionRemoveAll:
		dDEvent.MessageReactionRemoveAll = t
		// additional payload from state
		dDEvent.BotUser = session.State.User
		if t.ChannelID != "" {
			sourceChannel, err := session.State.Channel(t.ChannelID)
			if err == nil {
				dDEvent.SourceChannel = sourceChannel
				sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
		}
	}

	return dDEvent
}
