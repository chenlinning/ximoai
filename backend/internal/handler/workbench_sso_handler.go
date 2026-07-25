package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type WorkbenchSSOHandler struct {
	ssoService *service.WorkbenchSSOService
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

func NewWorkbenchSSOHandler(ssoService *service.WorkbenchSSOService) *WorkbenchSSOHandler {
	return &WorkbenchSSOHandler{ssoService: ssoService}
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
	if !h.authorizeAudience(c, req.Audience) {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	userContext, err := h.ssoService.ValidateTicket(c.Request.Context(), req.Ticket, req.Audience)
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
	audience, ok := h.resolveAudience(c)
	if !ok {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	authorization, err := h.ssoService.RefreshControlToken(c.Request.Context(), req.RefreshToken, audience)
	if response.ErrorFrom(c, err) {
		return
	}
	c.JSON(http.StatusOK, authorization)
}

func (h *WorkbenchSSOHandler) RevokeControlToken(c *gin.Context) {
	var req workbenchControlTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	audience, ok := h.resolveAudience(c)
	if !ok {
		response.Unauthorized(c, "Invalid Workbench SSO audience credentials")
		return
	}
	if response.ErrorFrom(c, h.ssoService.RevokeControlToken(c.Request.Context(), req.RefreshToken, audience)) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *WorkbenchSSOHandler) authorizeAudience(c *gin.Context, audience string) bool {
	credential, ok := workbenchBearerCredential(c)
	return ok && h != nil && h.ssoService != nil && h.ssoService.AuthorizeAudience(c.Request.Context(), audience, credential)
}

func (h *WorkbenchSSOHandler) resolveAudience(c *gin.Context) (string, bool) {
	if h == nil || h.ssoService == nil {
		return "", false
	}
	credential, ok := workbenchBearerCredential(c)
	if !ok {
		return "", false
	}
	return h.ssoService.ResolveAudienceCredential(c.Request.Context(), credential)
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
