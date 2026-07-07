//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchSSOHandler_CreateTicketRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewWorkbenchSSOHandler(nil)
	router.POST("/api/v1/workbench/sso-ticket", handler.CreateTicket)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sso-ticket", strings.NewReader(`{"audience":"http://127.0.0.1:4173"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWorkbenchSSOHandler_ValidateTicketRejectsInvalidInternalSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewWorkbenchSSOHandler(service.NewWorkbenchSSOService(
		&config.Config{WorkbenchSSO: config.WorkbenchSSOConfig{InternalSecret: "expected-secret"}},
		nil,
		nil,
		nil,
		nil,
		nil,
	))
	router.POST("/api/v1/workbench/sso-ticket/validate", handler.ValidateTicket)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sso-ticket/validate", strings.NewReader(`{"ticket":"t","audience":"http://127.0.0.1:4173"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
