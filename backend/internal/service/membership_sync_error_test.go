//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type membershipSyncErrorRepo struct {
	defaultMembershipRepoStub
	levels       []MembershipLevel
	memberships  []UserMembership
	repairBatch  []UserMembership
	repairListed bool
	syncErr      error
}

func (r *membershipSyncErrorRepo) ListMembershipLevelsByGroup(context.Context, int64) ([]MembershipLevel, error) {
	return r.levels, nil
}

func (r *membershipSyncErrorRepo) ListUserMembershipsByLevel(context.Context, int64) ([]UserMembership, error) {
	return r.memberships, nil
}

func (r *membershipSyncErrorRepo) ListActiveUserMembershipsAfterID(context.Context, int64, int) ([]UserMembership, error) {
	if r.repairListed {
		return nil, nil
	}
	r.repairListed = true
	return r.repairBatch, nil
}

func (r *membershipSyncErrorRepo) GetActiveUserMembership(_ context.Context, userID int64) (*UserMembership, error) {
	return &UserMembership{ID: userID, UserID: userID, MembershipLevelID: 10, Status: MembershipStatusActive}, nil
}

func (r *membershipSyncErrorRepo) GetMembershipLevel(context.Context, int64) (*MembershipLevel, error) {
	return nil, r.syncErr
}

func TestMembershipServiceSyncMembershipLevelReturnsUserErrors(t *testing.T) {
	repo := &membershipSyncErrorRepo{
		memberships: []UserMembership{{ID: 1, UserID: 7, MembershipLevelID: 10}},
		syncErr:     errors.New("membership level unavailable"),
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	err := svc.SyncMembershipLevel(context.Background(), 10)

	require.ErrorContains(t, err, "membership level unavailable")
}

func TestMembershipServiceSyncGroupRateReturnsLevelErrors(t *testing.T) {
	repo := &membershipSyncErrorRepo{
		levels:      []MembershipLevel{{ID: 10}},
		memberships: []UserMembership{{ID: 1, UserID: 7, MembershipLevelID: 10}},
		syncErr:     errors.New("membership level unavailable"),
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	err := svc.SyncGroupRate(context.Background(), 20)

	require.ErrorContains(t, err, "membership level unavailable")
}

func TestMembershipServiceRepairMembershipStateReturnsUserErrors(t *testing.T) {
	repo := &membershipSyncErrorRepo{
		repairBatch: []UserMembership{{ID: 1, UserID: 7, MembershipLevelID: 10}},
		syncErr:     errors.New("membership level unavailable"),
	}
	svc := NewMembershipService(repo, nil, nil, nil, nil)

	err := svc.RepairMembershipState(context.Background())

	require.ErrorContains(t, err, "membership level unavailable")
}
