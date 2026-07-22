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

func TestWorkbenchModelAccessHandlerRejectsInvalidInternalSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewWorkbenchModelAccessHandler(service.NewWorkbenchModelAccessService(
		&config.Config{WorkbenchSSO: config.WorkbenchSSOConfig{InternalSecret: "expected-secret"}},
		nil,
		nil,
	))
	router.POST("/api/v1/workbench/model-access", handler.Get)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/model-access", strings.NewReader(`{"userId":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
