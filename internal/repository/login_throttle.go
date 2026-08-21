package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type LoginThrottler interface {
	FailCount(ctx context.Context, username string) (int, error)
	RecordFailure(ctx context.Context, username string) error
	Reset(ctx context.Context, username string) error
	PenaltyTTL(ctx context.Context, username string) (time.Duration, error)
}

type redisLoginThrottler struct {
	client *redis.Client
}

func NewRedisLoginThrottler(client *redis.Client) LoginThrottler {
	return &redisLoginThrottler{client: client}
}

func failKey(username string) string {
	return "login_fail:" + username
}

func penaltyKey(username string) string {
	return "penalty:" + username
}

func (t *redisLoginThrottler) FailCount(ctx context.Context, username string) (int, error) {
	count, err := t.client.Get(ctx, failKey(username)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

func (t *redisLoginThrottler) RecordFailure(ctx context.Context, username string) error {
	key := failKey(username)

	count, err := t.client.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	if count < 5 {
		if count == 1 {
			return t.client.Expire(ctx, key, time.Minute).Err()
		}
		return nil
	}

	penaltyScript := redis.NewScript(`
	local penalty = tonumber(redis.call('GET', KEYS[2]))
	if not penalty then
		penalty = 60
	else
		penalty = penalty * 2
		if penalty > 1800 then
			penalty = 1800
		end
	end
	redis.call('SET', KEYS[2], penalty)
	redis.call('EXPIRE', KEYS[1], penalty)
	return penalty
`)
	_, err = penaltyScript.Run(ctx, t.client, []string{key, penaltyKey(username)}).Result()
	return err
}

func (t *redisLoginThrottler) Reset(ctx context.Context, username string) error {
	return t.client.Del(ctx, failKey(username)).Err()
}

func (t *redisLoginThrottler) PenaltyTTL(ctx context.Context, username string) (time.Duration, error) {
	return t.client.TTL(ctx, failKey(username)).Result()
}
