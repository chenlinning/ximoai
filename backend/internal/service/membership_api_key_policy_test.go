package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type managedAPIKeyPolicyStub struct {
	managed *MembershipManagedKey
	err     error
}

func (s managedAPIKeyPolicyStub) GetManagedKeyByAPIKeyID(context.Context, int64) (*MembershipManagedKey, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.managed, nil
}

func TestAPIKeyServiceManagedKeyPolicyBlocksUserDelete(t *testing.T) {
	svc := &APIKeyService{}
	svc.SetManagedAPIKeyPolicy(managedAPIKeyPolicyStub{
		managed: &MembershipManagedKey{APIKeyID: 10, Status: ManagedKeyStatusActive},
	})

	err := svc.ensureManagedAPIKeyUserCanDelete(context.Background(), 10)

	require.ErrorIs(t, err, ErrMembershipManagedKeyDeletion)
}

func TestAPIKeyServiceManagedKeyPolicyBlocksUserEnableWhenDisabled(t *testing.T) {
	svc := &APIKeyService{}
	svc.SetManagedAPIKeyPolicy(managedAPIKeyPolicyStub{
		managed: &MembershipManagedKey{APIKeyID: 10, Status: ManagedKeyStatusDisabled},
	})

	err := svc.ensureManagedAPIKeyUserCanEnable(context.Background(), 10)

	require.ErrorIs(t, err, ErrMembershipManagedKeyEnable)
}

func TestAPIKeyServiceManagedKeyPolicyAllowsNormalKey(t *testing.T) {
	svc := &APIKeyService{}
	svc.SetManagedAPIKeyPolicy(managedAPIKeyPolicyStub{err: ErrAPIKeyNotFound})

	require.NoError(t, svc.ensureManagedAPIKeyUserCanDelete(context.Background(), 10))
	require.NoError(t, svc.ensureManagedAPIKeyUserCanEnable(context.Background(), 10))
}

func TestAPIKeyServiceManagedKeyPolicyPropagatesLookupError(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	svc := &APIKeyService{}
	svc.SetManagedAPIKeyPolicy(managedAPIKeyPolicyStub{err: lookupErr})

	require.ErrorIs(t, svc.ensureManagedAPIKeyUserCanDelete(context.Background(), 10), lookupErr)
	require.ErrorIs(t, svc.ensureManagedAPIKeyUserCanEnable(context.Background(), 10), lookupErr)
}
