package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const workbenchSSOTicketKeyPrefix = "workbench:sso:ticket:"

type WorkbenchSSOTicketStore interface {
	StoreTicket(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error)
	ConsumeTicket(ctx context.Context, key string) (string, bool, error)
	Ping(ctx context.Context) error
}

type workbenchUserGetter interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type workbenchMembershipGetter interface {
	GetUserMembership(ctx context.Context, userID int64) (*MembershipSummary, error)
}

type workbenchAnnouncementLister interface {
	ListForUser(ctx context.Context, userID int64, unreadOnly bool) ([]UserAnnouncement, error)
}

type WorkbenchSSOService struct {
	cfg                *config.Config
	settingService     *SettingService
	ticketStore        WorkbenchSSOTicketStore
	userGetter         workbenchUserGetter
	membershipGetter   workbenchMembershipGetter
	announcementLister workbenchAnnouncementLister
}

type WorkbenchSSOTicket struct {
	Ticket    string
	ExpiresIn int
	EntryURL  string
}

type WorkbenchUserContext struct {
	UserID           string                         `json:"userId"`
	Email            string                         `json:"email,omitempty"`
	Username         string                         `json:"username,omitempty"`
	DisplayName      string                         `json:"displayName,omitempty"`
	AvatarURL        string                         `json:"avatarUrl,omitempty"`
	Balance          float64                        `json:"balance"`
	Announcements    []WorkbenchAnnouncementContext `json:"announcements"`
	Role             string                         `json:"role"`
	MembershipStatus string                         `json:"membershipStatus"`
	Quota            map[string]any                 `json:"quota"`
	ModelConfig      map[string]any                 `json:"modelConfig"`
	FeatureFlags     map[string]any                 `json:"featureFlags"`
	Permissions      []string                       `json:"permissions"`
}

type WorkbenchAnnouncementContext struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type workbenchTicketRecord struct {
	UserID    int64  `json:"user_id"`
	Audience  string `json:"audience"`
	ExpiresAt int64  `json:"expires_at"`
}

func NewWorkbenchSSOService(
	cfg *config.Config,
	settingService *SettingService,
	ticketStore WorkbenchSSOTicketStore,
	userService *UserService,
	membershipService *MembershipService,
	announcementService *AnnouncementService,
) *WorkbenchSSOService {
	return &WorkbenchSSOService{
		cfg:                cfg,
		settingService:     settingService,
		ticketStore:        ticketStore,
		userGetter:         userService,
		membershipGetter:   membershipService,
		announcementLister: announcementService,
	}
}

func (s *WorkbenchSSOService) IssueTicket(ctx context.Context, userID int64, audience string) (*WorkbenchSSOTicket, error) {
	settings, err := s.currentSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.enabled {
		return nil, infraerrors.Forbidden("WORKBENCH_SSO_DISABLED", "workbench sso is disabled")
	}
	normalizedAudience, err := s.validateAudience(audience, settings.baseURL)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRedis(ctx); err != nil {
		return nil, err
	}

	expiresIn := settings.ttlSeconds
	ttl := time.Duration(expiresIn) * time.Second
	for i := 0; i < 3; i++ {
		ticket, err := randomTicket()
		if err != nil {
			return nil, err
		}
		record := workbenchTicketRecord{
			UserID:    userID,
			Audience:  normalizedAudience,
			ExpiresAt: time.Now().Add(ttl).Unix(),
		}
		payload, err := json.Marshal(record)
		if err != nil {
			return nil, fmt.Errorf("marshal workbench sso ticket record: %w", err)
		}
		ok, err := s.ticketStore.StoreTicket(ctx, workbenchSSOTicketKey(ticket), payload, ttl)
		if err != nil {
			return nil, infraerrors.InternalServer("WORKBENCH_SSO_REDIS_UNAVAILABLE", "workbench sso ticket store is unavailable").WithCause(err)
		}
		if !ok {
			continue
		}
		entryURL, err := buildWorkbenchEntryURL(settings.baseURL, ticket)
		if err != nil {
			return nil, err
		}
		return &WorkbenchSSOTicket{
			Ticket:    ticket,
			ExpiresIn: expiresIn,
			EntryURL:  entryURL,
		}, nil
	}
	return nil, infraerrors.InternalServer("WORKBENCH_SSO_TICKET_COLLISION", "failed to create workbench sso ticket")
}

func (s *WorkbenchSSOService) ValidateTicket(ctx context.Context, ticket, audience string) (*WorkbenchUserContext, error) {
	settings, err := s.currentSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.enabled {
		return nil, infraerrors.Forbidden("WORKBENCH_SSO_DISABLED", "workbench sso is disabled")
	}
	normalizedAudience, err := s.validateAudience(audience, settings.baseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(ticket) == "" {
		return nil, infraerrors.BadRequest("WORKBENCH_SSO_TICKET_REQUIRED", "ticket is required")
	}
	if err := s.ensureRedis(ctx); err != nil {
		return nil, err
	}

	raw, ok, err := s.ticketStore.ConsumeTicket(ctx, workbenchSSOTicketKey(ticket))
	if err != nil {
		return nil, infraerrors.InternalServer("WORKBENCH_SSO_REDIS_UNAVAILABLE", "workbench sso ticket store is unavailable").WithCause(err)
	}
	if !ok {
		return nil, infraerrors.Unauthorized("WORKBENCH_SSO_TICKET_INVALID", "ticket is invalid or expired")
	}
	var record workbenchTicketRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, infraerrors.Unauthorized("WORKBENCH_SSO_TICKET_INVALID", "ticket is invalid or expired")
	}
	if record.ExpiresAt > 0 && time.Now().Unix() > record.ExpiresAt {
		return nil, infraerrors.Unauthorized("WORKBENCH_SSO_TICKET_EXPIRED", "ticket is expired")
	}
	if record.Audience != normalizedAudience {
		return nil, infraerrors.Unauthorized("WORKBENCH_SSO_AUDIENCE_MISMATCH", "ticket audience mismatch")
	}
	return s.buildUserContext(ctx, record.UserID)
}

func (s *WorkbenchSSOService) InternalSecret() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.WorkbenchSSO.InternalSecret)
}

type workbenchSSOSettings struct {
	enabled    bool
	baseURL    string
	ttlSeconds int
}

func (s *WorkbenchSSOService) currentSettings(ctx context.Context) (workbenchSSOSettings, error) {
	if s == nil || s.settingService == nil {
		return workbenchSSOSettings{}, infraerrors.InternalServer("WORKBENCH_SSO_NOT_CONFIGURED", "workbench sso is not configured")
	}
	enabled, baseURL, ttlSeconds, err := s.settingService.GetWorkbenchSSOSettings(ctx)
	if err != nil {
		return workbenchSSOSettings{}, err
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if enabled && baseURL == "" {
		return workbenchSSOSettings{}, infraerrors.InternalServer("WORKBENCH_SSO_BASE_URL_MISSING", "workbench base url is not configured")
	}
	return workbenchSSOSettings{enabled: enabled, baseURL: baseURL, ttlSeconds: ttlSeconds}, nil
}

func (s *WorkbenchSSOService) validateAudience(audience, baseURL string) (string, error) {
	normalizedAudience := normalizeWorkbenchAudience(audience)
	if normalizedAudience == "" {
		return "", infraerrors.BadRequest("WORKBENCH_SSO_AUDIENCE_INVALID", "audience must be an absolute http(s) URL")
	}
	for _, allowed := range allowedWorkbenchAudiences(baseURL) {
		if normalizedAudience == allowed {
			return normalizedAudience, nil
		}
	}
	return "", infraerrors.BadRequest("WORKBENCH_SSO_AUDIENCE_NOT_ALLOWED", "audience is not allowed")
}

func (s *WorkbenchSSOService) ensureRedis(ctx context.Context) error {
	if s == nil || s.ticketStore == nil {
		return infraerrors.InternalServer("WORKBENCH_SSO_REDIS_UNAVAILABLE", "workbench sso requires redis")
	}
	if err := s.ticketStore.Ping(ctx); err != nil {
		return infraerrors.InternalServer("WORKBENCH_SSO_REDIS_UNAVAILABLE", "workbench sso ticket store is unavailable").WithCause(err)
	}
	return nil
}

func (s *WorkbenchSSOService) buildUserContext(ctx context.Context, userID int64) (*WorkbenchUserContext, error) {
	if s.userGetter == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_SSO_USER_CONTEXT_UNAVAILABLE", "user context is unavailable")
	}
	user, err := s.userGetter.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	membershipStatus := ""
	if s.membershipGetter != nil {
		if summary, err := s.membershipGetter.GetUserMembership(ctx, userID); err == nil && summary != nil && summary.Level != nil {
			membershipStatus = strings.TrimSpace(summary.Level.Code)
		}
	}
	if membershipStatus == "" {
		membershipStatus = "free"
	}
	announcements := s.userAnnouncements(ctx, userID)
	return &WorkbenchUserContext{
		UserID:           strconv.FormatInt(user.ID, 10),
		Email:            strings.TrimSpace(user.Email),
		Username:         strings.TrimSpace(user.Username),
		DisplayName:      firstNonEmptyString(user.Username, user.Email, strconv.FormatInt(user.ID, 10)),
		AvatarURL:        strings.TrimSpace(user.AvatarURL),
		Balance:          user.Balance,
		Announcements:    announcements,
		Role:             user.Role,
		MembershipStatus: membershipStatus,
		Quota:            map[string]any{},
		ModelConfig:      map[string]any{},
		FeatureFlags:     map[string]any{},
		Permissions:      []string{},
	}, nil
}

func (s *WorkbenchSSOService) userAnnouncements(ctx context.Context, userID int64) []WorkbenchAnnouncementContext {
	if s.announcementLister == nil {
		return []WorkbenchAnnouncementContext{}
	}
	items, err := s.announcementLister.ListForUser(ctx, userID, false)
	if err != nil {
		return []WorkbenchAnnouncementContext{}
	}
	if len(items) > 20 {
		items = items[:20]
	}
	result := make([]WorkbenchAnnouncementContext, 0, len(items))
	for _, item := range items {
		result = append(result, WorkbenchAnnouncementContext{
			ID:        item.Announcement.ID,
			Title:     item.Announcement.Title,
			Content:   item.Announcement.Content,
			ReadAt:    item.ReadAt,
			CreatedAt: item.Announcement.CreatedAt,
		})
	}
	return result
}

func randomTicket() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate workbench sso ticket: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func workbenchSSOTicketKey(ticket string) string {
	sum := sha256.Sum256([]byte(ticket))
	return workbenchSSOTicketKeyPrefix + hex.EncodeToString(sum[:])
}

func normalizeWorkbenchAudience(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func allowedWorkbenchAudiences(baseURL string) []string {
	seen := map[string]struct{}{}
	var audiences []string
	add := func(raw string) {
		if normalized := normalizeWorkbenchAudience(raw); normalized != "" {
			if _, ok := seen[normalized]; !ok {
				seen[normalized] = struct{}{}
				audiences = append(audiences, normalized)
			}
		}
	}
	add(baseURL)
	add("http://127.0.0.1:4173")
	add("http://localhost:4173")
	add("https://workbench.ximoai.cn")
	return audiences
}

func buildWorkbenchEntryURL(baseURL, ticket string) (string, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/sso/entry")
	if err != nil || u.Host == "" {
		return "", infraerrors.InternalServer("WORKBENCH_SSO_BASE_URL_INVALID", "workbench base url is invalid")
	}
	q := u.Query()
	q.Set("ticket", ticket)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func ConstantTimeBearerEqual(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
