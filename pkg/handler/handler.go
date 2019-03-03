package handler

import (
	"encoding/json"

	raven "github.com/getsentry/raven-go"

	"gitlab.com/Cacophony/Gateway/pkg/whitelist"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis"
	"gitlab.com/Cacophony/Gateway/pkg/publisher"
	"gitlab.com/Cacophony/go-kit/events"
	"go.uber.org/zap"
)

// EventHandler handles discord events and puts them into rabbitMQ
type EventHandler struct {
	logger      *zap.Logger
	redisClient *redis.Client
	publisher   *publisher.Publisher
	checker     *whitelist.Checker
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	logger *zap.Logger,
	redisClient *redis.Client,
	publisher *publisher.Publisher,
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
	var routingKey string

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

	if event.GuildID != "" &&
		(!eh.checker.IsWhitelisted(event.GuildID) ||
			eh.checker.IsBlacklisted(event.GuildID)) {
		eh.logger.Debug("skipping event because guild is not whitelisted",
			zap.String("type", string(event.Type)),
			zap.String("guild_id", event.GuildID),
		)
		return
	}

	duplicate, err := eh.IsDuplicate(event.ID, expiration)
	if err != nil {
		raven.CaptureError(err, nil)
		eh.logger.Debug("unable to deduplicate event",
			zap.Error(err),
			zap.Any("event", eventItem),
		)
		return
	}

	if duplicate {
		eh.logger.Debug("skipping event, as it is a duplicate",
			zap.String("routing_key", routingKey),
			zap.String("id", event.ID),
		)
		return
	}

	routingKey = events.GenerateRoutingKey(event.Type)

	body, err := json.Marshal(event)
	if err != nil {
		raven.CaptureError(err, nil)
		eh.logger.Error("unable to marshal event",
			zap.Error(err),
		)
		return
	}

	err = eh.publisher.Publish(
		routingKey,
		body,
	)
	if err != nil {
		raven.CaptureError(err, nil)
		eh.logger.Error("unable to publish event",
			zap.Error(err),
			zap.String("routing_key", routingKey),
		)
		return
	}

	eh.logger.Debug("published event",
		zap.String("routing_key", routingKey),
		zap.String("id", event.ID),
	)
}
