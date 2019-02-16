package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/Gateway/pkg/publisher"
	"gitlab.com/Cacophony/go-kit/api"
	"gitlab.com/Cacophony/go-kit/logging"
	"gitlab.com/Cacophony/go-kit/state"
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
		panic(errors.Wrap(err, "unable to initialise logger"))
	}
	defer logger.Sync() // nolint: errcheck

	// init redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		logger.Fatal("unable to connect to Redis",
			zap.Error(err),
		)
	}

	// init AMQP session
	amqpConnection, err := amqp.Dial(config.AMQPDSN)
	if err != nil {
		logger.Fatal("unable to initialise AMQP session",
			zap.Error(err),
		)
	}

	// init state
	stateClient := state.NewSate(redisClient)

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

	// init http server
	httpRouter := api.NewRouter()
	httpServer := api.NewHTTPServer(config.Port, httpRouter)

	go func() {
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Fatal("http server error",
				zap.Error(err),
				zap.String("feature", "http-server"),
			)
		}
	}()

	// launch all sessions:
	discordCloseChannel := make(chan interface{}, len(config.DiscordTokens))
	for botID, token := range config.DiscordTokens {
		NewSession(
			logger.With(zap.String("bot_id", botID)),
			token,
			eventHandler,
			stateClient,
			discordCloseChannel,
		)
	}

	logger.Info("service is running",
		zap.Int("port", config.Port),
	)

	// wait for CTRL+C to stop the service
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-quitChannel

	// shutdown features

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// tell all sessions to close
	for i := 0; i < cap(discordCloseChannel); i++ {
		discordCloseChannel <- nil
	}
	// wait for all discord channels to close
	for i := 0; i < cap(discordCloseChannel); i++ {
		<-discordCloseChannel
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
