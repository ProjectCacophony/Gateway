package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/bwmarrin/discordgo"
	"github.com/json-iterator/go"
)

var (
	Token             string
	KinesisStreamName string
	Kinesis           *kinesis.Kinesis
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -sqs SQS_QUEUE_URL)
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&KinesisStreamName, "stream", "", "Amazon Kinesis Stream Name")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… KINESIS_STREAM_NAME=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		Token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("KINESIS_STREAM_NAME") != "" {
		KinesisStreamName = os.Getenv("KINESIS_STREAM_NAME")
	}
}

func main() {
	// setup Amazon Session
	fmt.Println("connecting to Amazon Kinesis, Stream:", KinesisStreamName)
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	// setup Amazon Kinesis
	Kinesis = kinesis.New(awsSession)

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
	marshalled, err := jsoniter.Marshal(m)
	if err != nil {
		fmt.Println("error packing event:", err.Error())
		return
	}

	event := &kinesis.PutRecordInput{
		Data:         marshalled,
		PartitionKey: aws.String("key1"),
		StreamName:   aws.String(KinesisStreamName),
	}

	// put the record into the kinesis stream
	result, err := Kinesis.PutRecord(event)
	if err != nil {
		fmt.Println("error sending event to kinesis:", err.Error())
		return
	}

	// log
	fmt.Printf("successfully sent #%s by #%s (%s) to Kinesis Stream: #%s (length: %d)\n",
		m.ID, m.Author.ID, m.Content, *result.SequenceNumber, len(marshalled))
}
