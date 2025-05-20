package server

import (
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
)

func newRedisClient() (*redis.Client, error) {
	redisURL, err := getRedisURL()
	if err != nil {
		return nil, err
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}
	redis := redis.NewClient(opt)
	_, err = redis.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	return redis, nil
}

func getRedisURL() (string, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return "", fmt.Errorf("REDIS_URL must be set")
	}
	return redisURL, nil
}
