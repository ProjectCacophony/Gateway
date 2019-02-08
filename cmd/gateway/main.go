package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/Gateway/pkg/publisher"
	"gitlab.com/Cacophony/go-kit/api"
	"gitlab.com/Cacophony/go-kit/logging"
	"go.uber.org/zap"
)

const (
	// ServiceName is the name of the service
	ServiceName = "gateway"
)

func main() {
	// init config
	var config config
	err := envconfig.Process("", &config)
	if err != nil {
		panic(errors.Wrap(err, "unable to load configuration"))
	}

	// init logger
	logger, err := logging.NewLogger(
		config.Environment,
		ServiceName,
		config.LoggingDiscordWebhook,
		&http.Client{
			Timeout: 10 * time.Second,
		},
	)
	if err != nil {
		panic(errors.Wrap(err, "unable to initialise launcher"))
	}

	// init discordgo session
	discordgo.Logger = logging.DiscordgoLogger(
		logger.With(zap.String("feature", "discordgo")),
	)

	discordSession, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		logger.Fatal("unable to initialise discord session",
			zap.Error(err),
		)
	}
	discordSession.LogLevel = discordgo.LogInformational
	discordSession.StateEnabled = false

	// init AMQP session
	amqpConnection, err := amqp.Dial(config.AMQPDSN)
	if err != nil {
		logger.Fatal("unable to initialise AMQP session",
			zap.Error(err),
		)
	}

	// init publisher
	publisherClient, err := publisher.NewPublisher(
		amqpConnection,
		"cacophony",
		config.EventTTL,
	)
	if err != nil {
		logger.Fatal("unable to initialise Publisher",
			zap.Error(err),
		)
	}

	// init event handler
	eventHandler := handler.NewEventHandler(
		logger.With(zap.String("feature", "EventHandler")),
		publisherClient,
	)

	discordSession.AddHandler(eventHandler.OnDiscordEvent)

	// init http server
	httpRouter := api.NewRouter()
	httpServer := api.NewHTTPServer(config.Port, httpRouter)

	// start discord session
	err = discordSession.Open()
	if err != nil {
		logger.Fatal("unable to start discord session",
			zap.Error(err),
		)
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Fatal("http server error",
				zap.Error(err),
				zap.String("feature", "http-server"),
			)
		}
	}()

	logger.Info("service is running",
		zap.String(
			"discord user",
			fmt.Sprintf("%s (#%s)", discordSession.State.User.String(), discordSession.State.User.ID),
		),
		zap.Int("port", config.Port),
	)

	// wait for CTRL+C to stop the service
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-quitChannel

	// shutdown features

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	err = discordSession.Close()
	if err != nil {
		logger.Error("unable to close discord session",
			zap.Error(err),
		)
	}

	err = amqpConnection.Close()
	if err != nil {
		logger.Error("unable to close AMQP session",
			zap.Error(err),
		)
	}

	err = httpServer.Shutdown(ctx)
	if err != nil {
		logger.Error("unable to shutdown HTTP Server",
			zap.Error(err),
		)
	}
}
