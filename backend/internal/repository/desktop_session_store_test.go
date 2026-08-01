package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestDesktopSessionStoreAuthorizationCodeIsOneTime(t *testing.T) {
	store, _ := newDesktopSessionRedisStore(t)
	ctx := context.Background()

	stored, err := store.StoreAuthorizationCode(ctx, "desktop:authorization:code", []byte(`{"user_id":1}`), time.Minute)
	require.NoError(t, err)
	require.True(t, stored)

	stored, err = store.StoreAuthorizationCode(ctx, "desktop:authorization:code", []byte(`duplicate`), time.Minute)
	require.NoError(t, err)
	require.False(t, stored)

	raw, ok, err := store.ConsumeAuthorizationCode(ctx, "desktop:authorization:code")
	require.NoError(t, err)
	require.True(t, ok)
	require.JSONEq(t, `{"user_id":1}`, raw)

	_, ok, err = store.ConsumeAuthorizationCode(ctx, "desktop:authorization:code")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDesktopSessionStoreRefreshRotationAndReplayRevocation(t *testing.T) {
	store, _ := newDesktopSessionRedisStore(t)
	ctx := context.Background()
	const (
		sessionKey = "desktop:session:session-1"
		oldKey     = "desktop:refresh:old"
		usedKey    = "desktop:refresh:used:old"
		newKey     = "desktop:refresh:new"
	)
	oldRefresh := []byte(`{"session_id":"session-1","user_id":1,"token_version":2,"jkt":"thumbprint","expires_at":9999999999}`)
	oldSession := desktopSessionStorePayload(t, "session-1", oldKey)

	stored, err := store.StoreSession(ctx, sessionKey, oldSession, oldKey, oldRefresh, time.Hour)
	require.NoError(t, err)
	require.True(t, stored)

	newRefresh := []byte(`{"session_id":"session-1","user_id":1,"token_version":2,"jkt":"thumbprint","expires_at":9999999999}`)
	newSession := desktopSessionStorePayload(t, "session-1", newKey)
	result, err := store.RotateRefresh(ctx, sessionKey, oldKey, usedKey, newKey, newRefresh, newSession, time.Hour)
	require.NoError(t, err)
	require.Equal(t, service.DesktopRefreshRotated, result)

	_, ok, err := store.GetRefresh(ctx, oldKey)
	require.NoError(t, err)
	require.False(t, ok)
	raw, ok, err := store.GetUsedRefresh(ctx, usedKey)
	require.NoError(t, err)
	require.True(t, ok)
	require.JSONEq(t, string(oldRefresh), raw)
	_, ok, err = store.GetRefresh(ctx, newKey)
	require.NoError(t, err)
	require.True(t, ok)

	result, err = store.RotateRefresh(ctx, sessionKey, oldKey, usedKey, "desktop:refresh:unused", newRefresh, newSession, time.Hour)
	require.NoError(t, err)
	require.Equal(t, service.DesktopRefreshReplayed, result)
	_, ok, err = store.GetSession(ctx, sessionKey)
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, err = store.GetRefresh(ctx, newKey)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDesktopSessionStoreExplicitRevocationAndDPoPReplay(t *testing.T) {
	store, _ := newDesktopSessionRedisStore(t)
	ctx := context.Background()
	const (
		sessionKey = "desktop:session:session-2"
		refreshKey = "desktop:refresh:current"
	)
	stored, err := store.StoreSession(ctx, sessionKey, desktopSessionStorePayload(t, "session-2", refreshKey), refreshKey, []byte(`{"refresh":true}`), time.Hour)
	require.NoError(t, err)
	require.True(t, stored)

	revoked, err := store.RevokeSession(ctx, sessionKey)
	require.NoError(t, err)
	require.True(t, revoked)
	_, ok, err := store.GetRefresh(ctx, refreshKey)
	require.NoError(t, err)
	require.False(t, ok)

	stored, err = store.StoreDPoPProof(ctx, "desktop:dpop:jti:key:jti", time.Minute)
	require.NoError(t, err)
	require.True(t, stored)
	stored, err = store.StoreDPoPProof(ctx, "desktop:dpop:jti:key:jti", time.Minute)
	require.NoError(t, err)
	require.False(t, stored)
}

func newDesktopSessionRedisStore(t *testing.T) (service.DesktopSessionStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewDesktopSessionStore(rdb), mr
}

func desktopSessionStorePayload(t *testing.T, sessionID, refreshKey string) []byte {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"session_id": sessionID, "user_id": 1, "token_version": 2, "jkt": "thumbprint",
		"current_refresh_key": refreshKey, "expires_at": int64(9999999999),
	})
	require.NoError(t, err)
	return payload
}
