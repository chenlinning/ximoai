//go:build unit

package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateBillableUsageCost(t *testing.T) {
	tests := []struct {
		name       string
		cost       *CostBreakdown
		imageCount int
		videoCount int
		wantErr    bool
	}{
		{
			name: "token zero usage is allowed",
			cost: &CostBreakdown{BillingMode: string(BillingModeToken)},
		},
		{
			name:       "image zero cost is rejected",
			cost:       &CostBreakdown{BillingMode: string(BillingModeImage)},
			imageCount: 1,
			wantErr:    true,
		},
		{
			name:       "video nil cost is rejected",
			videoCount: 1,
			wantErr:    true,
		},
		{
			name:       "per request positive cost is allowed",
			cost:       &CostBreakdown{BillingMode: string(BillingModePerRequest), ActualCost: 0.25},
			imageCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBillableUsageCost(tt.cost, tt.imageCount, tt.videoCount)
			if tt.wantErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrBillablePricingRequired))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestChannelPricingHasPositiveConfiguredPrice(t *testing.T) {
	require.False(t, channelPricingHasPositiveConfiguredPrice(nil))
	require.False(t, channelPricingHasPositiveConfiguredPrice(&ChannelModelPricing{InputPrice: testPtrFloat64(0)}))
	require.True(t, channelPricingHasPositiveConfiguredPrice(&ChannelModelPricing{InputPrice: testPtrFloat64(0.01)}))
	require.True(t, channelPricingHasPositiveConfiguredPrice(&ChannelModelPricing{
		Intervals: []PricingInterval{{PerRequestPrice: testPtrFloat64(0.01)}},
	}))
}
