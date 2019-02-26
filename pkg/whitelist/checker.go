package whitelist

import (
	"sync"
	"time"

	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

type Checker struct {
	redis    *redis.Client
	logger   *zap.Logger
	interval time.Duration
	enable   bool

	whitelist     map[string]interface{}
	whitelistLock sync.RWMutex
	blacklist     map[string]interface{}
	blacklistLock sync.RWMutex
}

func NewChecker(
	redis *redis.Client,
	logger *zap.Logger,
	interval time.Duration,
	enable bool,
) *Checker {
	return &Checker{
		redis:    redis,
		logger:   logger,
		interval: interval,
		enable:   enable,
	}
}

func (c *Checker) Start() error {
	var err error

	c.whitelistLock.Lock()
	c.whitelist, err = c.get(whitelistKey)
	c.whitelistLock.Unlock()
	if err != nil && err != redis.Nil {
		return err
	}

	c.blacklistLock.Lock()
	c.blacklist, err = c.get(blacklistKey)
	c.blacklistLock.Unlock()
	if err != nil && err != redis.Nil {
		return err
	}

	go func() {
		var err error
		var whitelist, blacklist map[string]interface{}
		for {
			time.Sleep(c.interval)

			whitelist, err = c.get(whitelistKey)
			if err != nil && err != redis.Nil {
				c.logger.Error("failed to retrieve whitelist", zap.Error(err))
			} else {
				c.whitelistLock.Lock()
				c.whitelist = whitelist
				c.whitelistLock.Unlock()
			}

			blacklist, err = c.get(blacklistKey)
			if err != nil && err != redis.Nil {
				c.logger.Error("failed to retrieve blacklist", zap.Error(err))
			} else {
				c.blacklistLock.Lock()
				c.blacklist = blacklist
				c.blacklistLock.Unlock()
			}

			c.logger.Debug("cached whitelist and blacklist")
		}
	}()

	return nil
}

func (c *Checker) IsWhitelisted(guildID string) bool {
	if !c.enable {
		return true
	}

	c.whitelistLock.RLock()
	defer c.whitelistLock.RUnlock()

	if c.whitelist == nil {
		return false
	}

	_, ok := c.whitelist[guildID]
	return ok
}

func (c *Checker) IsBlacklisted(guildID string) bool {
	if !c.enable {
		return false
	}

	c.blacklistLock.RLock()
	defer c.blacklistLock.RUnlock()

	if c.blacklist == nil {
		return false
	}

	_, ok := c.blacklist[guildID]
	return ok
}
