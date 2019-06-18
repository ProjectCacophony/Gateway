package whitelist

import (
	"strings"
)

const (
	whitelistKey = "cacophony.whitelist.whitelist"
	blacklistKey = "cacophony.whitelist.blacklist"
)

func (c *Checker) get(key string) ([]string, map[string]interface{}, error) {
	res, err := c.redis.Get(key).Result()
	if err != nil {
		return nil, nil, err
	}

	ids := strings.Split(res, ";")
	return ids, sliceIntoMap(ids), nil
}

func sliceIntoMap(list []string) map[string]interface{} {
	ids := make(map[string]interface{})

	for _, item := range list {
		ids[item] = true
	}

	return ids
}
