package main

import (
	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/go-kit/logging"
	"go.uber.org/zap"
)

func NewSession(
	logger *zap.Logger,
	token string,
	eventHandler *handler.EventHandler,
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

	// sets the necessary gateway intents https://discord.com/developers/docs/topics/gateway#gateway-intents
	discordSession.Identify.Intents = discordgo.MakeIntent(
		discordgo.IntentsAllWithoutPrivileged |
			discordgo.IntentsGuildMembers,
	)

	// start discord session
	err = discordSession.Open()
	if err != nil {
		logger.Fatal("unable to start discord session",
			zap.Error(err),
		)
	}

	logger.Info("connected Bot to Discord Gateway")

	err = discordSession.UpdateStatusComplex(discordgo.UpdateStatusData{
		Game: &discordgo.Game{
			Name: ".help",
			Type: discordgo.GameTypeGame,
		},
		Status: "online",
	})
	if err != nil {
		logger.Error("failure updating status", zap.Error(err))
	}

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
