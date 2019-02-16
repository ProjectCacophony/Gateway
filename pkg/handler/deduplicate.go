package handler

import (
	"errors"
	"time"
)

const (
	expiry = 15 * time.Minute
)

func (eh *EventHandler) IsDuplicate(key string) (bool, error) {
	if key == "" {
		return false, errors.New("passed key is empty")
	}

	// insert if not exists
	set, err := eh.redisClient.SetNX(key, true, expiry).Result()
	if err != nil {
		return false, err
	}

	return !set, nil
}
