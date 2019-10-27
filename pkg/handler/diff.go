package handler

import (
	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/go-kit/events"
)

func guildDiff(old, new *discordgo.Guild) (*events.Event, error) {
	if old == nil || new == nil {
		return nil, nil
	}

	event, err := events.New(events.CacophonyDiffGuild)
	if err != nil {
		return nil, err
	}
	event.GuildID = old.ID
	event.DiffGuild = &events.DiffGuild{
		Old: old,
		New: new,
	}

	return event, nil
}

func memberDiff(old, new *discordgo.Member) (*events.Event, error) {
	if old == nil || new == nil {
		return nil, nil
	}

	event, err := events.New(events.CacophonyDiffMember)
	if err != nil {
		return nil, err
	}
	event.GuildID = old.GuildID
	event.DiffMember = &events.DiffMember{
		Old: old,
		New: new,
	}

	return event, nil
}

func channelDiff(old, new *discordgo.Channel) (*events.Event, error) {
	if old == nil {
		return nil, nil
	}

	event, err := events.New(events.CacophonyDiffChannel)
	if err != nil {
		return nil, err
	}
	event.GuildID = old.GuildID
	event.DiffChannel = &events.DiffChannel{
		Old: old,
		New: new,
	}

	return event, nil
}

func roleDiff(guildID string, old, new *discordgo.Role) (*events.Event, error) {
	if old == nil {
		return nil, nil
	}

	event, err := events.New(events.CacophonyDiffRole)
	if err != nil {
		return nil, err
	}
	event.GuildID = guildID
	event.DiffRole = &events.DiffRole{
		Old: old,
		New: new,
	}

	return event, nil
}

func emojiDiff(guildID string, old, new []*discordgo.Emoji) (*events.Event, error) {
	event, err := events.New(events.CacophonyDiffEmoji)
	if err != nil {
		return nil, err
	}
	event.GuildID = guildID
	event.DiffEmoji = &events.DiffEmoji{
		Old: old,
		New: new,
	}

	return event, nil
}

func webhooksDiff(guildID string, old, new []*discordgo.Webhook) (*events.Event, error) {
	event, err := events.New(events.CacophonyDiffWebhooks)
	if err != nil {
		return nil, err
	}
	event.GuildID = guildID
	event.DiffWebhooks = &events.DiffWebhooks{
		Old: old,
		New: new,
	}

	return event, nil
}

func invitesDiff(guildID string, old, new []*discordgo.Invite) (*events.Event, error) {
	event, err := events.New(events.CacophonyDiffInvites)
	if err != nil {
		return nil, err
	}
	event.GuildID = guildID
	event.DiffInvites = &events.DiffInvites{
		Old: old,
		New: new,
	}

	return event, nil
}

func inviteSliceFindInvite(code string, list []*discordgo.Invite) *discordgo.Invite {
	for _, invite := range list {
		if invite.Code == code {
			return invite
		}
	}

	return nil
}

func inviteDiffFindUsed(diff *events.DiffInvites) *discordgo.Invite {
	var matches int
	var match *discordgo.Invite

	for _, newInvite := range diff.New {
		oldInvite := inviteSliceFindInvite(newInvite.Code, diff.Old)
		if oldInvite == nil {
			continue
		}

		if newInvite.Uses == oldInvite.Uses+1 {
			matches++
			match = newInvite
		}
		if matches > 1 {
			return nil
		}
	}

	return match
}
