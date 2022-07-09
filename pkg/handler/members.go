package handler

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var (
	requestedGuildMembers     = map[string]bool{}
	requestedGuildMembersLock sync.Mutex
)

func shouldRequestMembers(botID string) bool {
	requestedGuildMembersLock.Lock()
	defer requestedGuildMembersLock.Unlock()

	if requestedGuildMembers[botID] {
		return false
	}

	requestedGuildMembers[botID] = true
	return true
}

func (eh *EventHandler) requestGuildMembers(session *discordgo.Session, ready *discordgo.Ready) {
	if !shouldRequestMembers(session.State.User.ID) {
		return
	}

	time.Sleep(eh.requestGuildMembersDelay)

	for _, guild := range ready.Guilds {
		if eh.checker.IsBlacklisted(guild.ID) {
			continue
		}

		requestedBotID, err := eh.state.BotForGuild(guild.ID)
		if err != nil {
			continue
		}

		if requestedBotID != ready.User.ID {
			continue
		}

		eh.logger.Info("requesting members for guild",
			zap.String("guild_id", guild.ID),
			zap.String("bot_id", session.State.User.ID),
		)

		err = session.RequestGuildMembers(guild.ID, "", 0, "", false)
		if err != nil {
			eh.logger.Error("failure requesting guild members", zap.Error(err), zap.String("bot_id", session.State.User.ID))
		}

		time.Sleep(1 * time.Second)
	}
}
