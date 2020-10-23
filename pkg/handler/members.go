package handler

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func (eh *EventHandler) requestGuildMembers(session *discordgo.Session, ready *discordgo.Ready) {
	// do not request members on reconnect
	eh.requestOnce.Do(func() {
		time.Sleep(eh.requestGuildMembersDelay)

		guildIDs := make([]string, 0, len(ready.Guilds))
		var blacklisted bool

		for _, guild := range ready.Guilds {
			blacklisted = eh.checker.IsBlacklisted(guild.ID)
			if blacklisted {
				continue
			}

			requestedBotID, err := eh.state.BotForGuild(guild.ID)
			if err != nil {
				continue
			}

			if requestedBotID != ready.User.ID {
				continue
			}

			guildIDs = append(guildIDs, guild.ID)
		}

		eh.logger.Info("requesting members for guilds",
			zap.Int("count", len(guildIDs)),
			zap.String("bot_id", session.State.User.ID),
		)

		err := session.RequestGuildMembersBatch(guildIDs, "", 0, false)
		if err != nil {
			eh.logger.Error("failure requesting guild members", zap.Error(err), zap.String("bot_id", session.State.User.ID))
		}
	})
}
