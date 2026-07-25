package videocollector

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const internalAPIBase = "/internal/v1"

type HTTPServer struct {
	manager *Manager
	token   string
}

func NewHTTPServer(manager *Manager, token string) (*HTTPServer, error) {
	if manager == nil || len(token) < 32 {
		return nil, errors.New("video collector internal token must contain at least 32 characters")
	}
	return &HTTPServer{manager: manager, token: token}, nil
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path == "/health" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if !s.authenticate(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := strconv.ParseInt(r.Header.Get("X-Video-Collector-User"), 10, 64)
	if err != nil || userID <= 0 {
		writeJSONError(w, http.StatusUnauthorized, "invalid user identity")
		return
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == internalAPIBase+"/parse":
		s.handleParse(w, r)
	case r.Method == http.MethodPost && r.URL.Path == internalAPIBase+"/tasks":
		s.handleStart(w, r, userID)
	case strings.HasPrefix(r.URL.Path, internalAPIBase+"/tasks/"):
		s.handleTask(w, r, userID)
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}

func (s *HTTPServer) authenticate(r *http.Request) bool {
	provided := r.Header.Get("X-Video-Collector-Token")
	return len(provided) == len(s.token) && subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) == 1
}

func (s *HTTPServer) handleParse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if !decodeJSONBody(w, r, &input) {
		return
	}
	info, err := s.manager.Parse(r.Context(), input.URL)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, safeTaskError(err))
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *HTTPServer) handleStart(w http.ResponseWriter, r *http.Request, userID int64) {
	var input DownloadRequest
	if !decodeJSONBody(w, r, &input) {
		return
	}
	task, err := s.manager.Start(userID, input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrTaskAlreadyActive) {
			status = http.StatusConflict
		}
		writeJSONError(w, status, safeTaskError(err))
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (s *HTTPServer) handleTask(w http.ResponseWriter, r *http.Request, userID int64) {
	relative := strings.TrimPrefix(r.URL.Path, internalAPIBase+"/tasks/")
	parts := strings.Split(relative, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	taskID := parts[0]
	if len(parts) == 2 && parts[1] == "download" && r.Method == http.MethodGet {
		s.handleDownload(w, r, userID, taskID)
		return
	}
	if len(parts) != 1 {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		task, err := s.manager.Get(userID, taskID)
		if err != nil {
			writeManagerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, task)
	case http.MethodDelete:
		if err := s.manager.Cancel(userID, taskID); err != nil {
			writeManagerError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleDownload(w http.ResponseWriter, r *http.Request, userID int64, taskID string) {
	lease, err := s.manager.OpenDownload(userID, taskID)
	if err != nil {
		writeManagerError(w, err)
		return
	}
	defer func() { _ = lease.Close() }()
	fileInfo, err := lease.Stat()
	if err != nil {
		writeJSONError(w, http.StatusGone, "download file is no longer available")
		return
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": lease.FileName})
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Delete-At", lease.DeleteAt.UTC().Format("2006-01-02T15:04:05Z"))
	http.ServeContent(w, r, lease.FileName, fileInfo.ModTime(), lease.File)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeManagerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTaskNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrTaskNotReady):
		writeJSONError(w, http.StatusGone, err.Error())
	default:
		writeJSONError(w, http.StatusBadRequest, safeTaskError(err))
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"code": status, "message": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		_, _ = fmt.Fprint(w, "{}")
	}
}
