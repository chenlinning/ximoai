//go:build unit

package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type workbenchControlGrantStoreStub struct {
	mu    sync.Mutex
	items map[string]string
}

func newWorkbenchControlGrantStoreStub() *workbenchControlGrantStoreStub {
	return &workbenchControlGrantStoreStub{items: make(map[string]string)}
}

func (s *workbenchControlGrantStoreStub) StoreGrant(_ context.Context, key string, payload []byte, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[key]; ok {
		return false, nil
	}
	s.items[key] = string(payload)
	return true, nil
}

func (s *workbenchControlGrantStoreStub) ConsumeGrant(_ context.Context, key, ssoAudience string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[key]
	if !ok {
		return "", false, nil
	}
	var record workbenchControlRefreshRecord
	if json.Unmarshal([]byte(value), &record) != nil || record.SSOAudience != ssoAudience {
		return "", false, nil
	}
	delete(s.items, key)
	return value, true, nil
}

func (s *workbenchControlGrantStoreStub) RevokeGrant(_ context.Context, key, ssoAudience string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[key]
	if !ok {
		return false, nil
	}
	var record workbenchControlRefreshRecord
	if json.Unmarshal([]byte(value), &record) != nil || record.SSOAudience != ssoAudience {
		return false, nil
	}
	delete(s.items, key)
	return true, nil
}

func (s *workbenchControlGrantStoreStub) Ping(context.Context) error { return nil }

func newWorkbenchControlTokenTestService() (*WorkbenchControlTokenService, *AuthService, *workbenchControlGrantStoreStub) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "test-jwt-secret-32bytes-long!!!"
	user := &User{
		ID:           123,
		Email:        "alice@example.com",
		Role:         RoleUser,
		Status:       StatusActive,
		TokenVersion: 7,
	}
	store := newWorkbenchControlGrantStoreStub()
	authService := &AuthService{cfg: cfg}
	service := &WorkbenchControlTokenService{
		authService: authService,
		userGetter:  workbenchUserGetterStub{user: user},
		grantStore:  store,
	}
	return service, authService, store
}

func TestWorkbenchControlTokenIssueUsesScopedShortLivedJWTAndHashedRefreshGrant(t *testing.T) {
	control, authService, store := newWorkbenchControlTokenTestService()

	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)
	require.NotEmpty(t, grant.AccessToken)
	require.NotEmpty(t, grant.RefreshToken)
	require.Equal(t, WorkbenchControlAudience, grant.Audience)
	require.Equal(t, []string{WorkbenchModelControlReadScope}, grant.Scopes)
	require.Equal(t, int(workbenchControlAccessTTL.Seconds()), grant.ExpiresIn)

	claims, err := authService.ValidateToken(grant.AccessToken)
	require.NoError(t, err)
	require.Equal(t, int64(123), claims.UserID)
	require.Equal(t, WorkbenchControlTokenUse, claims.TokenUse)
	require.Equal(t, []string{WorkbenchModelControlReadScope}, claims.Scopes)
	require.Equal(t, []string{WorkbenchControlAudience}, []string(claims.Audience))

	store.mu.Lock()
	require.Len(t, store.items, 1)
	for key, raw := range store.items {
		require.NotContains(t, key, grant.RefreshToken)
		require.NotContains(t, raw, grant.RefreshToken)
		var record workbenchControlRefreshRecord
		require.NoError(t, json.Unmarshal([]byte(raw), &record))
		require.Equal(t, int64(123), record.UserID)
		require.Equal(t, "https://workbench.ximoai.cn", record.SSOAudience)
	}
	store.mu.Unlock()
}

func TestWorkbenchControlTokenRefreshAtomicallyRotatesOnce(t *testing.T) {
	control, _, _ := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)

	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, refreshErr := control.Refresh(context.Background(), grant.RefreshToken, "https://workbench.ximoai.cn")
			results <- refreshErr
		}()
	}

	var success, failure int
	for i := 0; i < 2; i++ {
		if <-results == nil {
			success++
		} else {
			failure++
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, failure)
}

func TestWorkbenchControlAccessTokenCannotUpgradeToRegularAccessToken(t *testing.T) {
	control, authService, _ := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)

	_, err = authService.RefreshToken(context.Background(), grant.AccessToken)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestWorkbenchControlTokenRejectsExpiredRefreshGrant(t *testing.T) {
	control, _, store := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)

	store.mu.Lock()
	for key, raw := range store.items {
		var record workbenchControlRefreshRecord
		require.NoError(t, json.Unmarshal([]byte(raw), &record))
		record.ExpiresAt = time.Now().Add(-time.Second).Unix()
		updated, marshalErr := json.Marshal(record)
		require.NoError(t, marshalErr)
		store.items[key] = string(updated)
	}
	store.mu.Unlock()

	_, err = control.Refresh(context.Background(), grant.RefreshToken, "https://workbench.ximoai.cn")
	require.Error(t, err)
}

func TestWorkbenchControlTokenRevokePreventsRefresh(t *testing.T) {
	control, _, _ := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)
	require.NoError(t, control.Revoke(context.Background(), grant.RefreshToken, "https://workbench.ximoai.cn"))

	_, err = control.Refresh(context.Background(), grant.RefreshToken, "https://workbench.ximoai.cn")
	require.Error(t, err)
}

func TestWorkbenchControlTokenWrongAudienceCannotConsumeOrRevokeGrant(t *testing.T) {
	control, _, _ := newWorkbenchControlTokenTestService()
	grant, err := control.Issue(context.Background(), 123, "https://workbench.ximoai.cn")
	require.NoError(t, err)

	_, err = control.Refresh(context.Background(), grant.RefreshToken, "https://novel.ximoai.cn")
	require.Error(t, err)
	require.Error(t, control.Revoke(context.Background(), grant.RefreshToken, "https://novel.ximoai.cn"))

	_, err = control.Refresh(context.Background(), grant.RefreshToken, "https://workbench.ximoai.cn")
	require.NoError(t, err)
}
