package events

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

// Type defines the type for a Event
type Type string

// defines various Event Types
const (
	MessageCreateEventType Type = "message_create"
)

// Event represents an Event
type Event struct {
	Type       Type
	ReceivedAt time.Time
	BotUserID  string

	// discordgo event data
	MessageCreate *discordgo.MessageCreate
}

// GenerateRoutingKey generates an Routing Key for AMQP based on a Event Type
func GenerateRoutingKey(eventType Type) string {
	return fmt.Sprintf("cacophony.discord.%s", eventType)
}

// GenerateEventFromDiscordgoEvent generates an Event from a Discordgo Event
func GenerateEventFromDiscordgoEvent(botUserID string, eventItem interface{}) (*Event, error) {
	event := &Event{
		ReceivedAt: time.Now(),
		BotUserID:  botUserID,
	}

	switch t := eventItem.(type) {
	case *discordgo.MessageCreate:
		event.Type = MessageCreateEventType
		event.MessageCreate = t

		return event, nil
	}

	return nil, errors.New("event type is not supported")
}
