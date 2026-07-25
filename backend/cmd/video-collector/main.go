package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/ximoai/videocollector"
)

func main() {
	config := loadConfig()
	if err := videocollector.EnsureRuntimeFiles(config.ytDLPPath, config.ffmpegPath); err != nil {
		log.Fatalf("video collector runtime unavailable: %v", err)
	}
	engine := videocollector.NewYTDLPEngine(config.ytDLPPath, config.ffmpegPath, nil)
	manager, err := videocollector.NewManager(videocollector.ManagerConfig{
		Root:               config.tempRoot,
		DownloadRetention:  10 * time.Minute,
		UnclaimedRetention: 30 * time.Minute,
		MaxConcurrent:      config.maxConcurrent,
	}, engine)
	if err != nil {
		log.Fatalf("initialize video collector: %v", err)
	}
	defer manager.Close()
	handler, err := videocollector.NewHTTPServer(manager, config.internalToken)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              config.listenAddress,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			manager.CleanupExpired()
		}
	}()

	go func() {
		log.Printf("video collector listening on %s", config.listenAddress)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("video collector server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

type appConfig struct {
	listenAddress string
	internalToken string
	tempRoot      string
	ytDLPPath     string
	ffmpegPath    string
	maxConcurrent int
}

func loadConfig() appConfig {
	maxConcurrent, _ := strconv.Atoi(envOrDefault("VIDEO_COLLECTOR_MAX_CONCURRENT", "2"))
	if maxConcurrent < 1 || maxConcurrent > 8 {
		maxConcurrent = 2
	}
	return appConfig{
		listenAddress: envOrDefault("VIDEO_COLLECTOR_LISTEN", "0.0.0.0:8090"),
		internalToken: os.Getenv("VIDEO_COLLECTOR_INTERNAL_TOKEN"),
		tempRoot:      envOrDefault("VIDEO_COLLECTOR_TEMP_ROOT", "/tmp/video-collector"),
		ytDLPPath:     envOrDefault("YTDLP_PATH", "/opt/yt-dlp/bin/yt-dlp"),
		ffmpegPath:    envOrDefault("FFMPEG_PATH", "/usr/bin/ffmpeg"),
		maxConcurrent: maxConcurrent,
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
