package main

import (
	"encoding/binary"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/emicklei/go-restful"
	"github.com/go-redis/redis"
	"github.com/json-iterator/go"
	"gitlab.com/project-d-collab/Gateway/api"
	"gitlab.com/project-d-collab/Gateway/metrics"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
	"gitlab.com/project-d-collab/dhelpers/components"
	"gitlab.com/project-d-collab/dhelpers/state"
)

var (
	routingConfig []dhelpers.RoutingRule
	started       time.Time
	didLaunch     bool
	sqsQueueURL   string
	redisClient   *redis.Client
	lambdaClient  *lambda.Lambda
	sqsClient     *sqs.SQS
)

func init() {
	// parse environment variables
	sqsQueueURL = os.Getenv("SQS_QUEUE_URL")
}

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
	err = components.InitAwsSqs()
	dhelpers.CheckErr(err)
	err = components.InitAwsLambda()
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

	// get cached client
	redisClient = cache.GetRedisClient()
	lambdaClient = cache.GetAwsLambdaSession()
	sqsClient = cache.GetAwsSqsSession()

	// open Discord Websocket connection
	err = cache.GetDiscord().Open()
	dhelpers.CheckErr(err)

	// start api
	go func() {
		restful.Add(api.New())
		cache.GetLogger().Fatal(http.ListenAndServe(os.Getenv("API_ADDRESS"), nil))
	}()
	cache.GetLogger().Infoln("started API on", os.Getenv("API_ADDRESS"))

	// wait for CTRL+C to stop the Bot
	cache.GetLogger().Infoln("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// disconnect from Discord Websocket
	err = cache.GetDiscord().Close()
	if err != nil {
		cache.GetLogger().Errorln("error closing connection,", err.Error())
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

	processorDestinations := make([]dhelpers.DestinationData, 0)

	var bytesSent int
	for _, destination := range destinations {
		switch destination.Type {
		case dhelpers.LambdaDestinationType:
			bytesSent, err = dhelpers.StartLambdaAsync(lambdaClient, eventContainer, destination.Name)
			if err != nil {
				cache.GetLogger().Errorln(
					eventContainer.Key+":", "error sending to lambda/"+destination.Name+":",
					err.Error(),
				)
			} else {
				cache.GetLogger().Infoln(
					eventContainer.Key+":", "sent to lambda/"+destination.Name,
					"(size: "+humanize.Bytes(uint64(bytesSent))+")",
				)
			}
		case dhelpers.SqsDestinationType:
			processorDestinations = append(processorDestinations, destination)
		}
	}

	if len(processorDestinations) > 0 {
		var destinationsText string
		for _, destination := range processorDestinations {
			destinationsText += destination.Name + ", "
		}
		destinationsText = strings.TrimRight(destinationsText, ", ")

		// send to SQS Queue
		_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
			DelaySeconds: aws.Int64(0),
			MessageBody:  aws.String(string(marshalled)),
			QueueUrl:     aws.String(sqsQueueURL),
		})
		if err != nil {
			cache.GetLogger().Errorln(
				eventContainer.Key+":", "error sending to sqs/"+destinationsText+":",
				err.Error(),
			)
		} else {
			cache.GetLogger().Infoln(
				eventContainer.Key+":", "sent to sqs/"+destinationsText,
				"(size: "+humanize.Bytes(uint64(binary.Size(marshalled)))+")",
			)
		}
	}

}
