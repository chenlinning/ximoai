package directassist

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSignalSmokeFlow(t *testing.T) {
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	cfg := DefaultConfig()
	cfg.DeviceTTL = 2 * time.Second
	cfg.DeviceSecretTTL = time.Minute
	cfg.UDPProbeTokenTTL = time.Minute
	cfg.SessionTTL = time.Minute
	cfg.EventTTL = time.Minute
	cfg.CandidateTTL = time.Minute
	cfg.FailureTTL = time.Minute
	cfg.LongPollTimeout = time.Second
	store := NewStore(redisClient, cfg)
	server := NewServer(cfg, store, slog.Default())
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	aSecret := "device-a-secret"
	bSecret := "device-b-secret"
	aHB := heartbeat(t, ts.URL, HeartbeatRequest{
		DeviceCode:      "A-CODE-001",
		DeviceID:        "device-a",
		DeviceSecret:    aSecret,
		DeviceSignature: sign(aSecret, canonicalHeartbeatMessage("device-a", "A-CODE-001")),
		AppVersion:      "1.0.0",
		Capabilities:    []string{"tcp", "udp"},
		LocalIPs:        []string{"192.168.1.10"},
		TCPPort:         47001,
		UDPPort:         47002,
		Status:          "online",
	})
	bHB := heartbeat(t, ts.URL, HeartbeatRequest{
		DeviceCode:      "B-CODE-001",
		DeviceID:        "device-b",
		DeviceSecret:    bSecret,
		DeviceSignature: sign(bSecret, canonicalHeartbeatMessage("device-b", "B-CODE-001")),
		AppVersion:      "1.0.0",
		Capabilities:    []string{"tcp", "udp"},
		LocalIPs:        []string{"192.168.1.11"},
		TCPPort:         47011,
		UDPPort:         47012,
		Status:          "online",
	})
	if aHB.UDPProbeToken == "" || bHB.UDPProbeToken == "" {
		t.Fatalf("expected udp probe tokens")
	}

	var query DeviceQueryResponse
	requestJSON(t, http.MethodGet, ts.URL+BasePath+"/devices/A-CODE-001", nil, "", "", "", http.StatusOK, &query)
	if query.Device.DeviceID != "device-a" || !query.Device.Online {
		t.Fatalf("unexpected device query: %+v", query.Device)
	}

	var sessionResp SessionResponse
	createBody := CreateSessionRequest{
		ControllerDeviceID: "device-b",
		DeviceCode:         "A-CODE-001",
		SessionType:        "control",
		AcceptMode:         "manual",
	}
	requestJSON(t, http.MethodPost, ts.URL+BasePath+"/sessions", createBody, bSecret, sign(bSecret, "device-b:create_session"), "", http.StatusCreated, &sessionResp)
	sessionID := sessionResp.Session.SessionID
	if sessionID == "" {
		t.Fatalf("expected session id")
	}

	var aEvents EventsResponse
	requestJSON(t, http.MethodGet, ts.URL+BasePath+"/events?deviceId=device-a", nil, aSecret, sign(aSecret, "device-a:events"), "", http.StatusOK, &aEvents)
	if len(aEvents.Events) != 1 || aEvents.Events[0].Type != "session_request" {
		t.Fatalf("expected session_request event, got %+v", aEvents.Events)
	}

	var accepted SessionResponse
	answerBody := AnswerSessionRequest{DeviceID: "device-a", Accepted: true}
	requestJSON(t, http.MethodPost, ts.URL+BasePath+"/sessions/"+sessionID+"/answer", answerBody, aSecret, sign(aSecret, "device-a:answer:"+sessionID), "", http.StatusOK, &accepted)
	if accepted.Session.Status != "accepted" {
		t.Fatalf("expected accepted session, got %+v", accepted.Session)
	}

	var bEvents EventsResponse
	requestJSON(t, http.MethodGet, ts.URL+BasePath+"/events?deviceId=device-b", nil, bSecret, sign(bSecret, "device-b:events"), "", http.StatusOK, &bEvents)
	if len(bEvents.Events) != 1 || bEvents.Events[0].Type != "session_accepted" {
		t.Fatalf("expected session_accepted event, got %+v", bEvents.Events)
	}

	addCandidate(t, ts.URL, sessionID, "device-a", aSecret, Candidate{Type: "local_udp", IP: "192.168.1.10", Port: 47002})
	addCandidate(t, ts.URL, sessionID, "device-b", bSecret, Candidate{Type: "reflexive_udp", IP: "203.0.113.10", Port: 50000})

	var candidates CandidatesResponse
	requestJSON(t, http.MethodGet, ts.URL+BasePath+"/sessions/"+sessionID+"/candidates?deviceId=device-a", nil, aSecret, sign(aSecret, "device-a:candidates:"+sessionID), "", http.StatusOK, &candidates)
	if len(candidates.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %+v", candidates.Candidates)
	}

	udpAddr := runUDPProbeServer(t, cfg, store)
	probeA := udpProbe(t, udpAddr, UDPProbeRequest{
		Type:      "udp_probe",
		DeviceID:  "device-a",
		SessionID: sessionID,
		Nonce:     "nonce-a",
		Token:     aHB.UDPProbeToken,
		Timestamp: time.Now().Unix(),
	})
	if probeA.Type != "udp_probe_result" || probeA.PublicIP == "" || probeA.PublicPort == 0 || probeA.Nonce != "nonce-a" {
		t.Fatalf("unexpected udp probe A response: %+v", probeA)
	}
	probeB := udpProbe(t, udpAddr, UDPProbeRequest{
		Type:      "udp_probe",
		DeviceID:  "device-b",
		SessionID: sessionID,
		Nonce:     "nonce-b",
		Token:     bHB.UDPProbeToken,
		Timestamp: time.Now().Unix(),
	})
	if probeB.Type != "udp_probe_result" || probeB.PublicIP == "" || probeB.PublicPort == 0 || probeB.Nonce != "nonce-b" {
		t.Fatalf("unexpected udp probe B response: %+v", probeB)
	}

	var failure FailureInfo
	failureBody := FailureRequest{DeviceID: "device-a", FailureReason: "udp_punch_failed", Detail: "smoke test"}
	requestJSON(t, http.MethodPost, ts.URL+BasePath+"/sessions/"+sessionID+"/failure", failureBody, aSecret, sign(aSecret, "device-a:failure:"+sessionID), "", http.StatusOK, &failure)
	if failure.FailureReason != "udp_punch_failed" {
		t.Fatalf("unexpected failure: %+v", failure)
	}

	mr.FastForward(3 * time.Second)
	var offline errorResponse
	requestJSON(t, http.MethodGet, ts.URL+BasePath+"/devices/A-CODE-001", nil, "", "", "", http.StatusNotFound, &offline)
	if offline.Error == "" {
		t.Fatalf("expected offline error")
	}
}

func heartbeat(t *testing.T, baseURL string, body HeartbeatRequest) HeartbeatResponse {
	t.Helper()
	var resp HeartbeatResponse
	requestJSON(t, http.MethodPost, baseURL+BasePath+"/heartbeat", body, "", "", "", http.StatusOK, &resp)
	return resp
}

func addCandidate(t *testing.T, baseURL, sessionID, deviceID, secret string, candidate Candidate) {
	t.Helper()
	var resp CandidatesResponse
	body := CandidateRequest{DeviceID: deviceID, Candidate: candidate}
	requestJSON(t, http.MethodPost, baseURL+BasePath+"/sessions/"+sessionID+"/candidates", body, secret, sign(secret, deviceID+":candidates:"+sessionID), "", http.StatusCreated, &resp)
	if len(resp.Candidates) != 1 {
		t.Fatalf("expected one candidate response, got %+v", resp.Candidates)
	}
}

func requestJSON(t *testing.T, method, url string, body interface{}, secret, signature, deviceID string, expectedStatus int, out interface{}) {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if secret != "" {
		req.Header.Set("X-DirectAssist-Device-Secret", secret)
	}
	if signature != "" {
		req.Header.Set("X-DirectAssist-Signature", signature)
	}
	if deviceID != "" {
		req.Header.Set("X-DirectAssist-Device-Id", deviceID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("%s %s status=%d want=%d", method, url, resp.StatusCode, expectedStatus)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
}

func runUDPProbeServer(t *testing.T, cfg Config, store *Store) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = conn.Close()
	})
	go func() {
		_ = ServeUDPProbe(ctx, conn, cfg, store, slog.Default())
	}()
	return conn.LocalAddr().String()
}

func udpProbe(t *testing.T, addr string, body UDPProbeRequest) UDPProbeResponse {
	t.Helper()
	conn, err := net.Dial("udp", addr)
	if err != nil {
		t.Fatalf("dial udp: %v", err)
	}
	defer conn.Close()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal udp probe: %v", err)
	}
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write udp probe: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set udp deadline: %v", err)
	}
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read udp probe response: %v", err)
	}
	var resp UDPProbeResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("decode udp probe response: %v", err)
	}
	return resp
}

func sign(secret, message string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
