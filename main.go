package main

import (
	"context"
	"encoding/binary"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sync"

	"github.com/Shopify/sarama"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/go-redis/redis"
	"github.com/json-iterator/go"
	"gitlab.com/Cacophony/Gateway/api"
	"gitlab.com/Cacophony/Gateway/metrics"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
	"gitlab.com/Cacophony/dhelpers/components"
	"gitlab.com/Cacophony/dhelpers/state"
)

var (
	routingConfig []dhelpers.RoutingRule
	started       time.Time
	didLaunch     bool
	redisClient   *redis.Client
	kafkaProducer sarama.SyncProducer
)

func main() {
	started = time.Now()
	var err error

	// init components
	components.InitMetrics()
	components.InitLogger("Gateway")
	err = components.InitSentry()
	dhelpers.CheckErr(err)
	components.InitRedis()
	err = components.InitDiscord()
	dhelpers.CheckErr(err)
	err = components.InitKafkaProducer()
	dhelpers.CheckErr(err)

	// get config
	routingConfig, err = dhelpers.GetRoutings()
	dhelpers.CheckErr(err)
	cache.GetLogger().Infoln("Found", len(routingConfig), "routing rules")
	// TODO: update routing at an interval

	// essentially only keep discordgo guild state
	cache.GetDiscord().State.TrackChannels = false
	cache.GetDiscord().State.TrackEmojis = false
	cache.GetDiscord().State.TrackMembers = false
	cache.GetDiscord().State.TrackPresences = false
	cache.GetDiscord().State.TrackRoles = false
	cache.GetDiscord().State.TrackVoice = false
	// add gateway ready handler
	cache.GetDiscord().AddHandler(onReady)
	// add the discord event handler
	cache.GetDiscord().AddHandler(eventHandler)

	// get cached clients
	redisClient = cache.GetRedisClient()
	kafkaProducer = cache.GetKafkaProducer()

	// open Discord Websocket connection
	err = cache.GetDiscord().Open()
	dhelpers.CheckErr(err)

	// start api server
	apiServer := &http.Server{
		Addr: os.Getenv("API_ADDRESS"),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      api.New(),
	}
	go func() {
		apiServerListenAndServeErr := apiServer.ListenAndServe()
		if err != nil && !strings.Contains(err.Error(), "http: Server closed") {
			cache.GetLogger().Fatal(apiServerListenAndServeErr)
		}
	}()
	cache.GetLogger().Infoln("started API on", os.Getenv("API_ADDRESS"))

	cache.GetLogger().Infoln("Gateway booting completed, took", time.Since(started).String())

	// wait for CTRL+C to stop the Bot
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// create sync.WaitGroup for all shutdown goroutines
	var exitGroup sync.WaitGroup

	// Discord Websocket disconnect goroutine
	exitGroup.Add(1)
	go func() {
		// disconnect from Discord Websocket
		cache.GetLogger().Infoln("Disconnecting from Discord bot gateway…")
		err = cache.GetDiscord().Close()
		dhelpers.LogError(err)
		cache.GetLogger().Infoln("Disconnected from Discord bot gateway")
		exitGroup.Done()
	}()

	// API Server shutdown goroutine
	exitGroup.Add(1)
	go func() {
		// shutdown api server
		cache.GetLogger().Infoln("Shutting API server down…")
		err = apiServer.Shutdown(context.Background())
		dhelpers.LogError(err)
		cache.GetLogger().Infoln("Shut API server down")
		exitGroup.Done()
	}()

	// Kafka Producer shutdown goroutine
	exitGroup.Add(1)
	go func() {
		// shutdown Kafka Producer
		cache.GetLogger().Infoln("Shutting Kafka Producer down…")
		err = cache.GetKafkaProducer().Close()
		dhelpers.LogError(err)
		cache.GetLogger().Infoln("Shut Kafka Producer down")
		exitGroup.Done()
	}()

	// wait for all shutdown goroutines to finish and then close channel finished
	finished := make(chan bool)
	go func() {
		exitGroup.Wait()
		close(finished)
	}()

	// wait 60 second for everything to finish, or shut it down anyway
	select {
	case <-finished:
		cache.GetLogger().Infoln("shutdown successful")
	case <-time.After(60 * time.Second):
		cache.GetLogger().Infoln("forcing shutdown after 60 seconds")
	}

}

func onReady(session *discordgo.Session, event *discordgo.Ready) {
	if !didLaunch {
		onFirstReady(session, event)
		didLaunch = true
	} else {
		onReconnect(session, event)
	}
}

func onFirstReady(session *discordgo.Session, event *discordgo.Ready) {
	for _, guild := range session.State.Guilds {
		if guild.Large {
			err := session.RequestGuildMembers(guild.ID, "", 0)
			if err != nil {
				cache.GetLogger().Errorln(err.Error())
			}
		}
	}
}

func onReconnect(session *discordgo.Session, event *discordgo.Ready) {
	for _, guild := range session.State.Guilds {
		if guild.Large {
			err := session.RequestGuildMembers(guild.ID, "", 0)
			if err != nil {
				cache.GetLogger().Errorln(err.Error())
			}
		}
	}
}

// discord event handler
func eventHandler(session *discordgo.Session, i interface{}) {
	receivedAt := time.Now()

	eventKey := dhelpers.GetEventKey(i)

	if eventKey == "" {
		return
	}

	if !dhelpers.IsNewEvent(redisClient, "gateway", eventKey) {
		cache.GetLogger().Infoln(eventKey+":", "ignored (handled by different gateway)")
		metrics.EventsDiscarded.Add(1)
		return
	}

	// update metrics
	metrics.CountEvent(i)

	// update shared state
	err := state.SharedStateEventHandler(session, i)
	if err != nil {
		cache.GetLogger().Errorln("state error:", err.Error())
	}

	eventContainer := dhelpers.CreateEventContainer(started, receivedAt, session, eventKey, i)

	if eventContainer.Type == "" {
		return
	}

	destinations := dhelpers.ContainerDestinations(
		session, routingConfig, eventContainer)
	eventContainer.Destinations = append(eventContainer.Destinations, destinations...)

	// pack the event data
	marshalled, err := jsoniter.Marshal(eventContainer)
	if err != nil {
		cache.GetLogger().Errorln(
			eventContainer.Key+":", "error marshalling", err.Error(),
		)
		return
	}

	kafkaDestinations := make([]dhelpers.DestinationData, 0)

	for _, destination := range destinations {
		switch destination.Type {
		case dhelpers.KafkaDestinationType:
			kafkaDestinations = append(kafkaDestinations, destination)
		}
	}

	if len(kafkaDestinations) > 0 {
		var destinationsText string
		for _, destination := range kafkaDestinations {
			destinationsText += destination.Name + ", "
		}
		destinationsText = strings.TrimRight(destinationsText, ", ")

		// publish to Kafka Producer
		_, _, err = kafkaProducer.SendMessage(&sarama.ProducerMessage{
			Topic: "cacophony",
			Value: sarama.ByteEncoder(marshalled),
		})
		if err != nil {
			cache.GetLogger().Errorln(
				eventContainer.Key+":", "error sending to kafka/"+destinationsText+":",
				err.Error(),
			)
		} else {
			cache.GetLogger().Infoln(
				eventContainer.Key+":", "sent to kafka/"+destinationsText,
				"(size: "+humanize.Bytes(uint64(binary.Size(marshalled)))+")",
			)
		}
	}
}
