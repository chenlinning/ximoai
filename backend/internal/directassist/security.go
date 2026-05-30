package directassist

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
)

var (
	errUnauthorized = errors.New("unauthorized")
	errForbidden    = errors.New("forbidden")
)

func generateToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hmacSHA256Hex(secret, message string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyHMAC(secret, message, signature string) bool {
	expected := hmacSHA256Hex(secret, message)
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(strings.TrimSpace(signature))))
}

func secretHash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func canonicalHeartbeatMessage(deviceID, deviceCode string) string {
	return strings.TrimSpace(deviceID) + ":" + strings.TrimSpace(deviceCode)
}

func signatureFromRequest(r *http.Request, fallback string) string {
	if value := r.Header.Get("X-DirectAssist-Signature"); value != "" {
		return value
	}
	return fallback
}

func deviceSecretFromRequest(r *http.Request, fallback string) string {
	if value := r.Header.Get("X-DirectAssist-Device-Secret"); value != "" {
		return value
	}
	return fallback
}

func remoteIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func normalizeStringList(values []string, maxItems int) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > 128 {
			value = value[:128]
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if len(out) >= maxItems {
			break
		}
	}
	return out
}
