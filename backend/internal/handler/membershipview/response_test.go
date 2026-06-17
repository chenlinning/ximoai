package membershipview

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSummaryFromServiceUsesFrontendFieldNamesAndEffectiveRates(t *testing.T) {
	summary := &service.MembershipSummary{
		Level: &service.MembershipLevel{
			ID:           10,
			Name:         "VIP",
			Code:         "vip",
			Color:        "#22c55e",
			DiscountRate: 0.8,
		},
		Groups: []service.Group{
			{
				ID:             20,
				Name:           "premium",
				RateMultiplier: 1.5,
			},
		},
		Levels: []service.MembershipLevel{
			{
				ID:           11,
				Name:         "VIP 2",
				Code:         "vip2",
				Color:        "#0ea5e9",
				DiscountRate: 0.7,
				Groups: []service.Group{
					{
						ID:             21,
						Name:           "exclusive",
						RateMultiplier: 2,
						IsExclusive:    true,
					},
				},
			},
		},
		ManagedKeys: []service.MembershipManagedKey{
			{
				ID:       30,
				APIKeyID: 40,
				APIKey: &service.APIKey{
					ID:     40,
					Key:    "sk-membership-secret-value",
					Name:   "Membership Key",
					Status: service.StatusAPIKeyActive,
				},
			},
		},
	}

	body, err := json.Marshal(SummaryFromService(summary))
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	level := payload["level"].(map[string]any)
	levels := payload["levels"].([]any)
	benefitLevel := levels[0].(map[string]any)
	benefitGroups := benefitLevel["groups"].([]any)
	benefitGroup := benefitGroups[0].(map[string]any)
	groups := payload["groups"].([]any)
	group := groups[0].(map[string]any)
	managedKeys := payload["managed_keys"].([]any)
	managedKey := managedKeys[0].(map[string]any)
	apiKey := managedKey["api_key"].(map[string]any)

	require.Equal(t, "#22c55e", level["color"])
	require.Equal(t, "VIP 2", benefitLevel["name"])
	require.Equal(t, true, benefitGroup["is_exclusive"])
	require.InEpsilon(t, 1.4, benefitGroup["effective_rate_multiplier"], 0.000001)
	require.Equal(t, float64(20), group["id"])
	require.Equal(t, "premium", group["name"])
	require.Equal(t, 1.5, group["rate_multiplier"])
	require.InEpsilon(t, 1.2, group["effective_rate_multiplier"], 0.000001)
	require.NotContains(t, group, "ID")
	require.NotContains(t, group, "Name")
	require.NotContains(t, group, "RateMultiplier")
	require.NotContains(t, apiKey, "key")
	require.Equal(t, "value", apiKey["key_suffix"])
	require.Equal(t, "sk-...value", apiKey["masked_key"])
	require.NotContains(t, string(body), "sk-membership-secret-value")
}
