// nolint: wrapcheck
package adapter

import (
	"context"
	"time"

	"github.com/joelrose/redisstore"
	"github.com/redis/go-redis/v9"
)

type GoRedisAdapter struct {
	redis.UniversalClient
}

var _ redisstore.Client = (*GoRedisAdapter)(nil)

func WithGoRedis(client redis.UniversalClient) *GoRedisAdapter {
	return &GoRedisAdapter{client}
}

func (a *GoRedisAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := a.UniversalClient.Get(ctx, key).Result()
	return []byte(val), err
}

func (a *GoRedisAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.UniversalClient.Set(ctx, key, value, 0).Err()
}

func (a *GoRedisAdapter) Del(ctx context.Context, key string) error {
	return a.UniversalClient.Del(ctx, key).Err()
}
