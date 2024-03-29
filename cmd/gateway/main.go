package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/honeycombio/opentelemetry-exporter-go/honeycomb"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Gateway/pkg/handler"
	"gitlab.com/Cacophony/Gateway/pkg/whitelist"
	"gitlab.com/Cacophony/go-kit/api"
	"gitlab.com/Cacophony/go-kit/discord"
	"gitlab.com/Cacophony/go-kit/errortracking"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/logging"
	"gitlab.com/Cacophony/go-kit/state"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	config.ErrorTracking.Version = config.Hash
	config.ErrorTracking.Environment = config.ClusterEnvironment

	discord.SetAPIBase(config.DiscordAPIBase)

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

	// init tracing
	if config.HoneycombAPIKey != "" {
		honeycombExporter, err := honeycomb.NewExporter(
			honeycomb.Config{
				APIKey: config.HoneycombAPIKey,
			},
			honeycomb.TargetingDataset(config.ClusterEnvironment),
			honeycomb.WithServiceName(ServiceName),
			// honeycomb.WithDebugEnabled(),
		)
		if err != nil {
			logger.Fatal("failure initialising honeycomb exporter", zap.Error(err))
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			honeycombExporter.Shutdown(ctx) // nolint: errcheck
			cancel()
		}()

		provider := sdktrace.NewTracerProvider(
			sdktrace.WithConfig(sdktrace.Config{
				DefaultSampler: sdktrace.ParentBased(
					sdktrace.TraceIDRatioBased(0.01),
					sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),
					sdktrace.WithRemoteParentNotSampled(sdktrace.TraceIDRatioBased(0.01)),
					sdktrace.WithLocalParentSampled(sdktrace.AlwaysSample()),
					sdktrace.WithLocalParentNotSampled(sdktrace.TraceIDRatioBased(0.01)),
				),
			}),
			sdktrace.WithSyncer(honeycombExporter),
		)
		global.SetTracerProvider(provider)

		b3Prop := b3.B3{}
		global.SetPropagators(propagation.New(
			propagation.WithExtractors(b3Prop),
			propagation.WithInjectors(b3Prop),
		))
	}

	// init raven
	err = errortracking.Init(&config.ErrorTracking)
	if err != nil {
		logger.Error("unable to initialise errortracking",
			zap.Error(err),
		)
	}

	// init redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:        config.RedisAddress,
		Password:    config.RedisPassword,
		ReadTimeout: time.Second * 15,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		logger.Fatal("unable to connect to Redis",
			zap.Error(err),
		)
	}

	// init whitelist checker
	checker := whitelist.NewChecker(
		redisClient,
		logger,
		time.Minute,
		config.EnableWhitelist,
	)
	err = checker.Start()
	if err != nil {
		logger.Fatal("unable to initialise whitelist checker",
			zap.Error(err),
		)
	}

	// init state
	stateClient := state.NewState(redisClient, nil)

	// init publisher
	publisher, err := events.NewPublisher(
		config.AMQPDSN, nil,
	)
	if err != nil {
		logger.Fatal("unable to initialise Publisher",
			zap.Error(err),
		)
	}

	// init event handler
	eventHandler := handler.NewEventHandler(
		logger.With(zap.String("feature", "EventHandler")),
		redisClient,
		publisher,
		checker,
		stateClient,
		config.Deduplicate,
		config.RequestMembersDelay,
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
			discordCloseChannel,
		)
	}

	logger.Info("service is running",
		zap.Int("port", config.Port),
		zap.Bool("whitelist_enabled", config.EnableWhitelist),
		zap.Bool("deduplicate", config.Deduplicate),
		zap.Duration("request_members_delay", config.RequestMembersDelay),
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

	err = httpServer.Shutdown(ctx)
	if err != nil {
		logger.Error("unable to shutdown HTTP Server",
			zap.Error(err),
		)
	}
}
