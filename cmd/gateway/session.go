package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/go-kit/logging"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

func NewSession(
	logger *zap.Logger,
	token string,
	eventHandler *handler.EventHandler,
	state *state.State,
	closeChannel chan interface{},
) {
	// init discordgo session
	discordgo.Logger = logging.DiscordgoLogger(
		logger.With(zap.String("feature", "discordgo")),
	)

	discordSession, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Fatal("unable to initialise discord session",
			zap.Error(err),
		)
	}
	discordSession.LogLevel = discordgo.LogInformational
	discordSession.StateEnabled = false

	discordSession.AddHandler(eventHandler.OnDiscordEvent)
	discordSession.AddHandler(func(session *discordgo.Session, eventItem interface{}) {
		err := state.SharedStateEventHandler(session, eventItem)
		if err != nil {
			logger.Error("state client failed to handle event", zap.Error(err))
		}
	})

	// start discord session
	err = discordSession.Open()
	if err != nil {
		logger.Fatal("unable to start discord session",
			zap.Error(err),
		)
	}

	logger.Info("connected Bot to Discord Gateway",
		zap.String(
			"discord user",
			fmt.Sprintf("%s (#%s)", discordSession.State.User.String(), discordSession.State.User.ID),
		),
	)

	go func() {
		<-closeChannel
		err := discordSession.Close()
		if err != nil {
			logger.Fatal("unable to close discord session",
				zap.Error(err),
			)
		}
		closeChannel <- true
	}()
}
