package redis

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

var (
	Client *redis.Client
)

func Init(redisAddr string, redisPass string) {

	Client = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	_, err := Client.Ping(context.TODO()).Result()
	if err != nil {
		panic(err)
	}
}

func ObtainLock(key string, expiration time.Duration) error {
	val, err := Client.SetNX(context.TODO(), key, 1, expiration).Result()
	if err != nil {
	}
	if !val {
		return errors.New("lock is been taken")
	}
	return nil
}

func ReleaseLock(key string) {
	Client.Del(context.TODO(), key)
}
