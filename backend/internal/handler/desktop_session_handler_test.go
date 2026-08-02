package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type desktopSessionHandlerServiceStub struct {
	authorizeUserID int64
	authorizeReq    service.DesktopAuthorizationRequest
	exchangeReq     service.DesktopCodeExchangeRequest
	refreshToken    string
	accessToken     string
	dpopProof       string
	target          service.DesktopDPoPTarget
	workbenchID     string
}

type desktopSSOBrokerHandlerStub struct {
	identity    *service.DesktopIdentity
	workbenchID string
}

func (s *desktopSSOBrokerHandlerStub) Issue(_ context.Context, identity *service.DesktopIdentity, workbenchID string) (*service.DesktopSSOBrokerCredential, error) {
	s.identity = identity
	s.workbenchID = workbenchID
	return &service.DesktopSSOBrokerCredential{
		Credential: "broker-credential", TokenType: "Bearer", ExpiresIn: 300,
		WorkbenchID: workbenchID, Audience: "https://image.ximoai.cn",
	}, nil
}

func (s *desktopSessionHandlerServiceStub) CreateAuthorizationCode(_ context.Context, userID int64, req service.DesktopAuthorizationRequest) (*service.DesktopAuthorizationGrant, error) {
	s.authorizeUserID = userID
	s.authorizeReq = req
	return &service.DesktopAuthorizationGrant{Code: "dca_code", ExpiresIn: 300}, nil
}

func (s *desktopSessionHandlerServiceStub) ExchangeAuthorizationCode(_ context.Context, req service.DesktopCodeExchangeRequest, proof string, target service.DesktopDPoPTarget) (*service.DesktopTokenPair, error) {
	s.exchangeReq, s.dpopProof, s.target = req, proof, target
	return desktopHandlerTokenPair(), nil
}

func (s *desktopSessionHandlerServiceStub) Refresh(_ context.Context, refreshToken, proof string, target service.DesktopDPoPTarget) (*service.DesktopTokenPair, error) {
	s.refreshToken, s.dpopProof, s.target = refreshToken, proof, target
	return desktopHandlerTokenPair(), nil
}

func (s *desktopSessionHandlerServiceStub) AuthenticateAccess(_ context.Context, accessToken, proof string, target service.DesktopDPoPTarget) (*service.DesktopIdentity, error) {
	s.accessToken, s.dpopProof, s.target = accessToken, proof, target
	if accessToken == "reject" {
		return nil, infraerrors.Unauthorized("DESKTOP_ACCESS_INVALID", "desktop access token is invalid or expired")
	}
	return &service.DesktopIdentity{UserID: 42, SessionID: "session"}, nil
}

func (s *desktopSessionHandlerServiceStub) IssueWorkbenchTicket(_ context.Context, _ *service.DesktopIdentity, workbenchID string) (*service.WorkbenchSSOTicket, error) {
	s.workbenchID = workbenchID
	return &service.WorkbenchSSOTicket{Ticket: "ticket", ExpiresIn: 60, EntryURL: "https://image.ximoai.cn/sso/entry?ticket=ticket"}, nil
}

func (s *desktopSessionHandlerServiceStub) RevokeAccess(_ context.Context, accessToken, proof string, target service.DesktopDPoPTarget) error {
	s.accessToken, s.dpopProof, s.target = accessToken, proof, target
	return nil
}

func (s *desktopSessionHandlerServiceStub) RevokeRefresh(_ context.Context, refreshToken, proof string, target service.DesktopDPoPTarget) error {
	s.refreshToken, s.dpopProof, s.target = refreshToken, proof, target
	return nil
}

func TestDesktopSessionHandlerAuthorizeUsesExistingLoginAndStrictJSON(t *testing.T) {
	stub := &desktopSessionHandlerServiceStub{}
	h := &DesktopSessionHandler{service: stub}
	router := gin.New()
	router.POST("/api/v1/desktop/authorize", func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 42})
		c.Next()
	}, h.Authorize)

	body := `{"code_challenge":"challenge","code_challenge_method":"S256","device_jwk":{"kty":"EC","crv":"P-256","x":"x","y":"y"},"redirect_uri":"ximoai://desktop/callback"}`
	w := desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/authorize", body, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(42), stub.authorizeUserID)
	require.Equal(t, "ximoai://desktop/callback", stub.authorizeReq.RedirectURI)
	require.NotContains(t, w.Body.String(), "access_token")
	require.NotContains(t, w.Body.String(), "refresh_token")

	unknown := strings.TrimSuffix(body, "}") + `,"audience":"https://evil.example"}`
	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/authorize", unknown, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDesktopSessionHandlerTokenAndTicketUseDPoP(t *testing.T) {
	stub := &desktopSessionHandlerServiceStub{}
	broker := &desktopSSOBrokerHandlerStub{}
	h := &DesktopSessionHandler{service: stub, broker: broker}
	router := gin.New()
	router.POST("/api/v1/desktop/token", h.Token)
	router.POST("/api/v1/desktop/sso-ticket", h.SSOTicket)
	router.POST("/api/v1/desktop/sso-broker-credential", h.SSOBrokerCredential)

	headers := map[string]string{"DPoP": "proof", "X-Forwarded-Proto": "https"}
	w := desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/token", `{"grant_type":"authorization_code","code":"dca_code","code_verifier":"verifier","redirect_uri":"ximoai://desktop/callback"}`, headers)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "dca_code", stub.exchangeReq.Code)
	require.Equal(t, "proof", stub.dpopProof)
	require.Equal(t, []string{"https://example.com/api/v1/desktop/token"}, stub.target.URLs)

	headers["Authorization"] = "DPoP desktop-access"
	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/sso-ticket", `{"workbench_id":"image"}`, headers)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "desktop-access", stub.accessToken)
	require.Equal(t, "image", stub.workbenchID)
	require.NotContains(t, w.Body.String(), "desktop-access")

	headers["Authorization"] = "Bearer desktop-access"
	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/sso-ticket", `{"workbench_id":"image"}`, headers)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	headers["Authorization"] = "DPoP desktop-access"
	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/sso-broker-credential", `{"workbench_id":"image"}`, headers)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "desktop-access", stub.accessToken)
	require.Equal(t, "image", broker.workbenchID)
	require.Equal(t, int64(42), broker.identity.UserID)
	require.Contains(t, w.Body.String(), "broker-credential")

	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/sso-broker-credential", `{"workbench_id":"image","audience":"https://evil.example"}`, headers)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDesktopSessionHandlerRefreshAndRevocation(t *testing.T) {
	stub := &desktopSessionHandlerServiceStub{}
	h := &DesktopSessionHandler{service: stub}
	router := gin.New()
	router.POST("/api/v1/desktop/token", h.Token)
	router.DELETE("/api/v1/desktop/session", h.RevokeSession)
	router.POST("/api/v1/desktop/revoke", h.RevokeRefresh)
	headers := map[string]string{"DPoP": "proof", "Authorization": "DPoP desktop-access"}

	w := desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/token", `{"grant_type":"refresh_token","refresh_token":"dtr_refresh"}`, headers)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "dtr_refresh", stub.refreshToken)

	w = desktopHandlerRequest(t, router, http.MethodDelete, "/api/v1/desktop/session", "", headers)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "desktop-access", stub.accessToken)

	w = desktopHandlerRequest(t, router, http.MethodPost, "/api/v1/desktop/revoke", `{"refresh_token":"dtr_refresh"}`, headers)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func desktopHandlerTokenPair() *service.DesktopTokenPair {
	return &service.DesktopTokenPair{
		AccessToken: "desktop-access", RefreshToken: "desktop-refresh", TokenType: "DPoP",
		ExpiresIn: 300, RefreshExpiresIn: 3600, DesktopSessionID: "session", Scope: service.DesktopWorkbenchSSOScope,
	}
}

func desktopHandlerRequest(t *testing.T, router http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
