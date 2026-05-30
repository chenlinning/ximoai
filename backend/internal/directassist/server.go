package directassist

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	cfg    Config
	store  *Store
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(cfg Config, store *Store, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		cfg:    cfg,
		store:  store,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              s.cfg.SignalAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      s.cfg.LongPollTimeout + 10*time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc(BasePath+"/health", s.withRateLimit("health", s.cfg.RateLimitPerMinute, s.handleHealth))
	s.mux.HandleFunc(BasePath+"/heartbeat", s.withRateLimit("heartbeat", s.cfg.HeartbeatRateLimitPerMinute, s.handleHeartbeat))
	s.mux.HandleFunc(BasePath+"/devices/", s.withRateLimit("devices", s.cfg.RateLimitPerMinute, s.handleDeviceQuery))
	s.mux.HandleFunc(BasePath+"/sessions", s.withRateLimit("sessions", s.cfg.RateLimitPerMinute, s.handleSessions))
	s.mux.HandleFunc(BasePath+"/sessions/", s.withRateLimit("session_action", s.cfg.RateLimitPerMinute, s.handleSessionAction))
	s.mux.HandleFunc(BasePath+"/events", s.withRateLimit("events", s.cfg.LongPollingRateLimitPerMinute, s.handleEvents))
}

type handlerFunc func(http.ResponseWriter, *http.Request)

func (s *Server) withRateLimit(scope string, limit int64, next handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := s.store.RateLimit(r.Context(), scope, remoteIP(r), limit)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "rate limit unavailable")
			return
		}
		if !allowed {
			s.writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := s.store.Ping(r.Context()); err != nil {
		s.writeError(w, http.StatusServiceUnavailable, "redis unavailable")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "direct-assist-signal",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req HeartbeatRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		req.Status = "online"
	}
	secret := deviceSecretFromRequest(r, req.DeviceSecret)
	signature := signatureFromRequest(r, req.DeviceSignature)
	if req.DeviceCode == "" || req.DeviceID == "" || secret == "" || signature == "" {
		s.writeError(w, http.StatusBadRequest, "deviceCode, deviceId, deviceSecret and deviceSignature are required")
		return
	}
	if !verifyHMAC(secret, canonicalHeartbeatMessage(req.DeviceID, req.DeviceCode), signature) {
		s.writeError(w, http.StatusUnauthorized, "invalid device signature")
		return
	}
	if err := s.store.VerifyDeviceSecret(r.Context(), req.DeviceID, secret); err != nil && !errors.Is(err, errNotFound) {
		s.handleAuthError(w, err)
		return
	}
	if err := s.store.SaveDeviceSecret(r.Context(), req.DeviceID, secret); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save device secret")
		return
	}
	udpToken, err := generateToken(24)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create udp token")
		return
	}
	device, err := s.store.SaveHeartbeat(r.Context(), req, remoteIP(r), udpToken)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save heartbeat")
		return
	}
	s.writeJSON(w, http.StatusOK, HeartbeatResponse{
		DeviceID:         device.DeviceID,
		Online:           device.Online,
		LastHeartbeatAt:  device.LastHeartbeatAt,
		PublicTCPIP:      device.PublicTCPIP,
		UDPProbeToken:    udpToken,
		UDPProbeTokenTTL: int64(s.cfg.UDPProbeTokenTTL / time.Second),
		DeviceTTLSeconds: int64(s.cfg.DeviceTTL / time.Second),
	})
}

func (s *Server) handleDeviceQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceCode := strings.TrimPrefix(r.URL.Path, BasePath+"/devices/")
	deviceCode = strings.Trim(deviceCode, "/")
	if deviceCode == "" || strings.Contains(deviceCode, "/") {
		s.writeError(w, http.StatusBadRequest, "deviceCode is required")
		return
	}
	device, err := s.store.GetDeviceByCode(r.Context(), deviceCode)
	if errors.Is(err, errNotFound) {
		s.writeError(w, http.StatusNotFound, "device offline")
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query device")
		return
	}
	device.DeviceCode = deviceCode
	s.writeJSON(w, http.StatusOK, DeviceQueryResponse{Device: device})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req CreateSessionRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.ControllerDeviceID = strings.TrimSpace(req.ControllerDeviceID)
	req.TargetDeviceID = strings.TrimSpace(req.TargetDeviceID)
	req.DeviceCode = strings.TrimSpace(req.DeviceCode)
	req.SessionType = strings.TrimSpace(req.SessionType)
	req.AcceptMode = strings.TrimSpace(req.AcceptMode)
	if req.ControllerDeviceID == "" || req.SessionType == "" || req.AcceptMode == "" {
		s.writeError(w, http.StatusBadRequest, "controllerDeviceId, sessionType and acceptMode are required")
		return
	}
	if !validSessionType(req.SessionType) || !validAcceptMode(req.AcceptMode) {
		s.writeError(w, http.StatusBadRequest, "invalid sessionType or acceptMode")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, req.ControllerDeviceID, req.ControllerDeviceID+":create_session"); err != nil {
		s.handleAuthError(w, err)
		return
	}
	if req.TargetDeviceID == "" {
		if req.DeviceCode == "" {
			s.writeError(w, http.StatusBadRequest, "targetDeviceId or deviceCode is required")
			return
		}
		device, err := s.store.GetDeviceByCode(r.Context(), req.DeviceCode)
		if errors.Is(err, errNotFound) {
			s.writeError(w, http.StatusNotFound, "target device offline")
			return
		}
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to query target device")
			return
		}
		req.TargetDeviceID = device.DeviceID
	} else if _, err := s.store.GetDevice(r.Context(), req.TargetDeviceID); err != nil {
		if errors.Is(err, errNotFound) {
			s.writeError(w, http.StatusNotFound, "target device offline")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "failed to query target device")
		return
	}
	session, err := s.store.CreateSession(r.Context(), req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	s.writeJSON(w, http.StatusCreated, SessionResponse{Session: session})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		s.writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, deviceID, deviceID+":events"); err != nil {
		s.handleAuthError(w, err)
		return
	}
	events, err := s.store.PopEvents(r.Context(), deviceID, s.cfg.LongPollTimeout)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read events")
		return
	}
	s.writeJSON(w, http.StatusOK, EventsResponse{Events: events})
}

func (s *Server) handleSessionAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, BasePath+"/sessions/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		s.writeError(w, http.StatusNotFound, "session route not found")
		return
	}
	sessionID, action := parts[0], parts[1]
	switch action {
	case "answer":
		s.handleSessionAnswer(w, r, sessionID)
	case "candidates":
		s.handleCandidates(w, r, sessionID)
	case "close":
		s.handleSessionClose(w, r, sessionID)
	case "failure":
		s.handleFailure(w, r, sessionID)
	default:
		s.writeError(w, http.StatusNotFound, "session route not found")
	}
}

func (s *Server) handleSessionAnswer(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req AnswerSessionRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		s.writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, req.DeviceID, req.DeviceID+":answer:"+sessionID); err != nil {
		s.handleAuthError(w, err)
		return
	}
	session, err := s.requireSessionParticipant(r.Context(), sessionID, req.DeviceID)
	if err != nil {
		s.handleSessionError(w, err)
		return
	}
	if req.DeviceID != session.TargetDeviceID {
		s.writeError(w, http.StatusForbidden, "only target device can answer")
		return
	}
	now := time.Now().UTC()
	eventType := "session_accepted"
	session.Status = "accepted"
	if !req.Accepted {
		eventType = "session_rejected"
		session.Status = "rejected"
	}
	if err := s.store.UpdateSession(r.Context(), session); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}
	event := SignalEvent{
		EventID:    mustToken(),
		Type:       eventType,
		SessionID:  sessionID,
		ToDevice:   session.ControllerDeviceID,
		FromDevice: req.DeviceID,
		Payload: map[string]interface{}{
			"accepted": req.Accepted,
			"reason":   req.Reason,
			"session":  session,
		},
		CreatedAt: now,
	}
	if err := s.store.AppendEvent(r.Context(), session.ControllerDeviceID, event); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to write event")
		return
	}
	s.writeJSON(w, http.StatusOK, SessionResponse{Session: session})
}

func (s *Server) handleCandidates(w http.ResponseWriter, r *http.Request, sessionID string) {
	switch r.Method {
	case http.MethodPost:
		s.handleAddCandidates(w, r, sessionID)
	case http.MethodGet:
		s.handleListCandidates(w, r, sessionID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAddCandidates(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req CandidateRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		s.writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, req.DeviceID, req.DeviceID+":candidates:"+sessionID); err != nil {
		s.handleAuthError(w, err)
		return
	}
	session, err := s.requireSessionParticipant(r.Context(), sessionID, req.DeviceID)
	if err != nil {
		s.handleSessionError(w, err)
		return
	}
	candidates := req.Candidates
	if req.Candidate.Type != "" {
		candidates = append([]Candidate{req.Candidate}, candidates...)
	}
	if len(candidates) == 0 {
		s.writeError(w, http.StatusBadRequest, "candidate is required")
		return
	}
	if len(candidates) > 16 {
		s.writeError(w, http.StatusBadRequest, "too many candidates")
		return
	}
	envelopes := make([]CandidateEnvelope, 0, len(candidates))
	for _, candidate := range candidates {
		if !validCandidateType(candidate.Type) || strings.TrimSpace(candidate.IP) == "" || candidate.Port <= 0 || candidate.Port > 65535 {
			s.writeError(w, http.StatusBadRequest, "invalid candidate")
			return
		}
		envelope, err := s.store.AddCandidate(r.Context(), sessionID, req.DeviceID, candidate)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "failed to save candidate")
			return
		}
		envelopes = append(envelopes, envelope)
	}
	otherDevice := session.TargetDeviceID
	if req.DeviceID == session.TargetDeviceID {
		otherDevice = session.ControllerDeviceID
	}
	event := SignalEvent{
		EventID:    mustToken(),
		Type:       "candidate_update",
		SessionID:  sessionID,
		ToDevice:   otherDevice,
		FromDevice: req.DeviceID,
		Payload:    envelopes,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.store.AppendEvent(r.Context(), otherDevice, event); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to write candidate event")
		return
	}
	s.writeJSON(w, http.StatusCreated, CandidatesResponse{Candidates: envelopes})
}

func (s *Server) handleListCandidates(w http.ResponseWriter, r *http.Request, sessionID string) {
	deviceID := strings.TrimSpace(r.URL.Query().Get("deviceId"))
	if deviceID == "" {
		s.writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, deviceID, deviceID+":candidates:"+sessionID); err != nil {
		s.handleAuthError(w, err)
		return
	}
	if _, err := s.requireSessionParticipant(r.Context(), sessionID, deviceID); err != nil {
		s.handleSessionError(w, err)
		return
	}
	candidates, err := s.store.ListCandidates(r.Context(), sessionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list candidates")
		return
	}
	s.writeJSON(w, http.StatusOK, CandidatesResponse{Candidates: candidates})
}

func (s *Server) handleSessionClose(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req CloseSessionRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" {
		s.writeError(w, http.StatusBadRequest, "deviceId is required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, req.DeviceID, req.DeviceID+":close:"+sessionID); err != nil {
		s.handleAuthError(w, err)
		return
	}
	session, err := s.requireSessionParticipant(r.Context(), sessionID, req.DeviceID)
	if err != nil {
		s.handleSessionError(w, err)
		return
	}
	session.Status = "closed"
	if err := s.store.UpdateSession(r.Context(), session); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to close session")
		return
	}
	otherDevice := session.TargetDeviceID
	if req.DeviceID == session.TargetDeviceID {
		otherDevice = session.ControllerDeviceID
	}
	event := SignalEvent{
		EventID:    mustToken(),
		Type:       "session_closed",
		SessionID:  sessionID,
		ToDevice:   otherDevice,
		FromDevice: req.DeviceID,
		Payload:    map[string]string{"reason": req.Reason},
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.store.AppendEvent(r.Context(), otherDevice, event); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to write close event")
		return
	}
	s.writeJSON(w, http.StatusOK, SessionResponse{Session: session})
}

func (s *Server) handleFailure(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req FailureRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.FailureReason = strings.TrimSpace(req.FailureReason)
	if req.DeviceID == "" || !validFailureReason(req.FailureReason) {
		s.writeError(w, http.StatusBadRequest, "valid deviceId and failureReason are required")
		return
	}
	if err := s.authenticateDevice(r.Context(), r, req.DeviceID, req.DeviceID+":failure:"+sessionID); err != nil {
		s.handleAuthError(w, err)
		return
	}
	session, err := s.requireSessionParticipant(r.Context(), sessionID, req.DeviceID)
	if err != nil {
		s.handleSessionError(w, err)
		return
	}
	failure := FailureInfo{
		SessionID:     sessionID,
		DeviceID:      req.DeviceID,
		FailureReason: req.FailureReason,
		Detail:        req.Detail,
		CreatedAt:     time.Now().UTC(),
	}
	session.Status = "failed"
	if err := s.store.UpdateSession(r.Context(), session); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}
	if err := s.store.SaveFailure(r.Context(), failure); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to save failure")
		return
	}
	otherDevice := session.TargetDeviceID
	if req.DeviceID == session.TargetDeviceID {
		otherDevice = session.ControllerDeviceID
	}
	event := SignalEvent{
		EventID:    mustToken(),
		Type:       "session_failure",
		SessionID:  sessionID,
		ToDevice:   otherDevice,
		FromDevice: req.DeviceID,
		Payload:    failure,
		CreatedAt:  failure.CreatedAt,
	}
	_ = s.store.AppendEvent(r.Context(), otherDevice, event)
	s.writeJSON(w, http.StatusOK, failure)
}

func (s *Server) authenticateDevice(ctx context.Context, r *http.Request, deviceID, message string) error {
	secret := deviceSecretFromRequest(r, "")
	signature := signatureFromRequest(r, "")
	if deviceID == "" || secret == "" || signature == "" {
		return errUnauthorized
	}
	if err := s.store.VerifyDeviceSecret(ctx, deviceID, secret); err != nil {
		return err
	}
	if !verifyHMAC(secret, message, signature) {
		return errUnauthorized
	}
	return nil
}

func (s *Server) requireSessionParticipant(ctx context.Context, sessionID, deviceID string) (SessionInfo, error) {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return SessionInfo{}, err
	}
	if deviceID != session.ControllerDeviceID && deviceID != session.TargetDeviceID {
		return SessionInfo{}, errForbidden
	}
	return session, nil
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, out interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.RequestBodyLimitBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, errorResponse{Error: message})
}

func (s *Server) handleAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotFound) || errors.Is(err, errUnauthorized) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized device")
		return
	}
	s.writeError(w, http.StatusInternalServerError, "authentication failed")
}

func (s *Server) handleSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errNotFound):
		s.writeError(w, http.StatusNotFound, "session not found")
	case errors.Is(err, errForbidden):
		s.writeError(w, http.StatusForbidden, "device is not a session participant")
	default:
		s.writeError(w, http.StatusInternalServerError, "failed to load session")
	}
}

func validSessionType(value string) bool {
	switch value {
	case "control", "view", "voice", "file":
		return true
	default:
		return false
	}
}

func validAcceptMode(value string) bool {
	switch value {
	case "manual", "auto", "verify_code":
		return true
	default:
		return false
	}
}

func validCandidateType(value string) bool {
	switch value {
	case "local_tcp", "local_udp", "reflexive_tcp", "reflexive_udp":
		return true
	default:
		return false
	}
}

func validFailureReason(value string) bool {
	switch value {
	case "offline", "timeout", "rejected", "verify_failed", "tcp_failed", "udp_probe_failed", "udp_punch_failed", "nat_failed", "firewall_blocked", "unknown":
		return true
	default:
		return false
	}
}
