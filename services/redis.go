package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// InitRedis inisialisasi koneksi ke Redis Enterprise Cloud
func InitRedis() error {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisUser := os.Getenv("REDIS_USERNAME")
	redisPass := os.Getenv("REDIS_PASSWORD")

	if redisHost == "" {
		log.Println("[Warning] REDIS_HOST tidak disetel.")
		return nil
	}

	addr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: redisUser,
		Password: redisPass,
		DB:       0,
	})

	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	RedisClient = client
	log.Println("[Redis] Berhasil terhubung ke cluster.")
	return nil
}
