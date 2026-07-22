package server

import "net/http"

const ximoAIAppPackageUploadPath = "/api/v1/admin/ximoapp/update/packages"
const ximoAIAppPackageUploadBodyLimit int64 = 1024 << 20

type ximoAIRequestBodyLimitHandler struct {
	next         http.Handler
	defaultLimit int64
}

func newXimoAIRequestBodyLimitHandler(next http.Handler, defaultLimit int64) http.Handler {
	return &ximoAIRequestBodyLimitHandler{next: next, defaultLimit: defaultLimit}
}

func (h *ximoAIRequestBodyLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limit := h.defaultLimit
	if r != nil && r.URL != nil && r.URL.Path == ximoAIAppPackageUploadPath && limit < ximoAIAppPackageUploadBodyLimit {
		limit = ximoAIAppPackageUploadBodyLimit
	}
	http.MaxBytesHandler(h.next, limit).ServeHTTP(w, r)
}
