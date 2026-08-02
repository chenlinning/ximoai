package service

import (
	"context"
	"crypto/hmac"
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
const workbenchSSOSecretContext = "ximoai-workbench-sso-audience:"

type WorkbenchSSOTicketStore interface {
	StoreTicket(ctx context.Context, key string, payload []byte, ttl time.Duration) (bool, error)
	ConsumeTicket(ctx context.Context, key string) (string, bool, error)
	ConsumeTicketForUser(ctx context.Context, key string, userID int64) (string, bool, error)
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
	controlTokens      *WorkbenchControlTokenService
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
	FeatureFlags     map[string]any                 `json:"featureFlags"`
	Permissions      []string                       `json:"permissions"`
}

type WorkbenchSSOValidation struct {
	*WorkbenchUserContext
	Authorization *WorkbenchControlAuthorization `json:"authorization"`
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
	controlTokens *WorkbenchControlTokenService,
) *WorkbenchSSOService {
	return &WorkbenchSSOService{
		cfg:                cfg,
		settingService:     settingService,
		ticketStore:        ticketStore,
		userGetter:         userService,
		membershipGetter:   membershipService,
		announcementLister: announcementService,
		controlTokens:      controlTokens,
	}
}

func (s *WorkbenchSSOService) IssueTicket(ctx context.Context, userID int64, audience string) (*WorkbenchSSOTicket, error) {
	settings, err := s.currentSettings(ctx)
	if err != nil {
		return nil, err
	}
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return nil, err
	}
	if err := s.requireDiamondMembershipForAudience(ctx, userID, normalizedAudience); err != nil {
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
		entryURL, err := buildWorkbenchEntryURL(normalizedAudience, ticket)
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

func (s *WorkbenchSSOService) ValidateTicket(ctx context.Context, ticket, audience string) (*WorkbenchSSOValidation, error) {
	return s.validateTicket(ctx, ticket, audience, 0)
}

func (s *WorkbenchSSOService) ValidateTicketForUser(ctx context.Context, ticket, audience string, userID int64) (*WorkbenchSSOValidation, error) {
	if userID <= 0 {
		return nil, infraerrors.Unauthorized("WORKBENCH_SSO_TICKET_INVALID", "ticket is invalid or expired")
	}
	return s.validateTicket(ctx, ticket, audience, userID)
}

func (s *WorkbenchSSOService) validateTicket(ctx context.Context, ticket, audience string, expectedUserID int64) (*WorkbenchSSOValidation, error) {
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(ticket) == "" {
		return nil, infraerrors.BadRequest("WORKBENCH_SSO_TICKET_REQUIRED", "ticket is required")
	}
	if err := s.ensureRedis(ctx); err != nil {
		return nil, err
	}

	key := workbenchSSOTicketKey(ticket)
	var raw string
	var ok bool
	if expectedUserID > 0 {
		raw, ok, err = s.ticketStore.ConsumeTicketForUser(ctx, key, expectedUserID)
	} else {
		raw, ok, err = s.ticketStore.ConsumeTicket(ctx, key)
	}
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
	if err := s.requireDiamondMembershipForAudience(ctx, record.UserID, normalizedAudience); err != nil {
		return nil, err
	}
	userContext, err := s.buildUserContext(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if s.controlTokens == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	authorization, err := s.controlTokens.Issue(ctx, record.UserID, normalizedAudience)
	if err != nil {
		return nil, err
	}
	return &WorkbenchSSOValidation{
		WorkbenchUserContext: userContext,
		Authorization:        authorization,
	}, nil
}

func (s *WorkbenchSSOService) requireDiamondMembershipForAudience(ctx context.Context, userID int64, audience string) error {
	if s == nil || s.settingService == nil || s.membershipGetter == nil {
		return infraerrors.InternalServer("WORKBENCH_SSO_MEMBERSHIP_UNAVAILABLE", "workbench membership check is unavailable")
	}
	tabs, err := s.settingService.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return err
	}
	requiresDiamond := false
	for _, tab := range tabs {
		if !tab.Enabled || !tab.WorkbenchSSO || !tab.DiamondOnly {
			continue
		}
		if normalizeWorkbenchAudience(tab.URL) == audience {
			requiresDiamond = true
			break
		}
	}
	if !requiresDiamond {
		return nil
	}
	return s.requireDiamondMembership(ctx, userID)
}

func (s *WorkbenchSSOService) requireDiamondMembership(ctx context.Context, userID int64) error {
	if s == nil || s.membershipGetter == nil {
		return infraerrors.InternalServer("WORKBENCH_SSO_MEMBERSHIP_UNAVAILABLE", "workbench membership check is unavailable")
	}
	summary, err := s.membershipGetter.GetUserMembership(ctx, userID)
	if err != nil {
		return infraerrors.InternalServer("WORKBENCH_SSO_MEMBERSHIP_UNAVAILABLE", "workbench membership check is unavailable").WithCause(err)
	}
	if summary == nil || summary.Level == nil || !strings.EqualFold(strings.TrimSpace(summary.Level.Code), "diamond") {
		return infraerrors.Forbidden("WORKBENCH_SSO_DIAMOND_REQUIRED", "diamond membership is required for this home tab")
	}
	return nil
}

func (s *WorkbenchSSOService) RefreshControlToken(ctx context.Context, refreshToken, audience string) (*WorkbenchControlAuthorization, error) {
	if s == nil || s.controlTokens == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return nil, err
	}
	return s.controlTokens.Refresh(ctx, refreshToken, normalizedAudience)
}

func (s *WorkbenchSSOService) RefreshControlTokenForUser(ctx context.Context, refreshToken, audience string, userID int64) (*WorkbenchControlAuthorization, error) {
	if s == nil || s.controlTokens == nil {
		return nil, infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return nil, err
	}
	return s.controlTokens.RefreshForUser(ctx, refreshToken, normalizedAudience, userID)
}

func (s *WorkbenchSSOService) RevokeControlToken(ctx context.Context, refreshToken, audience string) error {
	if s == nil || s.controlTokens == nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return err
	}
	return s.controlTokens.Revoke(ctx, refreshToken, normalizedAudience)
}

func (s *WorkbenchSSOService) RevokeControlTokenForUser(ctx context.Context, refreshToken, audience string, userID int64) error {
	if s == nil || s.controlTokens == nil {
		return infraerrors.InternalServer("WORKBENCH_CONTROL_UNAVAILABLE", "workbench control authorization is unavailable")
	}
	normalizedAudience, err := s.validateAudience(ctx, audience)
	if err != nil {
		return err
	}
	return s.controlTokens.RevokeForUser(ctx, refreshToken, normalizedAudience, userID)
}

func (s *WorkbenchSSOService) AuthorizeAudience(ctx context.Context, audience, actualSecret string) bool {
	normalizedAudience := normalizeWorkbenchAudience(audience)
	resolvedAudience, ok := s.ResolveAudienceCredential(ctx, actualSecret)
	return ok && normalizedAudience != "" && resolvedAudience == normalizedAudience
}

func (s *WorkbenchSSOService) ResolveAudienceCredential(ctx context.Context, actualSecret string) (string, bool) {
	if s == nil || s.cfg == nil || s.settingService == nil {
		return "", false
	}
	tabs, err := s.settingService.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return "", false
	}
	for _, audience := range XimoAIHomeTabSSOOrigins(tabs) {
		expectedSecret, deriveErr := DeriveWorkbenchAudienceSecret(s.cfg.WorkbenchSSO.InternalSecret, audience)
		if deriveErr == nil && ConstantTimeBearerEqual(actualSecret, expectedSecret) {
			return audience, true
		}
	}
	return "", false
}

func DeriveWorkbenchAudienceSecret(masterSecret, audience string) (string, error) {
	masterSecret = strings.TrimSpace(masterSecret)
	normalizedAudience := normalizeWorkbenchAudience(audience)
	if len([]byte(masterSecret)) < 32 || normalizedAudience == "" {
		return "", fmt.Errorf("workbench audience secret requires a 32-byte master secret and valid audience")
	}
	mac := hmac.New(sha256.New, []byte(masterSecret))
	_, _ = mac.Write([]byte(workbenchSSOSecretContext + normalizedAudience))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

type workbenchSSOSettings struct {
	ttlSeconds int
}

func (s *WorkbenchSSOService) currentSettings(ctx context.Context) (workbenchSSOSettings, error) {
	if s == nil || s.settingService == nil {
		return workbenchSSOSettings{}, infraerrors.InternalServer("WORKBENCH_SSO_NOT_CONFIGURED", "workbench sso is not configured")
	}
	if s.cfg == nil || len([]byte(strings.TrimSpace(s.cfg.WorkbenchSSO.InternalSecret))) < 32 {
		return workbenchSSOSettings{}, infraerrors.InternalServer("WORKBENCH_SSO_SECRET_MISSING", "workbench sso master secret is not configured")
	}
	ttlSeconds, err := s.settingService.GetWorkbenchTicketTTLSeconds(ctx)
	if err != nil {
		return workbenchSSOSettings{}, err
	}
	return workbenchSSOSettings{ttlSeconds: ttlSeconds}, nil
}

func (s *WorkbenchSSOService) validateAudience(ctx context.Context, audience string) (string, error) {
	if s == nil || s.settingService == nil {
		return "", infraerrors.InternalServer("WORKBENCH_SSO_NOT_CONFIGURED", "workbench sso is not configured")
	}
	normalizedAudience := normalizeWorkbenchAudience(audience)
	if normalizedAudience == "" {
		return "", infraerrors.BadRequest("WORKBENCH_SSO_AUDIENCE_INVALID", "audience must be an absolute http(s) URL")
	}
	tabs, err := s.settingService.GetXimoAIHomeTabs(ctx)
	if err != nil {
		return "", err
	}
	for _, allowed := range XimoAIHomeTabSSOOrigins(tabs) {
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

func buildWorkbenchEntryURL(audience, ticket string) (string, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(audience), "/") + "/sso/entry")
	if err != nil || u.Host == "" {
		return "", infraerrors.InternalServer("WORKBENCH_SSO_AUDIENCE_INVALID", "workbench audience is invalid")
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
