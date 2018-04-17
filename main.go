package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"net/http"

	"io/ioutil"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/json-iterator/go"
)

var (
	Token       string
	EndpointUrl string
	ApiKey      string
	HttpClient  http.Client
	ApiRequest  *http.Request
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -endpoint AWS_ENDPOINT_URL -apikey AWS_API_KEY)
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&EndpointUrl, "endpoint", "", "Complete AWS Endpoint URL")
	flag.StringVar(&ApiKey, "apikey", "", "AWS API Key")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… KINESIS_STREAM_NAME=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		Token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("AWS_ENDPOINT_URL") != "" {
		EndpointUrl = os.Getenv("AWS_ENDPOINT_URL")
	}
	if os.Getenv("AWS_API_KEY") != "" {
		ApiKey = os.Getenv("AWS_API_KEY")
	}
}

func main() {
	var err error
	// create a new Discordgo Bot Client
	fmt.Println("connecting to Discord, Token Length:", len(Token))
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err.Error())
		return
	}
	// create a new HTTP Client and prepare API request
	HttpClient = http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 10,
	}
	ApiRequest, err = http.NewRequest("POST", EndpointUrl, nil)
	if err != nil {
		fmt.Println("error creating http api request,", err.Error())
		return
	}
	ApiRequest.Header = http.Header{
		"X-APi-Key":    []string{ApiKey},
		"Content-Type": []string{"application/json"},
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

	// pack the event with additional information
	marshalled, err := jsoniter.Marshal(DMessageCreateEvent{
		Event:             m,
		BotUser:           s.State.User,
		GatewayReceivedAt: time.Now(),
	})
	if err != nil {
		fmt.Println("error packing event:", err.Error())
		return
	}

	// send to API Gateway
	req := ApiRequest
	req.Body = ioutil.NopCloser(bytes.NewReader(marshalled))

	resp, err := HttpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("error sending request:", string(body))
		return
	}

	// log
	fmt.Printf("successfully sent #%s by #%s (%s)\n",
		m.ID, m.Author.ID, m.Content)
}
