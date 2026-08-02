//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type workbenchBrokerAuthenticatorStub struct {
	identity *service.DesktopSSOBrokerIdentity
	err      error
}

func (s workbenchBrokerAuthenticatorStub) Authenticate(context.Context, string) (*service.DesktopSSOBrokerIdentity, error) {
	return s.identity, s.err
}

func (s workbenchBrokerAuthenticatorStub) AuthenticateForAudience(context.Context, string, string) (*service.DesktopSSOBrokerIdentity, error) {
	return s.identity, s.err
}

func TestWorkbenchSSOHandler_CreateTicketRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewWorkbenchSSOHandler(nil, nil)
	router.POST("/api/v1/workbench/sso-ticket", handler.CreateTicket)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sso-ticket", strings.NewReader(`{"audience":"http://127.0.0.1:4173"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWorkbenchSSOHandler_ValidateTicketRejectsInvalidAudienceCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewWorkbenchSSOHandler(service.NewWorkbenchSSOService(
		&config.Config{WorkbenchSSO: config.WorkbenchSSOConfig{InternalSecret: "test-workbench-master-secret-32-bytes-long"}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	), nil)
	router.POST("/api/v1/workbench/sso-ticket/validate", handler.ValidateTicket)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sso-ticket/validate", strings.NewReader(`{"ticket":"t","audience":"http://127.0.0.1:4173"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-secret")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWorkbenchSSOHandlerAcceptsDesktopBrokerCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &WorkbenchSSOHandler{
		ssoService: service.NewWorkbenchSSOService(&config.Config{}, nil, nil, nil, nil, nil, nil),
		broker: workbenchBrokerAuthenticatorStub{identity: &service.DesktopSSOBrokerIdentity{
			UserID: 42, WorkbenchID: "image", Audience: "https://image.ximoai.cn",
		}},
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/workbench/sso-ticket/validate", nil)
	c.Request.Header.Set("Authorization", "Bearer broker-credential")

	authorization, ok := h.resolveRequestAuthorization(c, "https://image.ximoai.cn/app")
	require.True(t, ok)
	require.Equal(t, int64(42), authorization.UserID)
	require.Equal(t, "https://image.ximoai.cn", authorization.Audience)
}
