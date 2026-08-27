package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyXimoAIHomeTabs = "ximoai_home_tabs"
	maxXimoAIHomeTabs        = 24
	maxXimoAIHomeTabLabelLen = 48
	maxXimoAIHomeTabURLLen   = 2048
)

var ximoAIHomeTabIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type XimoAIHomeTab struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	URL          string `json:"url"`
	Enabled      bool   `json:"enabled"`
	WorkbenchSSO bool   `json:"workbench_sso,omitempty"`
	DiamondOnly  bool   `json:"diamond_only,omitempty"`
	SortOrder    int    `json:"sort_order"`
}

func (s *SettingService) GetXimoAIHomeTabs(ctx context.Context) ([]XimoAIHomeTab, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyXimoAIHomeTabs)
	if errors.Is(err, ErrSettingNotFound) {
		return []XimoAIHomeTab{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get XimoAI home tabs: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return []XimoAIHomeTab{}, nil
	}

	var tabs []XimoAIHomeTab
	if err := json.Unmarshal([]byte(raw), &tabs); err != nil {
		return nil, fmt.Errorf("decode XimoAI home tabs: %w", err)
	}
	return normalizeXimoAIHomeTabs(tabs, false)
}

func (s *SettingService) UpdateXimoAIHomeTabs(ctx context.Context, tabs []XimoAIHomeTab) ([]XimoAIHomeTab, error) {
	normalized, err := normalizeXimoAIHomeTabs(tabs, true)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode XimoAI home tabs: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyXimoAIHomeTabs, string(payload)); err != nil {
		return nil, fmt.Errorf("save XimoAI home tabs: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return normalized, nil
}

func normalizeXimoAIHomeTabs(tabs []XimoAIHomeTab, generateMissingIDs bool) ([]XimoAIHomeTab, error) {
	if len(tabs) > maxXimoAIHomeTabs {
		return nil, infraerrors.BadRequest("INVALID_XIMOAI_HOME_TABS", fmt.Sprintf("home tabs cannot exceed %d entries", maxXimoAIHomeTabs))
	}

	normalized := make([]XimoAIHomeTab, 0, len(tabs))
	seenIDs := make(map[string]struct{}, len(tabs))
	for index, tab := range tabs {
		tab.ID = strings.TrimSpace(tab.ID)
		tab.Label = strings.TrimSpace(tab.Label)
		tab.URL = strings.TrimSpace(tab.URL)

		if tab.ID == "" && generateMissingIDs {
			generatedID, err := generateXimoAIHomeTabID()
			if err != nil {
				return nil, err
			}
			tab.ID = generatedID
		}
		if !ximoAIHomeTabIDPattern.MatchString(tab.ID) {
			return nil, invalidXimoAIHomeTabs("tab %d has an invalid id", index+1)
		}
		if _, exists := seenIDs[tab.ID]; exists {
			return nil, invalidXimoAIHomeTabs("tab %d duplicates id %q", index+1, tab.ID)
		}
		seenIDs[tab.ID] = struct{}{}

		if tab.Label == "" || len([]rune(tab.Label)) > maxXimoAIHomeTabLabelLen {
			return nil, invalidXimoAIHomeTabs("tab %d label must contain 1-%d characters", index+1, maxXimoAIHomeTabLabelLen)
		}
		if err := validateXimoAIHomePageURL(tab.URL); err != nil {
			return nil, invalidXimoAIHomeTabs("tab %d URL: %v", index+1, err)
		}
		tab.SortOrder = index
		normalized = append(normalized, tab)
	}
	return normalized, nil
}

func validateXimoAIHomePageURL(raw string) error {
	if raw == "" || len(raw) > maxXimoAIHomeTabURLLen {
		return fmt.Errorf("must contain 1-%d characters", maxXimoAIHomeTabURLLen)
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("must not contain embedded credentials")
	}
	return nil
}

func generateXimoAIHomeTabID() (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate XimoAI home tab id: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func invalidXimoAIHomeTabs(format string, args ...any) error {
	return infraerrors.BadRequest("INVALID_XIMOAI_HOME_TABS", fmt.Sprintf(format, args...))
}

func XimoAIHomeTabFrameOrigins(tabs []XimoAIHomeTab) []string {
	seen := make(map[string]struct{}, len(tabs))
	origins := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if !tab.Enabled {
			continue
		}
		parsed, err := url.Parse(strings.TrimSpace(tab.URL))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			continue
		}
		origin := parsed.Scheme + "://" + parsed.Host
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}

func XimoAIHomeTabSSOOrigins(tabs []XimoAIHomeTab) []string {
	filtered := make([]XimoAIHomeTab, 0, len(tabs))
	for _, tab := range tabs {
		if tab.WorkbenchSSO {
			filtered = append(filtered, tab)
		}
	}
	return XimoAIHomeTabFrameOrigins(filtered)
}
