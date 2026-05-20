package handler

import (
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSplitOpenAIVideoSubpath(t *testing.T) {
	require.Equal(t, []string{"video_123", "content"}, splitOpenAIVideoSubpath("/video_123/content"))
	require.Equal(t, []string{"characters", "char_123"}, splitOpenAIVideoSubpath("characters/char_123"))
	require.Nil(t, splitOpenAIVideoSubpath("/"))
}

func TestCanAccessOpenAIVideoCharacter(t *testing.T) {
	groupID := int64(10)
	apiKeyID := int64(20)
	userID := int64(30)
	apiKey := &service.APIKey{ID: apiKeyID, GroupID: &groupID}
	subject := middleware2.AuthSubject{UserID: userID}

	character := &service.OpenAIVideoCharacter{
		CharacterID: "char_123",
		Platform:    "acme",
		GroupID:     &groupID,
		APIKeyID:    &apiKeyID,
		UserID:      &userID,
	}
	require.True(t, canAccessOpenAIVideoCharacter(character, apiKey, subject, "acme"))
	require.False(t, canAccessOpenAIVideoCharacter(character, apiKey, subject, "openai"))

	otherUser := middleware2.AuthSubject{UserID: 31}
	require.False(t, canAccessOpenAIVideoCharacter(character, apiKey, otherUser, "acme"))
}
