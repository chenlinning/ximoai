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
local ok, record = pcall(cjson.decode, value)
if not ok or record["sso_audience"] ~= ARGV[1] then
	return false
end
redis.call("DEL", KEYS[1])
return value
`)

var consumeWorkbenchControlGrantForUserScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return false
end
local ok, record = pcall(cjson.decode, value)
if not ok or record["sso_audience"] ~= ARGV[1] or tonumber(record["user_id"]) ~= tonumber(ARGV[2]) then
	return false
end
redis.call("DEL", KEYS[1])
return value
`)

var revokeWorkbenchControlGrantScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return 0
end
local ok, record = pcall(cjson.decode, value)
if not ok or record["sso_audience"] ~= ARGV[1] then
	return 0
end
redis.call("DEL", KEYS[1])
return 1
`)

var revokeWorkbenchControlGrantForUserScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return 0
end
local ok, record = pcall(cjson.decode, value)
if not ok or record["sso_audience"] ~= ARGV[1] or tonumber(record["user_id"]) ~= tonumber(ARGV[2]) then
	return 0
end
redis.call("DEL", KEYS[1])
return 1
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

func (s *workbenchControlTokenStore) ConsumeGrant(ctx context.Context, key, ssoAudience string) (string, bool, error) {
	if s == nil || s.rdb == nil {
		return "", false, redis.Nil
	}
	raw, err := consumeWorkbenchControlGrantScript.Run(ctx, s.rdb, []string{key}, ssoAudience).Result()
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

func (s *workbenchControlTokenStore) ConsumeGrantForUser(ctx context.Context, key, ssoAudience string, userID int64) (string, bool, error) {
	if s == nil || s.rdb == nil {
		return "", false, redis.Nil
	}
	raw, err := consumeWorkbenchControlGrantForUserScript.Run(ctx, s.rdb, []string{key}, ssoAudience, userID).Result()
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

func (s *workbenchControlTokenStore) RevokeGrant(ctx context.Context, key, ssoAudience string) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	result, err := revokeWorkbenchControlGrantScript.Run(ctx, s.rdb, []string{key}, ssoAudience).Int64()
	return result == 1, err
}

func (s *workbenchControlTokenStore) RevokeGrantForUser(ctx context.Context, key, ssoAudience string, userID int64) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	result, err := revokeWorkbenchControlGrantForUserScript.Run(ctx, s.rdb, []string{key}, ssoAudience, userID).Int64()
	return result == 1, err
}

func (s *workbenchControlTokenStore) Ping(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return redis.Nil
	}
	return s.rdb.Ping(ctx).Err()
}
