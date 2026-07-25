package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type videoCollectorMembershipGetter interface {
	GetUserMembership(ctx context.Context, userID int64) (*service.MembershipSummary, error)
}

type VideoCollectorHandler struct {
	memberships   videoCollectorMembershipGetter
	internalURL   string
	internalToken string
	client        *http.Client
	now           func() time.Time
}

func NewVideoCollectorHandler(membershipService *service.MembershipService) *VideoCollectorHandler {
	return newVideoCollectorHandler(
		membershipService,
		videoCollectorEnvOrDefault("VIDEO_COLLECTOR_INTERNAL_URL", "http://video-collector:8090"),
		os.Getenv("VIDEO_COLLECTOR_INTERNAL_TOKEN"),
		&http.Client{},
	)
}

func newVideoCollectorHandler(memberships videoCollectorMembershipGetter, internalURL, internalToken string, client *http.Client) *VideoCollectorHandler {
	if client == nil {
		client = &http.Client{}
	}
	return &VideoCollectorHandler{
		memberships:   memberships,
		internalURL:   strings.TrimRight(internalURL, "/"),
		internalToken: internalToken,
		client:        client,
		now:           time.Now,
	}
}

func (h *VideoCollectorHandler) Parse(c *gin.Context) {
	h.proxy(c, http.MethodPost, "/internal/v1/parse", true)
}

func (h *VideoCollectorHandler) Start(c *gin.Context) {
	h.proxy(c, http.MethodPost, "/internal/v1/tasks", true)
}

func (h *VideoCollectorHandler) GetTask(c *gin.Context) {
	h.proxyTask(c, http.MethodGet, "")
}

func (h *VideoCollectorHandler) Cancel(c *gin.Context) {
	h.proxyTask(c, http.MethodDelete, "")
}

func (h *VideoCollectorHandler) Download(c *gin.Context) {
	h.proxyTask(c, http.MethodGet, "/download")
}

func (h *VideoCollectorHandler) proxyTask(c *gin.Context, method, suffix string) {
	taskID := strings.TrimSpace(c.Param("id"))
	if len(taskID) != 32 || strings.Trim(taskID, "0123456789abcdef") != "" {
		response.NotFound(c, "Video task not found")
		return
	}
	h.proxy(c, method, "/internal/v1/tasks/"+url.PathEscape(taskID)+suffix, false)
}

func (h *VideoCollectorHandler) proxy(c *gin.Context, method, path string, forwardBody bool) {
	userID, ok := h.authorize(c)
	if !ok {
		return
	}
	if len(h.internalToken) < 32 || h.internalURL == "" {
		response.Error(c, http.StatusServiceUnavailable, "Video collector is not configured")
		return
	}
	var body io.Reader
	if forwardBody {
		body = c.Request.Body
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, h.internalURL+path, body)
	if err != nil {
		response.InternalError(c, "Failed to create video collector request")
		return
	}
	req.Header.Set("X-Video-Collector-Token", h.internalToken)
	req.Header.Set("X-Video-Collector-User", fmt.Sprintf("%d", userID))
	if contentType := c.GetHeader("Content-Type"); contentType != "" && forwardBody {
		req.Header.Set("Content-Type", contentType)
	}
	for _, header := range []string{"Range", "If-Range"} {
		if value := c.GetHeader(header); value != "" {
			req.Header.Set(header, value)
		}
	}

	upstream, err := h.client.Do(req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "Video collector is unavailable")
		return
	}
	defer upstream.Body.Close()
	for _, header := range []string{
		"Content-Type", "Content-Length", "Content-Disposition", "Content-Range",
		"Accept-Ranges", "Cache-Control", "Last-Modified", "X-Accel-Buffering", "X-Delete-At",
	} {
		if value := upstream.Header.Get(header); value != "" {
			c.Header(header, value)
		}
	}
	c.Status(upstream.StatusCode)
	_, _ = io.Copy(c.Writer, upstream.Body)
}

func (h *VideoCollectorHandler) authorize(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if h.memberships == nil {
		response.Error(c, http.StatusServiceUnavailable, "Membership service is unavailable")
		return 0, false
	}
	summary, err := h.memberships.GetUserMembership(c.Request.Context(), subject.UserID)
	if err != nil {
		if response.ErrorFrom(c, err) {
			return 0, false
		}
	}
	now := h.now()
	if summary == nil || summary.Level == nil || !summary.Level.Enabled || summary.Level.Code != service.MembershipLevelCodeDiamond || summary.StartsAt.After(now) || (summary.ExpiresAt != nil && !summary.ExpiresAt.After(now)) {
		response.Forbidden(c, "Diamond membership is required")
		return 0, false
	}
	return subject.UserID, true
}

func videoCollectorEnvOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
