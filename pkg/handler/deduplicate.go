package handler

import (
	"errors"
	"time"
)

func (eh *EventHandler) IsDuplicate(key string, expiration time.Duration) (bool, error) {
	if key == "" {
		return false, errors.New("passed key is empty")
	}

	// insert if not exists
	set, err := eh.redisClient.SetNX(key, true, expiration).Result()
	if err != nil {
		return false, err
	}

	return !set, nil
}
