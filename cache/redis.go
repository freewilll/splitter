package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/freewilll/splitter/database"
	"github.com/freewilll/splitter/ledger"

	redis "github.com/go-redis/redis/v8"
)

// Config is the redis configuration
type Config struct {
	Addr     string
	Password string
	Db       int
}

var ctx = context.Background()

var cacheEntryTTL = 5 * time.Second

// RedisCache implements the Cache interface for redis
type RedisCache struct {
	config Config
}

// NewRedisCache creates an instance of RedisCache
func NewRedisCache(config Config) Cache {
	return new(RedisCache)

}

// connect returns a Redis client
func (r RedisCache) connect() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     r.config.Addr,
		Password: r.config.Password,
		DB:       r.config.Db,
	})
}

// makeKey makes a key from a userID
func (r RedisCache) makeKey(userID int) string {
	return fmt.Sprintf("key-%d", userID)
}

// setBalanceWithRdb writes the balance to redis for a userID
func (r RedisCache) setBalanceWithRdb(rdb *redis.Client, balance ledger.Balance, userID int) {
	key := r.makeKey(userID)

	value, err := json.Marshal(balance)
	if err != nil {
		panic(err)
	}

	err = rdb.Set(ctx, key, value, cacheEntryTTL).Err()
	if err != nil {
		panic(err)
	}
}

// SetBalance sets the userID/balance key/value in redis
func (r RedisCache) SetBalance(balance ledger.Balance, userID int) {
	rdb := r.connect()
	defer rdb.Close()
	r.setBalanceWithRdb(rdb, balance, userID)
}

// GetBalance gets the userID/balance key/value in redis. If the key doesn't exist,
// the expenses are read from the database, calculated and then written to the cache.
// A TTL ensures data doesn't remain stail in case of race conditions writing the
// data concurrently.
func (r RedisCache) GetBalance(db database.Database, userID int) ledger.Balance {
	rdb := r.connect()
	defer rdb.Close()

	key := r.makeKey(userID)
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		dbh := db.Connect()
		defer dbh.Close()

		expenses := dbh.GetExpenses(userID)
		balance := ledger.CalculateBalance(expenses, userID)
		r.setBalanceWithRdb(rdb, balance, userID)

		return balance
	} else if err != nil {
		panic(err)
	} else {
		var balance ledger.Balance
		err := json.Unmarshal([]byte(val), &balance)
		if err != nil {
			log.Fatalf("Unable to decode and parse json from cache")
		}

		return balance
	}
}
