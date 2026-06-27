package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ximoDeskHandlerSettingRepoStub struct {
	values map[string]string
}

func newXimoDeskHandlerSettingRepoStub() *ximoDeskHandlerSettingRepoStub {
	return &ximoDeskHandlerSettingRepoStub{values: map[string]string{}}
}

func (r *ximoDeskHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (r *ximoDeskHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (r *ximoDeskHandlerSettingRepoStub) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *ximoDeskHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *ximoDeskHandlerSettingRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		r.values[key] = value
	}
	return nil
}

func (r *ximoDeskHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *ximoDeskHandlerSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func newXimoDeskHandlerTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newXimoDeskHandlerSettingRepoStub()
	settingService := service.NewSettingService(repo, &config.Config{})
	err := settingService.SaveXimoDeskUpdateConfig(context.Background(), &service.XimoDeskUpdateConfig{
		Enabled: true,
		Releases: []service.XimoDeskUpdateRelease{
			{
				Channel:     "stable",
				OS:          "windows",
				Arch:        "x86_64",
				Version:     "1.0.1",
				DownloadURL: "https://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi",
				Notes:       "fixes",
				SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
		},
	})
	require.NoError(t, err)
	h := NewXimoDeskUpdateHandler(settingService)
	r := gin.New()
	r.POST("/api/ximoapp/:appKey/version/latest", h.LatestApp)
	r.GET("/downloads/ximoapp/:file", h.DownloadPackage)
	r.GET("/api/v1/admin/ximoapp/update", h.AdminGet)
	r.PUT("/api/v1/admin/ximoapp/update", h.AdminUpdate)
	r.POST("/api/v1/admin/ximoapp/update/packages", h.AdminUploadPackage)
	r.DELETE("/api/v1/admin/ximoapp/update/releases/:id", h.AdminDeleteRelease)
	r.GET("/api/v1/ximoapp/download-center", h.DownloadCenter)
	return r
}

func TestXimoDeskLatestUsesXimoAppEndpoint(t *testing.T) {
	r := newXimoDeskHandlerTestRouter(t)
	body := `{"app":"XimoDesk","typ":"ximodesk-client","version":"1.0.0","os":"windows","arch":"x86_64","channel":"stable","device_id":[]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ximoapp/ximodesk/version/latest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "1.0.1", got["version"])
	require.Equal(t, "https://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi", got["download_url"])
	require.NotContains(t, got, "code")
	require.NotContains(t, got, "data")
}

func TestXimoDeskLegacyEndpointIsNotRegistered(t *testing.T) {
	r := newXimoDeskHandlerTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ximodesk/version/latest", strings.NewReader(`{"app":"XimoDesk"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestXimoDeskLatestNoMatchReturnsNoContent(t *testing.T) {
	r := newXimoDeskHandlerTestRouter(t)
	body := `{"app":"XimoDesk","typ":"ximodesk-client","version":"1.0.0","os":"windows","arch":"aarch64","channel":"stable","device_id":[]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ximoapp/ximodesk/version/latest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Body.String())
}

func TestXimoDeskLatestInvalidJSONReturnsBadRequest(t *testing.T) {
	r := newXimoDeskHandlerTestRouter(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/ximoapp/ximodesk/version/latest", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestXimoDeskAdminUpdateRejectsBadSHA(t *testing.T) {
	r := newXimoDeskHandlerTestRouter(t)
	body := `{"enabled":true,"releases":[{"channel":"stable","os":"windows","arch":"x86_64","version":"1.0.1","download_url":"https://www.ximoai.cn/downloads/XimoDesk-1.0.1-x64.msi","sha256":"bad","force":false}]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/ximoapp/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestXimoDeskAdminUploadPackageAndDownload(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XIMOAPP_PACKAGE_DIR", dir)
	t.Setenv("XIMOAPP_DOWNLOAD_BASE_URL", "https://ximoai.cn/downloads/ximoapp")
	r := newXimoDeskHandlerTestRouter(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("channel", "stable"))
	require.NoError(t, writer.WriteField("os", "windows"))
	require.NoError(t, writer.WriteField("arch", "x86_64"))
	require.NoError(t, writer.WriteField("locale", "en-US"))
	require.NoError(t, writer.WriteField("version", "1.0.3"))
	require.NoError(t, writer.WriteField("enabled", "true"))
	require.NoError(t, writer.WriteField("notes", "english build"))
	part, err := writer.CreateFormFile("file", "XimoDesk-1.0.3-x64.zip")
	require.NoError(t, err)
	_, err = part.Write([]byte("zip-payload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ximoapp/update/packages", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Release service.XimoDeskUpdateRelease `json:"release"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "en-us", resp.Data.Release.Locale)
	require.Equal(t, "zip", resp.Data.Release.PackageType)
	require.FileExists(t, filepath.Join(dir, resp.Data.Release.FileName))
	require.Contains(t, resp.Data.Release.DownloadURL, "/downloads/ximoapp/")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/downloads/ximoapp/"+resp.Data.Release.FileName, nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "zip-payload", w.Body.String())
}

func TestXimoAppLatestMobileReturnsStandardMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XIMOAPP_PACKAGE_DIR", dir)
	t.Setenv("XIMOAPP_DOWNLOAD_BASE_URL", "https://ximoai.cn/downloads/ximoapp")
	r := newXimoDeskHandlerTestRouter(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("app_key", "ximo-mobile"))
	require.NoError(t, writer.WriteField("channel", "stable"))
	require.NoError(t, writer.WriteField("os", "android"))
	require.NoError(t, writer.WriteField("arch", "universal"))
	require.NoError(t, writer.WriteField("locale", "all"))
	require.NoError(t, writer.WriteField("version", "2.0.0"))
	require.NoError(t, writer.WriteField("min_supported_version", "1.5.0"))
	require.NoError(t, writer.WriteField("min_supported_version_code", "150"))
	require.NoError(t, writer.WriteField("enabled", "true"))
	require.NoError(t, writer.WriteField("notes", "android build"))
	part, err := writer.CreateFormFile("file", "XimoMobile-2.0.0.apk")
	require.NoError(t, err)
	_, err = part.Write([]byte("apk-payload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ximoapp/update/packages", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var uploadResp struct {
		Code int `json:"code"`
		Data struct {
			Release service.XimoDeskUpdateRelease `json:"release"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &uploadResp))
	require.GreaterOrEqual(t, uploadResp.Data.Release.VersionCode, int64(20000101000000))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/ximoapp/ximo-mobile/version/latest", strings.NewReader(`{"app_key":"ximo-mobile","version":"1.0.0","version_code":100,"platform":"android","arch":"aarch64","channel":"stable","locale":"zh-CN"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	require.Equal(t, "ximo-mobile", got["app_key"])
	require.Equal(t, "2.0.0", got["version"])
	require.Equal(t, float64(uploadResp.Data.Release.VersionCode), got["version_code"])
	require.Equal(t, "1.5.0", got["min_supported_version"])
	require.Equal(t, float64(150), got["min_supported_version_code"])
	require.Contains(t, got["download_url"], "/downloads/ximoapp/")
	require.NotContains(t, got, "code")
	require.NotContains(t, got, "data")
}

func TestXimoAppDownloadCenterReturnsPublishedPackages(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XIMOAPP_PACKAGE_DIR", dir)
	t.Setenv("XIMOAPP_DOWNLOAD_BASE_URL", "https://ximoai.cn/downloads/ximoapp")
	r := newXimoDeskHandlerTestRouter(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("app_key", "ximo-mobile"))
	require.NoError(t, writer.WriteField("channel", "stable"))
	require.NoError(t, writer.WriteField("os", "android"))
	require.NoError(t, writer.WriteField("arch", "universal"))
	require.NoError(t, writer.WriteField("locale", "zh-CN"))
	require.NoError(t, writer.WriteField("version", "2.1.0"))
	require.NoError(t, writer.WriteField("enabled", "true"))
	require.NoError(t, writer.WriteField("notes", "public app build"))
	part, err := writer.CreateFormFile("file", "XimoMobile-2.1.0.apk")
	require.NoError(t, err)
	_, err = part.Write([]byte("apk-download-center"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ximoapp/update/packages", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/ximoapp/download-center", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int                           `json:"code"`
		Data service.XimoAppDownloadCenter `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Apps, 2)
	require.Equal(t, "ximo-mobile", resp.Data.Apps[0].Key)
	require.Equal(t, "2.1.0", resp.Data.Apps[0].Releases[0].Version)
	require.Contains(t, resp.Data.Apps[0].Releases[0].DownloadURL, "/downloads/ximoapp/")
}
