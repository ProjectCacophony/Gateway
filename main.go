package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

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
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&SqsQueueUrl, "sqs", "", "Amazon SQS Queue URL")
	flag.Parse()
}

func main() {
	fmt.Println("setting up", Token, SqsQueueUrl)
	// setup Amazon Session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	// setup Amazon SQS queue
	Svc = sqs.New(sess)

	// create a new Discordgo Bot Client
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// add the MessageCreate handler
	dg.AddHandler(messageCreate)

	// open Discord Websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
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
		fmt.Println("Error packing event:", err)
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
		MessageBody: aws.String(string(marshalled)),
		QueueUrl:    &SqsQueueUrl,
	})
	if err != nil {
		fmt.Println("Error sending event to SQS:", err)
		return
	}

	// log
	fmt.Printf("Successfully sent #%s by #%s (%s) to SNS Queue: #%s\n",
		m.ID, m.Author.ID, m.Content, result.MessageId)
}
