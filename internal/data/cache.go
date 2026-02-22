package data

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

// UserCache wraps go-cache to provide type-safe user caching
type UserCache struct {
	c *cache.Cache
}

// NewUserCache creates a new user cache with the specified TTL and cleanup interval
func NewUserCache(defaultExpiration, cleanupInterval time.Duration) *UserCache {
	return &UserCache{
		c: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Get retrieves a user from the cache if it exists and hasn't expired
func (uc *UserCache) Get(userID int64) (*User, bool) {
	key := uc.key(userID)
	val, found := uc.c.Get(key)
	if !found {
		return nil, false
	}

	// Type assert and return a copy to prevent external modifications
	user, ok := val.(*User)
	if !ok {
		return nil, false
	}

	// Return a copy to prevent external modifications
	userCopy := *user
	return &userCopy, true
}

// Set stores a user in the cache with the default expiration time
func (uc *UserCache) Set(userID int64, user *User) {
	key := uc.key(userID)
	// Create a copy to prevent external modifications
	userCopy := *user
	uc.c.Set(key, &userCopy, cache.DefaultExpiration)
}

// Delete removes a user from the cache
func (uc *UserCache) Delete(userID int64) {
	key := uc.key(userID)
	uc.c.Delete(key)
}

// key generates a cache key for a user ID
func (uc *UserCache) key(userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}
