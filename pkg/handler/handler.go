package handler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	raven "github.com/getsentry/raven-go"
	"github.com/go-redis/redis"
	"gitlab.com/Cacophony/Gateway/pkg/whitelist"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/state"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/api/global"
	"go.uber.org/zap"
)

var b3Prop = b3.B3{}

// EventHandler handles discord events and puts them into rabbitMQ
type EventHandler struct {
	logger      *zap.Logger
	redisClient *redis.Client
	publisher   *events.Publisher
	checker     *whitelist.Checker
	state       *state.State
	deduplicate bool
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	logger *zap.Logger,
	redisClient *redis.Client,
	publisher *events.Publisher,
	checker *whitelist.Checker,
	state *state.State,
	deduplicate bool,
) *EventHandler {
	return &EventHandler{
		logger:      logger,
		redisClient: redisClient,
		publisher:   publisher,
		checker:     checker,
		state:       state,
		deduplicate: deduplicate,
	}
}

// OnDiscordEvent receives discord events
func (eh *EventHandler) OnDiscordEvent(session *discordgo.Session, eventItem interface{}) {
	var err error

	if session == nil || session.State == nil || session.State.User == nil {
		return
	}

	event, expiration, err := events.GenerateEventFromDiscordgoEvent(
		session.State.User.ID,
		eventItem,
	)
	if err != nil {
		raven.CaptureError(err, nil)
		eh.logger.Error("unable to generate event",
			zap.Error(err),
			zap.Any("event", eventItem),
		)

		err = eh.state.SharedStateEventHandler(session, eventItem)
		if err != nil {
			raven.CaptureError(err, nil)
			eh.logger.Error("state client failed to handle event", zap.Error(err))
		}
		return
	}

	if event == nil {
		err = eh.state.SharedStateEventHandler(session, eventItem)
		if err != nil {
			raven.CaptureError(err, nil)
			eh.logger.Error("state client failed to handle event", zap.Error(err))
		}
		return
	}

	ctx, span := global.Tracer("cacophony.dev/gateway").Start(context.Background(), "Recv.Discord")
	defer span.End()

	b3Prop.Inject(ctx, &event.SpanContext)
	span.SetAttributes(
		events.SpanLabelEventingType.String(string(event.Type)),
		events.SpanLabelDiscordBotUserID.String(event.BotUserID),
		events.SpanLabelDiscordGuildID.String(event.GuildID),
		events.SpanLabelDiscordChannelID.String(event.ChannelID),
		events.SpanLabelDiscordUserID.String(event.UserID),
		events.SpanLabelDiscordMessageID.String(event.MessageID),
	)

	l := eh.logger.With(
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)),
		zap.String("event_guild_id", event.GuildID),
		zap.String("event_user_id", event.UserID),
		zap.String("event_bot_user_id", event.BotUserID),
	)

	if event.GuildID != "" && eh.checker.IsBlacklisted(event.GuildID) {
		return
	}

	var oldGuild *discordgo.Guild
	var oldMember *discordgo.Member
	var oldChannel *discordgo.Channel
	var oldRole *discordgo.Role
	var oldEmoji []*discordgo.Emoji
	var oldWebhooks []*discordgo.Webhook
	// var oldInvites []*discordgo.Invite
	switch event.Type {
	case events.GuildUpdateType:
		oldGuild, _ = eh.state.Guild(event.GuildID)
	case events.GuildMemberUpdateType:
		oldMember, _ = eh.state.Member(event.GuildID, event.GuildMemberUpdate.Member.User.ID)
	case events.ChannelUpdateType:
		oldChannel, _ = eh.state.Channel(event.ChannelUpdate.ID)
	case events.ChannelDeleteType:
		oldChannel, _ = eh.state.Channel(event.ChannelDelete.ID)
	case events.GuildRoleUpdateType:
		oldRole, _ = eh.state.Role(event.GuildID, event.GuildRoleUpdate.Role.ID)
	case events.GuildRoleDeleteType:
		oldRole, _ = eh.state.Role(event.GuildID, event.GuildRoleDelete.RoleID)
	case events.GuildEmojisUpdateType:
		guild, _ := eh.state.Guild(event.GuildID)
		if guild != nil {
			oldEmoji = guild.Emojis
		}
	case events.WebhooksUpdateType:
		oldWebhooks, _ = eh.state.GuildWebhooks(event.GuildID)
	case events.GuildMemberAddType:
		// oldInvites, _ = eh.state.GuildInvites(event.GuildID)
	}

	if eh.deduplicate {
		duplicate, err := eh.IsDuplicate(event.CacheKey, expiration)
		if err != nil {
			raven.CaptureError(err, nil)
			l.Debug("unable to deduplicate event",
				zap.Error(err),
				zap.Any("event", eventItem),
			)

			err = eh.state.SharedStateEventHandler(session, eventItem)
			if err != nil {
				raven.CaptureError(err, nil)
				l.Error("state client failed to handle event", zap.Error(err))
			}
			return
		}
		if duplicate {
			l.Debug("skipping event, as it is a duplicate", zap.String("cache_key", event.CacheKey))
			return
		}
	}

	err = eh.state.SharedStateEventHandler(session, eventItem)
	if err != nil {
		raven.CaptureError(err, nil)
		l.Error("state client failed to handle event", zap.Error(err))
	}

	if event.GuildID != "" && !eh.checker.IsWhitelisted(event.GuildID) {
		l.Debug("skipping publishing event because guild is not whitelisted")
		return
	}

	var diffEvent *events.Event
	switch event.Type {
	case events.GuildUpdateType:
		newGuild, _ := eh.state.Guild(event.GuildID)
		diffEvent, err = guildDiff(oldGuild, newGuild)
	case events.GuildMemberUpdateType:
		newMember, _ := eh.state.Member(event.GuildID, event.GuildMemberUpdate.Member.User.ID)
		diffEvent, err = memberDiff(oldMember, newMember)
	case events.ChannelUpdateType:
		newChannel, _ := eh.state.Channel(event.ChannelUpdate.ID)
		diffEvent, err = channelDiff(oldChannel, newChannel)
	case events.ChannelDeleteType:
		newChannel, _ := eh.state.Channel(event.ChannelDelete.ID)
		diffEvent, err = channelDiff(oldChannel, newChannel)
	case events.GuildRoleUpdateType:
		newRole, _ := eh.state.Role(event.GuildID, event.GuildRoleUpdate.Role.ID)
		diffEvent, err = roleDiff(event.GuildID, oldRole, newRole)
	case events.GuildRoleDeleteType:
		newRole, _ := eh.state.Role(event.GuildID, event.GuildRoleDelete.RoleID)
		diffEvent, err = roleDiff(event.GuildID, oldRole, newRole)
	case events.GuildEmojisUpdateType:
		guild, _ := eh.state.Guild(event.GuildID)
		if guild != nil {
			newEmoji := guild.Emojis
			diffEvent, err = emojiDiff(event.GuildID, oldEmoji, newEmoji)
		}
	case events.WebhooksUpdateType:
		newWebhooks, _ := eh.state.GuildWebhooks(event.WebhooksUpdate.GuildID)
		diffEvent, err = webhooksDiff(event.GuildID, oldWebhooks, newWebhooks)
	case events.GuildMemberAddType:
		// disables this, calling invites endpoint every time someone joins a guild is not sustainable
		// go func(guildID string, oldInvites []*discordgo.Invite, memberAdd *discordgo.GuildMemberAdd) {
		// 	newInvites, err := eh.state.GuildInvitesWithRefresh(guildID, session)
		// 	if err != nil {
		// 		raven.CaptureError(err, nil)
		// 		l.Error("failure getting new invites", zap.Error(err))
		// 	}
		//
		// 	extraEvent, err := events.New(events.CacophonyGuildMemberAddExtra)
		// 	if err != nil {
		// 		raven.CaptureError(err, nil)
		// 		l.Error("failure generating member add extra event", zap.Error(err))
		// 		return
		// 	}
		// 	extraEvent.GuildID = guildID
		// 	extraEvent.GuildMemberAddExtra = &events.GuildMemberAddExtra{
		// 		GuildMemberAdd: memberAdd,
		// 	}
		//
		// 	var diffEvent *events.Event
		// 	if newInvites != nil {
		// 		diffEvent, err = invitesDiff(guildID, oldInvites, newInvites)
		// 		if err != nil {
		// 			raven.CaptureError(err, nil)
		// 			l.Error("failure generating invites diff", zap.Error(err))
		// 			return
		// 		}
		// 	}
		//
		// 	new, updated, removed := compareInvitesDiff(diffEvent.DiffInvites)
		// 	if len(new) <= 0 && len(updated) <= 0 && len(removed) <= 0 {
		// 		usedInvite := inviteDiffFindUsed(diffEvent.DiffInvites)
		// 		if usedInvite != nil {
		// 			extraEvent.GuildMemberAddExtra.UsedInvite = usedInvite
		// 		}
		//
		// 		// no change except possibly uses, do not send invites diff
		// 		diffEvent = nil
		// 	}
		//
		// 	if extraEvent != nil {
		// 		err, recoverable := eh.publisher.Publish(
		// 			context.TODO(),
		// 			extraEvent,
		// 		)
		// 		if err != nil {
		// 			raven.CaptureError(err, nil)
		// 			if !recoverable {
		// 				l.Fatal("unrecoverable publishing error, shutting down",
		// 					zap.Error(err),
		// 				)
		// 			}
		// 			l.Error("unable to publish event",
		// 				zap.Error(err),
		// 			)
		// 			return
		// 		}
		//
		// 		l.Debug("published member add extra event")
		// 	}
		//
		// 	if diffEvent != nil {
		// 		err, recoverable := eh.publisher.Publish(
		// 			context.TODO(),
		// 			diffEvent,
		// 		)
		// 		if err != nil {
		// 			raven.CaptureError(err, nil)
		// 			if !recoverable {
		// 				l.Fatal("unrecoverable publishing error, shutting down",
		// 					zap.Error(err),
		// 				)
		// 			}
		// 			l.Error("unable to publish event",
		// 				zap.Error(err),
		// 			)
		// 			return
		// 		}
		//
		// 		l.Debug("published invite diff event")
		// 	}
		//
		// }(event.GuildID, oldInvites, event.GuildMemberAdd)
	}
	if err != nil {
		raven.CaptureError(err, nil)
	}

	err, recoverable := eh.publisher.Publish(
		ctx,
		event,
	)
	if err != nil {
		raven.CaptureError(err, nil)
		if !recoverable {
			l.Fatal("unrecoverable publishing error, shutting down",
				zap.Error(err),
			)
		}
		l.Error("unable to publish event",
			zap.Error(err),
		)
		return
	}

	l.Debug("published event")

	if diffEvent != nil {
		err, recoverable = eh.publisher.Publish(
			context.TODO(),
			diffEvent,
		)
		if err != nil {
			raven.CaptureError(err, nil)
			if !recoverable {
				l.Fatal("unrecoverable publishing error, shutting down",
					zap.Error(err),
				)
			}
			l.Error("unable to publish event",
				zap.Error(err),
			)
			return
		}

		l.Debug("published diff event")
	}
}
