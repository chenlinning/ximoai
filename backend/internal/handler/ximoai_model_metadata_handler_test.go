package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelMetadataHandlerSettingRepoStub struct {
	values map[string]string
}

func (s *modelMetadataHandlerSettingRepoStub) Get(_ context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *modelMetadataHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *modelMetadataHandlerSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelMetadataHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *modelMetadataHandlerSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *modelMetadataHandlerSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *modelMetadataHandlerSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestModelPlazaMetadataHandlerSavesPartialOverridesAndResets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &modelMetadataHandlerSettingRepoStub{values: map[string]string{}}
	settingService := service.NewSettingService(repo, nil)
	h := &AvailableChannelHandler{
		settingService: settingService,
	}

	saved := runModelMetadataHandlerRequest(t, http.MethodPut,
		`{"platform":"openai","model":"gpt-5.4","types":["conversation","image"],"invocation_modes":["sync","batch"],"reasoning_levels":["low","high"],"thinking_supported":true}`,
		h.SaveModelPlazaMetadata,
	)
	require.Equal(t, http.StatusOK, saved.HTTPStatus)
	require.Zero(t, saved.Code)
	require.Nil(t, saved.Data.Brand)
	require.Equal(t, []string{"conversation", "image"}, *saved.Data.Types)
	require.Equal(t, []string{"sync", "batch"}, *saved.Data.InvocationModes)
	require.Equal(t, []string{"low", "high"}, *saved.Data.ReasoningLevels)
	require.NotNil(t, saved.Data.ThinkingSupported)
	require.True(t, *saved.Data.ThinkingSupported)

	overrides, err := settingService.GetXimoAIModelMetadataOverrides(context.Background(), []service.ModelMetadataOverride{{Platform: "openai", Model: "gpt-5.4"}})
	require.NoError(t, err)
	require.Equal(t, []string{"conversation", "image"}, *overrides[0].Types)

	reset := runModelMetadataHandlerRequest(t, http.MethodDelete,
		`{"platform":"openai","model":"gpt-5.4"}`,
		h.DeleteModelPlazaMetadata,
	)
	require.Equal(t, http.StatusOK, reset.HTTPStatus)
	require.Zero(t, reset.Code)
	overrides, err = settingService.GetXimoAIModelMetadataOverrides(context.Background(), []service.ModelMetadataOverride{{Platform: "openai", Model: "gpt-5.4"}})
	require.NoError(t, err)
	require.Nil(t, overrides[0])
}

func TestModelPlazaMetadataHandlerRejectsUnknownOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &modelMetadataHandlerSettingRepoStub{values: map[string]string{}}
	h := &AvailableChannelHandler{
		settingService: service.NewSettingService(repo, nil),
	}

	result := runModelMetadataHandlerRequest(t, http.MethodPut,
		`{"platform":"openai","model":"gpt-5.4","invocation_modes":["provider_magic"]}`,
		h.SaveModelPlazaMetadata,
	)
	require.Equal(t, http.StatusBadRequest, result.HTTPStatus)
	require.NotZero(t, result.Code)
}

type modelMetadataHandlerResponse struct {
	HTTPStatus int                           `json:"-"`
	Code       int                           `json:"code"`
	Data       modelMetadataOverrideResponse `json:"data"`
}

func runModelMetadataHandlerRequest(
	t *testing.T,
	method string,
	body string,
	handler gin.HandlerFunc,
) modelMetadataHandlerResponse {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/api/v1/admin/model-plaza/metadata", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)

	var result modelMetadataHandlerResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	result.HTTPStatus = w.Code
	return result
}
