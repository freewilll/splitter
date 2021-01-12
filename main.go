package main

import (
	"flag"
	"log"

	"github.com/freewilll/splitter/api"
	"github.com/freewilll/splitter/cache"
	"github.com/freewilll/splitter/database"
)

// General flags
var createSchema = flag.Bool("create-schema", false, "create schema")

// Postgresql flags
var dbHost = flag.String("db-host", "localhost", "database host")
var dbPort = flag.Int("db-port", 5432, "database port")
var dbUser = flag.String("db-user", "postgres", "database user")
var dbPassword = flag.String("db-password", "stream", "database password")
var dbName = flag.String("db-name", "postgres", "database name")

// Redis flags
var cacheAddr = flag.String("cache-addr", "localhost:6379", "redis cache address")
var cachePassword = flag.String("cache-password", "", "redis cache password")
var cacheDb = flag.Int("cache-db", 0, "redis cache db")

func main() {
	flag.Parse()

	// Configure Postgresql
	dbConfig := database.Config{
		Host:     *dbHost,
		Port:     *dbPort,
		User:     *dbUser,
		Password: *dbPassword,
		Name:     *dbName,
	}
	db := database.NewPgDatabase(dbConfig)

	// Create a schema is desired
	if *createSchema {
		dbh := db.Connect()
		dbh.CreateSchema()
		dbh.Close()
		log.Println("Database schema has been created")
		return
	}

	// Configure Redis
	cacheConfig := cache.Config{
		Addr:     *cacheAddr,
		Password: *cachePassword,
		Db:       *cacheDb,
	}
	cache := cache.NewRedisCache(cacheConfig)

	// All systems are go
	api.NewAPI(db, cache).Serve()
}
