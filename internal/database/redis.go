package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),     // e.g. "localhost:6379"
		Password: os.Getenv("REDIS_PASSWORD"), // "" if no password
		DB:       0,
	})

	// Ping to verify connection
	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	fmt.Println("Redis connected successfully")
}
