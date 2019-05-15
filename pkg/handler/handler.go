package handler

import (
	"context"

	"github.com/bwmarrin/discordgo"
	raven "github.com/getsentry/raven-go"
	"github.com/go-redis/redis"
	"gitlab.com/Cacophony/Gateway/pkg/whitelist"
	"gitlab.com/Cacophony/go-kit/events"
	"go.uber.org/zap"
)

// EventHandler handles discord events and puts them into rabbitMQ
type EventHandler struct {
	logger      *zap.Logger
	redisClient *redis.Client
	publisher   *events.Publisher
	checker     *whitelist.Checker
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	logger *zap.Logger,
	redisClient *redis.Client,
	publisher *events.Publisher,
	checker *whitelist.Checker,
) *EventHandler {
	return &EventHandler{
		logger:      logger,
		redisClient: redisClient,
		publisher:   publisher,
		checker:     checker,
	}
}

// OnDiscordEvent receives discord events
func (eh *EventHandler) OnDiscordEvent(session *discordgo.Session, eventItem interface{}) {
	var err error

	if session.State == nil || session.State.User == nil {
		return
	}

	event, expiration, err := events.GenerateEventFromDiscordgoEvent(
		session.State.User.ID,
		eventItem,
	)
	if err != nil {
		raven.CaptureError(err, nil)
		eh.logger.Debug("unable to generate event",
			zap.Error(err),
			zap.Any("event", eventItem),
		)
		return
	}

	if event == nil {
		return
	}

	l := eh.logger.With(
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)),
		zap.String("event_guild_id", event.GuildID),
		zap.String("event_user_id", event.UserID),
		zap.String("event_bot_user_id", event.BotUserID),
	)

	if event.GuildID != "" &&
		(!eh.checker.IsWhitelisted(event.GuildID) ||
			eh.checker.IsBlacklisted(event.GuildID)) {
		l.Debug("skipping event because guild is not whitelisted")
		return
	}

	duplicate, err := eh.IsDuplicate(event.CacheKey, expiration)
	if err != nil {
		raven.CaptureError(err, nil)
		l.Debug("unable to deduplicate event",
			zap.Error(err),
			zap.Any("event", eventItem),
		)
		return
	}

	if duplicate {
		l.Debug("skipping event, as it is a duplicate")
		return
	}

	err = eh.publisher.Publish(
		context.TODO(),
		event,
	)
	if err != nil {
		raven.CaptureError(err, nil)
		l.Error("unable to publish event",
			zap.Error(err),
		)
		return
	}

	eh.logger.Debug("published event")
}
