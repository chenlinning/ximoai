//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchBrokerUserBoundConsumesAreAtomic(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	ctx := context.Background()

	ticketStore := &workbenchSSOTicketStore{rdb: rdb}
	ticketPayload, err := json.Marshal(map[string]any{"user_id": 123, "audience": "https://image.ximoai.cn"})
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, "ticket", ticketPayload, time.Minute).Err())
	_, ok, err := ticketStore.ConsumeTicketForUser(ctx, "ticket", 456)
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, err = ticketStore.ConsumeTicketForUser(ctx, "ticket", 123)
	require.NoError(t, err)
	require.True(t, ok)

	controlStore := &workbenchControlTokenStore{rdb: rdb}
	controlPayload, err := json.Marshal(map[string]any{"user_id": 123, "sso_audience": "https://image.ximoai.cn"})
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, "refresh", controlPayload, time.Minute).Err())
	_, ok, err = controlStore.ConsumeGrantForUser(ctx, "refresh", "https://image.ximoai.cn", 456)
	require.NoError(t, err)
	require.False(t, ok)
	_, ok, err = controlStore.ConsumeGrantForUser(ctx, "refresh", "https://image.ximoai.cn", 123)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, rdb.Set(ctx, "revoke", controlPayload, time.Minute).Err())
	ok, err = controlStore.RevokeGrantForUser(ctx, "revoke", "https://image.ximoai.cn", 456)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = controlStore.RevokeGrantForUser(ctx, "revoke", "https://image.ximoai.cn", 123)
	require.NoError(t, err)
	require.True(t, ok)
}
