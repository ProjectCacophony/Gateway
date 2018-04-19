package main

import (
	"fmt"

	"time"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/project-d-collab/dhelpers"
)

// unique key for every discord event for deduplication
func getEventKey(receivedAt time.Time, i interface{}) (key string) {
	switch t := i.(type) {
	case *discordgo.GuildCreate:
		return "project-d:gateway:event-" + string(dhelpers.GuildCreateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Guild))
	case *discordgo.GuildUpdate:
		return "project-d:gateway:event-" + string(dhelpers.GuildUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Guild))
	case *discordgo.GuildDelete:
		return "project-d:gateway:event-" + string(dhelpers.GuildDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Guild))
	case *discordgo.GuildMemberAdd:
		return "project-d:gateway:event-" + string(dhelpers.GuildDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Member))
	case *discordgo.GuildMemberUpdate:
		return "project-d:gateway:event-" + string(dhelpers.GuildDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Member))
	case *discordgo.GuildMemberRemove:
		return "project-d:gateway:event-" + string(dhelpers.GuildDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Member))
	case *discordgo.GuildMembersChunk:
		return "project-d:gateway:event-" + string(dhelpers.GuildMembersChunkEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%s %v", t.GuildID, t.Members))
	case *discordgo.GuildRoleCreate:
		return "project-d:gateway:event-" + string(dhelpers.GuildRoleCreateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.GuildRole))
	case *discordgo.GuildRoleUpdate:
		return "project-d:gateway:event-" + string(dhelpers.GuildRoleUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.GuildRole))
	case *discordgo.GuildRoleDelete:
		return "project-d:gateway:event-" + string(dhelpers.GuildRoleDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%s %s", t.RoleID, t.GuildID))
	case *discordgo.GuildEmojisUpdate:
		return "project-d:gateway:event-" + string(dhelpers.GuildEmojisUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%s %v", t.GuildID, t.Emojis))
	case *discordgo.ChannelCreate:
		return "project-d:gateway:event-" + string(dhelpers.ChannelCreateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Channel))
	case *discordgo.ChannelUpdate:
		return "project-d:gateway:event-" + string(dhelpers.ChannelUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Channel))
	case *discordgo.ChannelDelete:
		return "project-d:gateway:event-" + string(dhelpers.ChannelDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Channel))
	case *discordgo.MessageCreate:
		return "project-d:gateway:event-" + string(dhelpers.MessageCreateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Message))
	case *discordgo.MessageUpdate:
		return "project-d:gateway:event-" + string(dhelpers.MessageUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Message))
	case *discordgo.MessageDelete:
		return "project-d:gateway:event-" + string(dhelpers.MessageDeleteEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.Message))
	case *discordgo.PresenceUpdate:
		return "project-d:gateway:event-" + string(dhelpers.PresenceUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v %s %v", t.Presence, t.GuildID, t.Roles))
	case *discordgo.ChannelPinsUpdate:
		return "project-d:gateway:event-" + string(dhelpers.ChannelPinsUpdateEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%s %s", t.LastPinTimestamp, t.ChannelID))
	case *discordgo.GuildBanAdd:
		return "project-d:gateway:event-" + string(dhelpers.GuildBanAddEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v %s", t.User, t.GuildID))
	case *discordgo.GuildBanRemove:
		return "project-d:gateway:event-" + string(dhelpers.GuildBanRemoveEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v %s", t.User, t.GuildID))
	case *discordgo.MessageReactionAdd:
		return "project-d:gateway:event-" + string(dhelpers.MessageReactionAddEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.MessageReaction))
	case *discordgo.MessageReactionRemove:
		return "project-d:gateway:event-" + string(dhelpers.MessageReactionRemoveEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.MessageReaction))
	case *discordgo.MessageReactionRemoveAll:
		return "project-d:gateway:event-" + string(dhelpers.MessageReactionRemoveAllEventType) + "-" + dhelpers.GetMD5Hash(fmt.Sprintf("%v", t.MessageReaction))
	}
	return ""
}

// Returns true if the event key is new, returns false if the event key has already been handled by other gateways
func isNewEvent(eventKey string) (new bool) {
	set, err := redisClient.SetNX(eventKey, true, time.Minute*5).Result()
	if err != nil {
		fmt.Println("error doing deduplication:", err.Error())
		return false
	}

	return set
}
