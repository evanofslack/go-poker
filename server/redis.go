package server

import (
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

func newRedisClient() *redis.Client {
	redisURL := getRedisURL()
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}
	redis := redis.NewClient(opt)
	_, err = redis.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	return redis
}

func getRedisURL() string {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	var redisURL = os.Getenv("REDIS_URL")
	log.Println(redisURL)
	if redisURL == "" {
		log.Panic("$REDIS_URL must be set")
	}
	return redisURL
}
