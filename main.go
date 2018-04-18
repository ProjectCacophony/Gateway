package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"flag"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
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
	lambdaClient  *lambda.Lambda
)

func init() {
	// Parse command line flags (-t DISCORD_BOT_TOKEN -aws-region AWS_REGION -redis REDIS_ADDRESS)
	flag.StringVar(&token, "t", "", "Discord Bot token")
	flag.StringVar(&awsRegion, "aws-region", "", "AWS Region")
	flag.StringVar(&redisAddress, "redis", "127.0.0.1:6379", "Redis Address")
	flag.Parse()
	// overwrite with environment variables if set DISCORD_BOT_TOKEN=… REDIS_ADDRESS=…
	if os.Getenv("DISCORD_BOT_TOKEN") != "" {
		token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if os.Getenv("AWS_REGION") != "" {
		awsRegion = os.Getenv("AWS_REGION")
	}
	if os.Getenv("REDIS_ADDRESS") != "" {
		redisAddress = os.Getenv("REDIS_ADDRESS")
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
	processEvent(session, i)
}

// processes discord events
func processEvent(session *discordgo.Session, i interface{}) {
	var err error
	receivedAt := time.Now()

	// create enhanced Event
	var handledByUs bool
	var handled int

	for _, routingEntry := range routingConfig {
		if handled > 0 && !routingEntry.Always {
			continue
		}

		switch t := i.(type) {
		case *discordgo.GuildCreate:
			if routingEntry.Type != dhelpers.GuildCreateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildCreate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			// add additional state payload
			dDEvent.BotUser = session.State.User
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
			continue
		case *discordgo.GuildUpdate:
			if routingEntry.Type != dhelpers.GuildUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildDelete:
			if routingEntry.Type != dhelpers.GuildDeleteEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Guild)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildDelete{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.Guild != nil {
				dDEvent.SourceGuild = t.Guild
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberAdd:
			if routingEntry.Type != dhelpers.GuildMemberAddEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildMemberAdd{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberUpdate:
			if routingEntry.Type != dhelpers.GuildMemberUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildMemberUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMemberRemove:
			if routingEntry.Type != dhelpers.GuildMemberRemoveEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Member)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildMemberRemove{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildMembersChunk:
			if routingEntry.Type != dhelpers.GuildMembersChunkEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %v", t.GuildID, t.Members)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildMembersChunk{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleCreate:
			if routingEntry.Type != dhelpers.GuildRoleCreateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.GuildRole)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildRoleCreate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleUpdate:
			if routingEntry.Type != dhelpers.GuildRoleUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.GuildRole)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildRoleUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildRoleDelete:
			if routingEntry.Type != dhelpers.GuildRoleDeleteEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %s", t.RoleID, t.GuildID)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildRoleDelete{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildEmojisUpdate:
			if routingEntry.Type != dhelpers.GuildEmojisUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %v", t.GuildID, t.Emojis)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildEmojisUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelCreate:
			if routingEntry.Type != dhelpers.ChannelCreateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventChannelCreate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelUpdate:
			if routingEntry.Type != dhelpers.ChannelUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventChannelUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelDelete:
			if routingEntry.Type != dhelpers.ChannelDeleteEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Channel)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventChannelDelete{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			if t.Channel != nil {
				dDEvent.SourceChannel = t.Channel
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageCreate:
			if routingEntry.Type != dhelpers.MessageCreateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			} else {
				handledByUs = true
			}
			// args and prefix
			args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Author, session.State.User, t.Content, args, prefix) {
				continue
			}
			dDEvent := dhelpers.EventMessageCreate{}
			dDEvent.Args = args
			dDEvent.Prefix = prefix
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageUpdate:
			if routingEntry.Type != dhelpers.MessageUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			} else {
				handledByUs = true
			}
			// args and prefix
			args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Author, session.State.User, t.Content, args, prefix) {
				continue
			}
			dDEvent := dhelpers.EventMessageUpdate{}
			dDEvent.Args = args
			dDEvent.Prefix = prefix
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageDelete:
			if routingEntry.Type != dhelpers.MessageDeleteEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.Message)) {
				return
			} else {
				handledByUs = true
			}
			// args and prefix
			args, prefix := dhelpers.GetMessageArguments(t.Content, PREFIXES)
			// check requirements
			if !dhelpers.RoutingMatchMessage(routingEntry, t.Author, session.State.User, t.Content, args, prefix) {
				continue
			}
			dDEvent := dhelpers.EventMessageDelete{}
			dDEvent.Args = args
			dDEvent.Prefix = prefix
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, "#", t.ID, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "#", t.ID, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.PresenceUpdate:
			if routingEntry.Type != dhelpers.PresenceUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s %v", t.Presence, t.GuildID, t.Roles)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventPresenceUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.ChannelPinsUpdate:
			if routingEntry.Type != dhelpers.ChannelPinsUpdateEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%s %s", t.LastPinTimestamp, t.ChannelID)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventChannelPinsUpdate{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildBanAdd:
			if routingEntry.Type != dhelpers.GuildBanAddEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s", t.User, t.GuildID)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildBanAdd{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.GuildBanRemove:
			if routingEntry.Type != dhelpers.GuildBanRemoveEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v %s", t.User, t.GuildID)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventGuildBanRemove{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
			if t.GuildID != "" {
				sourceGuild, err := session.State.Guild(t.GuildID)
				if err == nil {
					dDEvent.SourceGuild = sourceGuild
				}
			}
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionAdd:
			if routingEntry.Type != dhelpers.MessageReactionAddEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventMessageReactionAdd{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemove:
			if routingEntry.Type != dhelpers.MessageReactionRemoveEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventMessageReactionRemove{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		case *discordgo.MessageReactionRemoveAll:
			if routingEntry.Type != dhelpers.MessageReactionRemoveAllEventType {
				continue
			}
			// deduplication
			if !handledByUs && !IsNewEvent(routingEntry.Type, fmt.Sprintf("%v", t.MessageReaction)) {
				return
			} else {
				handledByUs = true
			}
			dDEvent := dhelpers.EventMessageReactionRemoveAll{}
			dDEvent.Alias = routingEntry.Alias
			dDEvent.Event = t
			// additional payload from state
			dDEvent.BotUser = session.State.User
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
			err = SendEvent(started, receivedAt, routingEntry.Type, dDEvent, routingEntry.Function)
			handled++
			if err != nil {
				fmt.Println("error processing event", routingEntry.Type, ":", err.Error())
			} else {
				fmt.Println("sent event", routingEntry.Type, "to", routingEntry.Function, "alias", routingEntry.Alias)
			}
		}
	}
}
