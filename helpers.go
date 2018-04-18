package main

import (
	"gitlab.com/project-d-collab/dhelpers"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// sends an event to the given AWS Endpoint
func SendEvent(start, receive time.Time, theType dhelpers.EventType, event interface{}, function string) error {
	// pack the event data
	marshalled, err := jsoniter.Marshal(event)
	if err != nil {
		return err
	}

	// create event container
	eventContainer := dhelpers.EventContainer{
		Type:           theType,
		ReceivedAt:     receive,
		GatewayStarted: start,
		Data:           marshalled,
	}
	// pack the event container
	marshalledContainer, err := jsoniter.Marshal(eventContainer)
	if err != nil {
		return err
	}

	// invoke lambda
	_, err = lambdaClient.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String(function),
		InvocationType: aws.String("Event"), // Async
		Payload:        marshalledContainer,
	})
	if err != nil {
		return errors.New("error invoking lambda: " + err.Error())
	}
	return nil
}
