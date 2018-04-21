package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"time"

	"encoding/binary"

	"strings"

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
	dhelpersState "gitlab.com/project-d-collab/dhelpers/state"
)

var (
	// PREFIXES are all allowed prefixes, TODO: replace with dynamic prefix
	PREFIXES      = []string{"/"}
	token         string
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
	// Parse command line flags (-t DISCORD_BOT_TOKEN -aws-region AWS_REGION -redis REDIS_ADDRESS -sqs SQS_QUEUE_URL)
	flag.StringVar(&token, "t", "", "Discord Bot token")
	flag.StringVar(&awsRegion, "aws-region", "", "AWS Region")
	flag.StringVar(&redisAddress, "redis", "127.0.0.1:6379", "Redis Address")
	flag.StringVar(&sqsQueueURL, "sqs", "", "SQS Queue Url")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… AWS_REGION=… REDIS_ADDRESS=… SQS_QUEUE_URL=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		token = os.Getenv("DISCORD_BOT_TOKEN")
	}
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
	// get config
	routingConfig, err = dhelpers.GetRoutings()
	if err != nil {
		fmt.Println("error getting routing config", err.Error())
		return
	}
	fmt.Println("Found", len(routingConfig), "routing rules")

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

	// create a new Discordgo Bot Client
	fmt.Println("Connecting to Discord, token Length:", len(token))
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err.Error())
		return
	}
	// essentially only keep discordgo guild state
	dg.State.TrackChannels = false
	dg.State.TrackEmojis = false
	dg.State.TrackMembers = false
	dg.State.TrackPresences = false
	dg.State.TrackRoles = false
	dg.State.TrackVoice = false
	// add gateway ready handler
	dg.AddHandler(onReady)
	// add the discord event handler
	dg.AddHandler(eventHandler)

	// open Discord Websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err.Error())
		return
	}

	// wait for CTRL+C to stop the Bot
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// disconnect from Discord Websocket
	err = dg.Close()
	if err != nil {
		fmt.Println("error closing connection,", err.Error())
		return
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
				fmt.Println(err.Error())
			}
		}
	}
}

func onReconnect(session *discordgo.Session, event *discordgo.Ready) {
	for _, guild := range session.State.Guilds {
		if guild.Large {
			err := session.RequestGuildMembers(guild.ID, "", 0)
			if err != nil {
				fmt.Println(err.Error())
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
		fmt.Println(eventKey+":", "ignored (handled by different gateway)")
		return
	}

	// update shared state
	err := dhelpersState.SharedStateEventHandler(session, i) // TODO: deduplication?
	if err != nil {
		fmt.Println("state error:", err.Error())
	}

	eventContainer := createEventContainer(receivedAt, session, eventKey, i)

	if eventContainer.Type == "" {
		return
	}

	lambdaDestinations, processorDestinations, aliases := dhelpers.ContainerDestinations(
		session, routingConfig, eventContainer)
	eventContainer.Alias = aliases
	eventContainer.Destinations = append(eventContainer.Destinations, lambdaDestinations...)
	eventContainer.Destinations = append(eventContainer.Destinations, processorDestinations...)

	// pack the event data
	marshalled, err := jsoniter.Marshal(eventContainer)
	if err != nil {
		fmt.Println(
			eventContainer.Key+":", "error marshalling", err.Error(),
		)
		return
	}

	if len(processorDestinations) > 0 {
		// send to SQS Queue
		_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
			DelaySeconds: aws.Int64(0),
			MessageBody:  aws.String(string(marshalled)),
			QueueUrl:     aws.String(sqsQueueURL),
		})
		if err != nil {
			fmt.Println(
				eventContainer.Key+":", "error sending to sqs/"+strings.Join(processorDestinations, ",")+":",
				err.Error(),
			)
		} else {
			fmt.Println(
				eventContainer.Key+":", "sent to sqs/"+strings.Join(processorDestinations, ","),
				"(size: "+humanize.Bytes(uint64(binary.Size(marshalled)))+")",
			)
		}
	}

	if len(lambdaDestinations) > 0 {
		for _, lambdaDestination := range lambdaDestinations {
			bytesSent, err := dhelpers.StartLambdaAsync(lambdaClient, eventContainer, lambdaDestination)
			if err != nil {
				fmt.Println(
					eventContainer.Key+":", "error sending to lambda/"+lambdaDestination+":",
					err.Error(),
				)
			} else {
				fmt.Println(
					eventContainer.Key+":", "sent to lambda/"+lambdaDestination,
					"(size: "+humanize.Bytes(uint64(bytesSent))+")",
				)
			}
		}
	}
}
