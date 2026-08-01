package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	DesktopAudience          = "desktop"
	DesktopTokenUse          = "desktop_access"
	DesktopWorkbenchSSOScope = "workbench:sso:issue"

	desktopAuthorizationCodePrefix = "dca_"
	desktopRefreshTokenPrefix      = "dtr_"
	desktopAuthorizationKeyPrefix  = "desktop:authorization:"
	desktopSessionKeyPrefix        = "desktop:session:"
	desktopRefreshKeyPrefix        = "desktop:refresh:"
	desktopUsedRefreshKeyPrefix    = "desktop:refresh:used:"
	desktopDPoPProofKeyPrefix      = "desktop:dpop:jti:"
)

const (
	desktopAuthorizationCodeTTL = 5 * time.Minute
	desktopAccessTokenTTL       = 5 * time.Minute
	desktopSessionTTL           = 30 * 24 * time.Hour
	desktopDPoPProofTTL         = 5 * time.Minute
	desktopDPoPClockSkew        = 30 * time.Second
	desktopDPoPMaxAge           = 5 * time.Minute
)

type DesktopRefreshRotationResult int

const (
	DesktopRefreshInvalid DesktopRefreshRotationResult = iota
	DesktopRefreshRotated
	DesktopRefreshReplayed
	DesktopRefreshCollision
)

type DesktopSessionStore interface {
	StoreAuthorizationCode(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error)
	ConsumeAuthorizationCode(ctx context.Context, key string) (string, bool, error)
	StoreSession(ctx context.Context, sessionKey string, sessionPayload []byte, refreshKey string, refreshPayload []byte, ttl time.Duration) (bool, error)
	GetSession(ctx context.Context, key string) (string, bool, error)
	GetRefresh(ctx context.Context, key string) (string, bool, error)
	GetUsedRefresh(ctx context.Context, key string) (string, bool, error)
	RotateRefresh(ctx context.Context, sessionKey, oldRefreshKey, usedRefreshKey, newRefreshKey string, newRefreshPayload, newSessionPayload []byte, ttl time.Duration) (DesktopRefreshRotationResult, error)
	RevokeSession(ctx context.Context, sessionKey string) (bool, error)
	RevokeRefreshReplay(ctx context.Context, usedRefreshKey string) (bool, error)
	StoreDPoPProof(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Ping(ctx context.Context) error
}

type desktopWorkbenchTicketIssuer interface {
	IssueTicketForWorkbenchID(ctx context.Context, userID int64, workbenchID string) (*WorkbenchSSOTicket, error)
}

type DesktopSessionService struct {
	authService  *AuthService
	userGetter   workbenchUserGetter
	ticketIssuer desktopWorkbenchTicketIssuer
	store        DesktopSessionStore
	now          func() time.Time
}

type DesktopPublicJWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Alg string `json:"alg,omitempty"`
	D   string `json:"d,omitempty"`
}

type desktopAccessTokenConfirmation struct {
	JKT string `json:"jkt"`
}

type desktopAccessTokenClaims struct {
	UserID       int64                          `json:"user_id"`
	TokenVersion int64                          `json:"token_version"`
	TokenUse     string                         `json:"token_use"`
	Scopes       []string                       `json:"scopes"`
	SessionID    string                         `json:"sid"`
	Confirmation desktopAccessTokenConfirmation `json:"cnf"`
	jwt.RegisteredClaims
}

type DesktopAuthorizationRequest struct {
	CodeChallenge       string           `json:"code_challenge"`
	CodeChallengeMethod string           `json:"code_challenge_method"`
	DeviceJWK           DesktopPublicJWK `json:"device_jwk"`
	RedirectURI         string           `json:"redirect_uri"`
}

type DesktopAuthorizationGrant struct {
	Code      string `json:"code"`
	ExpiresIn int    `json:"expires_in"`
}

type DesktopCodeExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

type DesktopTokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	DesktopSessionID string `json:"desktop_session_id"`
	Scope            string `json:"scope"`
}

type DesktopIdentity struct {
	UserID       int64
	SessionID    string
	TokenVersion int64
	JKT          string
}

type DesktopDPoPTarget struct {
	Method string
	URLs   []string
}

type desktopAuthorizationCodeRecord struct {
	UserID        int64  `json:"user_id"`
	TokenVersion  int64  `json:"token_version"`
	CodeChallenge string `json:"code_challenge"`
	RedirectURI   string `json:"redirect_uri"`
	JKT           string `json:"jkt"`
	ExpiresAt     int64  `json:"expires_at"`
}

type desktopSessionRecord struct {
	SessionID         string `json:"session_id"`
	UserID            int64  `json:"user_id"`
	TokenVersion      int64  `json:"token_version"`
	JKT               string `json:"jkt"`
	CurrentRefreshKey string `json:"current_refresh_key"`
	ExpiresAt         int64  `json:"expires_at"`
}

type desktopRefreshRecord struct {
	SessionID    string `json:"session_id"`
	UserID       int64  `json:"user_id"`
	TokenVersion int64  `json:"token_version"`
	JKT          string `json:"jkt"`
	ExpiresAt    int64  `json:"expires_at"`
}

type desktopDPoPClaims struct {
	HTU string `json:"htu"`
	HTM string `json:"htm"`
	ATH string `json:"ath,omitempty"`
	jwt.RegisteredClaims
}

func NewDesktopSessionService(authService *AuthService, userService *UserService, workbenchSSO *WorkbenchSSOService, store DesktopSessionStore) *DesktopSessionService {
	return &DesktopSessionService{
		authService:  authService,
		userGetter:   userService,
		ticketIssuer: workbenchSSO,
		store:        store,
		now:          time.Now,
	}
}

func (s *DesktopSessionService) CreateAuthorizationCode(ctx context.Context, userID int64, req DesktopAuthorizationRequest) (*DesktopAuthorizationGrant, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	challenge, err := normalizeDesktopPKCEChallenge(req.CodeChallenge, req.CodeChallengeMethod)
	if err != nil {
		return nil, err
	}
	redirectURI, err := normalizeDesktopRedirectURI(req.RedirectURI)
	if err != nil {
		return nil, err
	}
	_, jkt, err := parseDesktopPublicJWK(req.DeviceJWK)
	if err != nil {
		return nil, infraerrors.BadRequest("DESKTOP_DEVICE_KEY_INVALID", "device_jwk must be a public P-256 key").WithCause(err)
	}

	now := s.currentTime()
	record := desktopAuthorizationCodeRecord{
		UserID:        user.ID,
		TokenVersion:  resolvedTokenVersion(user),
		CodeChallenge: challenge,
		RedirectURI:   redirectURI,
		JKT:           jkt,
		ExpiresAt:     now.Add(desktopAuthorizationCodeTTL).Unix(),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal desktop authorization code: %w", err)
	}
	for i := 0; i < 3; i++ {
		code, err := randomDesktopSecret(desktopAuthorizationCodePrefix)
		if err != nil {
			return nil, err
		}
		key, err := desktopAuthorizationCodeKey(code)
		if err != nil {
			return nil, err
		}
		stored, err := s.store.StoreAuthorizationCode(ctx, key, payload, desktopAuthorizationCodeTTL)
		if err != nil {
			return nil, desktopStoreError(err)
		}
		if stored {
			return &DesktopAuthorizationGrant{Code: code, ExpiresIn: int(desktopAuthorizationCodeTTL.Seconds())}, nil
		}
	}
	return nil, infraerrors.InternalServer("DESKTOP_AUTHORIZATION_COLLISION", "failed to create desktop authorization code")
}

func (s *DesktopSessionService) ExchangeAuthorizationCode(ctx context.Context, req DesktopCodeExchangeRequest, dpopProof string, target DesktopDPoPTarget) (*DesktopTokenPair, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	key, err := desktopAuthorizationCodeKey(req.Code)
	if err != nil {
		return nil, err
	}
	raw, ok, err := s.store.ConsumeAuthorizationCode(ctx, key)
	if err != nil {
		return nil, desktopStoreError(err)
	}
	if !ok {
		return nil, infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_INVALID", "authorization code is invalid or expired")
	}
	var record desktopAuthorizationCodeRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || !validDesktopAuthorizationRecord(record) {
		return nil, infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_INVALID", "authorization code is invalid or expired")
	}
	if !s.currentTime().Before(time.Unix(record.ExpiresAt, 0)) {
		return nil, infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_EXPIRED", "authorization code is expired")
	}
	redirectURI, err := normalizeDesktopRedirectURI(req.RedirectURI)
	if err != nil || redirectURI != record.RedirectURI || !verifyDesktopPKCE(req.CodeVerifier, record.CodeChallenge) {
		return nil, infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_INVALID", "authorization code verification failed")
	}
	if err := s.verifyDPoP(ctx, dpopProof, target, record.JKT, ""); err != nil {
		return nil, err
	}
	user, err := s.activeUser(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if resolvedTokenVersion(user) != record.TokenVersion {
		return nil, infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_REVOKED", "authorization code is no longer valid")
	}
	return s.issueNewSession(ctx, user, record.JKT)
}

func (s *DesktopSessionService) Refresh(ctx context.Context, refreshToken, dpopProof string, target DesktopDPoPTarget) (*DesktopTokenPair, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	refreshKey, usedKey, err := desktopRefreshKeys(refreshToken)
	if err != nil {
		return nil, err
	}
	raw, ok, err := s.store.GetRefresh(ctx, refreshKey)
	if err != nil {
		return nil, desktopStoreError(err)
	}
	if !ok {
		return s.handleRefreshReplay(ctx, usedKey, dpopProof, target)
	}
	var refreshRecord desktopRefreshRecord
	if err := json.Unmarshal([]byte(raw), &refreshRecord); err != nil || !validDesktopRefreshRecord(refreshRecord) {
		return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	if err := s.verifyDPoP(ctx, dpopProof, target, refreshRecord.JKT, ""); err != nil {
		return nil, err
	}
	sessionKey := desktopSessionKey(refreshRecord.SessionID)
	session, err := s.loadSession(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if session.CurrentRefreshKey != refreshKey || !desktopSessionMatchesRefresh(session, refreshRecord) {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	now := s.currentTime()
	sessionExpiresAt := time.Unix(session.ExpiresAt, 0)
	if !now.Before(sessionExpiresAt) {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_EXPIRED", "desktop session is expired")
	}
	user, err := s.activeUser(ctx, session.UserID)
	if err != nil {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, err
	}
	if resolvedTokenVersion(user) != session.TokenVersion {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_REVOKED", "desktop session is no longer valid")
	}

	refreshTTL := sessionExpiresAt.Sub(now)
	for i := 0; i < 3; i++ {
		newRefreshToken, err := randomDesktopSecret(desktopRefreshTokenPrefix)
		if err != nil {
			return nil, err
		}
		newRefreshKey, _, err := desktopRefreshKeys(newRefreshToken)
		if err != nil {
			return nil, err
		}
		pair, newRefreshPayload, newSessionPayload, err := s.buildRotatedTokenMaterial(user, session, newRefreshToken, newRefreshKey, now, sessionExpiresAt)
		if err != nil {
			return nil, err
		}
		result, err := s.store.RotateRefresh(ctx, sessionKey, refreshKey, usedKey, newRefreshKey, newRefreshPayload, newSessionPayload, refreshTTL)
		if err != nil {
			return nil, desktopStoreError(err)
		}
		switch result {
		case DesktopRefreshRotated:
			return pair, nil
		case DesktopRefreshReplayed:
			return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_REUSED", "refresh token reuse revoked the desktop session")
		case DesktopRefreshCollision:
			continue
		default:
			return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
		}
	}
	return nil, infraerrors.InternalServer("DESKTOP_REFRESH_COLLISION", "failed to rotate desktop refresh token")
}

func (s *DesktopSessionService) AuthenticateAccess(ctx context.Context, accessToken, dpopProof string, target DesktopDPoPTarget) (*DesktopIdentity, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	claims, err := s.authService.validateDesktopAccessToken(accessToken)
	if err != nil {
		return nil, infraerrors.Unauthorized("DESKTOP_ACCESS_INVALID", "desktop access token is invalid or expired")
	}
	if !validDesktopAccessClaims(claims) {
		return nil, infraerrors.Forbidden("DESKTOP_ACCESS_SCOPE_INVALID", "token is not authorized for desktop SSO")
	}
	if err := s.verifyDPoP(ctx, dpopProof, target, claims.Confirmation.JKT, accessToken); err != nil {
		return nil, err
	}
	sessionKey := desktopSessionKey(claims.SessionID)
	session, err := s.loadSession(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	if session.UserID != claims.UserID || session.TokenVersion != claims.TokenVersion || session.JKT != claims.Confirmation.JKT || session.SessionID != claims.SessionID {
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	if !s.currentTime().Before(time.Unix(session.ExpiresAt, 0)) {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_EXPIRED", "desktop session is expired")
	}
	user, err := s.activeUser(ctx, session.UserID)
	if err != nil {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, err
	}
	if resolvedTokenVersion(user) != session.TokenVersion {
		_, _ = s.store.RevokeSession(ctx, sessionKey)
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_REVOKED", "desktop session is no longer valid")
	}
	return &DesktopIdentity{UserID: session.UserID, SessionID: session.SessionID, TokenVersion: session.TokenVersion, JKT: session.JKT}, nil
}

func (s *DesktopSessionService) IssueWorkbenchTicket(ctx context.Context, identity *DesktopIdentity, workbenchID string) (*WorkbenchSSOTicket, error) {
	if identity == nil || identity.UserID <= 0 || strings.TrimSpace(identity.SessionID) == "" {
		return nil, infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	if _, err := s.activeUser(ctx, identity.UserID); err != nil {
		return nil, err
	}
	if s.ticketIssuer == nil {
		return nil, infraerrors.InternalServer("DESKTOP_SSO_UNAVAILABLE", "desktop SSO is unavailable")
	}
	return s.ticketIssuer.IssueTicketForWorkbenchID(ctx, identity.UserID, workbenchID)
}

func (s *DesktopSessionService) RevokeAccess(ctx context.Context, accessToken, dpopProof string, target DesktopDPoPTarget) error {
	identity, err := s.AuthenticateAccess(ctx, accessToken, dpopProof, target)
	if err != nil {
		return err
	}
	revoked, err := s.store.RevokeSession(ctx, desktopSessionKey(identity.SessionID))
	if err != nil {
		return desktopStoreError(err)
	}
	if !revoked {
		return infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	return nil
}

func (s *DesktopSessionService) RevokeRefresh(ctx context.Context, refreshToken, dpopProof string, target DesktopDPoPTarget) error {
	if err := s.ensureAvailable(ctx); err != nil {
		return err
	}
	refreshKey, usedKey, err := desktopRefreshKeys(refreshToken)
	if err != nil {
		return err
	}
	raw, ok, err := s.store.GetRefresh(ctx, refreshKey)
	if err != nil {
		return desktopStoreError(err)
	}
	if !ok {
		_, replayErr := s.handleRefreshReplay(ctx, usedKey, dpopProof, target)
		if infraerrors.Reason(replayErr) == "DESKTOP_REFRESH_REUSED" {
			return nil
		}
		return replayErr
	}
	var record desktopRefreshRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || !validDesktopRefreshRecord(record) {
		return infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	if err := s.verifyDPoP(ctx, dpopProof, target, record.JKT, ""); err != nil {
		return err
	}
	_, err = s.store.RevokeSession(ctx, desktopSessionKey(record.SessionID))
	if err != nil {
		return desktopStoreError(err)
	}
	return nil
}

func (s *DesktopSessionService) handleRefreshReplay(ctx context.Context, usedKey, dpopProof string, target DesktopDPoPTarget) (*DesktopTokenPair, error) {
	raw, ok, err := s.store.GetUsedRefresh(ctx, usedKey)
	if err != nil {
		return nil, desktopStoreError(err)
	}
	if !ok {
		return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	var record desktopRefreshRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || !validDesktopRefreshRecord(record) {
		return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	if err := s.verifyDPoP(ctx, dpopProof, target, record.JKT, ""); err != nil {
		return nil, err
	}
	replayed, err := s.store.RevokeRefreshReplay(ctx, usedKey)
	if err != nil {
		return nil, desktopStoreError(err)
	}
	if !replayed {
		return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	return nil, infraerrors.Unauthorized("DESKTOP_REFRESH_REUSED", "refresh token reuse revoked the desktop session")
}

func (s *DesktopSessionService) issueNewSession(ctx context.Context, user *User, jkt string) (*DesktopTokenPair, error) {
	now := s.currentTime()
	sessionExpiresAt := now.Add(desktopSessionTTL)
	for i := 0; i < 3; i++ {
		sessionID, err := randomHexString(16)
		if err != nil {
			return nil, fmt.Errorf("generate desktop session id: %w", err)
		}
		refreshToken, err := randomDesktopSecret(desktopRefreshTokenPrefix)
		if err != nil {
			return nil, err
		}
		refreshKey, _, err := desktopRefreshKeys(refreshToken)
		if err != nil {
			return nil, err
		}
		session := desktopSessionRecord{
			SessionID: sessionID, UserID: user.ID, TokenVersion: resolvedTokenVersion(user), JKT: jkt,
			CurrentRefreshKey: refreshKey, ExpiresAt: sessionExpiresAt.Unix(),
		}
		pair, refreshPayload, sessionPayload, err := s.buildRotatedTokenMaterial(user, session, refreshToken, refreshKey, now, sessionExpiresAt)
		if err != nil {
			return nil, err
		}
		stored, err := s.store.StoreSession(ctx, desktopSessionKey(sessionID), sessionPayload, refreshKey, refreshPayload, desktopSessionTTL)
		if err != nil {
			return nil, desktopStoreError(err)
		}
		if stored {
			return pair, nil
		}
	}
	return nil, infraerrors.InternalServer("DESKTOP_SESSION_COLLISION", "failed to create desktop session")
}

func (s *DesktopSessionService) buildRotatedTokenMaterial(user *User, session desktopSessionRecord, refreshToken, refreshKey string, now, sessionExpiresAt time.Time) (*DesktopTokenPair, []byte, []byte, error) {
	session.CurrentRefreshKey = refreshKey
	session.ExpiresAt = sessionExpiresAt.Unix()
	accessExpiresAt := now.Add(desktopAccessTokenTTL)
	if accessExpiresAt.After(sessionExpiresAt) {
		accessExpiresAt = sessionExpiresAt
	}
	accessToken, err := s.authService.generateDesktopAccessToken(user, session.SessionID, session.JKT, now, accessExpiresAt)
	if err != nil {
		return nil, nil, nil, err
	}
	refreshRecord := desktopRefreshRecord{
		SessionID: session.SessionID, UserID: session.UserID, TokenVersion: session.TokenVersion,
		JKT: session.JKT, ExpiresAt: session.ExpiresAt,
	}
	refreshPayload, err := json.Marshal(refreshRecord)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal desktop refresh record: %w", err)
	}
	sessionPayload, err := json.Marshal(session)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal desktop session record: %w", err)
	}
	return &DesktopTokenPair{
		AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "DPoP",
		ExpiresIn: int(accessExpiresAt.Sub(now).Seconds()), RefreshExpiresIn: int(sessionExpiresAt.Sub(now).Seconds()),
		DesktopSessionID: session.SessionID, Scope: DesktopWorkbenchSSOScope,
	}, refreshPayload, sessionPayload, nil
}

func (s *DesktopSessionService) loadSession(ctx context.Context, key string) (desktopSessionRecord, error) {
	raw, ok, err := s.store.GetSession(ctx, key)
	if err != nil {
		return desktopSessionRecord{}, desktopStoreError(err)
	}
	if !ok {
		return desktopSessionRecord{}, infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	var record desktopSessionRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || !validDesktopSessionRecord(record) {
		return desktopSessionRecord{}, infraerrors.Unauthorized("DESKTOP_SESSION_INVALID", "desktop session is invalid or revoked")
	}
	return record, nil
}

func (s *DesktopSessionService) activeUser(ctx context.Context, userID int64) (*User, error) {
	if s == nil || s.userGetter == nil || userID <= 0 {
		return nil, infraerrors.Unauthorized("DESKTOP_USER_INVALID", "user is unavailable")
	}
	user, err := s.userGetter.GetByID(ctx, userID)
	if err != nil || user == nil || user.ID != userID || !user.IsActive() {
		return nil, infraerrors.Unauthorized("DESKTOP_USER_INVALID", "user is unavailable")
	}
	return user, nil
}

func (s *DesktopSessionService) ensureAvailable(ctx context.Context) error {
	if s == nil || s.authService == nil || s.authService.cfg == nil || s.userGetter == nil || s.store == nil || strings.TrimSpace(s.authService.cfg.JWT.Secret) == "" {
		return infraerrors.InternalServer("DESKTOP_SESSION_UNAVAILABLE", "desktop session service is unavailable")
	}
	if err := s.store.Ping(ctx); err != nil {
		return desktopStoreError(err)
	}
	return nil
}

func (s *DesktopSessionService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *DesktopSessionService) verifyDPoP(ctx context.Context, proof string, target DesktopDPoPTarget, expectedJKT, accessToken string) error {
	proof = strings.TrimSpace(proof)
	if proof == "" || len(proof) > maxTokenLength {
		return infraerrors.Unauthorized("DESKTOP_DPOP_INVALID", "DPoP proof is required")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodES256.Name}), jwt.WithoutClaimsValidation())
	claims := &desktopDPoPClaims{}
	var proofJKT string
	token, err := parser.ParseWithClaims(proof, claims, func(token *jwt.Token) (any, error) {
		if !strings.EqualFold(strings.TrimSpace(fmt.Sprint(token.Header["typ"])), "dpop+jwt") {
			return nil, fmt.Errorf("invalid DPoP typ")
		}
		rawJWK, ok := token.Header["jwk"]
		if !ok {
			return nil, fmt.Errorf("missing DPoP jwk")
		}
		payload, err := json.Marshal(rawJWK)
		if err != nil {
			return nil, err
		}
		var jwk DesktopPublicJWK
		if err := json.Unmarshal(payload, &jwk); err != nil {
			return nil, err
		}
		publicKey, jkt, err := parseDesktopPublicJWK(jwk)
		if err != nil {
			return nil, err
		}
		proofJKT = jkt
		return publicKey, nil
	})
	if err != nil || token == nil || !token.Valid {
		return infraerrors.Unauthorized("DESKTOP_DPOP_INVALID", "DPoP proof is invalid")
	}
	if !constantTimeStringEqual(proofJKT, expectedJKT) {
		return infraerrors.Unauthorized("DESKTOP_DPOP_KEY_MISMATCH", "DPoP proof does not match the desktop session key")
	}
	if claims.IssuedAt == nil || strings.TrimSpace(claims.ID) == "" || len(claims.ID) > 128 {
		return infraerrors.Unauthorized("DESKTOP_DPOP_INVALID", "DPoP proof claims are invalid")
	}
	now := s.currentTime()
	issuedAt := claims.IssuedAt.Time
	if issuedAt.After(now.Add(desktopDPoPClockSkew)) || now.Sub(issuedAt) > desktopDPoPMaxAge {
		return infraerrors.Unauthorized("DESKTOP_DPOP_EXPIRED", "DPoP proof is outside the allowed time window")
	}
	if !desktopDPoPTargetMatches(target, claims.HTM, claims.HTU) {
		return infraerrors.Unauthorized("DESKTOP_DPOP_TARGET_MISMATCH", "DPoP proof target does not match this request")
	}
	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		expectedATH := base64.RawURLEncoding.EncodeToString(sum[:])
		if !constantTimeStringEqual(claims.ATH, expectedATH) {
			return infraerrors.Unauthorized("DESKTOP_DPOP_ATH_MISMATCH", "DPoP proof is not bound to the access token")
		}
	}
	proofKey := desktopDPoPProofKey(proofJKT, claims.ID)
	stored, err := s.store.StoreDPoPProof(ctx, proofKey, desktopDPoPProofTTL)
	if err != nil {
		return desktopStoreError(err)
	}
	if !stored {
		return infraerrors.Unauthorized("DESKTOP_DPOP_REPLAYED", "DPoP proof has already been used")
	}
	return nil
}

func (s *AuthService) generateDesktopAccessToken(user *User, sessionID, jkt string, now, expiresAt time.Time) (string, error) {
	tokenID, err := randomHexString(16)
	if err != nil {
		return "", fmt.Errorf("generate desktop access token id: %w", err)
	}
	claims := &desktopAccessTokenClaims{
		UserID: user.ID, TokenVersion: resolvedTokenVersion(user),
		TokenUse: DesktopTokenUse, Scopes: []string{DesktopWorkbenchSSOScope}, SessionID: sessionID,
		Confirmation: desktopAccessTokenConfirmation{JKT: jkt},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{DesktopAudience}, Subject: strconv.FormatInt(user.ID, 10), ID: tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign desktop access token: %w", err)
	}
	return signed, nil
}

func (s *AuthService) validateDesktopAccessToken(tokenString string) (*desktopAccessTokenClaims, error) {
	if s == nil || s.cfg == nil || strings.TrimSpace(s.cfg.JWT.Secret) == "" {
		return nil, ErrInvalidToken
	}
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" || len(tokenString) > maxTokenLength {
		return nil, ErrInvalidToken
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, err := parser.ParseWithClaims(tokenString, &desktopAccessTokenClaims{}, func(*jwt.Token) (any, error) {
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*desktopAccessTokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func parseDesktopPublicJWK(jwk DesktopPublicJWK) (*ecdsa.PublicKey, string, error) {
	if strings.TrimSpace(jwk.D) != "" {
		return nil, "", fmt.Errorf("private JWK material is not allowed")
	}
	if jwk.Kty != "EC" || jwk.Crv != "P-256" || (jwk.Alg != "" && jwk.Alg != "ES256") {
		return nil, "", fmt.Errorf("unsupported JWK")
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil || len(xBytes) != 32 {
		return nil, "", fmt.Errorf("invalid JWK x coordinate")
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil || len(yBytes) != 32 {
		return nil, "", fmt.Errorf("invalid JWK y coordinate")
	}
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}
	if !publicKey.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, "", fmt.Errorf("JWK point is not on P-256")
	}
	thumbprintPayload, err := json.Marshal(struct {
		Crv string `json:"crv"`
		Kty string `json:"kty"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}{Crv: "P-256", Kty: "EC", X: jwk.X, Y: jwk.Y})
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(thumbprintPayload)
	return publicKey, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func normalizeDesktopPKCEChallenge(challenge, method string) (string, error) {
	challenge = strings.TrimSpace(challenge)
	if method != "S256" {
		return "", infraerrors.BadRequest("DESKTOP_PKCE_INVALID", "code_challenge_method must be S256")
	}
	raw, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil || len(raw) != sha256.Size || len(challenge) < 43 || len(challenge) > 128 {
		return "", infraerrors.BadRequest("DESKTOP_PKCE_INVALID", "code_challenge is invalid")
	}
	return challenge, nil
}

func verifyDesktopPKCE(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	for _, r := range verifier {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && !strings.ContainsRune("-._~", r) {
			return false
		}
	}
	sum := sha256.Sum256([]byte(verifier))
	return constantTimeStringEqual(base64.RawURLEncoding.EncodeToString(sum[:]), challenge)
}

func normalizeDesktopRedirectURI(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Fragment != "" || u.RawQuery != "" || u.User != nil {
		return "", infraerrors.BadRequest("DESKTOP_REDIRECT_URI_INVALID", "redirect_uri is not allowed")
	}
	if u.Scheme == "ximoai" && u.Host == "desktop" && u.Path == "/callback" {
		return u.String(), nil
	}
	if u.Scheme == "http" && u.Path == "/callback" && u.Port() != "" {
		host := strings.TrimSpace(u.Hostname())
		if host == "127.0.0.1" || host == "::1" {
			return "http://" + net.JoinHostPort(host, u.Port()) + "/callback", nil
		}
	}
	return "", infraerrors.BadRequest("DESKTOP_REDIRECT_URI_INVALID", "redirect_uri must be the XimoAI desktop callback or a loopback callback")
}

func desktopDPoPTargetMatches(target DesktopDPoPTarget, method, htu string) bool {
	if !strings.EqualFold(strings.TrimSpace(method), strings.TrimSpace(target.Method)) {
		return false
	}
	normalizedHTU := normalizeDesktopHTU(htu)
	if normalizedHTU == "" {
		return false
	}
	for _, candidate := range target.URLs {
		if normalizeDesktopHTU(candidate) == normalizedHTU {
			return true
		}
	}
	return false
}

func normalizeDesktopHTU(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Host == "" || u.Fragment != "" || u.RawQuery != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	hostname := strings.ToLower(u.Hostname())
	port := u.Port()
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		port = ""
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	return strings.ToLower(u.Scheme) + "://" + host + path
}

func validDesktopAuthorizationRecord(record desktopAuthorizationCodeRecord) bool {
	return record.UserID > 0 && record.TokenVersion >= 0 && record.CodeChallenge != "" && record.RedirectURI != "" && record.JKT != "" && record.ExpiresAt > 0
}

func validDesktopSessionRecord(record desktopSessionRecord) bool {
	return record.SessionID != "" && record.UserID > 0 && record.TokenVersion >= 0 && record.JKT != "" && record.CurrentRefreshKey != "" && record.ExpiresAt > 0
}

func validDesktopRefreshRecord(record desktopRefreshRecord) bool {
	return record.SessionID != "" && record.UserID > 0 && record.TokenVersion >= 0 && record.JKT != "" && record.ExpiresAt > 0
}

func desktopSessionMatchesRefresh(session desktopSessionRecord, refresh desktopRefreshRecord) bool {
	return session.SessionID == refresh.SessionID && session.UserID == refresh.UserID && session.TokenVersion == refresh.TokenVersion && session.JKT == refresh.JKT && session.ExpiresAt == refresh.ExpiresAt
}

func validDesktopAccessClaims(claims *desktopAccessTokenClaims) bool {
	if claims == nil || claims.UserID <= 0 || claims.TokenUse != DesktopTokenUse || claims.SessionID == "" || claims.Confirmation.JKT == "" {
		return false
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != DesktopWorkbenchSSOScope {
		return false
	}
	for _, audience := range claims.Audience {
		if audience == DesktopAudience {
			return true
		}
	}
	return false
}

func randomDesktopSecret(prefix string) (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate desktop secret: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func desktopAuthorizationCodeKey(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, desktopAuthorizationCodePrefix) || len(code) != len(desktopAuthorizationCodePrefix)+43 {
		return "", infraerrors.Unauthorized("DESKTOP_AUTHORIZATION_INVALID", "authorization code is invalid or expired")
	}
	return desktopAuthorizationKeyPrefix + hashDesktopSecret(code), nil
}

func desktopRefreshKeys(token string) (string, string, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, desktopRefreshTokenPrefix) || len(token) != len(desktopRefreshTokenPrefix)+43 {
		return "", "", infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	hash := hashDesktopSecret(token)
	return desktopRefreshKeyPrefix + hash, desktopUsedRefreshKeyPrefix + hash, nil
}

func desktopSessionKey(sessionID string) string {
	return desktopSessionKeyPrefix + strings.TrimSpace(sessionID)
}

func desktopDPoPProofKey(jkt, jti string) string {
	return desktopDPoPProofKeyPrefix + hashDesktopSecret(jkt+"\n"+jti)
}

func hashDesktopSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func constantTimeStringEqual(actual, expected string) bool {
	if len(actual) != len(expected) || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func desktopStoreError(err error) error {
	return infraerrors.InternalServer("DESKTOP_SESSION_STORE_UNAVAILABLE", "desktop session store is unavailable").WithCause(err)
}
