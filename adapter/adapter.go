// nolint: wrapcheck
package adapter

import (
	"context"
	"fmt"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/joelrose/redisstore"
	goredis "github.com/redis/go-redis/v9"
)

type GoRedisAdapter struct {
	goredis.UniversalClient
}

var _ redisstore.Client = (*GoRedisAdapter)(nil)

func UseGoRedis(client goredis.UniversalClient) *GoRedisAdapter {
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

type RedigoAdapter struct {
	*redigo.Pool
}

var _ redisstore.Client = (*RedigoAdapter)(nil)

func UseRedigo(pool *redigo.Pool) *RedigoAdapter {
	return &RedigoAdapter{pool}
}

func (a *RedigoAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	conn, err := a.Pool.GetContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting connection from pool: %v", err)
	}
	defer conn.Close()

	val, err := redigo.DoContext(conn, ctx, "GET", key)
	if err != nil {
		return nil, fmt.Errorf("getting value from redis: %v", err)
	}

	v, ok := val.([]byte)
	if !ok {
		return nil, fmt.Errorf("value is not a []byte: %v", val)
	}

	return v, nil
}

func (a *RedigoAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	conn, err := a.Pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("getting connection from pool: %v", err)
	}
	defer conn.Close()

	_, err = redigo.DoContext(conn, ctx, "SET", key, value)
	if err != nil {
		return fmt.Errorf("setting value in redis: %v", err)
	}

	return nil
}

func (a *RedigoAdapter) Del(ctx context.Context, key string) error {
	conn, err := a.Pool.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("getting connection from pool: %v", err)
	}
	defer conn.Close()

	_, err = redigo.DoContext(conn, ctx, "DEL", key)
	if err != nil {
		return fmt.Errorf("deleting value from redis: %v", err)
	}

	return nil
}
