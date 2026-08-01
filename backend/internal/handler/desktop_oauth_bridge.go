package handler

import (
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) redirectDesktopAuthorizationIfRequested(
	c *gin.Context,
	frontendCallback string,
	userID int64,
	redirectTo string,
) bool {
	callbackURL, handled, err := h.issueDesktopAuthorizationCallback(c, userID, redirectTo)
	if !handled {
		return false
	}
	if err != nil {
		redirectOAuthError(c, frontendCallback, infraerrors.Reason(err), infraerrors.Message(err), "")
		return true
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Redirect(http.StatusFound, callbackURL)
	return true
}

func (h *AuthHandler) writeDesktopAwareOAuthTokenPair(
	c *gin.Context,
	userID int64,
	redirectTo string,
	tokenPair *service.TokenPair,
) {
	callbackURL, handled, err := h.issueDesktopAuthorizationCallback(c, userID, redirectTo)
	if !handled {
		writeOAuthTokenPairResponse(c, tokenPair)
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{"desktop_callback_url": callbackURL})
}

func (h *AuthHandler) issueDesktopAuthorizationCallback(c *gin.Context, userID int64, redirectTo string) (string, bool, error) {
	if h == nil || h.desktopSession == nil || userID <= 0 || strings.TrimSpace(redirectTo) == "" {
		return "", false, nil
	}
	return h.desktopSession.IssueAuthorizationCodeForRedirect(c.Request.Context(), userID, redirectTo)
}
