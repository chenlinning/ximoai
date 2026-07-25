package videocollector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type engineStub struct{}

func (engineStub) Parse(context.Context, string) (*MediaInfo, error) {
	return &MediaInfo{ID: "media-1", Formats: []MediaFormat{{ID: "best", HasVideo: true}}}, nil
}

func (engineStub) Download(_ context.Context, _ DownloadRequest, outputDir string, progress func(ProgressUpdate)) (*DownloadResult, error) {
	progress(ProgressUpdate{State: TaskStateDownloading, Percent: 50})
	path := filepath.Join(outputDir, "output.mp4")
	if err := os.WriteFile(path, []byte("video-data"), 0o600); err != nil {
		return nil, err
	}
	return &DownloadResult{Path: path, Extension: "mp4"}, nil
}

func waitForCompletedTask(t *testing.T, manager *Manager, userID int64, taskID string) TaskSnapshot {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := manager.Get(userID, taskID)
		require.NoError(t, err)
		if task.State == TaskStateCompleted {
			return task
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("task did not complete")
	return TaskSnapshot{}
}

func TestManagerBindsTasksToUserAndExpiresTenMinutesAfterDownload(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	manager, err := NewManager(ManagerConfig{
		Root:               t.TempDir(),
		DownloadRetention:  10 * time.Minute,
		UnclaimedRetention: 30 * time.Minute,
		MaxConcurrent:      1,
		Now:                func() time.Time { return now },
	}, engineStub{})
	require.NoError(t, err)

	task, err := manager.Start(101, DownloadRequest{
		SourceURL: "https://example.com/video",
		MediaID:   "media-1",
		Title:     "Test video",
		FormatID:  "best",
		HasAudio:  true,
	})
	require.NoError(t, err)
	completed := waitForCompletedTask(t, manager, 101, task.ID)
	require.NotEmpty(t, completed.FileName)

	_, err = manager.Get(202, task.ID)
	require.ErrorIs(t, err, ErrTaskNotFound)
	_, err = manager.OpenDownload(202, task.ID)
	require.ErrorIs(t, err, ErrTaskNotFound)

	lease, err := manager.OpenDownload(101, task.ID)
	require.NoError(t, err)
	require.Equal(t, now.Add(10*time.Minute), lease.DeleteAt)
	require.NoError(t, lease.Close())
	firstDeleteAt := lease.DeleteAt

	now = now.Add(5 * time.Minute)
	secondLease, err := manager.OpenDownload(101, task.ID)
	require.NoError(t, err)
	require.Equal(t, firstDeleteAt, secondLease.DeleteAt)
	require.NoError(t, secondLease.Close())

	now = firstDeleteAt.Add(-time.Second)
	manager.CleanupExpired()
	_, err = manager.Get(101, task.ID)
	require.NoError(t, err)

	now = firstDeleteAt
	manager.CleanupExpired()
	_, err = manager.Get(101, task.ID)
	require.ErrorIs(t, err, ErrTaskNotFound)
	require.NoDirExists(t, filepath.Join(manager.root, task.ID))
}

func TestManagerRemovesUnclaimedFiles(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	manager, err := NewManager(ManagerConfig{
		Root:               t.TempDir(),
		DownloadRetention:  10 * time.Minute,
		UnclaimedRetention: 30 * time.Minute,
		MaxConcurrent:      1,
		Now:                func() time.Time { return now },
	}, engineStub{})
	require.NoError(t, err)

	task, err := manager.Start(101, DownloadRequest{SourceURL: "https://example.com/video", FormatID: "best"})
	require.NoError(t, err)
	waitForCompletedTask(t, manager, 101, task.ID)

	now = now.Add(30 * time.Minute)
	manager.CleanupExpired()
	_, err = manager.Get(101, task.ID)
	require.ErrorIs(t, err, ErrTaskNotFound)
}
