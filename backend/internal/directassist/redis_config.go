package directassist

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func RedisOptionsFromEnv() *redis.Options {
	host := envString("REDIS_HOST", "localhost")
	port := int(envInt64("REDIS_PORT", 6379))
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     envString("REDIS_PASSWORD", ""),
		DB:           int(envInt64("REDIS_DB", 0)),
		DialTimeout:  time.Duration(envInt64("REDIS_DIAL_TIMEOUT_SECONDS", 5)) * time.Second,
		ReadTimeout:  time.Duration(envInt64("REDIS_READ_TIMEOUT_SECONDS", 3)) * time.Second,
		WriteTimeout: time.Duration(envInt64("REDIS_WRITE_TIMEOUT_SECONDS", 3)) * time.Second,
		PoolSize:     int(envInt64("REDIS_POOL_SIZE", 128)),
		MinIdleConns: int(envInt64("REDIS_MIN_IDLE_CONNS", 10)),
	}
	if envBool("REDIS_ENABLE_TLS", false) {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host,
		}
	}
	return opts
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(envString(key, ""))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
