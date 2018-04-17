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
)

var (
	PREFIX          = "/" // TODO
	Token           string
	EndpointBaseUrl string
	ApiKey          string
	HttpClient      http.Client
	ApiRequest      *http.Request
	RoutingConfig   []RoutingEntry
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -endpoint AWS_ENDPOINT_BASE -apikey AWS_API_KEY)
	flag.StringVar(&Token, "t", "", "Discord Bot Token")
	flag.StringVar(&EndpointBaseUrl, "endpoint", "", "AWS Endpoint Base URL")
	flag.StringVar(&ApiKey, "apikey", "", "AWS API Key")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… AWS_ENDPOINT_BASE=… AWS_API_KEY=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		Token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("AWS_ENDPOINT_BASE") != "" {
		EndpointBaseUrl = os.Getenv("AWS_ENDPOINT_BASE")
	}
	if os.Getenv("AWS_API_KEY") != "" {
		ApiKey = os.Getenv("AWS_API_KEY")
	}
}

func main() {
	var err error
	// get config
	RoutingConfig, err = GetRoutings()
	if err != nil {
		fmt.Println("error getting routing config", err.Error())
		return
	}
	fmt.Println("Found", len(RoutingConfig), "routing rules")

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
	var err error

	// create enhanced Event
	dDEvent := DDiscordEvent{
		Type:              "",
		Event:             i,
		BotUser:           nil,
		SourceChannel:     nil,
		SourceGuild:       nil,
		GatewayReceivedAt: time.Now(),
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
			if routingEntry.Type != GuildCreateEventType {
				continue
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
			if routingEntry.Type != GuildUpdateEventType {
				continue
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
			if routingEntry.Type != GuildDeleteEventType {
				continue
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
			if routingEntry.Type != GuildMemberAddEventType {
				continue
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
			if routingEntry.Type != GuildMemberUpdateEventType {
				continue
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
			if routingEntry.Type != GuildMemberRemoveEventType {
				continue
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
			if routingEntry.Type != GuildMembersChunkEventType {
				continue
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
			if routingEntry.Type != GuildRoleCreateEventType {
				continue
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
			if routingEntry.Type != GuildRoleUpdateEventType {
				continue
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
			if routingEntry.Type != GuildRoleDeleteEventType {
				continue
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
			if routingEntry.Type != GuildEmojisUpdateEventType {
				continue
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
			if routingEntry.Type != ChannelCreateEventType {
				continue
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
			if routingEntry.Type != ChannelUpdateEventType {
				continue
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
			if routingEntry.Type != ChannelDeleteEventType {
				continue
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
			if routingEntry.Type != MessageCreateEventType {
				continue
			}
			// check requirements
			if !MatchMessageRequirements(routingEntry, t.Content) {
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
			if routingEntry.Type != MessageUpdateEventType {
				continue
			}
			// check requirements
			if !MatchMessageRequirements(routingEntry, t.Content) {
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
			if routingEntry.Type != MessageDeleteEventType {
				continue
			}
			// check requirements
			if !MatchMessageRequirements(routingEntry, t.Content) {
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
			if routingEntry.Type != PresenceUpdateEventType {
				continue
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
			if routingEntry.Type != ChannelPinsUpdateEventType {
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
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildBanAdd:
			if routingEntry.Type != GuildBanAddEventType {
				continue
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
			if routingEntry.Type != GuildBanRemoveEventType {
				continue
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
			if routingEntry.Type != MessageReactionAddEventType {
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
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemove:
			if routingEntry.Type != MessageReactionRemoveEventType {
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
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemoveAll:
			if routingEntry.Type != MessageReactionRemoveAllEventType {
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
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Endpoint, "alias", routingEntry.Alias)
			}
		}
	}
}
