package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type schedulerSnapshotRecordingCache struct {
	buckets []SchedulerBucket
}

func (c *schedulerSnapshotRecordingCache) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (c *schedulerSnapshotRecordingCache) SetSnapshot(_ context.Context, bucket SchedulerBucket, _ []Account) error {
	c.buckets = append(c.buckets, bucket)
	return nil
}

func (c *schedulerSnapshotRecordingCache) GetAccount(context.Context, int64) (*Account, error) {
	return nil, nil
}

func (c *schedulerSnapshotRecordingCache) SetAccount(context.Context, *Account) error {
	return nil
}

func (c *schedulerSnapshotRecordingCache) DeleteAccount(context.Context, int64) error {
	return nil
}

func (c *schedulerSnapshotRecordingCache) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (c *schedulerSnapshotRecordingCache) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (bool, error) {
	return true, nil
}

func (c *schedulerSnapshotRecordingCache) UnlockBucket(context.Context, SchedulerBucket) error {
	return nil
}

func (c *schedulerSnapshotRecordingCache) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *schedulerSnapshotRecordingCache) GetOutboxWatermark(context.Context) (int64, error) {
	return 0, nil
}

func (c *schedulerSnapshotRecordingCache) SetOutboxWatermark(context.Context, int64) error {
	return nil
}

type schedulerSnapshotAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r schedulerSnapshotAccountRepo) ListActive(context.Context) ([]Account, error) {
	return r.accounts, nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatform(r.accounts, platform), nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, _ int64, platforms []string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatforms(r.accounts, platforms), nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatform(r.accounts, platform), nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatforms(r.accounts, platforms), nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatform(r.accounts, platform), nil
}

func (r schedulerSnapshotAccountRepo) ListSchedulableUngroupedByPlatforms(_ context.Context, platforms []string) ([]Account, error) {
	return filterSchedulerSnapshotAccountsByPlatforms(r.accounts, platforms), nil
}

func filterSchedulerSnapshotAccountsByPlatform(accounts []Account, platform string) []Account {
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out
}

func filterSchedulerSnapshotAccountsByPlatforms(accounts []Account, platforms []string) []Account {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if _, ok := allowed[account.Platform]; ok {
			out = append(out, account)
		}
	}
	return out
}

type schedulerSnapshotGroupRepo struct {
	GroupRepository
	groups map[int64]Group
	active []Group
}

func (r schedulerSnapshotGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	if group, ok := r.groups[id]; ok {
		cp := group
		return &cp, nil
	}
	return nil, ErrGroupNotFound
}

func (r schedulerSnapshotGroupRepo) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return r.GetByID(ctx, id)
}

func (r schedulerSnapshotGroupRepo) ListActive(context.Context) ([]Group, error) {
	return r.active, nil
}

func (r schedulerSnapshotGroupRepo) ListActiveByPlatform(_ context.Context, platform string) ([]Group, error) {
	out := make([]Group, 0, len(r.active))
	for _, group := range r.active {
		if group.Platform == platform {
			out = append(out, group)
		}
	}
	return out, nil
}

func (r schedulerSnapshotGroupRepo) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r schedulerSnapshotGroupRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func TestSchedulerSnapshotService_RebuildByGroupIDsIncludesCustomPlatforms(t *testing.T) {
	cache := &schedulerSnapshotRecordingCache{}
	groupID := int64(7)
	customPlatform := "acme-openai"
	svc := NewSchedulerSnapshotService(
		cache,
		nil,
		schedulerSnapshotAccountRepo{accounts: []Account{
			{
				ID:          100,
				Platform:    customPlatform,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				GroupIDs:    []int64{groupID},
			},
		}},
		schedulerSnapshotGroupRepo{
			groups: map[int64]Group{
				groupID: {ID: groupID, Platform: customPlatform, Status: StatusActive},
			},
			active: []Group{{ID: groupID, Platform: customPlatform, Status: StatusActive}},
		},
		nil,
	)

	err := svc.rebuildByGroupIDs(context.Background(), []int64{groupID}, "test", nil)
	require.NoError(t, err)

	require.Contains(t, cache.buckets, SchedulerBucket{GroupID: groupID, Platform: customPlatform, Mode: SchedulerModeSingle})
	require.Contains(t, cache.buckets, SchedulerBucket{GroupID: groupID, Platform: customPlatform, Mode: SchedulerModeForced})
}

func TestSchedulerSnapshotService_DefaultBucketsIncludeCustomAccountPlatform(t *testing.T) {
	customPlatform := "acme-openai"
	svc := NewSchedulerSnapshotService(
		nil,
		nil,
		schedulerSnapshotAccountRepo{accounts: []Account{{ID: 101, Platform: customPlatform, Status: StatusActive}}},
		nil,
		nil,
	)

	buckets, err := svc.defaultBuckets(context.Background())
	require.NoError(t, err)

	require.Contains(t, buckets, SchedulerBucket{GroupID: 0, Platform: customPlatform, Mode: SchedulerModeSingle})
	require.Contains(t, buckets, SchedulerBucket{GroupID: 0, Platform: customPlatform, Mode: SchedulerModeForced})
}
