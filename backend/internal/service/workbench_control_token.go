package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	WorkbenchControlAudience         = "workbench"
	WorkbenchControlTokenUse         = "workbench_control"
	WorkbenchModelControlReadScope   = "workbench:model-control:read"
	workbenchControlRefreshKeyPrefix = "workbench:control:refresh:"
	workbenchControlRefreshPrefix    = "wbr_"
	workbenchControlRefreshMaxLength = len(workbenchControlRefreshPrefix) + 64
)

const (
	workbenchControlAccessTTL  = 5 * time.Minute
	workbenchControlSessionTTL = 24 * time.Hour
)

type WorkbenchControlGrantStore interface {
	StoreGrant(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error)
	ConsumeGrant(ctx context.Context, key, ssoAudience string) (string, bool, error)
	ConsumeGrantForUser(ctx context.Context, key, ssoAudience string, userID int64) (string, bool, error)
	RevokeGrant(ctx context.Context, key, ssoAudience string) (bool, error)
	RevokeGrantForUser(ctx context.Context, key, ssoAudience string, userID int64) (bool, error)
	Ping(ctx context.Context) error
}

type WorkbenchControlTokenService struct {
	authService *AuthService
	userGetter  workbenchUserGetter
	grantStore  WorkbenchControlGrantStore
}

type WorkbenchControlAuthorization struct {
	AccessToken      string   `json:"accessToken"`
	RefreshToken     string   `json:"refreshToken"`
	TokenType        string   `json:"tokenType"`
	ExpiresIn        int      `json:"expiresIn"`
	RefreshExpiresIn int      `json:"refreshExpiresIn"`
	Audience         string   `json:"audience"`
	Scopes           []string `json:"scopes"`
}

type workbenchControlRefreshRecord struct {
	UserID       int64    `json:"user_id"`
	TokenVersion int64    `json:"token_version"`
	SessionID    string   `json:"session_id"`
	Audience     string   `json:"audience"`
	SSOAudience  string   `json:"sso_audience"`
	Scopes       []string `json:"scopes"`
	ExpiresAt    int64    `json:"expires_at"`
}

func NewWorkbenchControlTokenService(
	authService *AuthService,
	userService *UserService,
	grantStore WorkbenchControlGrantStore,
) *WorkbenchControlTokenService {
	return &WorkbenchControlTokenService{
		authService: authService,
		userGetter:  userService,
		grantStore:  grantStore,
	}
}

func (s *WorkbenchControlTokenService) Issue(ctx context.Context, userID int64, ssoAudience string) (*WorkbenchControlAuthorization, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	user, err := s.activeUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	sessionID, err := randomHexString(16)
	if err != nil {
		return nil, fmt.Errorf("generate workbench control session id: %w", err)
	}
	return s.issueForUser(ctx, user, sessionID, ssoAudience, time.Now().Add(workbenchControlSessionTTL))
}

func (s *WorkbenchControlTokenService) Refresh(ctx context.Context, refreshToken, ssoAudience string) (*WorkbenchControlAuthorization, error) {
	return s.refresh(ctx, refreshToken, ssoAudience, 0)
}

func (s *WorkbenchControlTokenService) RefreshForUser(ctx context.Context, refreshToken, ssoAudience string, userID int64) (*WorkbenchControlAuthorization, error) {
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	return s.refresh(ctx, refreshToken, ssoAudience, userID)
}

func (s *WorkbenchControlTokenService) refresh(ctx context.Context, refreshToken, ssoAudience string, expectedUserID int64) (*WorkbenchControlAuthorization, error) {
	if err := s.ensureAvailable(ctx); err != nil {
		return nil, err
	}
	key, err := workbenchControlRefreshKey(refreshToken)
	if err != nil {
		return nil, err
	}
	var raw string
	var ok bool
	if expectedUserID > 0 {
		raw, ok, err = s.grantStore.ConsumeGrantForUser(ctx, key, ssoAudience, expectedUserID)
	} else {
		raw, ok, err = s.grantStore.ConsumeGrant(ctx, key, ssoAudience)
	}
	if err != nil {
		return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_REDIS_UNAVAILABLE", "workbench control token store is unavailable").WithCause(err)
	}
	if !ok {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}

	var record workbenchControlRefreshRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil || !validWorkbenchControlRefreshRecord(record) {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	sessionExpiresAt := time.Unix(record.ExpiresAt, 0)
	if !time.Now().Before(sessionExpiresAt) {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_EXPIRED", "refresh token is expired")
	}
	user, err := s.activeUser(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if resolvedTokenVersion(user) != record.TokenVersion {
		return nil, ErrTokenRevoked
	}
	return s.issueForUser(ctx, user, record.SessionID, record.SSOAudience, sessionExpiresAt)
}

func (s *WorkbenchControlTokenService) Revoke(ctx context.Context, refreshToken, ssoAudience string) error {
	return s.revoke(ctx, refreshToken, ssoAudience, 0)
}

func (s *WorkbenchControlTokenService) RevokeForUser(ctx context.Context, refreshToken, ssoAudience string, userID int64) error {
	if userID <= 0 {
		return infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	return s.revoke(ctx, refreshToken, ssoAudience, userID)
}

func (s *WorkbenchControlTokenService) revoke(ctx context.Context, refreshToken, ssoAudience string, expectedUserID int64) error {
	if s == nil || s.grantStore == nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	key, err := workbenchControlRefreshKey(refreshToken)
	if err != nil {
		return err
	}
	var revoked bool
	if expectedUserID > 0 {
		revoked, err = s.grantStore.RevokeGrantForUser(ctx, key, ssoAudience, expectedUserID)
	} else {
		revoked, err = s.grantStore.RevokeGrant(ctx, key, ssoAudience)
	}
	if err != nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_REDIS_UNAVAILABLE", "workbench control token store is unavailable").WithCause(err)
	}
	if !revoked {
		return infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	return nil
}

func (s *WorkbenchControlTokenService) ensureAvailable(ctx context.Context) error {
	if s == nil || s.authService == nil || s.authService.cfg == nil || s.userGetter == nil || s.grantStore == nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	if strings.TrimSpace(s.authService.cfg.JWT.Secret) == "" {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	if err := s.grantStore.Ping(ctx); err != nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_REDIS_UNAVAILABLE", "workbench control token store is unavailable").WithCause(err)
	}
	return nil
}

func (s *WorkbenchControlTokenService) activeUser(ctx context.Context, userID int64) (*User, error) {
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_USER_INVALID", "user is unavailable")
	}
	user, err := s.userGetter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.ID != userID || !user.IsActive() {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_USER_INVALID", "user is unavailable")
	}
	return user, nil
}

func (s *WorkbenchControlTokenService) issueForUser(
	ctx context.Context,
	user *User,
	sessionID string,
	ssoAudience string,
	sessionExpiresAt time.Time,
) (*WorkbenchControlAuthorization, error) {
	now := time.Now()
	if sessionID == "" || strings.TrimSpace(ssoAudience) == "" || !now.Before(sessionExpiresAt) {
		return nil, infraerrors.Unauthorized("WORKBENCH_CONTROL_SESSION_EXPIRED", "workbench control session is expired")
	}
	accessExpiresAt := now.Add(workbenchControlAccessTTL)
	if accessExpiresAt.After(sessionExpiresAt) {
		accessExpiresAt = sessionExpiresAt
	}
	accessToken, err := s.authService.generateWorkbenchControlAccessToken(user, sessionID, now, accessExpiresAt)
	if err != nil {
		return nil, err
	}

	record := workbenchControlRefreshRecord{
		UserID:       user.ID,
		TokenVersion: resolvedTokenVersion(user),
		SessionID:    sessionID,
		Audience:     WorkbenchControlAudience,
		SSOAudience:  ssoAudience,
		Scopes:       []string{WorkbenchModelControlReadScope},
		ExpiresAt:    sessionExpiresAt.Unix(),
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal workbench control refresh record: %w", err)
	}

	refreshTTL := time.Until(sessionExpiresAt)
	for i := 0; i < 3; i++ {
		refreshToken, err := randomWorkbenchControlRefreshToken()
		if err != nil {
			return nil, err
		}
		key, err := workbenchControlRefreshKey(refreshToken)
		if err != nil {
			return nil, err
		}
		ok, err := s.grantStore.StoreGrant(ctx, key, payload, refreshTTL)
		if err != nil {
			return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_REDIS_UNAVAILABLE", "workbench control token store is unavailable").WithCause(err)
		}
		if !ok {
			continue
		}
		return &WorkbenchControlAuthorization{
			AccessToken:      accessToken,
			RefreshToken:     refreshToken,
			TokenType:        "Bearer",
			ExpiresIn:        int(accessExpiresAt.Sub(now).Seconds()),
			RefreshExpiresIn: int(refreshTTL.Seconds()),
			Audience:         WorkbenchControlAudience,
			Scopes:           []string{WorkbenchModelControlReadScope},
		}, nil
	}
	return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_TOKEN_COLLISION", "failed to create workbench control token")
}

func (s *AuthService) generateWorkbenchControlAccessToken(
	user *User,
	sessionID string,
	now time.Time,
	expiresAt time.Time,
) (string, error) {
	tokenID, err := randomHexString(16)
	if err != nil {
		return "", fmt.Errorf("generate workbench control token id: %w", err)
	}
	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: resolvedTokenVersion(user),
		SessionID:    sessionID,
		TokenUse:     WorkbenchControlTokenUse,
		Scopes:       []string{WorkbenchModelControlReadScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{WorkbenchControlAudience},
			Subject:   strconv.FormatInt(user.ID, 10),
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign workbench control token: %w", err)
	}
	return signed, nil
}

func validWorkbenchControlRefreshRecord(record workbenchControlRefreshRecord) bool {
	return record.UserID > 0 &&
		record.SessionID != "" &&
		record.Audience == WorkbenchControlAudience &&
		record.SSOAudience != "" &&
		len(record.Scopes) == 1 &&
		record.Scopes[0] == WorkbenchModelControlReadScope &&
		record.ExpiresAt > 0
}

func randomWorkbenchControlRefreshToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate workbench control refresh token: %w", err)
	}
	return workbenchControlRefreshPrefix + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func workbenchControlRefreshKey(token string) (string, error) {
	token = strings.TrimSpace(token)
	if len(token) <= len(workbenchControlRefreshPrefix) || len(token) > workbenchControlRefreshMaxLength || !strings.HasPrefix(token, workbenchControlRefreshPrefix) {
		return "", infraerrors.Unauthorized("WORKBENCH_CONTROL_REFRESH_INVALID", "refresh token is invalid or expired")
	}
	sum := sha256.Sum256([]byte(token))
	return workbenchControlRefreshKeyPrefix + hex.EncodeToString(sum[:]), nil
}
