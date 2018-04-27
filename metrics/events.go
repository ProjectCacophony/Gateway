package metrics

import (
	"expvar"

	"github.com/bwmarrin/discordgo"
)

var (
	// EventsDiscarded counts all events discarded because they have been handled by different gateways
	EventsDiscarded = expvar.NewInt("events_discarded")
	// EventsGuildCreate counts the events of the specific type
	EventsGuildCreate = expvar.NewInt("events_guildcreate")
	// EventsGuildUpdate counts the events of the specific type
	EventsGuildUpdate = expvar.NewInt("events_guildupdate")
	// EventsGuildDelete counts the events of the specific type
	EventsGuildDelete = expvar.NewInt("events_guilddelete")
	// EventsGuildMemberAdd counts the events of the specific type
	EventsGuildMemberAdd = expvar.NewInt("events_guildmemberadd")
	// EventsGuildMemberUpdate counts the events of the specific type
	EventsGuildMemberUpdate = expvar.NewInt("events_guildmemberupdate")
	// EventsGuildMemberRemove counts the events of the specific type
	EventsGuildMemberRemove = expvar.NewInt("events_guildmemberremove")
	// EventsGuildMembersChunk counts the events of the specific type
	EventsGuildMembersChunk = expvar.NewInt("events_guildmemberschunk")
	// EventsGuildRoleCreate counts the events of the specific type
	EventsGuildRoleCreate = expvar.NewInt("events_guildrolecreate")
	// EventsGuildRoleUpdate counts the events of the specific type
	EventsGuildRoleUpdate = expvar.NewInt("events_guildroleupdate")
	// EventsGuildRoleDelete counts the events of the specific type
	EventsGuildRoleDelete = expvar.NewInt("events_guildroledelete")
	// EventsGuildEmojisUpdate counts the events of the specific type
	EventsGuildEmojisUpdate = expvar.NewInt("events_guildemojisupdate")
	// EventsChannelCreate counts the events of the specific type
	EventsChannelCreate = expvar.NewInt("events_channelcreate")
	// EventsChannelUpdate counts the events of the specific type
	EventsChannelUpdate = expvar.NewInt("events_channelupdate")
	// EventsChannelDelete counts the events of the specific type
	EventsChannelDelete = expvar.NewInt("events_channeldelete")
	// EventsMessageCreate counts the events of the specific type
	EventsMessageCreate = expvar.NewInt("events_messagecreate")
	// EventsMessageUpdate counts the events of the specific type
	EventsMessageUpdate = expvar.NewInt("events_messageupdate")
	// EventsMessageDelete counts the events of the specific type
	EventsMessageDelete = expvar.NewInt("events_messagedelete")
	// EventsPresenceUpdate counts the events of the specific type
	EventsPresenceUpdate = expvar.NewInt("events_presenceupdate")
	// EventsChannelPinsUpdate counts the events of the specific type
	EventsChannelPinsUpdate = expvar.NewInt("events_channelpinsupdate")
	// EventsGuildBanAdd counts the events of the specific type
	EventsGuildBanAdd = expvar.NewInt("events_guildbanadd")
	// EventsGuildBanRemove counts the events of the specific type
	EventsGuildBanRemove = expvar.NewInt("events_guildbanremove")
	// EventsMessageReactionAdd counts the events of the specific type
	EventsMessageReactionAdd = expvar.NewInt("events_messagecreationadd")
	// EventsMessageReactionRemove counts the events of the specific type
	EventsMessageReactionRemove = expvar.NewInt("events_messagereactionremove")
	// EventsMessageReactionRemoveAll counts the events of the specific type
	EventsMessageReactionRemoveAll = expvar.NewInt("events_messagereactionremoveall")
)

// CountEvent counts all events
func CountEvent(i interface{}) {
	switch i.(type) {
	case *discordgo.GuildCreate:
		EventsGuildCreate.Add(1)
	case *discordgo.GuildUpdate:
		EventsGuildUpdate.Add(1)
	case *discordgo.GuildDelete:
		EventsGuildDelete.Add(1)
	case *discordgo.GuildMemberAdd:
		EventsGuildMemberAdd.Add(1)
	case *discordgo.GuildMemberUpdate:
		EventsGuildMemberUpdate.Add(1)
	case *discordgo.GuildMemberRemove:
		EventsGuildMemberRemove.Add(1)
	case *discordgo.GuildMembersChunk:
		EventsGuildMembersChunk.Add(1)
	case *discordgo.GuildRoleCreate:
		EventsGuildRoleCreate.Add(1)
	case *discordgo.GuildRoleUpdate:
		EventsGuildRoleUpdate.Add(1)
	case *discordgo.GuildRoleDelete:
		EventsGuildRoleDelete.Add(1)
	case *discordgo.GuildEmojisUpdate:
		EventsGuildEmojisUpdate.Add(1)
	case *discordgo.ChannelCreate:
		EventsChannelCreate.Add(1)
	case *discordgo.ChannelUpdate:
		EventsChannelUpdate.Add(1)
	case *discordgo.ChannelDelete:
		EventsChannelDelete.Add(1)
	case *discordgo.MessageCreate:
		EventsMessageCreate.Add(1)
	case *discordgo.MessageUpdate:
		EventsMessageUpdate.Add(1)
	case *discordgo.MessageDelete:
		EventsMessageDelete.Add(1)
	case *discordgo.PresenceUpdate:
		EventsPresenceUpdate.Add(1)
	case *discordgo.ChannelPinsUpdate:
		EventsChannelPinsUpdate.Add(1)
	case *discordgo.GuildBanAdd:
		EventsGuildBanAdd.Add(1)
	case *discordgo.GuildBanRemove:
		EventsGuildBanRemove.Add(1)
	case *discordgo.MessageReactionAdd:
		EventsMessageReactionAdd.Add(1)
	case *discordgo.MessageReactionRemove:
		EventsMessageReactionRemove.Add(1)
	case *discordgo.MessageReactionRemoveAll:
		EventsMessageReactionRemoveAll.Add(1)
	}
}
