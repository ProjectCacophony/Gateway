package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Gateway/pkg/whitelist"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

// TODO: more sophisticated logic, do not request all members multiple times

func requestGuildMembers(
	session *discordgo.Session,
	state *state.State,
	checker *whitelist.Checker,
	logger *zap.Logger,
) {
	var blacklisted bool

	for _, guildID := range checker.GetWhitelist() {
		blacklisted = checker.IsBlacklisted(guildID)
		if blacklisted {
			continue
		}

		_, err := state.Guild(guildID)
		if err != nil {
			continue
		}

		requestedBotID, err := state.BotForGuild(guildID)
		if err != nil {
			continue
		}

		if requestedBotID != session.State.User.ID {
			continue
		}

		l := logger.With(
			zap.String("guild_id", guildID),
			zap.String("bot_id", guildID),
		)

		l.Info("requesting guild members")

		err = session.RequestGuildMembers(guildID, "", 0)
		if err != nil {
			l.Error("failure requesting guild members", zap.Error(err))
		}

		time.Sleep(5 * time.Second)
	}
}
