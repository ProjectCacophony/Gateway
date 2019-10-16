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
