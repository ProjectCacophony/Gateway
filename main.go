package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"encoding/base64"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/bwmarrin/discordgo"
	"github.com/vmihailenco/msgpack"
)

var (
	Token       string
	SqsQueueUrl string
	Svc         *sqs.SQS
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -sqs SQS_QUEUE_URL)
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&SqsQueueUrl, "sqs", "", "Amazon SQS Queue URL")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… SQS_QUEUE_URL=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		Token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("SQS_QUEUE_URL") != "" {
		SqsQueueUrl = os.Getenv("SQS_QUEUE_URL")
	}
}

func main() {
	// setup Amazon Session
	fmt.Println("connecting to Amazon SQS, URL:", SqsQueueUrl)
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	// setup Amazon SQS queue
	Svc = sqs.New(awsSession)

	// create a new Discordgo Bot Client
	fmt.Println("connecting to Discord, Token Length:", len(Token))
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err.Error())
		return
	}

	// add the MessageCreate handler
	dg.AddHandler(messageCreate)

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

// MessageCreate handler
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages without author
	if m.Author == nil {
		return
	}
	// ignore messages by the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// pack the event
	marshalled, err := msgpack.Marshal(m)
	if err != nil {
		fmt.Println("error packing event:", err.Error())
		return
	}

	// send the message to SQS queue
	result, err := Svc.SendMessage(&sqs.SendMessageInput{
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"EventType": {
				DataType:    aws.String("String"),
				StringValue: aws.String("discordgo.MessageCreate"),
			},
		},
		QueueUrl:               &SqsQueueUrl,
		MessageGroupId:         aws.String("discord-events"),
		MessageDeduplicationId: aws.String(m.ID),
		MessageBody:            aws.String(base64.StdEncoding.EncodeToString(marshalled)),
	})
	if err != nil {
		fmt.Println("error sending event to SQS:", err.Error())
		return
	}

	// log
	fmt.Printf("successfully sent #%s by #%s (%s) to SNS Queue: #%s\n",
		m.ID, m.Author.ID, m.Content, *result.MessageId)
}
