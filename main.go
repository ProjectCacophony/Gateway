package main

import (
	"encoding/binary"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/go-redis/redis"
	"github.com/json-iterator/go"
	"gitlab.com/project-d-collab/dhelpers"
	dhelpersCache "gitlab.com/project-d-collab/dhelpers/cache"
	"gitlab.com/project-d-collab/dhelpers/components"
	dhelpersState "gitlab.com/project-d-collab/dhelpers/state"
)

var (
	// PREFIXES are all allowed prefixes, TODO: replace with dynamic prefix
	PREFIXES      = []string{"/"}
	awsRegion     string
	redisAddress  string
	routingConfig []dhelpers.RoutingRule
	redisClient   *redis.Client
	started       time.Time
	didLaunch     bool
	sqsClient     *sqs.SQS
	sqsQueueURL   string
	lambdaClient  *lambda.Lambda
)

func init() {
	// Parse command line flags (-aws-region AWS_REGION -redis REDIS_ADDRESS -sqs SQS_QUEUE_URL)
	flag.StringVar(&awsRegion, "aws-region", "", "AWS Region")
	flag.StringVar(&redisAddress, "redis", "127.0.0.1:6379", "Redis Address")
	flag.StringVar(&sqsQueueURL, "sqs", "", "SQS Queue Url")
	flag.Parse()
	// overwrite with environment variables if set AWS_REGION=… REDIS_ADDRESS=… SQS_QUEUE_URL=…
	if os.Getenv("AWS_REGION") != "" {
		awsRegion = os.Getenv("AWS_REGION")
	}
	if os.Getenv("REDIS_ADDRESS") != "" {
		redisAddress = os.Getenv("REDIS_ADDRESS")
	}
	if os.Getenv("SQS_QUEUE_URL") != "" {
		sqsQueueURL = os.Getenv("SQS_QUEUE_URL")
	}
}

func main() {
	started = time.Now()
	var err error

	// init components
	components.InitLogger("Gateway")
	err = components.InitSentry()
	dhelpers.CheckErr(err)
	err = components.InitDiscord()
	dhelpers.CheckErr(err)

	// get config
	routingConfig, err = dhelpers.GetRoutings()
	dhelpers.CheckErr(err)
	dhelpersCache.GetLogger().Infoln("Found", len(routingConfig), "routing rules")

	// connect to aws
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))
	sqsClient = sqs.New(sess)
	lambdaClient = lambda.New(sess)

	// connect to redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "",
		DB:       0,
	})
	dhelpersCache.SetRedisClient(redisClient)

	// essentially only keep discordgo guild state
	dhelpersCache.GetDiscord().State.TrackChannels = false
	dhelpersCache.GetDiscord().State.TrackEmojis = false
	dhelpersCache.GetDiscord().State.TrackMembers = false
	dhelpersCache.GetDiscord().State.TrackPresences = false
	dhelpersCache.GetDiscord().State.TrackRoles = false
	dhelpersCache.GetDiscord().State.TrackVoice = false
	// add gateway ready handler
	dhelpersCache.GetDiscord().AddHandler(onReady)
	// add the discord event handler
	dhelpersCache.GetDiscord().AddHandler(eventHandler)

	// open Discord Websocket connection
	err = dhelpersCache.GetDiscord().Open()
	dhelpers.CheckErr(err)

	// wait for CTRL+C to stop the Bot
	dhelpersCache.GetLogger().Infoln("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// disconnect from Discord Websocket
	err = dhelpersCache.GetDiscord().Close()
	if err != nil {
		dhelpersCache.GetLogger().Errorln("error closing connection,", err.Error())
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
	PREFIXES = append(PREFIXES, "<@"+session.State.User.ID+">")  // add bot mention to prefixes
	PREFIXES = append(PREFIXES, "<@!"+session.State.User.ID+">") // add bot alias mention to prefixes

	for _, guild := range session.State.Guilds {
		if guild.Large {
			err := session.RequestGuildMembers(guild.ID, "", 0)
			if err != nil {
				dhelpersCache.GetLogger().Errorln(err.Error())
			}
		}
	}
}

func onReconnect(session *discordgo.Session, event *discordgo.Ready) {
	for _, guild := range session.State.Guilds {
		if guild.Large {
			err := session.RequestGuildMembers(guild.ID, "", 0)
			if err != nil {
				dhelpersCache.GetLogger().Errorln(err.Error())
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
		dhelpersCache.GetLogger().Infoln(eventKey+":", "ignored (handled by different gateway)")
		return
	}

	// update shared state
	err := dhelpersState.SharedStateEventHandler(session, i) // TODO: deduplication?
	if err != nil {
		dhelpersCache.GetLogger().Errorln("state error:", err.Error())
	}

	eventContainer := createEventContainer(receivedAt, session, eventKey, i)

	if eventContainer.Type == "" {
		return
	}

	destinations := dhelpers.ContainerDestinations(
		session, routingConfig, eventContainer)
	eventContainer.Destinations = append(eventContainer.Destinations, destinations...)

	// pack the event data
	marshalled, err := jsoniter.Marshal(eventContainer)
	if err != nil {
		dhelpersCache.GetLogger().Errorln(
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
				dhelpersCache.GetLogger().Errorln(
					eventContainer.Key+":", "error sending to lambda/"+destination.Name+":",
					err.Error(),
				)
			} else {
				dhelpersCache.GetLogger().Infoln(
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
			dhelpersCache.GetLogger().Errorln(
				eventContainer.Key+":", "error sending to sqs/"+destinationsText+":",
				err.Error(),
			)
		} else {
			dhelpersCache.GetLogger().Infoln(
				eventContainer.Key+":", "sent to sqs/"+destinationsText,
				"(size: "+humanize.Bytes(uint64(binary.Size(marshalled)))+")",
			)
		}
	}

}
