package main

import (
	"bytes"
	"io/ioutil"

	"gitlab.com/project-d-collab/dhelpers"

	"net/url"

	"github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// sends an event to the given AWS Endpoint
func SendEvent(event dhelpers.Event, endpoint string) error {
	// pack the event
	marshalled, err := jsoniter.Marshal(event)
	if err != nil {
		return err
	}

	// send to API Gateway
	req := ApiRequest
	req.URL, err = url.Parse(EndpointBaseUrl + endpoint +
		"?type=" + url.QueryEscape(string(event.Type)))
	if err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(bytes.NewReader(marshalled))

	resp, err := HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New("error sending request: " + string(body))
	}
	return nil
}
