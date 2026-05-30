package directassist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisPrefix = "directassist:"

var errNotFound = errors.New("not found")

type Store struct {
	redis *redis.Client
	cfg   Config
	now   func() time.Time
}

func NewStore(redisClient *redis.Client, cfg Config) *Store {
	return &Store{
		redis: redisClient,
		cfg:   cfg,
		now:   time.Now,
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.redis.Ping(ctx).Err()
}

func (s *Store) SaveHeartbeat(ctx context.Context, req HeartbeatRequest, publicIP string, udpToken string) (DeviceInfo, error) {
	now := s.now().UTC()
	device := DeviceInfo{
		DeviceCode:      req.DeviceCode,
		DeviceID:        req.DeviceID,
		AppVersion:      req.AppVersion,
		Capabilities:    normalizeStringList(req.Capabilities, 32),
		LocalIPs:        normalizeStringList(req.LocalIPs, 32),
		TCPPort:         req.TCPPort,
		UDPPort:         req.UDPPort,
		Status:          req.Status,
		Online:          true,
		LastHeartbeatAt: now,
		PublicTCPIP:     publicIP,
	}

	payload, err := json.Marshal(device)
	if err != nil {
		return DeviceInfo{}, err
	}

	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, keyDeviceCode(req.DeviceCode), req.DeviceID, s.cfg.DeviceTTL)
	pipe.Set(ctx, keyDevice(req.DeviceID), payload, s.cfg.DeviceTTL)
	pipe.Set(ctx, keyUDPToken(req.DeviceID), udpToken, s.cfg.UDPProbeTokenTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return DeviceInfo{}, err
	}
	return device, nil
}

func (s *Store) SaveDeviceSecret(ctx context.Context, deviceID, secret string) error {
	return s.redis.Set(ctx, keyDeviceSecret(deviceID), secretHash(secret), s.cfg.DeviceSecretTTL).Err()
}

func (s *Store) VerifyDeviceSecret(ctx context.Context, deviceID, secret string) error {
	stored, err := s.redis.Get(ctx, keyDeviceSecret(deviceID)).Result()
	if errors.Is(err, redis.Nil) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	if stored != secretHash(secret) {
		return errUnauthorized
	}
	return nil
}

func (s *Store) GetDeviceByCode(ctx context.Context, deviceCode string) (DeviceInfo, error) {
	deviceID, err := s.redis.Get(ctx, keyDeviceCode(deviceCode)).Result()
	if errors.Is(err, redis.Nil) {
		return DeviceInfo{}, errNotFound
	}
	if err != nil {
		return DeviceInfo{}, err
	}
	return s.GetDevice(ctx, deviceID)
}

func (s *Store) GetDevice(ctx context.Context, deviceID string) (DeviceInfo, error) {
	raw, err := s.redis.Get(ctx, keyDevice(deviceID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return DeviceInfo{}, errNotFound
	}
	if err != nil {
		return DeviceInfo{}, err
	}
	var device DeviceInfo
	if err := json.Unmarshal(raw, &device); err != nil {
		return DeviceInfo{}, err
	}
	return device, nil
}

func (s *Store) CreateSession(ctx context.Context, req CreateSessionRequest) (SessionInfo, error) {
	now := s.now().UTC()
	sessionID, err := generateToken(16)
	if err != nil {
		return SessionInfo{}, err
	}
	session := SessionInfo{
		SessionID:          sessionID,
		ControllerDeviceID: req.ControllerDeviceID,
		TargetDeviceID:     req.TargetDeviceID,
		SessionType:        req.SessionType,
		AcceptMode:         req.AcceptMode,
		Status:             "pending",
		CreatedAt:          now,
		UpdatedAt:          now,
		ExpiresAt:          now.Add(s.cfg.SessionTTL),
	}
	if err := s.saveSession(ctx, session); err != nil {
		return SessionInfo{}, err
	}
	event := SignalEvent{
		EventID:    mustToken(),
		Type:       "session_request",
		SessionID:  session.SessionID,
		ToDevice:   session.TargetDeviceID,
		FromDevice: session.ControllerDeviceID,
		Payload:    session,
		CreatedAt:  now,
	}
	if err := s.AppendEvent(ctx, session.TargetDeviceID, event); err != nil {
		return SessionInfo{}, err
	}
	return session, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (SessionInfo, error) {
	raw, err := s.redis.Get(ctx, keySession(sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return SessionInfo{}, errNotFound
	}
	if err != nil {
		return SessionInfo{}, err
	}
	var session SessionInfo
	if err := json.Unmarshal(raw, &session); err != nil {
		return SessionInfo{}, err
	}
	return session, nil
}

func (s *Store) UpdateSession(ctx context.Context, session SessionInfo) error {
	session.UpdatedAt = s.now().UTC()
	return s.saveSession(ctx, session)
}

func (s *Store) saveSession(ctx context.Context, session SessionInfo) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, keySession(session.SessionID), payload, s.cfg.SessionTTL).Err()
}

func (s *Store) AppendEvent(ctx context.Context, deviceID string, event SignalEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	key := keyEvents(deviceID)
	pipe := s.redis.TxPipeline()
	pipe.RPush(ctx, key, payload)
	pipe.Expire(ctx, key, s.cfg.EventTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) PopEvents(ctx context.Context, deviceID string, timeout time.Duration) ([]SignalEvent, error) {
	key := keyEvents(deviceID)
	values, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 && timeout > 0 {
		result, err := s.redis.BLPop(ctx, timeout, key).Result()
		if errors.Is(err, redis.Nil) {
			return []SignalEvent{}, nil
		}
		if err != nil {
			return nil, err
		}
		if len(result) == 2 {
			values = append(values, result[1])
			more, err := s.redis.LRange(ctx, key, 0, -1).Result()
			if err != nil {
				return nil, err
			}
			values = append(values, more...)
			if len(more) > 0 {
				_ = s.redis.Del(ctx, key).Err()
			}
		}
	} else if len(values) > 0 {
		_ = s.redis.Del(ctx, key).Err()
	}

	events := make([]SignalEvent, 0, len(values))
	for _, value := range values {
		var event SignalEvent
		if err := json.Unmarshal([]byte(value), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (s *Store) AddCandidate(ctx context.Context, sessionID, deviceID string, candidate Candidate) (CandidateEnvelope, error) {
	now := s.now().UTC()
	candidate.CreatedAt = now
	envelope := CandidateEnvelope{
		SessionID: sessionID,
		DeviceID:  deviceID,
		Candidate: candidate,
		CreatedAt: now,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return CandidateEnvelope{}, err
	}
	key := keyCandidates(sessionID)
	pipe := s.redis.TxPipeline()
	pipe.RPush(ctx, key, payload)
	pipe.Expire(ctx, key, s.cfg.CandidateTTL)
	_, err = pipe.Exec(ctx)
	return envelope, err
}

func (s *Store) ListCandidates(ctx context.Context, sessionID string) ([]CandidateEnvelope, error) {
	values, err := s.redis.LRange(ctx, keyCandidates(sessionID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	candidates := make([]CandidateEnvelope, 0, len(values))
	for _, value := range values {
		var envelope CandidateEnvelope
		if err := json.Unmarshal([]byte(value), &envelope); err != nil {
			continue
		}
		candidates = append(candidates, envelope)
	}
	return candidates, nil
}

func (s *Store) SaveFailure(ctx context.Context, failure FailureInfo) error {
	payload, err := json.Marshal(failure)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, keyFailure(failure.SessionID), payload, s.cfg.FailureTTL).Err()
}

func (s *Store) GetUDPToken(ctx context.Context, deviceID string) (string, error) {
	token, err := s.redis.Get(ctx, keyUDPToken(deviceID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", errUnauthorized
	}
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) RateLimit(ctx context.Context, scope, identity string, limit int64) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	now := s.now().UTC()
	key := fmt.Sprintf("%sratelimit:%s:%s:%s", redisPrefix, scope, identity, now.Format("200601021504"))
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = s.redis.Expire(ctx, key, 2*time.Minute).Err()
	}
	return count <= limit, nil
}

func mustToken() string {
	token, err := generateToken(16)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return token
}

func keyDeviceCode(deviceCode string) string {
	return redisPrefix + "device_code:" + deviceCode
}

func keyDevice(deviceID string) string {
	return redisPrefix + "device:" + deviceID
}

func keyDeviceSecret(deviceID string) string {
	return redisPrefix + "device_secret:" + deviceID
}

func keyUDPToken(deviceID string) string {
	return redisPrefix + "udp_token:" + deviceID
}

func keySession(sessionID string) string {
	return redisPrefix + "session:" + sessionID
}

func keyEvents(deviceID string) string {
	return redisPrefix + "events:" + deviceID
}

func keyCandidates(sessionID string) string {
	return redisPrefix + "candidates:" + sessionID
}

func keyFailure(sessionID string) string {
	return redisPrefix + "failure:" + sessionID
}
