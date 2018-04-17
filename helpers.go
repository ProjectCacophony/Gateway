package main

import (
	"bytes"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// checks if a message content matches the requirements of the routing rule
func MatchMessageRequirements(routingEntry RoutingEntry, content string) (match bool) {
	// match beginning if beginning is set
	if routingEntry.Beginning != "" {
		if routingEntry.CaseSensitive {
			if routingEntry.DoNotPrependPrefix {
				if !strings.HasPrefix(content, routingEntry.Beginning) {
					return false
				}
			} else {
				if !strings.HasPrefix(content, PREFIX+routingEntry.Beginning) {
					return false
				}
			}
		} else {
			if routingEntry.DoNotPrependPrefix {
				if !strings.HasPrefix(strings.ToLower(content), strings.ToLower(routingEntry.Beginning)) {
					return false
				}
			} else {
				if !strings.HasPrefix(strings.ToLower(content), PREFIX+strings.ToLower(routingEntry.Beginning)) {
					return false
				}
			}
		}
	}
	// match regex if regex is set
	if routingEntry.Regex != nil {
		if routingEntry.DoNotPrependPrefix {
			if !routingEntry.Regex.MatchString(content) {
				return false
			}
		} else {
			if !strings.HasPrefix(content, PREFIX) {
				return false
			}
			matchContent := strings.TrimLeft(content, PREFIX)
			if !routingEntry.Regex.MatchString(matchContent) {
				return false
			}
		}
	}

	return true
}

// sends an event to the given AWS Endpoint
func SendEvent(event DDiscordEvent, endpoint string) error {
	// pack the event
	marshalled, err := jsoniter.Marshal(event)
	if err != nil {
		return err
	}

	// send to API Gateway
	req := ApiRequest
	req.URL, err = url.Parse(EndpointBaseUrl + endpoint + "?type=" + string(event.Type))
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
