package publisher

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Publisher publishes Events to AMQP
type Publisher struct {
	amqpConnection   *amqp.Connection
	amqpExchangeName string
	eventTTL         time.Duration

	amqpChannel    *amqp.Channel
	amqpExpiration string
}

// NewPublisher creates new Publisher
func NewPublisher(
	amqpConnection *amqp.Connection,
	amqpExchangeName string,
	eventTTL time.Duration,
) (*Publisher, error) {
	publisher := &Publisher{
		amqpConnection:   amqpConnection,
		amqpExchangeName: amqpExchangeName,
		eventTTL:         eventTTL,
	}

	if eventTTL > 0 {
		publisher.amqpExpiration = strconv.Itoa(int(eventTTL.Seconds() * 1000))
	}

	err := publisher.init()
	if err != nil {
		return nil, err
	}

	return publisher, nil
}

// init initialises the channel and exchange
func (p *Publisher) init() error {
	var err error
	p.amqpChannel, err = p.amqpConnection.Channel()
	if err != nil {
		return errors.Wrap(err, "cannot open channel")
	}

	err = p.amqpChannel.ExchangeDeclare(
		p.amqpExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "cannot declare exchange")
	}

	return nil
}

// Publish publishes a specific event
func (p *Publisher) Publish(routingKey string, body json.RawMessage) error {
	if p.amqpChannel == nil {
		return errors.New("no channel setup")
	}

	err := p.amqpChannel.Publish(
		p.amqpExchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "application/json",
			ContentEncoding: "",
			Body:            body,
			DeliveryMode:    amqp.Transient,
			Priority:        0,
			Expiration:      p.amqpExpiration,
		},
	)

	return err
}
