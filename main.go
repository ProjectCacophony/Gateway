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
)

var (
	PREFIXES      = []string{"/"} // TODO
	token         string
	awsRegion     string
	redisAddress  string
	routingConfig []dhelpers.RoutingRule
	redisClient   *redis.Client
	started       time.Time
	didLaunch     bool
	sqsClient     *sqs.SQS
	sqsQueueUrl   string
	lambdaClient  *lambda.Lambda
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -aws-region AWS_REGION -redis REDIS_ADDRESS -sqs SQS_QUEUE_URL)
	flag.StringVar(&token, "t", "", "Discord Bot token")
	flag.StringVar(&awsRegion, "aws-region", "", "AWS Region")
	flag.StringVar(&redisAddress, "redis", "127.0.0.1:6379", "Redis Address")
	flag.StringVar(&sqsQueueUrl, "sqs", "", "SQS Queue Url")
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
		sqsQueueUrl = os.Getenv("SQS_QUEUE_URL")
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

	// create a new Discordgo Bot Client
	fmt.Println("Connecting to Discord, token Length:", len(token))
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err.Error())
		return
	}

	// add gateway ready handler
	dg.AddHandler(BotOnReady)
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
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// disconnect from Discord Websocket
	dg.Close()
}

func BotOnReady(session *discordgo.Session, event *discordgo.Ready) {
	if !didLaunch {
		OnFirstReady(session, event)
		didLaunch = true
	} else {
		OnReconnect(session, event)
	}
}

func OnFirstReady(session *discordgo.Session, event *discordgo.Ready) {
	PREFIXES = append(PREFIXES, "<@"+session.State.User.ID+">")  // add bot mention to prefixes
	PREFIXES = append(PREFIXES, "<@!"+session.State.User.ID+">") // add bot alias mention to prefixes
	// TODO: request guild member chunks for large guilds, and for new guilds
	// TODO: persist and restore state?
}

func OnReconnect(session *discordgo.Session, event *discordgo.Ready) {
	// TODO: request guild member chunks for large guilds, and for new guilds
	// TODO: persist and restore state?
}

// discord event handler
func eventHandler(session *discordgo.Session, i interface{}) {
	receivedAt := time.Now()

	eventKey := dhelpers.GetEventKey(receivedAt, i)

	if eventKey == "" {
		return
	}

	if !dhelpers.IsNewEvent(redisClient, "gateway", eventKey) {
		fmt.Println(eventKey+":", "ignored (handled by different gateway)")
		return
	}

	eventContainer := createEventContainer(receivedAt, session, eventKey, i)

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
			QueueUrl:     aws.String(sqsQueueUrl),
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
