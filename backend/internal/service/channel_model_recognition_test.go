package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedModelsKeepsDisplayNameAndUpstreamRecognitionNameSeparate(t *testing.T) {
	channel := XimoAIAvailableChannel{
		ModelMapping: map[string]map[string]string{
			"openai": {"ximo-gpt": "gpt-5"},
		},
	}

	require.Equal(t, "gpt-5", channel.RecognitionName("openai", "ximo-gpt"))
	require.Equal(t, "gpt-5", channel.RecognitionName("openai", "gpt-5"))
}

func TestSupportedModelsUsesDisplayNameWhenNoMappingExists(t *testing.T) {
	channel := XimoAIAvailableChannel{}
	require.Equal(t, "gpt-5", channel.RecognitionName("openai", "gpt-5"))
}
