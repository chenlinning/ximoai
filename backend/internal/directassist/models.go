package directassist

import "time"

type HeartbeatRequest struct {
	DeviceCode      string   `json:"deviceCode"`
	DeviceID        string   `json:"deviceId"`
	DeviceSecret    string   `json:"deviceSecret,omitempty"`
	DeviceSignature string   `json:"deviceSignature"`
	AppVersion      string   `json:"appVersion"`
	Capabilities    []string `json:"capabilities"`
	LocalIPs        []string `json:"localIps"`
	TCPPort         int      `json:"tcpPort"`
	UDPPort         int      `json:"udpPort"`
	Status          string   `json:"status"`
}

type HeartbeatResponse struct {
	DeviceID         string    `json:"deviceId"`
	Online           bool      `json:"online"`
	LastHeartbeatAt  time.Time `json:"lastHeartbeatAt"`
	PublicTCPIP      string    `json:"publicTcpIp"`
	UDPProbeToken    string    `json:"udpProbeToken"`
	UDPProbeTokenTTL int64     `json:"udpProbeTokenTtlSeconds"`
	DeviceTTLSeconds int64     `json:"deviceTtlSeconds"`
}

type DeviceInfo struct {
	DeviceCode      string    `json:"deviceCode,omitempty"`
	DeviceID        string    `json:"deviceId"`
	AppVersion      string    `json:"appVersion,omitempty"`
	Capabilities    []string  `json:"capabilities,omitempty"`
	LocalIPs        []string  `json:"localIps,omitempty"`
	TCPPort         int       `json:"tcpPort,omitempty"`
	UDPPort         int       `json:"udpPort,omitempty"`
	Status          string    `json:"status"`
	Online          bool      `json:"online"`
	LastHeartbeatAt time.Time `json:"lastHeartbeatAt"`
	PublicTCPIP     string    `json:"publicTcpIp,omitempty"`
}

type DeviceQueryResponse struct {
	Device DeviceInfo `json:"device"`
}

type CreateSessionRequest struct {
	ControllerDeviceID string `json:"controllerDeviceId"`
	TargetDeviceID     string `json:"targetDeviceId"`
	DeviceCode         string `json:"deviceCode"`
	SessionType        string `json:"sessionType"`
	AcceptMode         string `json:"acceptMode"`
	VerifyCode         string `json:"verifyCode,omitempty"`
	ClientNonce        string `json:"clientNonce,omitempty"`
}

type SessionInfo struct {
	SessionID          string    `json:"sessionId"`
	ControllerDeviceID string    `json:"controllerDeviceId"`
	TargetDeviceID     string    `json:"targetDeviceId"`
	SessionType        string    `json:"sessionType"`
	AcceptMode         string    `json:"acceptMode"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

type SessionResponse struct {
	Session SessionInfo `json:"session"`
}

type AnswerSessionRequest struct {
	DeviceID    string `json:"deviceId"`
	Accepted    bool   `json:"accepted"`
	Reason      string `json:"reason,omitempty"`
	VerifyCode  string `json:"verifyCode,omitempty"`
	ClientNonce string `json:"clientNonce,omitempty"`
}

type CandidateRequest struct {
	DeviceID   string      `json:"deviceId"`
	Candidate  Candidate   `json:"candidate"`
	Candidates []Candidate `json:"candidates,omitempty"`
}

type Candidate struct {
	Type      string    `json:"type"`
	Protocol  string    `json:"protocol,omitempty"`
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	Priority  int       `json:"priority,omitempty"`
	Nonce     string    `json:"nonce,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

type CandidatesResponse struct {
	Candidates []CandidateEnvelope `json:"candidates"`
}

type CandidateEnvelope struct {
	SessionID string    `json:"sessionId"`
	DeviceID  string    `json:"deviceId"`
	Candidate Candidate `json:"candidate"`
	CreatedAt time.Time `json:"createdAt"`
}

type CloseSessionRequest struct {
	DeviceID string `json:"deviceId"`
	Reason   string `json:"reason,omitempty"`
}

type FailureRequest struct {
	DeviceID      string `json:"deviceId"`
	FailureReason string `json:"failureReason"`
	Detail        string `json:"detail,omitempty"`
}

type FailureInfo struct {
	SessionID     string    `json:"sessionId"`
	DeviceID      string    `json:"deviceId"`
	FailureReason string    `json:"failureReason"`
	Detail        string    `json:"detail,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type EventsResponse struct {
	Events []SignalEvent `json:"events"`
}

type SignalEvent struct {
	EventID    string      `json:"eventId"`
	Type       string      `json:"type"`
	SessionID  string      `json:"sessionId,omitempty"`
	ToDevice   string      `json:"toDevice,omitempty"`
	FromDevice string      `json:"fromDevice,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

type UDPProbeRequest struct {
	Type      string `json:"type"`
	DeviceID  string `json:"deviceId"`
	SessionID string `json:"sessionId"`
	Nonce     string `json:"nonce"`
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
}

type UDPProbeResponse struct {
	Type       string `json:"type"`
	PublicIP   string `json:"publicIp"`
	PublicPort int    `json:"publicPort"`
	Nonce      string `json:"nonce"`
	ServerTime int64  `json:"serverTime"`
}

type errorResponse struct {
	Error string `json:"error"`
}
