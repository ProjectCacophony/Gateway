package main

import (
	"fmt"

	"time"

	"gitlab.com/project-d-collab/dhelpers"
)

// Returns true if the event is new, returns false if the event has already been handled by other gateways
func IsNewEvent(theType dhelpers.EventType, id string) (new bool) {
	key := "project-d:gateway:event-" + string(theType) + "-" + id

	set, err := RedisClient.SetNX(key, true, time.Minute*15).Result()
	if err != nil {
		fmt.Println("error doing deduplication:", err.Error())
		return true
	}
	if !set {
		return false
	}

	return true
}
