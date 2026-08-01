//go:build unit

package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type desktopSessionMemoryStore struct {
	codes    map[string]string
	sessions map[string]string
	refresh  map[string]string
	used     map[string]desktopUsedRefreshMemoryRecord
	proofs   map[string]struct{}
}

type desktopUsedRefreshMemoryRecord struct {
	sessionKey string
	payload    string
}

func newDesktopSessionMemoryStore() *desktopSessionMemoryStore {
	return &desktopSessionMemoryStore{
		codes:    map[string]string{},
		sessions: map[string]string{},
		refresh:  map[string]string{},
		used:     map[string]desktopUsedRefreshMemoryRecord{},
		proofs:   map[string]struct{}{},
	}
}

func (s *desktopSessionMemoryStore) StoreAuthorizationCode(_ context.Context, key string, payload []byte, _ time.Duration) (bool, error) {
	if _, exists := s.codes[key]; exists {
		return false, nil
	}
	s.codes[key] = string(payload)
	return true, nil
}

func (s *desktopSessionMemoryStore) ConsumeAuthorizationCode(_ context.Context, key string) (string, bool, error) {
	value, ok := s.codes[key]
	delete(s.codes, key)
	return value, ok, nil
}

func (s *desktopSessionMemoryStore) StoreSession(_ context.Context, sessionKey string, sessionPayload []byte, refreshKey string, refreshPayload []byte, _ time.Duration) (bool, error) {
	if _, exists := s.sessions[sessionKey]; exists {
		return false, nil
	}
	if _, exists := s.refresh[refreshKey]; exists {
		return false, nil
	}
	s.sessions[sessionKey] = string(sessionPayload)
	s.refresh[refreshKey] = string(refreshPayload)
	return true, nil
}

func (s *desktopSessionMemoryStore) GetSession(_ context.Context, key string) (string, bool, error) {
	value, ok := s.sessions[key]
	return value, ok, nil
}

func (s *desktopSessionMemoryStore) GetRefresh(_ context.Context, key string) (string, bool, error) {
	value, ok := s.refresh[key]
	return value, ok, nil
}

func (s *desktopSessionMemoryStore) GetUsedRefresh(_ context.Context, key string) (string, bool, error) {
	record, ok := s.used[key]
	return record.payload, ok, nil
}

func (s *desktopSessionMemoryStore) RotateRefresh(_ context.Context, sessionKey, oldRefreshKey, usedRefreshKey, newRefreshKey string, newRefreshPayload, newSessionPayload []byte, _ time.Duration) (DesktopRefreshRotationResult, error) {
	if _, ok := s.refresh[oldRefreshKey]; !ok {
		if replayRecord, replay := s.used[usedRefreshKey]; replay {
			s.revoke(replayRecord.sessionKey)
			return DesktopRefreshReplayed, nil
		}
		return DesktopRefreshInvalid, nil
	}
	if _, ok := s.sessions[sessionKey]; !ok {
		return DesktopRefreshInvalid, nil
	}
	if _, collision := s.refresh[newRefreshKey]; collision {
		return DesktopRefreshCollision, nil
	}
	oldRefreshPayload := s.refresh[oldRefreshKey]
	delete(s.refresh, oldRefreshKey)
	s.used[usedRefreshKey] = desktopUsedRefreshMemoryRecord{sessionKey: sessionKey, payload: oldRefreshPayload}
	s.refresh[newRefreshKey] = string(newRefreshPayload)
	s.sessions[sessionKey] = string(newSessionPayload)
	return DesktopRefreshRotated, nil
}

func (s *desktopSessionMemoryStore) RevokeSession(_ context.Context, sessionKey string) (bool, error) {
	return s.revoke(sessionKey), nil
}

func (s *desktopSessionMemoryStore) RevokeRefreshReplay(_ context.Context, usedRefreshKey string) (bool, error) {
	record, ok := s.used[usedRefreshKey]
	if !ok {
		return false, nil
	}
	s.revoke(record.sessionKey)
	return true, nil
}

func (s *desktopSessionMemoryStore) StoreDPoPProof(_ context.Context, key string, _ time.Duration) (bool, error) {
	if _, exists := s.proofs[key]; exists {
		return false, nil
	}
	s.proofs[key] = struct{}{}
	return true, nil
}

func (s *desktopSessionMemoryStore) Ping(context.Context) error { return nil }

func (s *desktopSessionMemoryStore) revoke(sessionKey string) bool {
	raw, ok := s.sessions[sessionKey]
	if !ok {
		return false
	}
	var record desktopSessionRecord
	_ = json.Unmarshal([]byte(raw), &record)
	delete(s.sessions, sessionKey)
	delete(s.refresh, record.CurrentRefreshKey)
	return true
}

type desktopUserGetterStub struct {
	user *User
}

func (s *desktopUserGetterStub) GetByID(_ context.Context, id int64) (*User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}

type desktopWorkbenchIssuerStub struct {
	lastUserID      int64
	lastWorkbenchID string
}

func (s *desktopWorkbenchIssuerStub) IssueTicketForWorkbenchID(_ context.Context, userID int64, workbenchID string) (*WorkbenchSSOTicket, error) {
	s.lastUserID = userID
	s.lastWorkbenchID = workbenchID
	if workbenchID != "image" {
		return nil, infraerrors.Forbidden("WORKBENCH_SSO_FORBIDDEN", "workbench is unavailable")
	}
	return &WorkbenchSSOTicket{Ticket: "ticket", ExpiresIn: 60, EntryURL: "https://image.ximoai.cn/sso/entry?ticket=ticket"}, nil
}

func TestDesktopSession_PKCEDPoPAndRefreshReplay(t *testing.T) {
	svc, store, userSource := newDesktopSessionTestService(t)
	deviceKey := generateDesktopTestKey(t)
	deviceJWK := desktopTestJWK(deviceKey)
	verifier := strings.Repeat("v", 64)
	challengeHash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])

	grant, err := svc.CreateAuthorizationCode(context.Background(), userSource.user.ID, DesktopAuthorizationRequest{
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		DeviceJWK:           deviceJWK,
		RedirectURI:         "http://127.0.0.1:49152/callback",
	})
	require.NoError(t, err)
	require.NotEmpty(t, grant.Code)
	require.Equal(t, int(desktopAuthorizationCodeTTL.Seconds()), grant.ExpiresIn)

	tokenTarget := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/token"}}
	proof := signDesktopTestDPoP(t, deviceKey, deviceJWK, tokenTarget.URLs[0], tokenTarget.Method, "exchange-1", "")
	pair, err := svc.ExchangeAuthorizationCode(context.Background(), DesktopCodeExchangeRequest{
		Code:         grant.Code,
		CodeVerifier: verifier,
		RedirectURI:  "http://127.0.0.1:49152/callback",
	}, proof, tokenTarget)
	require.NoError(t, err)
	require.Equal(t, "DPoP", pair.TokenType)
	require.Equal(t, DesktopWorkbenchSSOScope, pair.Scope)
	require.NotEmpty(t, pair.DesktopSessionID)
	mainSiteClaims, err := svc.authService.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, DesktopTokenUse, mainSiteClaims.TokenUse)
	require.Equal(t, []string{DesktopWorkbenchSSOScope}, mainSiteClaims.Scopes)

	accessTarget := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/sso-ticket"}}
	accessProof := signDesktopTestDPoP(t, deviceKey, deviceJWK, accessTarget.URLs[0], accessTarget.Method, "access-1", pair.AccessToken)
	identity, err := svc.AuthenticateAccess(context.Background(), pair.AccessToken, accessProof, accessTarget)
	require.NoError(t, err)
	require.Equal(t, userSource.user.ID, identity.UserID)

	otherKey := generateDesktopTestKey(t)
	wrongProof := signDesktopTestDPoP(t, otherKey, desktopTestJWK(otherKey), accessTarget.URLs[0], accessTarget.Method, "access-2", pair.AccessToken)
	_, err = svc.AuthenticateAccess(context.Background(), pair.AccessToken, wrongProof, accessTarget)
	require.Equal(t, "DESKTOP_DPOP_KEY_MISMATCH", infraerrors.Reason(err))

	refreshProof := signDesktopTestDPoP(t, deviceKey, deviceJWK, tokenTarget.URLs[0], tokenTarget.Method, "refresh-1", "")
	rotated, err := svc.Refresh(context.Background(), pair.RefreshToken, refreshProof, tokenTarget)
	require.NoError(t, err)
	require.NotEqual(t, pair.RefreshToken, rotated.RefreshToken)
	require.Equal(t, pair.DesktopSessionID, rotated.DesktopSessionID)

	replayProof := signDesktopTestDPoP(t, deviceKey, deviceJWK, tokenTarget.URLs[0], tokenTarget.Method, "refresh-replay", "")
	_, err = svc.Refresh(context.Background(), pair.RefreshToken, replayProof, tokenTarget)
	require.Equal(t, "DESKTOP_REFRESH_REUSED", infraerrors.Reason(err))

	rotatedAccessProof := signDesktopTestDPoP(t, deviceKey, deviceJWK, accessTarget.URLs[0], accessTarget.Method, "access-after-replay", rotated.AccessToken)
	_, err = svc.AuthenticateAccess(context.Background(), rotated.AccessToken, rotatedAccessProof, accessTarget)
	require.Equal(t, "DESKTOP_SESSION_INVALID", infraerrors.Reason(err))
	require.Empty(t, store.sessions)
}

func TestDesktopSession_RefreshReplayRequiresBoundDevice(t *testing.T) {
	svc, _, userSource := newDesktopSessionTestService(t)
	deviceKey := generateDesktopTestKey(t)
	deviceJWK := desktopTestJWK(deviceKey)
	verifier := strings.Repeat("v", 64)
	challengeHash := sha256.Sum256([]byte(verifier))

	grant, err := svc.CreateAuthorizationCode(context.Background(), userSource.user.ID, DesktopAuthorizationRequest{
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(challengeHash[:]),
		CodeChallengeMethod: "S256",
		DeviceJWK:           deviceJWK,
		RedirectURI:         "ximoai://desktop/callback",
	})
	require.NoError(t, err)

	tokenTarget := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/token"}}
	pair, err := svc.ExchangeAuthorizationCode(context.Background(), DesktopCodeExchangeRequest{
		Code: grant.Code, CodeVerifier: verifier, RedirectURI: "ximoai://desktop/callback",
	}, signDesktopTestDPoP(t, deviceKey, deviceJWK, tokenTarget.URLs[0], tokenTarget.Method, "exchange-bound", ""), tokenTarget)
	require.NoError(t, err)

	rotated, err := svc.Refresh(context.Background(), pair.RefreshToken,
		signDesktopTestDPoP(t, deviceKey, deviceJWK, tokenTarget.URLs[0], tokenTarget.Method, "rotate-bound", ""), tokenTarget)
	require.NoError(t, err)

	otherKey := generateDesktopTestKey(t)
	otherJWK := desktopTestJWK(otherKey)
	_, err = svc.Refresh(context.Background(), pair.RefreshToken,
		signDesktopTestDPoP(t, otherKey, otherJWK, tokenTarget.URLs[0], tokenTarget.Method, "replay-other-device", ""), tokenTarget)
	require.Equal(t, "DESKTOP_DPOP_KEY_MISMATCH", infraerrors.Reason(err))

	accessTarget := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/sso-ticket"}}
	_, err = svc.AuthenticateAccess(context.Background(), rotated.AccessToken,
		signDesktopTestDPoP(t, deviceKey, deviceJWK, accessTarget.URLs[0], accessTarget.Method, "access-after-wrong-replay", rotated.AccessToken), accessTarget)
	require.NoError(t, err)
}

func TestDesktopSession_MultipleDevicesAreIndependent(t *testing.T) {
	svc, _, userSource := newDesktopSessionTestService(t)
	firstKey := generateDesktopTestKey(t)
	secondKey := generateDesktopTestKey(t)
	first := issueDesktopTestSession(t, svc, userSource.user.ID, firstKey, "first")
	second := issueDesktopTestSession(t, svc, userSource.user.ID, secondKey, "second")
	require.NotEqual(t, first.DesktopSessionID, second.DesktopSessionID)

	revokeTarget := DesktopDPoPTarget{Method: "DELETE", URLs: []string{"https://ximoai.cn/api/v1/desktop/session"}}
	err := svc.RevokeAccess(context.Background(), first.AccessToken,
		signDesktopTestDPoP(t, firstKey, desktopTestJWK(firstKey), revokeTarget.URLs[0], revokeTarget.Method, "revoke-first", first.AccessToken), revokeTarget)
	require.NoError(t, err)

	accessTarget := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/sso-ticket"}}
	_, err = svc.AuthenticateAccess(context.Background(), first.AccessToken,
		signDesktopTestDPoP(t, firstKey, desktopTestJWK(firstKey), accessTarget.URLs[0], accessTarget.Method, "first-after-revoke", first.AccessToken), accessTarget)
	require.Equal(t, "DESKTOP_SESSION_INVALID", infraerrors.Reason(err))

	_, err = svc.AuthenticateAccess(context.Background(), second.AccessToken,
		signDesktopTestDPoP(t, secondKey, desktopTestJWK(secondKey), accessTarget.URLs[0], accessTarget.Method, "second-still-active", second.AccessToken), accessTarget)
	require.NoError(t, err)
}

func TestDesktopSession_RechecksUserAndIssuesByWorkbenchID(t *testing.T) {
	svc, _, userSource := newDesktopSessionTestService(t)
	issuer := &desktopWorkbenchIssuerStub{}
	svc.ticketIssuer = issuer

	ticket, err := svc.IssueWorkbenchTicket(context.Background(), &DesktopIdentity{
		UserID:    userSource.user.ID,
		SessionID: "session",
	}, "image")
	require.NoError(t, err)
	require.Equal(t, userSource.user.ID, issuer.lastUserID)
	require.Equal(t, "image", issuer.lastWorkbenchID)
	require.Equal(t, "ticket", ticket.Ticket)

	userSource.user.Status = StatusDisabled
	_, err = svc.IssueWorkbenchTicket(context.Background(), &DesktopIdentity{UserID: userSource.user.ID, SessionID: "session"}, "image")
	require.Equal(t, "DESKTOP_USER_INVALID", infraerrors.Reason(err))
}

func newDesktopSessionTestService(t *testing.T) (*DesktopSessionService, *desktopSessionMemoryStore, *desktopUserGetterStub) {
	t.Helper()
	store := newDesktopSessionMemoryStore()
	users := &desktopUserGetterStub{user: &User{
		ID:                   123,
		Email:                "desktop@example.com",
		Role:                 RoleUser,
		Status:               StatusActive,
		TokenVersion:         7,
		TokenVersionResolved: true,
	}}
	auth := &AuthService{cfg: &config.Config{JWT: config.JWTConfig{Secret: "desktop-test-jwt-secret-32-bytes-long"}}}
	return &DesktopSessionService{
		authService: auth,
		userGetter:  users,
		store:       store,
		now:         time.Now,
	}, store, users
}

func generateDesktopTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func desktopTestJWK(key *ecdsa.PrivateKey) DesktopPublicJWK {
	return DesktopPublicJWK{
		Kty: "EC",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		Y:   base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
	}
}

func issueDesktopTestSession(t *testing.T, svc *DesktopSessionService, userID int64, key *ecdsa.PrivateKey, jtiPrefix string) *DesktopTokenPair {
	t.Helper()
	verifier := strings.Repeat("v", 64)
	challengeHash := sha256.Sum256([]byte(verifier))
	grant, err := svc.CreateAuthorizationCode(context.Background(), userID, DesktopAuthorizationRequest{
		CodeChallenge:       base64.RawURLEncoding.EncodeToString(challengeHash[:]),
		CodeChallengeMethod: "S256",
		DeviceJWK:           desktopTestJWK(key),
		RedirectURI:         "ximoai://desktop/callback",
	})
	require.NoError(t, err)
	target := DesktopDPoPTarget{Method: "POST", URLs: []string{"https://ximoai.cn/api/v1/desktop/token"}}
	pair, err := svc.ExchangeAuthorizationCode(context.Background(), DesktopCodeExchangeRequest{
		Code: grant.Code, CodeVerifier: verifier, RedirectURI: "ximoai://desktop/callback",
	}, signDesktopTestDPoP(t, key, desktopTestJWK(key), target.URLs[0], target.Method, jtiPrefix+"-exchange", ""), target)
	require.NoError(t, err)
	return pair
}

func signDesktopTestDPoP(t *testing.T, key *ecdsa.PrivateKey, jwk DesktopPublicJWK, htu, htm, jti, accessToken string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"htu": htu,
		"htm": htm,
		"iat": time.Now().Unix(),
		"jti": jti,
	}
	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = map[string]string{"kty": jwk.Kty, "crv": jwk.Crv, "x": jwk.X, "y": jwk.Y}
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func TestDesktopPublicJWKRejectsPrivateMaterial(t *testing.T) {
	key := generateDesktopTestKey(t)
	jwk := desktopTestJWK(key)
	jwk.D = "must-not-be-accepted"
	_, _, err := parseDesktopPublicJWK(jwk)
	require.Error(t, err)
	require.Contains(t, fmt.Sprint(err), "private")
}
