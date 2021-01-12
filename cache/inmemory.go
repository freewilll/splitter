package cache

import (
	"github.com/freewilll/splitter/database"
	"github.com/freewilll/splitter/ledger"
)

// InMemoryCache implements the Cache interface for an in memory cache
type InMemoryCache struct {
	entries map[int]ledger.Balance
}

// NewInMemoryCache creates an instance of InMemoryCache
func NewInMemoryCache() Cache {
	cache := new(InMemoryCache)
	cache.entries = make(map[int]ledger.Balance)
	return cache
}

// SetBalance sets the userID/balance key/value
func (c *InMemoryCache) SetBalance(balance ledger.Balance, userID int) {
	c.entries[userID] = balance
}

// GetBalance gets the userID/balance key/value
func (c *InMemoryCache) GetBalance(_ database.Database, userID int) ledger.Balance {
	return c.entries[userID]
}
