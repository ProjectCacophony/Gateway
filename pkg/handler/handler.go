package handler

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Gateway/pkg/kit/events"
	"gitlab.com/Cacophony/Gateway/pkg/publisher"
	"go.uber.org/zap"
)

// EventHandler handles discord events and puts them into rabbitMQ
type EventHandler struct {
	logger    *zap.Logger
	publisher *publisher.Publisher
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(
	logger *zap.Logger,
	publisher *publisher.Publisher,
) *EventHandler {
	return &EventHandler{
		logger:    logger,
		publisher: publisher,
	}
}

// OnDiscordEvent receives discord events
func (eh *EventHandler) OnDiscordEvent(session *discordgo.Session, eventItem interface{}) {
	var err error
	var routingKey string

	event, err := events.GenerateEventFromDiscordgoEvent(
		session.State.User.ID,
		eventItem,
	)
	if err != nil {
		eh.logger.Debug("unable to generate event",
			zap.Error(err),
		)
		return
	}

	routingKey = events.GenerateRoutingKey(event.Type)

	body, err := json.Marshal(event)
	if err != nil {
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
		eh.logger.Error("unable to publish event",
			zap.Error(err),
			zap.String("routing_key", routingKey),
		)
		return
	}

	eh.logger.Debug("published event",
		zap.String("routing_key", routingKey),
	)
}
