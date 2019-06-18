package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
	raven "github.com/getsentry/raven-go"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/Gateway/pkg/whitelist"
	"gitlab.com/Cacophony/go-kit/logging"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

func NewSession(
	logger *zap.Logger,
	token string,
	eventHandler *handler.EventHandler,
	state *state.State,
	checker *whitelist.Checker,
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
	discordSession.SyncEvents = true

	discordSession.AddHandler(func(session *discordgo.Session, eventItem interface{}) {
		err := state.SharedStateEventHandler(session, eventItem)
		if err != nil {
			raven.CaptureError(err, nil)
			logger.Error("state client failed to handle event", zap.Error(err))
		}
	})
	discordSession.AddHandler(eventHandler.OnDiscordEvent)

	// start discord session
	err = discordSession.Open()
	if err != nil {
		logger.Fatal("unable to start discord session",
			zap.Error(err),
		)
	}

	logger.Info("connected Bot to Discord Gateway")

	go func() {
		time.Sleep(5 * time.Minute)

		requestGuildMembers(
			discordSession,
			state,
			checker,
			logger.With(zap.String("feature", "members")),
		)
	}()

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
