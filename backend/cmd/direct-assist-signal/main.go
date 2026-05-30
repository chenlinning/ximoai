package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/directassist"
	"github.com/redis/go-redis/v9"
)

var (
	Commit = "unknown"
	Date   = "unknown"
)

func main() {
	redisClient := redis.NewClient(directassist.RedisOptionsFromEnv())
	defer redisClient.Close()

	signalCfg := directassist.LoadConfigFromEnv()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	store := directassist.NewStore(redisClient, signalCfg)
	server := directassist.NewServer(signalCfg, store, logger)
	httpServer := server.HTTPServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		logger.Info("direct assist signal http server started", "addr", signalCfg.SignalAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("direct assist udp probe server started", "addr", signalCfg.UDPProbeAddr)
		errCh <- directassist.ListenAndServeUDPProbe(ctx, signalCfg, store, logger)
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			logger.Error("direct assist signal service failed", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("direct assist http shutdown failed", "error", err)
	}
	logger.Info("direct assist signal service stopped")
}
