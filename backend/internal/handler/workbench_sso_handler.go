package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type WorkbenchSSOHandler struct {
	ssoService *service.WorkbenchSSOService
	broker     desktopSSOBrokerAuthenticator
}

type desktopSSOBrokerAuthenticator interface {
	Authenticate(ctx context.Context, credential string) (*service.DesktopSSOBrokerIdentity, error)
	AuthenticateForAudience(ctx context.Context, credential, audience string) (*service.DesktopSSOBrokerIdentity, error)
}

type workbenchSSOTicketRequest struct {
	Audience string `json:"audience" binding:"required"`
}

type workbenchSSOTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
	EntryURL  string `json:"entry_url"`
}

type workbenchSSOValidateRequest struct {
	Ticket   string `json:"ticket" binding:"required"`
	Audience string `json:"audience" binding:"required"`
}

type workbenchControlTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func NewWorkbenchSSOHandler(ssoService *service.WorkbenchSSOService, brokerService *service.DesktopSSOBrokerService) *WorkbenchSSOHandler {
	return &WorkbenchSSOHandler{ssoService: ssoService, broker: brokerService}
}

func (h *WorkbenchSSOHandler) CreateTicket(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req workbenchSSOTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ticket, err := h.ssoService.IssueTicket(c.Request.Context(), subject.UserID, req.Audience)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, workbenchSSOTicketResponse{
		Ticket:    ticket.Ticket,
		ExpiresIn: ticket.ExpiresIn,
		EntryURL:  ticket.EntryURL,
	})
}

func (h *WorkbenchSSOHandler) ValidateTicket(c *gin.Context) {
	var req workbenchSSOValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	authorization, ok := h.resolveRequestAuthorization(c, req.Audience)
	if !ok {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	var userContext *service.WorkbenchSSOValidation
	var err error
	if authorization.UserID > 0 {
		userContext, err = h.ssoService.ValidateTicketForUser(c.Request.Context(), req.Ticket, authorization.Audience, authorization.UserID)
	} else {
		userContext, err = h.ssoService.ValidateTicket(c.Request.Context(), req.Ticket, authorization.Audience)
	}
	if response.ErrorFrom(c, err) {
		return
	}
	c.JSON(http.StatusOK, userContext)
}

func (h *WorkbenchSSOHandler) RefreshControlToken(c *gin.Context) {
	var req workbenchControlTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	authorization, ok := h.resolveRequestAuthorization(c, "")
	if !ok {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	var refreshed *service.WorkbenchControlAuthorization
	var err error
	if authorization.UserID > 0 {
		refreshed, err = h.ssoService.RefreshControlTokenForUser(c.Request.Context(), req.RefreshToken, authorization.Audience, authorization.UserID)
	} else {
		refreshed, err = h.ssoService.RefreshControlToken(c.Request.Context(), req.RefreshToken, authorization.Audience)
	}
	if response.ErrorFrom(c, err) {
		return
	}
	c.JSON(http.StatusOK, refreshed)
}

func (h *WorkbenchSSOHandler) RevokeControlToken(c *gin.Context) {
	var req workbenchControlTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	authorization, ok := h.resolveRequestAuthorization(c, "")
	if !ok {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	var err error
	if authorization.UserID > 0 {
		err = h.ssoService.RevokeControlTokenForUser(c.Request.Context(), req.RefreshToken, authorization.Audience, authorization.UserID)
	} else {
		err = h.ssoService.RevokeControlToken(c.Request.Context(), req.RefreshToken, authorization.Audience)
	}
	if response.ErrorFrom(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

type workbenchRequestAuthorization struct {
	Audience string
	UserID   int64
}

func (h *WorkbenchSSOHandler) resolveRequestAuthorization(c *gin.Context, requestedAudience string) (workbenchRequestAuthorization, bool) {
	if h == nil || h.ssoService == nil {
		return workbenchRequestAuthorization{}, false
	}
	credential, ok := workbenchBearerCredential(c)
	if !ok {
		return workbenchRequestAuthorization{}, false
	}
	if requestedAudience != "" {
		if h.ssoService.AuthorizeAudience(c.Request.Context(), requestedAudience, credential) {
			return workbenchRequestAuthorization{Audience: requestedAudience}, true
		}
	} else if audience, authorized := h.ssoService.ResolveAudienceCredential(c.Request.Context(), credential); authorized {
		return workbenchRequestAuthorization{Audience: audience}, true
	}
	if h.broker == nil {
		return workbenchRequestAuthorization{}, false
	}
	var identity *service.DesktopSSOBrokerIdentity
	var err error
	if requestedAudience != "" {
		identity, err = h.broker.AuthenticateForAudience(c.Request.Context(), credential, requestedAudience)
	} else {
		identity, err = h.broker.Authenticate(c.Request.Context(), credential)
	}
	if err != nil || identity == nil || identity.UserID <= 0 || identity.Audience == "" {
		return workbenchRequestAuthorization{}, false
	}
	return workbenchRequestAuthorization{Audience: identity.Audience, UserID: identity.UserID}, true
}

func workbenchBearerCredential(c *gin.Context) (string, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	credential := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return credential, credential != ""
}
