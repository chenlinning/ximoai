package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type desktopSessionStore struct {
	rdb *redis.Client
}

type usedDesktopRefreshRecord struct {
	SessionKey     string `json:"session_key"`
	RefreshPayload string `json:"refresh_payload"`
}

var consumeDesktopAuthorizationCodeScript = redis.NewScript(`
local value = redis.call("GET", KEYS[1])
if not value then
	return false
end
redis.call("DEL", KEYS[1])
return value
`)

var storeDesktopSessionScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 1 or redis.call("EXISTS", KEYS[2]) == 1 then
	return 0
end
redis.call("PSETEX", KEYS[1], ARGV[3], ARGV[1])
redis.call("PSETEX", KEYS[2], ARGV[3], ARGV[2])
return 1
`)

var rotateDesktopRefreshScript = redis.NewScript(`
local old_refresh = redis.call("GET", KEYS[2])
if not old_refresh then
	local used = redis.call("GET", KEYS[3])
	if not used then
		return 0
	end
	local used_ok, used_record = pcall(cjson.decode, used)
	if used_ok and used_record["session_key"] then
		local replay_session = redis.call("GET", used_record["session_key"])
		if replay_session then
			local session_ok, session_record = pcall(cjson.decode, replay_session)
			if session_ok and session_record["current_refresh_key"] then
				redis.call("DEL", session_record["current_refresh_key"])
			end
			redis.call("DEL", used_record["session_key"])
		end
	end
	return 2
end

local session = redis.call("GET", KEYS[1])
if not session then
	return 0
end
local session_ok, session_record = pcall(cjson.decode, session)
if not session_ok or session_record["current_refresh_key"] ~= KEYS[2] then
	return 0
end
if redis.call("EXISTS", KEYS[4]) == 1 then
	return 3
end

local used_record = cjson.encode({session_key = KEYS[1], refresh_payload = old_refresh})
redis.call("DEL", KEYS[2])
redis.call("PSETEX", KEYS[3], ARGV[3], used_record)
redis.call("PSETEX", KEYS[4], ARGV[3], ARGV[1])
redis.call("PSETEX", KEYS[1], ARGV[3], ARGV[2])
return 1
`)

var revokeDesktopSessionScript = redis.NewScript(`
local session = redis.call("GET", KEYS[1])
if not session then
	return 0
end
local ok, record = pcall(cjson.decode, session)
if ok and record["current_refresh_key"] then
	redis.call("DEL", record["current_refresh_key"])
end
redis.call("DEL", KEYS[1])
return 1
`)

var revokeDesktopRefreshReplayScript = redis.NewScript(`
local used = redis.call("GET", KEYS[1])
if not used then
	return 0
end
local used_ok, used_record = pcall(cjson.decode, used)
if not used_ok or not used_record["session_key"] then
	return 0
end
local session = redis.call("GET", used_record["session_key"])
if session then
	local session_ok, session_record = pcall(cjson.decode, session)
	if session_ok and session_record["current_refresh_key"] then
		redis.call("DEL", session_record["current_refresh_key"])
	end
	redis.call("DEL", used_record["session_key"])
end
return 1
`)

func NewDesktopSessionStore(rdb *redis.Client) service.DesktopSessionStore {
	return &desktopSessionStore{rdb: rdb}
}

func (s *desktopSessionStore) StoreAuthorizationCode(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	return s.rdb.SetNX(ctx, key, payload, ttl).Result()
}

func (s *desktopSessionStore) ConsumeAuthorizationCode(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.rdb == nil {
		return "", false, redis.Nil
	}
	return desktopScriptStringResult(consumeDesktopAuthorizationCodeScript.Run(ctx, s.rdb, []string{key}).Result())
}

func (s *desktopSessionStore) StoreSession(ctx context.Context, sessionKey string, sessionPayload []byte, refreshKey string, refreshPayload []byte, ttl time.Duration) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	result, err := storeDesktopSessionScript.Run(ctx, s.rdb, []string{sessionKey, refreshKey}, sessionPayload, refreshPayload, desktopTTLMillis(ttl)).Int64()
	return result == 1, err
}

func (s *desktopSessionStore) GetSession(ctx context.Context, key string) (string, bool, error) {
	return s.get(ctx, key)
}

func (s *desktopSessionStore) GetRefresh(ctx context.Context, key string) (string, bool, error) {
	return s.get(ctx, key)
}

func (s *desktopSessionStore) GetUsedRefresh(ctx context.Context, key string) (string, bool, error) {
	raw, ok, err := s.get(ctx, key)
	if err != nil || !ok {
		return "", ok, err
	}
	var record usedDesktopRefreshRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || record.SessionKey == "" || record.RefreshPayload == "" {
		return "", false, nil
	}
	return record.RefreshPayload, true, nil
}

func (s *desktopSessionStore) RotateRefresh(ctx context.Context, sessionKey, oldRefreshKey, usedRefreshKey, newRefreshKey string, newRefreshPayload, newSessionPayload []byte, ttl time.Duration) (service.DesktopRefreshRotationResult, error) {
	if s == nil || s.rdb == nil {
		return service.DesktopRefreshInvalid, redis.Nil
	}
	result, err := rotateDesktopRefreshScript.Run(ctx, s.rdb, []string{sessionKey, oldRefreshKey, usedRefreshKey, newRefreshKey}, newRefreshPayload, newSessionPayload, desktopTTLMillis(ttl)).Int64()
	if err != nil {
		return service.DesktopRefreshInvalid, err
	}
	return service.DesktopRefreshRotationResult(result), nil
}

func (s *desktopSessionStore) RevokeSession(ctx context.Context, sessionKey string) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	result, err := revokeDesktopSessionScript.Run(ctx, s.rdb, []string{sessionKey}).Int64()
	return result == 1, err
}

func (s *desktopSessionStore) RevokeRefreshReplay(ctx context.Context, usedRefreshKey string) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	result, err := revokeDesktopRefreshReplayScript.Run(ctx, s.rdb, []string{usedRefreshKey}).Int64()
	return result == 1, err
}

func (s *desktopSessionStore) StoreDPoPProof(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if s == nil || s.rdb == nil {
		return false, redis.Nil
	}
	return s.rdb.SetNX(ctx, key, 1, ttl).Result()
}

func (s *desktopSessionStore) Ping(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return redis.Nil
	}
	return s.rdb.Ping(ctx).Err()
}

func (s *desktopSessionStore) get(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.rdb == nil {
		return "", false, redis.Nil
	}
	raw, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	return raw, err == nil, err
}

func desktopScriptStringResult(raw any, err error) (string, bool, error) {
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil || raw == nil || raw == false {
		return "", false, err
	}
	return fmt.Sprint(raw), true, nil
}

func desktopTTLMillis(ttl time.Duration) int64 {
	if ttl < time.Millisecond {
		return 1
	}
	return ttl.Milliseconds()
}
