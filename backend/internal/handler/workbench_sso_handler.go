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
	if !h.authorizeInternal(c) {
		response.Unauthorized(c, "Invalid Workbench SSO internal secret")
		return
	}
	var req workbenchSSOValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	userContext, err := h.ssoService.ValidateTicket(c.Request.Context(), req.Ticket, req.Audience)
	if response.ErrorFrom(c, err) {
		return
	}
	c.JSON(http.StatusOK, userContext)
}

func (h *WorkbenchSSOHandler) authorizeInternal(c *gin.Context) bool {
	if h == nil || h.ssoService == nil {
		return false
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	return service.ConstantTimeBearerEqual(strings.TrimSpace(strings.TrimPrefix(header, prefix)), h.ssoService.InternalSecret())
}
