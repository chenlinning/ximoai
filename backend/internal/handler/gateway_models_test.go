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

type gatewayModelsPlatformRepoStub struct {
	service.PlatformRepository

	platforms map[string]service.Platform
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
		platformService: service.NewPlatformService(nil),
	}
}

func newGatewayModelsHandlerWithPlatformService(repo service.AccountRepository, channelSvc *service.ChannelService, rateRepo service.UserGroupRateRepository, platformSvc *service.PlatformService) *GatewayHandler {
	h := newGatewayModelsHandlerWithDeps(repo, channelSvc, rateRepo)
	h.platformService = platformSvc
	return h
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

func (s *gatewayModelsPlatformRepoStub) GetBySlug(ctx context.Context, slug string) (*service.Platform, error) {
	platform, ok := s.platforms[service.NormalizePlatformSlug(slug)]
	if !ok {
		return nil, service.ErrPlatformNotFound
	}
	return &platform, nil
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
	require.Equal(t, "gemini", ximoai["default_entry_protocol"])
	require.Equal(t, "/v1beta/models/NanoBanana2:generateContent", ximoai["default_endpoint"])
	require.Equal(t, "image", ximoai["model_type"])
	require.Equal(t, "image_generation", ximoai["operation_type"])
	require.Equal(t, "sync", ximoai["execution_mode"])
	require.Equal(t, false, ximoai["supports_stream"])
	require.Equal(t, false, ximoai["supports_polling"])
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
	require.InDelta(t, 0.72, pricing["per_request_price"], 1e-12)
	require.Nil(t, pricing["input_price"])
	require.Nil(t, pricing["output_price"])
}

func TestGatewayModels_IncludeEntryProtocolsUsesCustomPlatformProtocol(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(31)
	customPlatform := "custom-gemini"
	channelSvc := service.NewChannelService(
		&gatewayModelsChannelRepoStub{
			channels: []service.Channel{
				{
					ID:       1,
					Status:   service.StatusActive,
					GroupIDs: []int64{groupID},
					ModelPricing: []service.ChannelModelPricing{
						{
							Platform:    customPlatform,
							Models:      []string{"custom-gemini-chat"},
							BillingMode: service.BillingModeToken,
							InputPrice:  testPrice(0.01),
							OutputPrice: testPrice(0.02),
						},
					},
				},
			},
			groupPlatforms: map[int64]string{groupID: customPlatform},
		},
		nil,
		nil,
		nil,
	)
	platformSvc := service.NewPlatformService(&gatewayModelsPlatformRepoStub{
		platforms: map[string]service.Platform{
			customPlatform: {
				Slug:         customPlatform,
				DisplayName:  "Custom Gemini",
				Protocol:     service.PlatformProtocolGemini,
				Capabilities: []string{service.PlatformCapabilityMessages, service.PlatformCapabilityNativeGemini},
				Enabled:      true,
			},
		},
	})
	h := newGatewayModelsHandlerWithPlatformService(&gatewayModelsAccountRepoStub{}, channelSvc, nil, platformSvc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models?include_entry_protocols=1", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		UserID: 1002,
		Group: &service.Group{
			ID:       groupID,
			Name:     "custom gemini",
			Platform: customPlatform,
		},
		GroupID: &groupID,
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	data := got["data"].([]any)
	require.Len(t, data, 1)
	item := data[0].(map[string]any)
	ximoai := item["ximoai"].(map[string]any)
	require.Equal(t, "gemini", ximoai["default_entry_protocol"])
	require.Equal(t, "/v1beta/models/custom-gemini-chat:generateContent", ximoai["default_endpoint"])
	require.Equal(t, "chat", ximoai["model_type"])
	require.Equal(t, "chat", ximoai["operation_type"])
	require.Equal(t, "sync", ximoai["execution_mode"])
	require.Equal(t, true, ximoai["supports_stream"])
	require.Equal(t, false, ximoai["supports_polling"])
}

func TestGatewayModels_IncludeEntryProtocolsReturnsAudioContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(32)
	channelSvc := service.NewChannelService(
		&gatewayModelsChannelRepoStub{
			channels: []service.Channel{
				{
					ID:       1,
					Status:   service.StatusActive,
					GroupIDs: []int64{groupID},
					ModelPricing: []service.ChannelModelPricing{
						{
							Platform:        service.PlatformKlingAudio,
							Models:          []string{"kling-audio", "kling-custom-voices"},
							BillingMode:     service.BillingModePerRequest,
							PerRequestPrice: testPrice(0.25),
						},
					},
				},
			},
			groupPlatforms: map[int64]string{groupID: service.PlatformKlingAudio},
		},
		nil,
		nil,
		nil,
	)
	h := newGatewayModelsHandlerWithDeps(&gatewayModelsAccountRepoStub{}, channelSvc, nil)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models?include_entry_protocols=1", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		UserID:  1003,
		Group:   &service.Group{ID: groupID, Name: "可灵audio", Platform: service.PlatformKlingAudio},
		GroupID: &groupID,
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	data := got["data"].([]any)
	require.Len(t, data, 2)

	byID := map[string]map[string]any{}
	for _, raw := range data {
		item := raw.(map[string]any)
		byID[item["id"].(string)] = item["ximoai"].(map[string]any)
	}

	tts := byID["kling-audio"]
	require.Equal(t, "audio_tts", tts["operation_type"])
	ttsRequest := tts["request_contract"].(map[string]any)
	require.Contains(t, ttsRequest["required_fields"], "voice_id")
	require.NotContains(t, ttsRequest["required_fields"], "voice")
	ttsNotes := ttsRequest["field_notes"].(map[string]any)
	require.Contains(t, ttsNotes["voice_id"], "Kling voice id")
	ttsResponse := tts["response_contract"].(map[string]any)
	require.Equal(t, "json_url", ttsResponse["delivery"])
	require.Equal(t, "data.task_result.audios[0].url", ttsResponse["audio_url_path"])

	custom := byID["kling-custom-voices"]
	require.Equal(t, "voice_management", custom["operation_type"])
	customRequest := custom["request_contract"].(map[string]any)
	require.Contains(t, customRequest["create_required_fields"], "voice_name")
	require.Contains(t, customRequest["create_required_fields"], "voice_url")
	require.Contains(t, customRequest["query_required_fields"], "voice_id")
	customResponse := custom["response_contract"].(map[string]any)
	require.Equal(t, "json", customResponse["delivery"])
	require.Equal(t, "data.task_result.voices[0].voice_id", customResponse["voice_id_path"])
}

func TestPublicEntryMetadataForPricedModel_DetectsPublicCapabilities(t *testing.T) {
	imagePrice := testPrice(0.00003)
	tests := []struct {
		name                string
		detail              service.GatewayPricedModelDetail
		platform            *service.Platform
		wantProtocol        string
		wantEndpoint        string
		wantModelType       string
		wantOperationType   string
		wantExecutionMode   string
		wantSupportsStream  bool
		wantSupportsPolling bool
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
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/images/generations",
			wantModelType:       "image",
			wantOperationType:   "image_generation",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
		{
			name: "gemini image model stays on gemini entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "NanoBanana2",
				Platform: service.PlatformGemini,
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModeImage,
					PerRequestPrice: testPrice(0.4),
				},
			},
			wantProtocol:        "gemini",
			wantEndpoint:        "/v1beta/models/NanoBanana2:generateContent",
			wantModelType:       "image",
			wantOperationType:   "image_generation",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
		{
			name: "gemini chat model stays on gemini generateContent entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "gemini-3.5-flash",
				Platform: service.PlatformGemini,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.0000015),
					OutputPrice: testPrice(0.000009),
				},
			},
			wantProtocol:        "gemini",
			wantEndpoint:        "/v1beta/models/gemini-3.5-flash:generateContent",
			wantModelType:       "chat",
			wantOperationType:   "chat",
			wantExecutionMode:   "sync",
			wantSupportsStream:  true,
			wantSupportsPolling: false,
		},
		{
			name: "custom gemini protocol platform uses gemini public entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "custom-gemini-chat",
				Platform: "custom-gemini",
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.000001),
					OutputPrice: testPrice(0.000002),
				},
			},
			platform: &service.Platform{
				Slug:     "custom-gemini",
				Protocol: service.PlatformProtocolGemini,
			},
			wantProtocol:        "gemini",
			wantEndpoint:        "/v1beta/models/custom-gemini-chat:generateContent",
			wantModelType:       "chat",
			wantOperationType:   "chat",
			wantExecutionMode:   "sync",
			wantSupportsStream:  true,
			wantSupportsPolling: false,
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
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/audio/speech",
			wantModelType:       "audio",
			wantOperationType:   "audio_tts",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
		{
			name: "openai audio preview uses chat completions audio entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-4o-audio-preview",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.0000025),
					OutputPrice: testPrice(0.00000318),
				},
			},
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/chat/completions",
			wantModelType:       "audio",
			wantOperationType:   "chat_audio",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
		{
			name: "openai transcription model uses transcriptions endpoint",
			detail: service.GatewayPricedModelDetail{
				Name:     "whisper-1",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.000001),
				},
			},
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/audio/transcriptions",
			wantModelType:       "transcription",
			wantOperationType:   "audio_transcription",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
		{
			name: "custom openai compatible video platform uses openai videos entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "custom-video-model",
				Platform: "custom-openai-video",
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModeVideo,
					PerRequestPrice: testPrice(0.4),
				},
			},
			platform: &service.Platform{
				Slug:     "custom-openai-video",
				Protocol: service.PlatformProtocolOpenAICompatible,
			},
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/videos",
			wantModelType:       "video",
			wantOperationType:   "video_generation",
			wantExecutionMode:   "async",
			wantSupportsStream:  false,
			wantSupportsPolling: true,
		},
		{
			name: "grok custom adapter exposes openai videos entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "grok-video-3",
				Platform: service.PlatformGrok,
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModeVideo,
					PerRequestPrice: testPrice(0.4),
				},
			},
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/videos",
			wantModelType:       "video",
			wantOperationType:   "video_generation",
			wantExecutionMode:   "async",
			wantSupportsStream:  false,
			wantSupportsPolling: true,
		},
		{
			name: "kling audio custom adapter exposes openai speech entry",
			detail: service.GatewayPricedModelDetail{
				Name:     "kling-audio",
				Platform: service.PlatformKlingAudio,
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModePerRequest,
					PerRequestPrice: testPrice(0.25),
				},
			},
			wantProtocol:        "openai",
			wantEndpoint:        "/v1/audio/speech",
			wantModelType:       "audio",
			wantOperationType:   "audio_tts",
			wantExecutionMode:   "sync",
			wantSupportsStream:  false,
			wantSupportsPolling: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := publicEntryMetadataForPricedModel(tt.detail, tt.platform)
			require.Equal(t, tt.wantProtocol, got.DefaultEntryProtocol)
			require.Equal(t, tt.wantEndpoint, got.DefaultEndpoint)
			require.Equal(t, tt.wantModelType, got.ModelType)
			require.Equal(t, tt.wantOperationType, got.OperationType)
			require.Equal(t, tt.wantExecutionMode, got.ExecutionMode)
			require.Equal(t, tt.wantSupportsStream, got.SupportsStream)
			require.Equal(t, tt.wantSupportsPolling, got.SupportsPolling)
		})
	}
}

func TestPublicEntryMetadataForPricedModel_ReturnsProtocolSpecificContracts(t *testing.T) {
	tests := []struct {
		name               string
		detail             service.GatewayPricedModelDetail
		platform           *service.Platform
		wantRequired       []string
		wantOptional       []string
		wantNotRequired    []string
		wantDelivery       string
		wantStreamDelivery string
		wantStreamEndpoint string
	}{
		{
			name: "openai responses chat uses input contract",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-5.4-mini",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.00000075),
					OutputPrice: testPrice(0.0000045),
				},
			},
			wantRequired:       []string{"model", "input"},
			wantOptional:       []string{"stream", "reasoning", "max_output_tokens"},
			wantNotRequired:    []string{"messages"},
			wantDelivery:       "openai_responses_json",
			wantStreamDelivery: "openai_responses_sse",
			wantStreamEndpoint: "/v1/responses",
		},
		{
			name: "anthropic messages chat exposes thinking",
			detail: service.GatewayPricedModelDetail{
				Name:     "claude-opus-4-6",
				Platform: service.PlatformAnthropic,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.000005),
					OutputPrice: testPrice(0.000025),
				},
			},
			wantRequired:       []string{"model", "messages", "max_tokens"},
			wantOptional:       []string{"stream", "thinking", "system"},
			wantDelivery:       "anthropic_messages_json",
			wantStreamDelivery: "anthropic_messages_sse",
			wantStreamEndpoint: "/v1/messages",
		},
		{
			name: "gemini native chat uses contents contract",
			detail: service.GatewayPricedModelDetail{
				Name:     "gemini-3.5-flash",
				Platform: service.PlatformGemini,
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  testPrice(0.0000015),
					OutputPrice: testPrice(0.000009),
				},
			},
			wantRequired:       []string{"contents"},
			wantOptional:       []string{"systemInstruction", "generationConfig", "tools"},
			wantNotRequired:    []string{"messages"},
			wantDelivery:       "gemini_generate_content_json",
			wantStreamDelivery: "gemini_sse",
			wantStreamEndpoint: "/v1beta/models/gemini-3.5-flash:streamGenerateContent?alt=sse",
		},
		{
			name: "openai image exposes size enum aliases",
			detail: service.GatewayPricedModelDetail{
				Name:     "gpt-image-2",
				Platform: service.PlatformOpenAI,
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModePerRequest,
					PerRequestPrice: testPrice(0.05),
				},
			},
			wantRequired:    []string{"model", "prompt"},
			wantOptional:    []string{"size", "quality", "background"},
			wantNotRequired: []string{"contents"},
			wantDelivery:    "openai_image_json",
		},
		{
			name: "gemini image exposes image config contract",
			detail: service.GatewayPricedModelDetail{
				Name:     "NanoBanana2",
				Platform: service.PlatformGemini,
				Pricing: &service.ChannelModelPricing{
					BillingMode:     service.BillingModeImage,
					PerRequestPrice: testPrice(0.2),
				},
			},
			wantRequired:    []string{"contents"},
			wantOptional:    []string{"generationConfig", "safetySettings"},
			wantNotRequired: []string{"model", "prompt", "size"},
			wantDelivery:    "gemini_generate_content_json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := publicEntryMetadataForPricedModel(tt.detail, tt.platform)
			for _, field := range tt.wantRequired {
				require.Contains(t, got.RequestContract["required_fields"], field)
			}
			for _, field := range tt.wantOptional {
				require.Contains(t, got.RequestContract["optional_fields"], field)
			}
			for _, field := range tt.wantNotRequired {
				require.NotContains(t, got.RequestContract["required_fields"], field)
			}
			require.Equal(t, tt.wantDelivery, got.ResponseContract["delivery"])
			if tt.wantStreamDelivery != "" {
				require.Equal(t, tt.wantStreamDelivery, got.ResponseContract["stream_delivery"])
			}
			if tt.wantStreamEndpoint != "" {
				require.Equal(t, tt.wantStreamEndpoint, got.StreamEndpoint)
			}
			if got.OperationType == "image_generation" && got.DefaultEntryProtocol == "openai" {
				sizeContract := got.RequestContract["size"].(map[string]any)
				require.Contains(t, sizeContract["values"], "1536x1024")
				aliases := sizeContract["aliases"].(map[string]any)
				require.Equal(t, "1536x1024", aliases["landscape"])
				require.Equal(t, "1024x1536", aliases["mobile_wallpaper"])
				require.Equal(t, "data[].b64_json", got.ResponseContract["image_data_path"])
			}
			if got.OperationType == "image_generation" && got.DefaultEntryProtocol == "gemini" {
				generationConfig := got.RequestContract["generationConfig"].(map[string]any)
				imageConfig := generationConfig["imageConfig"].(map[string]any)
				aspectRatio := imageConfig["aspectRatio"].(map[string]any)
				imageSize := imageConfig["imageSize"].(map[string]any)
				require.Contains(t, aspectRatio["values"], "16:9")
				require.Contains(t, imageSize["values"], "4K")
				require.Equal(t, "9:16", aspectRatio["aliases"].(map[string]any)["mobile_wallpaper"])
				require.Equal(t, "candidates[].content.parts[].inlineData.data", got.ResponseContract["image_data_path"])
			}
			require.NotEmpty(t, got.EntryProtocols)
		})
	}
}

func TestOpenAIChannelRoutingModelUsesChannelMappedModel(t *testing.T) {
	got := openAIChannelRoutingModel("gpt-audio-tts-hd", service.ChannelMappingResult{
		Mapped:      true,
		MappedModel: "tts-1-hd",
	})

	require.Equal(t, "tts-1-hd", got)
}

func TestOpenAIChannelRoutingModelFallsBackToRequestedModel(t *testing.T) {
	got := openAIChannelRoutingModel("gpt-audio-tts-hd", service.ChannelMappingResult{
		Mapped:      false,
		MappedModel: "tts-1-hd",
	})

	require.Equal(t, "gpt-audio-tts-hd", got)
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
