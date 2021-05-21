package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
)

func Publish(ctx context.Context, channel string, message interface{}) error {
	// Publish a message.
	err := Client.Publish(ctx, channel, message).Err()
	if err != nil {
		return err
	}
	return nil
}

func Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return Client.Subscribe(ctx, channels...)
}
