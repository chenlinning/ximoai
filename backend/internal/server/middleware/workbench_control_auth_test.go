//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchControlTokenUsesExactReadOnlyNativeEndpointAllowlist(t *testing.T) {
	claims := &service.JWTClaims{
		UserID:    123,
		SessionID: "session-1",
		TokenUse:  service.WorkbenchControlTokenUse,
		Scopes:    []string{service.WorkbenchModelControlReadScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{service.WorkbenchControlAudience},
			Subject:  "123",
			ID:       "token-1",
		},
	}

	for _, path := range []string{
		"/api/v1/keys",
		"/api/v1/groups/available",
		"/api/v1/groups/rates",
		"/api/v1/channels/model-plaza",
	} {
		require.True(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, path), path)
		require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodPost, path), path)
	}

	for _, path := range []string{
		"/api/v1/keys/1",
		"/api/v1/user/profile",
		"/api/v1/admin/users",
		"/v1/responses",
		"/api/v1/workbench/model-access",
	} {
		require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, path), path)
	}
}

func TestWorkbenchControlTokenRejectsWrongAudienceOrScope(t *testing.T) {
	claims := &service.JWTClaims{
		UserID:    123,
		SessionID: "session-1",
		TokenUse:  service.WorkbenchControlTokenUse,
		Scopes:    []string{service.WorkbenchModelControlReadScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience: jwt.ClaimStrings{"other"},
			Subject:  "123",
			ID:       "token-1",
		},
	}
	require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, "/api/v1/keys"))

	claims.Audience = jwt.ClaimStrings{service.WorkbenchControlAudience}
	claims.Scopes = nil
	require.False(t, isWorkbenchControlRequestAllowed(claims, http.MethodGet, "/api/v1/keys"))
}

func TestJWTAuthEnforcesWorkbenchControlTokenScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-jwt-secret-32bytes-long!!!"}}
	user := &service.User{
		ID:                   123,
		Email:                "alice@example.com",
		Role:                 service.RoleUser,
		Status:               service.StatusActive,
		TokenVersion:         7,
		TokenVersionResolved: true,
	}
	userRepo := &stubJWTUserRepo{users: map[int64]*service.User{user.ID: user}}
	authService := service.NewAuthService(nil, userRepo, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	userService := service.NewUserService(userRepo, nil, nil, nil)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewJWTAuthMiddleware(authService, userService, nil, nil)))
	router.GET("/api/v1/keys", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/v1/keys", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/user/profile", func(c *gin.Context) { c.Status(http.StatusOK) })

	now := time.Now()
	claims := &service.JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		SessionID:    "session-1",
		TokenUse:     service.WorkbenchControlTokenUse,
		Scopes:       []string{service.WorkbenchModelControlReadScope},
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{service.WorkbenchControlAudience},
			Subject:   "123",
			ID:        "token-1",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWT.Secret))
	require.NoError(t, err)

	request := func(method, path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, nil).WithContext(context.Background())
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, req)
		return recorder
	}

	require.Equal(t, http.StatusOK, request(http.MethodGet, "/api/v1/keys").Code)
	require.Equal(t, http.StatusForbidden, request(http.MethodPost, "/api/v1/keys").Code)
	require.Equal(t, http.StatusForbidden, request(http.MethodGet, "/api/v1/user/profile").Code)
}
