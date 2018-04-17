package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"net/http"

	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	"gitlab.com/project-d-collab/dhelpers"
)

var (
	PREFIX          = "/" // TODO
	Token           string
	EndpointBaseUrl string
	ApiKey          string
	RedisAddress    string
	HttpClient      http.Client
	ApiRequest      *http.Request
	RoutingConfig   []dhelpers.RoutingRule
	RedisClient     *redis.Client
	Started         time.Time
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -endpoint AWS_ENDPOINT_BASE -apikey AWS_API_KEY -redis REDIS_ADDRESS)
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&EndpointBaseUrl, "endpoint", "", "AWS Endpoint Base URL")
	flag.StringVar(&ApiKey, "apikey", "", "AWS API Key")
	flag.StringVar(&RedisAddress, "redis", "127.0.0.1:6379", "Redis Address")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… AWS_ENDPOINT_BASE=… AWS_API_KEY=… REDIS_ADDRESS=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		Token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("AWS_ENDPOINT_BASE") != "" {
		EndpointBaseUrl = os.Getenv("AWS_ENDPOINT_BASE")
	}
	if os.Getenv("AWS_API_KEY") != "" {
		ApiKey = os.Getenv("AWS_API_KEY")
	}
	if os.Getenv("REDIS_ADDRESS") != "" {
		RedisAddress = os.Getenv("REDIS_ADDRESS")
	}
}

func main() {
	Started = time.Now()
	var err error
	// get config
	RoutingConfig, err = dhelpers.GetRoutings()
	if err != nil {
		fmt.Println("error getting routing config", err.Error())
		return
	}
	fmt.Println("Found", len(RoutingConfig), "routing rules")

	// connect to redis
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: "",
		DB:       0,
	})

	// create a new Discordgo Bot Client
	fmt.Println("Connecting to Discord, Token Length:", len(Token))
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
	ApiRequest, err = http.NewRequest("POST", EndpointBaseUrl, nil)
	if err != nil {
		fmt.Println("error creating http api request,", err.Error())
		return
	}
	ApiRequest.Header = http.Header{
		"X-APi-Key":    []string{ApiKey},
		"Content-Type": []string{"application/json"},
	}

	// add the MessageCreate handler
	dg.AddHandler(eventHandler)

	// open Discord Websocket connection
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err.Error())
		return
	}

	// TODO: request guild member chunks for large guilds, and for new guilds
	// TODO: persist and restore state?

	// wait for CTRL+C to stop the Bot
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// disconnect from Discord Websocket
	dg.Close()
}

// discord event handler
func eventHandler(session *discordgo.Session, i interface{}) {
	processEvent(session, i)
	/*
		if IsNewEvent(i) {
			processEvent(session, i)
		} else {
			fmt.Println(fmt.Sprintf("%v", i))
		}
	*/
}

// discord event handler
func processEvent(session *discordgo.Session, i interface{}) {
	var err error

	// create enhanced Event
	dDEvent := dhelpers.Event{
		Type:              "",
		Event:             i,
		BotUser:           nil,
		SourceChannel:     nil,
		SourceGuild:       nil,
		GatewayReceivedAt: time.Now(),
		GatewayStarted:    Started,
		Prefix:            PREFIX,
	}
	// add additional state payload
	if session != nil && session.State != nil {
		if session.State.User != nil {
			dDEvent.BotUser = session.State.User
		}
	}

	var handled int

	for _, routingEntry := range RoutingConfig {
		if handled > 0 && !routingEntry.Always {
			continue
		}

		switch t := i.(type) {
		case *discordgo.GuildCreate:
			if routingEntry.Type != dhelpers.GuildCreateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
			continue
		case *discordgo.GuildUpdate:
			if routingEntry.Type != dhelpers.GuildUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildDelete:
			if routingEntry.Type != dhelpers.GuildDeleteEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberAdd:
			if routingEntry.Type != dhelpers.GuildMemberAddEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberUpdate:
			if routingEntry.Type != dhelpers.GuildMemberUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberRemove:
			if routingEntry.Type != dhelpers.GuildMemberRemoveEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMembersChunk:
			if routingEntry.Type != dhelpers.GuildMembersChunkEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %v", t.GuildID, t.Members)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleCreate:
			if routingEntry.Type != dhelpers.GuildRoleCreateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.GuildRole)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleUpdate:
			if routingEntry.Type != dhelpers.GuildRoleUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.GuildRole)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleDelete:
			if routingEntry.Type != dhelpers.GuildRoleDeleteEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %s", t.RoleID, t.GuildID)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildEmojisUpdate:
			if routingEntry.Type != dhelpers.GuildEmojisUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %v", t.GuildID, t.Emojis)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelCreate:
			if routingEntry.Type != dhelpers.ChannelCreateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelUpdate:
			if routingEntry.Type != dhelpers.ChannelUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelDelete:
			if routingEntry.Type != dhelpers.ChannelDeleteEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageCreate:
			if routingEntry.Type != dhelpers.MessageCreateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			}
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Content, PREFIX) {
				continue
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageUpdate:
			if routingEntry.Type != dhelpers.MessageUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			}
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Content, PREFIX) {
				continue
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageDelete:
			if routingEntry.Type != dhelpers.MessageDeleteEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			}
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Content, PREFIX) {
				continue
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.PresenceUpdate:
			if routingEntry.Type != dhelpers.PresenceUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s %v", t.Presence, t.GuildID, t.Roles)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelPinsUpdate:
			if routingEntry.Type != dhelpers.ChannelPinsUpdateEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %s", t.LastPinTimestamp, t.ChannelID)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildBanAdd:
			if routingEntry.Type != dhelpers.GuildBanAddEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s", t.User, t.GuildID)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildBanRemove:
			if routingEntry.Type != dhelpers.GuildBanRemoveEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s", t.User, t.GuildID)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionAdd:
			if routingEntry.Type != dhelpers.MessageReactionAddEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemove:
			if routingEntry.Type != dhelpers.MessageReactionRemoveEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemoveAll:
			if routingEntry.Type != dhelpers.MessageReactionRemoveAllEventType {
				continue
			}
			// deduplication
			if !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			}
			dDEvent.Type = routingEntry.Type
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.ChannelID != "" {
				sourceChannel, err := session.State.Channel(t.ChannelID)
				if err == nil {
					dDEvent.SourceChannel = sourceChannel
					sourceGuild, err := session.State.Guild(sourceChannel.GuildID)
					if err == nil {
						dDEvent.SourceGuild = sourceGuild
					}
				}
			}
			err = SendEvent(dDEvent, routingEntry.Endpoint)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		}
	}
}
