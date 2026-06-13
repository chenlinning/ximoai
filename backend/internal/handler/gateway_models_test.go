package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewayModelsAccountRepoStub struct {
	service.AccountRepository

	byGroup map[int64][]service.Account
}

type gatewayModelsChannelRepoStub struct {
	service.ChannelRepository

	channels       []service.Channel
	groupPlatforms map[int64]string
}

type gatewayModelsResponseForTest struct {
	Object string                    `json:"object"`
	Data   []gatewayModelItemForTest `json:"data"`
}

type gatewayModelItemForTest struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int64  `json:"created"`
	OwnedBy   string `json:"owned_by"`
	CreatedAt string `json:"created_at"`
}

type gatewayModelsUserGroupRateRepoStub struct {
	service.UserGroupRateRepository

	rate *float64
}

func (s *gatewayModelsAccountRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]service.Account, error) {
	accounts, ok := s.byGroup[groupID]
	if !ok {
		return nil, nil
	}
	out := make([]service.Account, len(accounts))
	copy(out, accounts)
	return out, nil
}

func newGatewayModelsHandlerForTest(repo service.AccountRepository) *GatewayHandler {
	return newGatewayModelsHandlerWithChannelService(repo, nil)
}

func newGatewayModelsHandlerWithPricing(channels []service.Channel, groupPlatforms map[int64]string) *GatewayHandler {
	channelSvc := service.NewChannelService(
		&gatewayModelsChannelRepoStub{
			channels:       channels,
			groupPlatforms: groupPlatforms,
		},
		nil,
		nil,
		nil,
	)
	return newGatewayModelsHandlerWithChannelService(&gatewayModelsAccountRepoStub{}, channelSvc)
}

func newGatewayModelsHandlerWithChannelService(repo service.AccountRepository, channelSvc *service.ChannelService) *GatewayHandler {
	return newGatewayModelsHandlerWithDeps(repo, channelSvc, nil)
}

func newGatewayModelsHandlerWithDeps(repo service.AccountRepository, channelSvc *service.ChannelService, rateRepo service.UserGroupRateRepository) *GatewayHandler {
	return &GatewayHandler{
		gatewayService: service.NewGatewayService(
			repo,
			nil, nil, nil, nil, nil, rateRepo, nil, nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, channelSvc, nil, nil, nil,
		),
	}
}

func (s *gatewayModelsChannelRepoStub) ListAll(ctx context.Context) ([]service.Channel, error) {
	out := make([]service.Channel, len(s.channels))
	copy(out, s.channels)
	return out, nil
}

func (s *gatewayModelsChannelRepoStub) GetGroupPlatforms(ctx context.Context, groupIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(groupIDs))
	for _, groupID := range groupIDs {
		if platform, ok := s.groupPlatforms[groupID]; ok {
			out[groupID] = platform
		}
	}
	return out, nil
}

func (s *gatewayModelsUserGroupRateRepoStub) GetByUserAndGroup(ctx context.Context, userID, groupID int64) (*float64, error) {
	if s == nil {
		return nil, nil
	}
	return s.rate, nil
}

func testPrice(v float64) *float64 {
	return &v
}

func testPricedChannel(groupID int64, platform string, models ...string) service.Channel {
	pricing := make([]service.ChannelModelPricing, 0, len(models))
	for _, model := range models {
		pricing = append(pricing, service.ChannelModelPricing{
			Platform:    platform,
			Models:      []string{model},
			InputPrice:  testPrice(0.01),
			OutputPrice: testPrice(0.02),
		})
	}
	return service.Channel{
		ID:           groupID,
		Status:       service.StatusActive,
		GroupIDs:     []int64{groupID},
		ModelPricing: pricing,
	}
}

func TestGatewayModels_GeminiGroupUsesPricedChannelModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(20)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{
			{
				ID:       1,
				Status:   service.StatusActive,
				GroupIDs: []int64{groupID},
				ModelPricing: []service.ChannelModelPricing{
					{
						Platform:    service.PlatformGemini,
						Models:      []string{"gemini-priced"},
						InputPrice:  testPrice(0.01),
						OutputPrice: testPrice(0.02),
					},
					{
						Platform:    service.PlatformAnthropic,
						Models:      []string{"claude-priced"},
						InputPrice:  testPrice(0.01),
						OutputPrice: testPrice(0.02),
					},
				},
			},
		},
		map[int64]string{groupID: service.PlatformGemini},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformGemini},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "list", got.Object)
	require.Equal(t, []string{"gemini-priced"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_IgnoresAccountMappingsAndFiltersPricedModelsByPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(21)
	channelSvc := service.NewChannelService(
		&gatewayModelsChannelRepoStub{
			channels: []service.Channel{
				{
					ID:       1,
					Status:   service.StatusActive,
					GroupIDs: []int64{groupID},
					ModelPricing: []service.ChannelModelPricing{
						{
							Platform:    service.PlatformAnthropic,
							Models:      []string{"claude-priced"},
							InputPrice:  testPrice(0.01),
							OutputPrice: testPrice(0.02),
						},
						{
							Platform:    service.PlatformGemini,
							Models:      []string{"gemini-priced"},
							InputPrice:  testPrice(0.01),
							OutputPrice: testPrice(0.02),
						},
					},
				},
			},
			groupPlatforms: map[int64]string{groupID: service.PlatformGemini},
		},
		nil,
		nil,
		nil,
	)
	h := newGatewayModelsHandlerWithChannelService(
		&gatewayModelsAccountRepoStub{
			byGroup: map[int64][]service.Account{
				groupID: {
					{
						ID:       1,
						Platform: service.PlatformAnthropic,
						Credentials: map[string]any{
							"model_mapping": map[string]any{
								"claude-sonnet-4-6": "claude-sonnet-4-6",
							},
						},
					},
					{
						ID:       2,
						Platform: service.PlatformGemini,
						Credentials: map[string]any{
							"model_mapping": map[string]any{
								"gemini-2.5-flash": "gemini-2.5-flash",
							},
						},
					},
				},
			},
		},
		channelSvc,
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformGemini},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"gemini-priced"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListDisabledKeepsPricedModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(22)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{testPricedChannel(groupID, service.PlatformOpenAI, "gpt-5.4", "gpt-5.5")},
		map[int64]string{groupID: service.PlatformOpenAI},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: false,
				Models:  []string{"gpt-5.5"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"gpt-5.4", "gpt-5.5"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListFiltersAndOrdersPricedModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(23)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{testPricedChannel(groupID, service.PlatformOpenAI, "gpt-5.4", "gpt-5.5", "legacy-gpt-2024")},
		map[int64]string{groupID: service.PlatformOpenAI},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gpt-5.5", "missing-model", "gpt-5.4"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"gpt-5.5", "gpt-5.4"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListKeepsConcreteModelAllowedByChannelWildcardMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(26)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{
			{
				ID:       1,
				Status:   service.StatusActive,
				GroupIDs: []int64{groupID},
				ModelPricing: []service.ChannelModelPricing{
					{
						Platform:    service.PlatformAnthropic,
						Models:      []string{"claude-sonnet-4-6"},
						InputPrice:  testPrice(0.01),
						OutputPrice: testPrice(0.02),
					},
				},
				ModelMapping: map[string]map[string]string{
					service.PlatformAnthropic: {"claude-*": "claude-sonnet-4-6"},
				},
			},
		},
		map[int64]string{groupID: service.PlatformAnthropic},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformAnthropic,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"claude-sonnet-4-6"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"claude-sonnet-4-6"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListCanReturnEmptyWhenSelectionsUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(24)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{testPricedChannel(groupID, service.PlatformOpenAI, "gpt-5.4")},
		map[int64]string{groupID: service.PlatformOpenAI},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gpt-5.5"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListEnabledWithEmptyListReturnsEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(28)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{testPricedChannel(groupID, service.PlatformOpenAI, "gpt-5.4")},
		map[int64]string{groupID: service.PlatformOpenAI},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  nil,
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, modelIDsForTest(got.Data))
}

func TestGatewayModels_CustomModelsListDoesNotFallbackToDefaultModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(25)
	h := newGatewayModelsHandlerWithPricing(nil, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gpt-5.5", "legacy-gpt-2024", "gpt-5.4"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, modelIDsForTest(got.Data))
}

func TestGatewayModels_OpenAIPricedModelsKeepOpenAIResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(27)
	h := newGatewayModelsHandlerWithPricing(
		[]service.Channel{testPricedChannel(groupID, service.PlatformOpenAI, "gpt-5.4", "gpt-5.5")},
		map[int64]string{groupID: service.PlatformOpenAI},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformOpenAI,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"gpt-5.5", "gpt-5.4"},
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"gpt-5.5", "gpt-5.4"}, modelIDsForTest(got.Data))
	require.Equal(t, "model", got.Data[0].Object)
	require.NotZero(t, got.Data[0].Created)
	require.Equal(t, "openai", got.Data[0].OwnedBy)
	require.Empty(t, got.Data[0].CreatedAt)
}

func TestGatewayModels_IncludeEntryProtocolsReturnsCompactXimoAIMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(30)
	userRate := 1.8
	channelSvc := service.NewChannelService(
		&gatewayModelsChannelRepoStub{
			channels: []service.Channel{
				{
					ID:       1,
					Status:   service.StatusActive,
					GroupIDs: []int64{groupID},
					ModelPricing: []service.ChannelModelPricing{
						{
							Platform:        service.PlatformGemini,
							Models:          []string{"NanoBanana2"},
							BillingMode:     service.BillingModeImage,
							PerRequestPrice: testPrice(0.4),
						},
						{
							Platform:    service.PlatformGemini,
							Models:      []string{"gemini-3.5-flash"},
							BillingMode: service.BillingModeToken,
							InputPrice:  testPrice(0.01),
							OutputPrice: testPrice(0.02),
						},
					},
				},
			},
			groupPlatforms: map[int64]string{groupID: service.PlatformGemini},
		},
		nil,
		nil,
		nil,
	)
	h := newGatewayModelsHandlerWithDeps(
		&gatewayModelsAccountRepoStub{},
		channelSvc,
		&gatewayModelsUserGroupRateRepoStub{rate: &userRate},
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models?include_entry_protocols=1", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		UserID: 1001,
		Group: &service.Group{
			ID:               groupID,
			Name:             "梦工厂 gemini",
			Platform:         service.PlatformGemini,
			SubscriptionType: service.SubscriptionTypeStandard,
			RateMultiplier:   1.5,
			IsExclusive:      true,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: true,
				Models:  []string{"NanoBanana2"},
			},
		},
		GroupID: &groupID,
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "list", got["object"])

	data, ok := got["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	item, ok := data[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{
		"id":     "NanoBanana2",
		"object": "model",
	}, map[string]any{
		"id":     item["id"],
		"object": item["object"],
	})
	require.NotContains(t, item, "created")
	require.NotContains(t, item, "owned_by")
	require.NotContains(t, item, "type")
	require.NotContains(t, item, "display_name")

	ximoai, ok := item["ximoai"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "openai", ximoai["default_entry_protocol"])
	require.Equal(t, "/v1/images/generations", ximoai["default_endpoint"])
	require.NotContains(t, ximoai, "platform")
	require.NotContains(t, ximoai, "supported_entry_protocols")

	group, ok := ximoai["group"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(groupID), group["id"])
	require.Equal(t, "梦工厂 gemini", group["name"])
	require.Equal(t, service.SubscriptionTypeStandard, group["subscription_type"])
	require.Equal(t, 1.5, group["rate_multiplier"])
	require.Equal(t, 1.8, group["effective_rate_multiplier"])
	require.Equal(t, true, group["is_exclusive"])
	require.NotContains(t, group, "platform")

	pricing, ok := ximoai["pricing"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, string(service.BillingModeImage), pricing["billing_mode"])
	require.Equal(t, 0.4, pricing["per_request_price"])
	require.Nil(t, pricing["input_price"])
	require.Nil(t, pricing["output_price"])
}

func TestDefaultEntryProtocolForPricedModel_DetectsImageAndSpeechEndpoints(t *testing.T) {
	imagePrice := testPrice(0.00003)
	tests := []struct {
		name         string
		detail       service.GatewayPricedModelDetail
		wantProtocol string
		wantEndpoint string
	}{
		{
			name: "openai image model billed per request uses images endpoint",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-image-2",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode:      service.BillingModePerRequest,
					ImageOutputPrice: imagePrice,
					PerRequestPrice:  testPrice(0.1),
				},
			},
			wantProtocol: "openai",
			wantEndpoint: "/v1/images/generations",
		},
		{
			name: "openai tts model uses speech endpoint",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-audio-tts-hd",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.0000003),
					OutputPrice: testPrice(0.000018),
				},
			},
			wantProtocol: "openai",
			wantEndpoint: "/v1/audio/speech",
		},
		{
			name: "openai audio preview is not treated as tts",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-4o-audio-preview",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.0000025),
					OutputPrice: testPrice(0.00000318),
				},
			},
			wantProtocol: "openai",
			wantEndpoint: "/v1/responses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProtocol, gotEndpoint := defaultEntryProtocolForPricedModel(tt.detail)
			require.Equal(t, tt.wantProtocol, gotProtocol)
			require.Equal(t, tt.wantEndpoint, gotEndpoint)
		})
	}
}

func TestGatewayModels_NoChannelPricingReturnsEmptyEvenWhenAccountHasMappings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(29)
	h := newGatewayModelsHandlerWithChannelService(
		&gatewayModelsAccountRepoStub{
			byGroup: map[int64][]service.Account{
				groupID: {
					{
						ID:       1,
						Platform: service.PlatformOpenAI,
						Credentials: map[string]any{
							"model_mapping": map[string]any{
								"account-only": "upstream-only",
							},
						},
					},
				},
			},
		},
		nil,
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Empty(t, modelIDsForTest(got.Data))
}

func modelIDsForTest(models []gatewayModelItemForTest) []string {
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	return ids
}
