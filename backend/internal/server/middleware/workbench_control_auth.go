package middleware

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var workbenchControlReadPaths = map[string]struct{}{
	"/api/v1/keys":                 {},
	"/api/v1/groups/available":     {},
	"/api/v1/groups/rates":         {},
	"/api/v1/platforms":            {},
	"/api/v1/channels/model-plaza": {},
}

func isWorkbenchControlRequestAllowed(claims *service.JWTClaims, method, path string) bool {
	if claims == nil || claims.TokenUse != service.WorkbenchControlTokenUse || method != http.MethodGet {
		return false
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != service.WorkbenchControlAudience {
		return false
	}
	if claims.Subject != strconv.FormatInt(claims.UserID, 10) || claims.ID == "" || claims.SessionID == "" {
		return false
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != service.WorkbenchModelControlReadScope {
		return false
	}
	_, ok := workbenchControlReadPaths[path]
	return ok
}
