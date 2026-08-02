package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const desktopRequestBodyLimit = 16 << 10

type desktopSessionAPI interface {
	CreateAuthorizationCode(ctx context.Context, userID int64, req service.DesktopAuthorizationRequest) (*service.DesktopAuthorizationGrant, error)
	ExchangeAuthorizationCode(ctx context.Context, req service.DesktopCodeExchangeRequest, dpopProof string, target service.DesktopDPoPTarget) (*service.DesktopTokenPair, error)
	Refresh(ctx context.Context, refreshToken, dpopProof string, target service.DesktopDPoPTarget) (*service.DesktopTokenPair, error)
	AuthenticateAccess(ctx context.Context, accessToken, dpopProof string, target service.DesktopDPoPTarget) (*service.DesktopIdentity, error)
	IssueWorkbenchTicket(ctx context.Context, identity *service.DesktopIdentity, workbenchID string) (*service.WorkbenchSSOTicket, error)
	RevokeAccess(ctx context.Context, accessToken, dpopProof string, target service.DesktopDPoPTarget) error
	RevokeRefresh(ctx context.Context, refreshToken, dpopProof string, target service.DesktopDPoPTarget) error
}

type DesktopSessionHandler struct {
	service desktopSessionAPI
	broker  desktopSSOBrokerAPI
}

type desktopSSOBrokerAPI interface {
	Issue(ctx context.Context, identity *service.DesktopIdentity, workbenchID string) (*service.DesktopSSOBrokerCredential, error)
}

type desktopTokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type desktopWorkbenchTicketRequest struct {
	WorkbenchID string `json:"workbench_id"`
}

type desktopRefreshRevokeRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func NewDesktopSessionHandler(desktopService *service.DesktopSessionService, brokerService *service.DesktopSSOBrokerService) *DesktopSessionHandler {
	return &DesktopSessionHandler{service: desktopService, broker: brokerService}
}

func (h *DesktopSessionHandler) DesktopService() *service.DesktopSessionService {
	if h == nil {
		return nil
	}
	desktopService, _ := h.service.(*service.DesktopSessionService)
	return desktopService
}

func (h *DesktopSessionHandler) Authorize(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.ErrorFrom(c, infraerrors.Unauthorized("DESKTOP_LOGIN_REQUIRED", "main site authentication is required"))
		return
	}
	var req service.DesktopAuthorizationRequest
	if !decodeDesktopJSON(c, &req) {
		return
	}
	grant, err := h.service.CreateAuthorizationCode(c.Request.Context(), subject.UserID, req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, grant)
}

func (h *DesktopSessionHandler) Token(c *gin.Context) {
	var req desktopTokenRequest
	if !decodeDesktopJSON(c, &req) {
		return
	}
	proof := strings.TrimSpace(c.GetHeader("DPoP"))
	if proof == "" {
		response.ErrorFrom(c, infraerrors.Unauthorized("DESKTOP_DPOP_INVALID", "DPoP proof is required"))
		return
	}
	target := desktopRequestTarget(c)

	var (
		pair *service.DesktopTokenPair
		err  error
	)
	switch req.GrantType {
	case "authorization_code":
		if req.Code == "" || req.CodeVerifier == "" || req.RedirectURI == "" || req.RefreshToken != "" {
			desktopInvalidRequest(c)
			return
		}
		pair, err = h.service.ExchangeAuthorizationCode(c.Request.Context(), service.DesktopCodeExchangeRequest{
			Code: req.Code, CodeVerifier: req.CodeVerifier, RedirectURI: req.RedirectURI,
		}, proof, target)
	case "refresh_token":
		if req.RefreshToken == "" || req.Code != "" || req.CodeVerifier != "" || req.RedirectURI != "" {
			desktopInvalidRequest(c)
			return
		}
		pair, err = h.service.Refresh(c.Request.Context(), req.RefreshToken, proof, target)
	default:
		desktopInvalidRequest(c)
		return
	}
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, pair)
}

func (h *DesktopSessionHandler) SSOTicket(c *gin.Context) {
	var req desktopWorkbenchTicketRequest
	if !decodeDesktopJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.WorkbenchID) == "" {
		desktopInvalidRequest(c)
		return
	}
	accessToken, ok := desktopDPoPAccessToken(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("DESKTOP_ACCESS_INVALID", "DPoP access token is required"))
		return
	}
	identity, err := h.service.AuthenticateAccess(c.Request.Context(), accessToken, strings.TrimSpace(c.GetHeader("DPoP")), desktopRequestTarget(c))
	if response.ErrorFrom(c, err) {
		return
	}
	ticket, err := h.service.IssueWorkbenchTicket(c.Request.Context(), identity, strings.TrimSpace(req.WorkbenchID))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, ticket)
}

func (h *DesktopSessionHandler) SSOBrokerCredential(c *gin.Context) {
	var req desktopWorkbenchTicketRequest
	if !decodeDesktopJSON(c, &req) {
		return
	}
	workbenchID := strings.TrimSpace(req.WorkbenchID)
	if workbenchID == "" {
		desktopInvalidRequest(c)
		return
	}
	accessToken, ok := desktopDPoPAccessToken(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("DESKTOP_ACCESS_INVALID", "DPoP access token is required"))
		return
	}
	identity, err := h.service.AuthenticateAccess(c.Request.Context(), accessToken, strings.TrimSpace(c.GetHeader("DPoP")), desktopRequestTarget(c))
	if response.ErrorFrom(c, err) {
		return
	}
	if h.broker == nil {
		response.ErrorFrom(c, infraerrors.InternalServer("DESKTOP_SSO_BROKER_UNAVAILABLE", "desktop sso broker is unavailable"))
		return
	}
	credential, err := h.broker.Issue(c.Request.Context(), identity, workbenchID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, credential)
}

func (h *DesktopSessionHandler) RevokeSession(c *gin.Context) {
	accessToken, ok := desktopDPoPAccessToken(c)
	if !ok {
		response.ErrorFrom(c, infraerrors.Unauthorized("DESKTOP_ACCESS_INVALID", "DPoP access token is required"))
		return
	}
	if response.ErrorFrom(c, h.service.RevokeAccess(c.Request.Context(), accessToken, strings.TrimSpace(c.GetHeader("DPoP")), desktopRequestTarget(c))) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DesktopSessionHandler) RevokeRefresh(c *gin.Context) {
	var req desktopRefreshRevokeRequest
	if !decodeDesktopJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		desktopInvalidRequest(c)
		return
	}
	if response.ErrorFrom(c, h.service.RevokeRefresh(c.Request.Context(), req.RefreshToken, strings.TrimSpace(c.GetHeader("DPoP")), desktopRequestTarget(c))) {
		return
	}
	c.Status(http.StatusNoContent)
}

func decodeDesktopJSON(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, desktopRequestBodyLimit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		desktopInvalidRequest(c)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		desktopInvalidRequest(c)
		return false
	}
	return true
}

func desktopInvalidRequest(c *gin.Context) {
	response.ErrorFrom(c, infraerrors.BadRequest("DESKTOP_REQUEST_INVALID", "desktop request is invalid"))
}

func desktopDPoPAccessToken(c *gin.Context) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(c.GetHeader("Authorization")))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "DPoP") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func desktopRequestTarget(c *gin.Context) service.DesktopDPoPTarget {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	u := url.URL{Scheme: scheme, Host: c.Request.Host, Path: c.Request.URL.Path}
	return service.DesktopDPoPTarget{Method: c.Request.Method, URLs: []string{u.String()}}
}
