package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolcengineAgentPlanResolveUpstreamModelUsesExplicitAccountMapping(t *testing.T) {
	account := &Account{
		Platform: PlatformVolcengineAgentPlan,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"doubao-seed-tts-2.0": "seed-tts-2.0",
			},
		},
	}

	require.Equal(t, "seed-tts-2.0", ResolveVolcengineAgentPlanUpstreamModel(account, "doubao-seed-tts-2.0"))
	require.Equal(t, "seed-tts-2.0", ResolveVolcengineAgentPlanUpstreamModel(account, "seed-tts-2.0"))
}

func TestVolcengineAgentPlanAccountSupportsPublicOrChannelMappedModel(t *testing.T) {
	account := &Account{
		Platform: PlatformVolcengineAgentPlan,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"seed-tts-2.0": "seed-tts-2.0",
			},
		},
	}

	require.True(t, VolcengineAgentPlanAccountSupportsModel(account, "doubao-seed-tts-2.0", "seed-tts-2.0"))
	require.False(t, VolcengineAgentPlanAccountSupportsModel(account, "doubao-unknown-model", "doubao-unknown-model"))
}

func TestVolcengineAgentPlanModelRouteContextRoundTrips(t *testing.T) {
	route := VolcengineAgentPlanModelRoute{
		PublicModel:        "doubao-seed-asr-2.0",
		ChannelMappedModel: "volc.seedasr.sauc.duration",
	}

	ctx := WithVolcengineAgentPlanModelRoute(context.Background(), route)
	got, ok := VolcengineAgentPlanModelRouteFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, route, got)
}
