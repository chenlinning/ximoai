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

type modelBrandSettingRepoStub struct {
	values map[string]string
}

func (s *modelBrandSettingRepoStub) Get(_ context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *modelBrandSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *modelBrandSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelBrandSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s *modelBrandSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}

func (s *modelBrandSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	values := make(map[string]string, len(s.values))
	for key, value := range s.values {
		values[key] = value
	}
	return values, nil
}

func (s *modelBrandSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestModelPlazaBrandHandlerSaveAndReset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &modelBrandSettingRepoStub{values: map[string]string{}}
	settingService := service.NewSettingService(repo, nil)
	h := &AvailableChannelHandler{
		settingService:  settingService,
		platformService: service.NewPlatformService(nil),
	}

	saved := runModelBrandHandlerRequest(t, http.MethodPut, `{"platform":"openai","model":"gpt-5.4","brand":"XimoAI Lab"}`, h.SaveModelPlazaBrand)
	require.Equal(t, http.StatusOK, saved.HTTPStatus)
	require.Zero(t, saved.Code)
	require.Equal(t, "XimoAI Lab", saved.Data.Brand)
	require.Equal(t, ximoAIModelBrandSourceAdministrator, saved.Data.Editor.Source)

	overrides, err := settingService.GetXimoAIModelBrandOverrides(context.Background(), []service.ModelBrandOverride{{Platform: "openai", Model: "gpt-5.4"}})
	require.NoError(t, err)
	require.Equal(t, "XimoAI Lab", overrides[0].Brand)

	reset := runModelBrandHandlerRequest(t, http.MethodDelete, `{"platform":"openai","model":"gpt-5.4"}`, h.DeleteModelPlazaBrand)
	require.Equal(t, http.StatusOK, reset.HTTPStatus)
	require.Zero(t, reset.Code)
	require.Equal(t, "OpenAI", reset.Data.Brand)
	require.Equal(t, ximoAIModelBrandSourceAutomatic, reset.Data.Editor.Source)
}

type modelBrandHandlerResponse struct {
	HTTPStatus int               `json:"-"`
	Code       int               `json:"code"`
	Data       modelBrandDetails `json:"data"`
}

func runModelBrandHandlerRequest(
	t *testing.T,
	method string,
	body string,
	handler gin.HandlerFunc,
) modelBrandHandlerResponse {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/api/v1/admin/model-plaza/brand", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler(c)

	var response modelBrandHandlerResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	response.HTTPStatus = w.Code
	return response
}
