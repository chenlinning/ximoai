package directassist

import (
	"os"
	"strconv"
	"time"
)

const BasePath = "/api/direct-assist/signal/v1"

type Config struct {
	SignalAddr                    string
	UDPProbeAddr                  string
	DeviceTTL                     time.Duration
	DeviceSecretTTL               time.Duration
	UDPProbeTokenTTL              time.Duration
	SessionTTL                    time.Duration
	EventTTL                      time.Duration
	CandidateTTL                  time.Duration
	FailureTTL                    time.Duration
	LongPollTimeout               time.Duration
	RequestBodyLimitBytes         int64
	UDPPacketLimitBytes           int
	RateLimitPerMinute            int64
	HeartbeatRateLimitPerMinute   int64
	LongPollingRateLimitPerMinute int64
}

func DefaultConfig() Config {
	return Config{
		SignalAddr:                    "0.0.0.0:47880",
		UDPProbeAddr:                  "0.0.0.0:47822",
		DeviceTTL:                     60 * time.Second,
		DeviceSecretTTL:               30 * time.Minute,
		UDPProbeTokenTTL:              60 * time.Second,
		SessionTTL:                    10 * time.Minute,
		EventTTL:                      10 * time.Minute,
		CandidateTTL:                  10 * time.Minute,
		FailureTTL:                    10 * time.Minute,
		LongPollTimeout:               25 * time.Second,
		RequestBodyLimitBytes:         64 * 1024,
		UDPPacketLimitBytes:           1024,
		RateLimitPerMinute:            120,
		HeartbeatRateLimitPerMinute:   240,
		LongPollingRateLimitPerMinute: 180,
	}
}

func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()
	cfg.SignalAddr = envString("DIRECT_ASSIST_SIGNAL_ADDR", cfg.SignalAddr)
	cfg.UDPProbeAddr = envString("DIRECT_ASSIST_UDP_PROBE_ADDR", cfg.UDPProbeAddr)
	cfg.DeviceTTL = envDurationSeconds("DIRECT_ASSIST_DEVICE_TTL_SECONDS", cfg.DeviceTTL)
	cfg.DeviceSecretTTL = envDurationSeconds("DIRECT_ASSIST_DEVICE_SECRET_TTL_SECONDS", cfg.DeviceSecretTTL)
	cfg.UDPProbeTokenTTL = envDurationSeconds("DIRECT_ASSIST_UDP_PROBE_TOKEN_TTL_SECONDS", cfg.UDPProbeTokenTTL)
	cfg.SessionTTL = envDurationSeconds("DIRECT_ASSIST_SESSION_TTL_SECONDS", cfg.SessionTTL)
	cfg.EventTTL = envDurationSeconds("DIRECT_ASSIST_EVENT_TTL_SECONDS", cfg.EventTTL)
	cfg.CandidateTTL = envDurationSeconds("DIRECT_ASSIST_CANDIDATE_TTL_SECONDS", cfg.CandidateTTL)
	cfg.FailureTTL = envDurationSeconds("DIRECT_ASSIST_FAILURE_TTL_SECONDS", cfg.FailureTTL)
	cfg.LongPollTimeout = envDurationSeconds("DIRECT_ASSIST_LONG_POLL_TIMEOUT_SECONDS", cfg.LongPollTimeout)
	cfg.RequestBodyLimitBytes = envInt64("DIRECT_ASSIST_REQUEST_BODY_LIMIT_BYTES", cfg.RequestBodyLimitBytes)
	cfg.UDPPacketLimitBytes = int(envInt64("DIRECT_ASSIST_UDP_PACKET_LIMIT_BYTES", int64(cfg.UDPPacketLimitBytes)))
	cfg.RateLimitPerMinute = envInt64("DIRECT_ASSIST_RATE_LIMIT_PER_MINUTE", cfg.RateLimitPerMinute)
	cfg.HeartbeatRateLimitPerMinute = envInt64("DIRECT_ASSIST_HEARTBEAT_RATE_LIMIT_PER_MINUTE", cfg.HeartbeatRateLimitPerMinute)
	cfg.LongPollingRateLimitPerMinute = envInt64("DIRECT_ASSIST_LONG_POLL_RATE_LIMIT_PER_MINUTE", cfg.LongPollingRateLimitPerMinute)
	return cfg
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := envInt64(key, int64(fallback/time.Second))
	if value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Second
}

func envInt64(key string, fallback int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
