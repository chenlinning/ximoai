//go:build unit

package middleware

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestDesktopAccessTokenCannotUseMainSiteOrWorkbenchControlEndpoints(t *testing.T) {
	claims := &service.JWTClaims{
		UserID:    123,
		SessionID: "desktop-session",
		TokenUse:  service.DesktopTokenUse,
		Scopes:    []string{service.DesktopWorkbenchSSOScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{service.DesktopAudience},
			Subject:  "123",
			ID:       "desktop-access-token",
		},
	}

	for _, path := range []string{
		"/api/v1/user/profile",
		"/api/v1/keys",
		"/api/v1/groups/available",
		"/api/v1/admin/users",
		"/v1/responses",
	} {
		require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, path), path)
	}
}

func TestDesktopSSOBrokerCredentialCannotUseCatalogEndpoints(t *testing.T) {
	claims := &service.JWTClaims{
		UserID:    123,
		SessionID: "desktop-session",
		TokenUse:  service.DesktopSSOBrokerTokenUse,
		Scopes:    []string{service.DesktopSSOBrokerScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{service.DesktopSSOBrokerAudience},
			Subject:  "123",
			ID:       "desktop-sso-broker",
		},
	}

	for _, path := range []string{
		"/api/v1/keys",
		"/api/v1/groups/available",
		"/api/v1/groups/rates",
		"/api/v1/channels/model-plaza",
	} {
		require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, path), path)
	}
}
