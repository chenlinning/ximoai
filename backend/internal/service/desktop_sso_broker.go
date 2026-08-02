package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	DesktopSSOBrokerAudience = "workbench-sso-broker"
	DesktopSSOBrokerTokenUse = "desktop_sso_broker"
	DesktopSSOBrokerScope    = "workbench:sso:broker"
)

const desktopSSOBrokerCredentialTTL = 5 * time.Minute

const desktopSSOBrokerSigningContext = "ximoai-desktop-sso-broker:v1"

type DesktopSSOBrokerCredential struct {
	Credential  string `json:"credential"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	WorkbenchID string `json:"workbench_id"`
	Audience    string `json:"audience"`
}

type DesktopSSOBrokerIdentity struct {
	UserID           int64
	DesktopSessionID string
	TokenVersion     int64
	JKT              string
	WorkbenchID      string
	Audience         string
}

type desktopSSOBrokerClaims struct {
	UserID       int64                          `json:"user_id"`
	TokenVersion int64                          `json:"token_version"`
	TokenUse     string                         `json:"token_use"`
	Scopes       []string                       `json:"scopes"`
	SessionID    string                         `json:"sid"`
	WorkbenchID  string                         `json:"workbench_id"`
	Confirmation desktopAccessTokenConfirmation `json:"cnf"`
	jwt.RegisteredClaims
}

type DesktopSSOBrokerService struct {
	authService *AuthService
	desktop     *DesktopSessionService
	workbench   *WorkbenchSSOService
	now         func() time.Time
}

func NewDesktopSSOBrokerService(authService *AuthService, desktop *DesktopSessionService, workbench *WorkbenchSSOService) *DesktopSSOBrokerService {
	return &DesktopSSOBrokerService{authService: authService, desktop: desktop, workbench: workbench, now: time.Now}
}

func (s *DesktopSSOBrokerService) Issue(ctx context.Context, identity *DesktopIdentity, workbenchID string) (*DesktopSSOBrokerCredential, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	validated, err := s.desktop.validateIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}
	audience, err := s.workbench.ResolveWorkbenchForUser(ctx, validated.UserID, workbenchID)
	if err != nil {
		return nil, err
	}
	now := s.currentTime()
	expiresAt := now.Add(desktopSSOBrokerCredentialTTL)
	tokenID, err := randomHexString(16)
	if err != nil {
		return nil, fmt.Errorf("generate desktop sso broker token id: %w", err)
	}
	claims := &desktopSSOBrokerClaims{
		UserID: validated.UserID, TokenVersion: validated.TokenVersion,
		TokenUse: DesktopSSOBrokerTokenUse, Scopes: []string{DesktopSSOBrokerScope},
		SessionID: validated.SessionID, WorkbenchID: strings.TrimSpace(workbenchID),
		Confirmation: desktopAccessTokenConfirmation{JKT: validated.JKT},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{DesktopSSOBrokerAudience}, Subject: strconv.FormatInt(validated.UserID, 10), ID: tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(desktopSSOBrokerSigningKey(s.authService.cfg.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign desktop sso broker credential: %w", err)
	}
	return &DesktopSSOBrokerCredential{
		Credential: signed, TokenType: "Bearer", ExpiresIn: int(desktopSSOBrokerCredentialTTL.Seconds()),
		WorkbenchID: claims.WorkbenchID, Audience: audience,
	}, nil
}

func (s *DesktopSSOBrokerService) Authenticate(ctx context.Context, credential string) (*DesktopSSOBrokerIdentity, error) {
	if err := s.ensureAvailable(); err != nil {
		return nil, err
	}
	credential = strings.TrimSpace(credential)
	if credential == "" || len(credential) > maxTokenLength {
		return nil, invalidDesktopSSOBrokerCredential()
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithTimeFunc(s.currentTime),
	)
	token, err := parser.ParseWithClaims(credential, &desktopSSOBrokerClaims{}, func(*jwt.Token) (any, error) {
		return desktopSSOBrokerSigningKey(s.authService.cfg.JWT.Secret), nil
	})
	if err != nil || token == nil || !token.Valid {
		return nil, invalidDesktopSSOBrokerCredential()
	}
	claims, ok := token.Claims.(*desktopSSOBrokerClaims)
	if !ok || !validDesktopSSOBrokerClaims(claims, s.currentTime()) {
		return nil, invalidDesktopSSOBrokerCredential()
	}
	validated, err := s.desktop.validateIdentity(ctx, &DesktopIdentity{
		UserID: claims.UserID, SessionID: claims.SessionID, TokenVersion: claims.TokenVersion, JKT: claims.Confirmation.JKT,
	})
	if err != nil {
		return nil, err
	}
	audience, err := s.workbench.ResolveWorkbenchForUser(ctx, validated.UserID, claims.WorkbenchID)
	if err != nil {
		return nil, err
	}
	return &DesktopSSOBrokerIdentity{
		UserID: validated.UserID, DesktopSessionID: validated.SessionID, TokenVersion: validated.TokenVersion,
		JKT: validated.JKT, WorkbenchID: claims.WorkbenchID, Audience: audience,
	}, nil
}

func (s *DesktopSSOBrokerService) AuthenticateForAudience(ctx context.Context, credential, audience string) (*DesktopSSOBrokerIdentity, error) {
	identity, err := s.Authenticate(ctx, credential)
	if err != nil {
		return nil, err
	}
	if normalizeWorkbenchAudience(audience) != identity.Audience {
		return nil, invalidDesktopSSOBrokerCredential()
	}
	return identity, nil
}

func (s *DesktopSSOBrokerService) ensureAvailable() error {
	if s == nil || s.authService == nil || s.authService.cfg == nil || s.desktop == nil || s.workbench == nil || strings.TrimSpace(s.authService.cfg.JWT.Secret) == "" {
		return infraerrors.InternalServer("DESKTOP_SSO_BROKER_UNAVAILABLE", "desktop sso broker is unavailable")
	}
	return nil
}

func (s *DesktopSSOBrokerService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func validDesktopSSOBrokerClaims(claims *desktopSSOBrokerClaims, now time.Time) bool {
	if claims == nil || claims.UserID <= 0 || claims.TokenUse != DesktopSSOBrokerTokenUse || claims.SessionID == "" || claims.WorkbenchID == "" || claims.Confirmation.JKT == "" {
		return false
	}
	if claims.Subject != strconv.FormatInt(claims.UserID, 10) || strings.TrimSpace(claims.ID) == "" {
		return false
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != DesktopSSOBrokerAudience || len(claims.Scopes) != 1 || claims.Scopes[0] != DesktopSSOBrokerScope {
		return false
	}
	if claims.IssuedAt == nil || claims.NotBefore == nil || claims.ExpiresAt == nil {
		return false
	}
	issuedAt := claims.IssuedAt.Time
	expiresAt := claims.ExpiresAt.Time
	return !issuedAt.After(now.Add(desktopDPoPClockSkew)) && expiresAt.After(now) && expiresAt.Sub(issuedAt) <= desktopSSOBrokerCredentialTTL
}

func invalidDesktopSSOBrokerCredential() error {
	return infraerrors.Unauthorized("DESKTOP_SSO_BROKER_INVALID", "desktop sso broker credential is invalid or expired")
}

func desktopSSOBrokerSigningKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(desktopSSOBrokerSigningContext))
	return mac.Sum(nil)
}
