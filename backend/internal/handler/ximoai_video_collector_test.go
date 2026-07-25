package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type videoCollectorMembershipStub struct {
	summary *service.MembershipSummary
	err     error
}

const videoCollectorTestToken = "0123456789abcdef0123456789abcdef"

func (s videoCollectorMembershipStub) GetUserMembership(context.Context, int64) (*service.MembershipSummary, error) {
	return s.summary, s.err
}

func videoCollectorTestRouter(h *VideoCollectorHandler, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if authenticated {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		}
		c.Next()
	})
	r.POST("/parse", h.Parse)
	r.GET("/tasks/:id/download", h.Download)
	return r
}

func TestVideoCollectorRequiresDiamondMembership(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	defer upstream.Close()

	h := newVideoCollectorHandler(videoCollectorMembershipStub{summary: &service.MembershipSummary{
		Level:    &service.MembershipLevel{Code: service.MembershipLevelCodeGold, Enabled: true},
		StartsAt: time.Now().Add(-time.Hour),
	}}, upstream.URL, videoCollectorTestToken, upstream.Client())
	r := videoCollectorTestRouter(h, true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parse", strings.NewReader(`{"url":"https://example.com/video"}`))
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestVideoCollectorRequiresAuthentication(t *testing.T) {
	h := newVideoCollectorHandler(videoCollectorMembershipStub{}, "http://127.0.0.1", videoCollectorTestToken, http.DefaultClient)
	r := videoCollectorTestRouter(h, false)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parse", strings.NewReader(`{"url":"https://example.com/video"}`))
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestVideoCollectorRejectsExpiredDiamondMembership(t *testing.T) {
	expiresAt := time.Now().Add(-time.Minute)
	h := newVideoCollectorHandler(videoCollectorMembershipStub{summary: &service.MembershipSummary{
		Level:     &service.MembershipLevel{Code: service.MembershipLevelCodeDiamond, Enabled: true},
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: &expiresAt,
	}}, "http://127.0.0.1", videoCollectorTestToken, http.DefaultClient)
	r := videoCollectorTestRouter(h, true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parse", strings.NewReader(`{"url":"https://example.com/video"}`))
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestVideoCollectorProxiesOnlyAuthenticatedUserIdentity(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "42", r.Header.Get("X-Video-Collector-User"))
		require.Equal(t, videoCollectorTestToken, r.Header.Get("X-Video-Collector-Token"))
		require.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"media-1","formats":[{"id":"best"}]}`)
	}))
	defer upstream.Close()

	h := newVideoCollectorHandler(videoCollectorMembershipStub{summary: &service.MembershipSummary{
		Level:    &service.MembershipLevel{Code: service.MembershipLevelCodeDiamond, Enabled: true},
		StartsAt: time.Now().Add(-time.Hour),
	}}, upstream.URL, videoCollectorTestToken, upstream.Client())
	r := videoCollectorTestRouter(h, true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/parse", strings.NewReader(`{"url":"https://example.com/video"}`))
	req.Header.Set("Authorization", "Bearer user-jwt")
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "media-1")
}

func TestVideoCollectorStreamsDownloadHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", `attachment; filename="video.mp4"`)
		w.Header().Set("X-Delete-At", "2026-07-25T12:10:00Z")
		_, _ = io.WriteString(w, "video-bytes")
	}))
	defer upstream.Close()

	h := newVideoCollectorHandler(videoCollectorMembershipStub{summary: &service.MembershipSummary{
		Level:    &service.MembershipLevel{Code: service.MembershipLevelCodeDiamond, Enabled: true},
		StartsAt: time.Now().Add(-time.Hour),
	}}, upstream.URL, videoCollectorTestToken, upstream.Client())
	r := videoCollectorTestRouter(h, true)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/download", nil)
	r.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	require.Equal(t, `attachment; filename="video.mp4"`, recorder.Header().Get("Content-Disposition"))
	require.Equal(t, "2026-07-25T12:10:00Z", recorder.Header().Get("X-Delete-At"))
	require.Equal(t, "video-bytes", recorder.Body.String())
}
