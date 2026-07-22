package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type workbenchControlTokenStore struct {
	rdb *redis.Client
}

var consumeWorkbenchControlGrantScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return false
end
redis.call("DEL", KEYS[1])
return value
`)

func NewWorkbenchControlTokenStore(rdb *redis.Client) service.WorkbenchControlGrantStore {
	return &workbenchControlTokenStore{rdb: rdb}
}

func (s *workbenchControlTokenStore) StoreGrant(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	return s.rdb.SetNX(ctx, key, payload, ttl).Result()
}

func (s *workbenchControlTokenStore) ConsumeGrant(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.rdb == nil {
		return "", false, redis.Nil
	}
	raw, err := consumeWorkbenchControlGrantScript.Run(ctx, s.rdb, []string{key}).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if raw == nil || raw == false {
		return "", false, nil
	}
	return fmt.Sprint(raw), true, nil
}

func (s *workbenchControlTokenStore) DeleteGrant(ctx context.Context, key string) error {
	if s == nil || s.rdb == nil {
		return redis.Nil
	}
	return s.rdb.Del(ctx, key).Err()
}

func (s *workbenchControlTokenStore) Ping(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return redis.Nil
	}
	return s.rdb.Ping(ctx).Err()
}
