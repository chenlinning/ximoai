package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyXimoAppUpdateConfig  = "ximoai_ximoapp_update_config"
	SettingKeyXimoDeskUpdateConfig = "ximoai_ximodesk_update_config"
	defaultXimoAppKeyXimoDesk      = "ximodesk"
	defaultXimoAppKeyMobile        = "ximo-mobile"
)

var (
	ximoDeskSHA256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	ximoDeskLocalePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	ximoAppKeyPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
)

type XimoAppUpdateApp struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ClientType   string `json:"client_type,omitempty"`
	ResponseMode string `json:"response_mode,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type XimoDeskUpdateConfig struct {
	Enabled  bool                    `json:"enabled"`
	Apps     []XimoAppUpdateApp      `json:"apps,omitempty"`
	Releases []XimoDeskUpdateRelease `json:"releases"`
}

type XimoDeskUpdateRelease struct {
	ID                      string `json:"id,omitempty"`
	AppKey                  string `json:"app_key,omitempty"`
	Enabled                 *bool  `json:"enabled,omitempty"`
	Channel                 string `json:"channel"`
	OS                      string `json:"os"`
	Arch                    string `json:"arch"`
	Locale                  string `json:"locale,omitempty"`
	PackageType             string `json:"package_type,omitempty"`
	Version                 string `json:"version"`
	VersionCode             int64  `json:"version_code,omitempty"`
	MinSupportedVersion     string `json:"min_supported_version,omitempty"`
	MinSupportedVersionCode int64  `json:"min_supported_version_code,omitempty"`
	DownloadURL             string `json:"download_url"`
	Notes                   string `json:"notes"`
	SHA256                  string `json:"sha256"`
	Force                   bool   `json:"force"`
	FileName                string `json:"file_name,omitempty"`
	FileSize                int64  `json:"file_size,omitempty"`
	UploadedAt              string `json:"uploaded_at,omitempty"`
	PublishedAt             string `json:"published_at,omitempty"`
}

type XimoDeskVersionRequest struct {
	App         string          `json:"app"`
	AppKey      string          `json:"app_key,omitempty"`
	Type        string          `json:"typ"`
	Version     string          `json:"version"`
	VersionCode int64           `json:"version_code,omitempty"`
	BuildNumber int64           `json:"build_number,omitempty"`
	OS          string          `json:"os"`
	Platform    string          `json:"platform,omitempty"`
	OSVersion   string          `json:"os_version"`
	Arch        string          `json:"arch"`
	Channel     string          `json:"channel"`
	Locale      string          `json:"locale,omitempty"`
	Language    string          `json:"language,omitempty"`
	DeviceID    json.RawMessage `json:"device_id,omitempty"`
}

type XimoAppUpdateResponse struct {
	AppKey                  string `json:"app_key,omitempty"`
	Version                 string `json:"version"`
	VersionCode             int64  `json:"version_code,omitempty"`
	MinSupportedVersion     string `json:"min_supported_version,omitempty"`
	MinSupportedVersionCode int64  `json:"min_supported_version_code,omitempty"`
	DownloadURL             string `json:"download_url"`
	Notes                   string `json:"notes"`
	SHA256                  string `json:"sha256"`
	Force                   bool   `json:"force"`
	PackageType             string `json:"package_type,omitempty"`
	FileSize                int64  `json:"file_size,omitempty"`
	PublishedAt             string `json:"published_at,omitempty"`
}

type XimoAppDownloadCenter struct {
	Apps []XimoAppDownloadApp `json:"apps"`
}

type XimoAppDownloadApp struct {
	Key         string                   `json:"key"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	ClientType  string                   `json:"client_type,omitempty"`
	Releases    []XimoAppDownloadRelease `json:"releases"`
}

type XimoAppDownloadRelease struct {
	ID                      string `json:"id,omitempty"`
	AppKey                  string `json:"app_key,omitempty"`
	Channel                 string `json:"channel"`
	OS                      string `json:"os"`
	Arch                    string `json:"arch"`
	Locale                  string `json:"locale,omitempty"`
	PackageType             string `json:"package_type,omitempty"`
	Version                 string `json:"version"`
	VersionCode             int64  `json:"version_code,omitempty"`
	MinSupportedVersion     string `json:"min_supported_version,omitempty"`
	MinSupportedVersionCode int64  `json:"min_supported_version_code,omitempty"`
	DownloadURL             string `json:"download_url"`
	Notes                   string `json:"notes"`
	SHA256                  string `json:"sha256"`
	Force                   bool   `json:"force"`
	FileName                string `json:"file_name,omitempty"`
	FileSize                int64  `json:"file_size,omitempty"`
	UploadedAt              string `json:"uploaded_at,omitempty"`
	PublishedAt             string `json:"published_at,omitempty"`
}

func DefaultXimoDeskUpdateConfig() *XimoDeskUpdateConfig {
	return &XimoDeskUpdateConfig{
		Enabled:  false,
		Apps:     defaultXimoAppUpdateApps(),
		Releases: []XimoDeskUpdateRelease{},
	}
}

func defaultXimoAppUpdateApps() []XimoAppUpdateApp {
	enabled := true
	return []XimoAppUpdateApp{
		{
			Key:          defaultXimoAppKeyXimoDesk,
			Name:         "XimoDesk",
			ClientType:   "desktop",
			ResponseMode: "ximodesk",
			Enabled:      ximoDeskBoolPtr(enabled),
		},
		{
			Key:          defaultXimoAppKeyMobile,
			Name:         "Ximo Mobile",
			ClientType:   "mobile",
			ResponseMode: "standard",
			Enabled:      ximoDeskBoolPtr(enabled),
		},
	}
}

func (s *SettingService) GetXimoDeskUpdateConfig(ctx context.Context) (*XimoDeskUpdateConfig, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultXimoDeskUpdateConfig(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyXimoAppUpdateConfig)
	if err != nil {
		if !errors.Is(err, ErrSettingNotFound) {
			return nil, fmt.Errorf("get XimoAPP update config: %w", err)
		}
		raw, err = s.settingRepo.GetValue(ctx, SettingKeyXimoDeskUpdateConfig)
		if err != nil {
			if errors.Is(err, ErrSettingNotFound) {
				return DefaultXimoDeskUpdateConfig(), nil
			}
			return nil, fmt.Errorf("get legacy XimoDesk update config: %w", err)
		}
	}
	if strings.TrimSpace(raw) == "" {
		return DefaultXimoDeskUpdateConfig(), nil
	}
	var cfg XimoDeskUpdateConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultXimoDeskUpdateConfig(), nil
	}
	normalizeXimoDeskUpdateConfig(&cfg)
	return &cfg, nil
}

func (s *SettingService) SaveXimoDeskUpdateConfig(ctx context.Context, cfg *XimoDeskUpdateConfig) error {
	if s == nil || s.settingRepo == nil {
		return fmt.Errorf("setting service is not configured")
	}
	if cfg == nil {
		return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_CONFIG", "config is required")
	}
	normalizeXimoDeskUpdateConfig(cfg)
	if err := validateXimoDeskUpdateConfig(cfg); err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal XimoAPP update config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyXimoAppUpdateConfig, string(data)); err != nil {
		return fmt.Errorf("save XimoAPP update config: %w", err)
	}
	return nil
}

func (s *SettingService) GetXimoAppDownloadCenter(ctx context.Context) (*XimoAppDownloadCenter, error) {
	cfg, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, err
	}
	out := &XimoAppDownloadCenter{Apps: []XimoAppDownloadApp{}}
	if cfg == nil || !cfg.Enabled {
		return out, nil
	}

	appsByKey := map[string]XimoAppUpdateApp{}
	for _, app := range cfg.Apps {
		if app.Key == "" || (app.Enabled != nil && !*app.Enabled) {
			continue
		}
		appsByKey[app.Key] = app
	}

	releasesByApp := map[string][]XimoAppDownloadRelease{}
	for _, release := range cfg.Releases {
		appKey := normalizeXimoAppKey(release.AppKey)
		if appKey == "" || !isXimoDeskReleaseEnabled(release) {
			continue
		}
		if _, ok := appsByKey[appKey]; !ok {
			continue
		}
		releasesByApp[appKey] = append(releasesByApp[appKey], ximoAppDownloadReleaseFromUpdate(release))
	}

	appKeys := make([]string, 0, len(releasesByApp))
	for appKey := range releasesByApp {
		appKeys = append(appKeys, appKey)
	}
	sort.Slice(appKeys, func(i, j int) bool {
		left := appsByKey[appKeys[i]]
		right := appsByKey[appKeys[j]]
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Key < right.Key
	})

	for _, appKey := range appKeys {
		releases := releasesByApp[appKey]
		sort.Slice(releases, func(i, j int) bool {
			cmp := compareXimoDeskVersions(releases[i].Version, releases[j].Version)
			if cmp != 0 {
				return cmp > 0
			}
			if releases[i].VersionCode != releases[j].VersionCode {
				return releases[i].VersionCode > releases[j].VersionCode
			}
			return releases[i].PublishedAt > releases[j].PublishedAt
		})
		app := appsByKey[appKey]
		out.Apps = append(out.Apps, XimoAppDownloadApp{
			Key:         app.Key,
			Name:        app.Name,
			Description: app.Description,
			ClientType:  app.ClientType,
			Releases:    releases,
		})
	}

	return out, nil
}

func (s *SettingService) ResolveXimoAppUpdate(ctx context.Context, appKey string, req XimoDeskVersionRequest) (*XimoAppUpdateResponse, bool, error) {
	appKey = normalizeXimoAppKey(firstNonEmpty(appKey, req.AppKey))
	if appKey == "" {
		if isSupportedXimoDeskUpdateClient(req) {
			appKey = defaultXimoAppKeyXimoDesk
		} else {
			return nil, false, nil
		}
	}
	cfg, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, false, err
	}
	if cfg == nil || !cfg.Enabled || !isXimoAppEnabled(cfg.Apps, appKey) {
		return nil, false, nil
	}
	channel := normalizeXimoDeskChannel(req.Channel)
	osName := normalizeXimoDeskOS(firstNonEmpty(req.OS, req.Platform))
	arch := normalizeXimoDeskArch(req.Arch)
	locale := normalizeXimoDeskLocale(req.Locale)
	if locale == "all" && strings.TrimSpace(req.Language) != "" {
		locale = normalizeXimoDeskLocale(req.Language)
	}
	bestRank := 100
	var best *XimoDeskUpdateRelease
	for _, release := range cfg.Releases {
		rank, localeMatches := matchXimoDeskLocale(release.Locale, locale)
		if !isXimoDeskReleaseEnabled(release) ||
			normalizeXimoAppKey(release.AppKey) != appKey ||
			!localeMatches ||
			release.Channel != channel ||
			release.OS != osName ||
			!matchXimoDeskArch(release.Arch, arch) {
			continue
		}
		current := release
		if best == nil ||
			rank < bestRank ||
			(rank == bestRank && compareXimoDeskVersions(current.Version, best.Version) > 0) ||
			(rank == bestRank && compareXimoDeskVersions(current.Version, best.Version) == 0 && current.VersionCode > best.VersionCode) ||
			(rank == bestRank && compareXimoDeskVersions(current.Version, best.Version) == 0 && current.VersionCode == best.VersionCode && current.PublishedAt > best.PublishedAt) {
			best = &current
			bestRank = rank
		}
	}
	if best == nil {
		return nil, false, nil
	}
	return best.XimoAppUpdateResponse(), true, nil
}

func (r XimoDeskUpdateRelease) XimoAppUpdateResponse() *XimoAppUpdateResponse {
	return &XimoAppUpdateResponse{
		AppKey:                  r.AppKey,
		Version:                 r.Version,
		VersionCode:             r.VersionCode,
		MinSupportedVersion:     r.MinSupportedVersion,
		MinSupportedVersionCode: r.MinSupportedVersionCode,
		DownloadURL:             r.DownloadURL,
		Notes:                   r.Notes,
		SHA256:                  r.SHA256,
		Force:                   r.Force,
		PackageType:             r.PackageType,
		FileSize:                r.FileSize,
		PublishedAt:             r.PublishedAt,
	}
}

func ximoAppDownloadReleaseFromUpdate(r XimoDeskUpdateRelease) XimoAppDownloadRelease {
	return XimoAppDownloadRelease{
		ID:                      r.ID,
		AppKey:                  r.AppKey,
		Channel:                 r.Channel,
		OS:                      r.OS,
		Arch:                    r.Arch,
		Locale:                  r.Locale,
		PackageType:             r.PackageType,
		Version:                 r.Version,
		VersionCode:             r.VersionCode,
		MinSupportedVersion:     r.MinSupportedVersion,
		MinSupportedVersionCode: r.MinSupportedVersionCode,
		DownloadURL:             r.DownloadURL,
		Notes:                   r.Notes,
		SHA256:                  r.SHA256,
		Force:                   r.Force,
		FileName:                r.FileName,
		FileSize:                r.FileSize,
		UploadedAt:              r.UploadedAt,
		PublishedAt:             r.PublishedAt,
	}
}

func validateXimoDeskUpdateConfig(cfg *XimoDeskUpdateConfig) error {
	if cfg == nil {
		return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_CONFIG", "config is required")
	}
	for i, app := range cfg.Apps {
		if !isAllowedXimoAppKey(app.Key) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_APP", fmt.Sprintf("app %d key is not supported", i+1))
		}
	}
	for i, release := range cfg.Releases {
		if !isAllowedXimoAppKey(release.AppKey) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d app_key is not supported", i+1))
		}
		if release.Version == "" {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d version is required", i+1))
		}
		if release.DownloadURL == "" {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d download_url is required", i+1))
		}
		if err := validateXimoDeskDownloadURL(release.DownloadURL); err != nil {
			return err
		}
		if !ximoDeskSHA256Pattern.MatchString(release.SHA256) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d sha256 must be a 64-character hexadecimal string", i+1))
		}
		if !isAllowedXimoDeskChannel(release.Channel) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d channel is not supported", i+1))
		}
		if !isAllowedXimoDeskOS(release.OS) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d os is not supported", i+1))
		}
		if !isAllowedXimoDeskArch(release.Arch) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d arch is not supported", i+1))
		}
		if !isAllowedXimoDeskLocale(release.Locale) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d locale is not supported", i+1))
		}
		if release.PackageType != "" && !isAllowedXimoDeskPackageType(release.PackageType) {
			return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", fmt.Sprintf("release %d package_type is not supported", i+1))
		}
	}
	return nil
}

func validateXimoDeskDownloadURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", "download_url must be a valid URL")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", "download_url must use https")
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "ximoai.cn", "www.ximoai.cn":
		return nil
	default:
		return infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_RELEASE", "download_url host is not allowed")
	}
}

func normalizeXimoDeskUpdateConfig(cfg *XimoDeskUpdateConfig) {
	if cfg == nil {
		return
	}
	cfg.Apps = normalizeXimoAppUpdateApps(cfg.Apps)
	if cfg.Releases == nil {
		cfg.Releases = []XimoDeskUpdateRelease{}
	}
	for i := range cfg.Releases {
		if cfg.Releases[i].Enabled == nil {
			cfg.Releases[i].Enabled = ximoDeskBoolPtr(true)
		}
		cfg.Releases[i].AppKey = normalizeXimoAppKey(cfg.Releases[i].AppKey)
		if cfg.Releases[i].AppKey == "" {
			cfg.Releases[i].AppKey = defaultXimoAppKeyXimoDesk
		}
		cfg.Releases[i].Channel = normalizeXimoDeskChannel(cfg.Releases[i].Channel)
		cfg.Releases[i].OS = normalizeXimoDeskOS(cfg.Releases[i].OS)
		cfg.Releases[i].Arch = normalizeXimoDeskArch(cfg.Releases[i].Arch)
		cfg.Releases[i].Locale = normalizeXimoDeskLocale(cfg.Releases[i].Locale)
		cfg.Releases[i].PackageType = normalizeXimoDeskPackageType(cfg.Releases[i].PackageType)
		cfg.Releases[i].Version = strings.TrimSpace(cfg.Releases[i].Version)
		cfg.Releases[i].MinSupportedVersion = strings.TrimSpace(cfg.Releases[i].MinSupportedVersion)
		cfg.Releases[i].DownloadURL = strings.TrimSpace(cfg.Releases[i].DownloadURL)
		cfg.Releases[i].Notes = strings.TrimSpace(cfg.Releases[i].Notes)
		cfg.Releases[i].SHA256 = strings.ToLower(strings.TrimSpace(cfg.Releases[i].SHA256))
		cfg.Releases[i].FileName = sanitizeXimoDeskFileName(strings.TrimSpace(cfg.Releases[i].FileName))
		cfg.Releases[i].UploadedAt = strings.TrimSpace(cfg.Releases[i].UploadedAt)
		cfg.Releases[i].PublishedAt = strings.TrimSpace(cfg.Releases[i].PublishedAt)
		cfg.Releases[i].ID = normalizeXimoDeskReleaseID(cfg.Releases[i])
		cfg.Apps = ensureXimoAppInList(cfg.Apps, cfg.Releases[i].AppKey)
	}
}

func normalizeXimoAppUpdateApps(apps []XimoAppUpdateApp) []XimoAppUpdateApp {
	orderedKeys := []string{}
	byKey := map[string]XimoAppUpdateApp{}
	for _, app := range defaultXimoAppUpdateApps() {
		orderedKeys = append(orderedKeys, app.Key)
		byKey[app.Key] = app
	}
	for _, app := range apps {
		app.Key = normalizeXimoAppKey(app.Key)
		if app.Key == "" {
			continue
		}
		if strings.TrimSpace(app.Name) == "" {
			app.Name = defaultXimoAppName(app.Key)
		}
		app.Name = strings.TrimSpace(app.Name)
		app.Description = strings.TrimSpace(app.Description)
		app.ClientType = strings.TrimSpace(app.ClientType)
		app.ResponseMode = strings.TrimSpace(app.ResponseMode)
		if app.Enabled == nil {
			app.Enabled = ximoDeskBoolPtr(true)
		}
		if _, exists := byKey[app.Key]; !exists {
			orderedKeys = append(orderedKeys, app.Key)
		}
		byKey[app.Key] = app
	}
	out := make([]XimoAppUpdateApp, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		out = append(out, byKey[key])
	}
	return out
}

func ensureXimoAppInList(apps []XimoAppUpdateApp, appKey string) []XimoAppUpdateApp {
	appKey = normalizeXimoAppKey(appKey)
	if appKey == "" {
		return apps
	}
	for _, app := range apps {
		if app.Key == appKey {
			return apps
		}
	}
	return append(apps, XimoAppUpdateApp{
		Key:          appKey,
		Name:         defaultXimoAppName(appKey),
		ClientType:   "custom",
		ResponseMode: "standard",
		Enabled:      ximoDeskBoolPtr(true),
	})
}

func isSupportedXimoDeskUpdateClient(req XimoDeskVersionRequest) bool {
	return strings.EqualFold(strings.TrimSpace(req.App), "XimoDesk") &&
		strings.EqualFold(strings.TrimSpace(req.Type), "ximodesk-client")
}

func isXimoAppEnabled(apps []XimoAppUpdateApp, appKey string) bool {
	appKey = normalizeXimoAppKey(appKey)
	for _, app := range apps {
		if app.Key == appKey {
			return app.Enabled == nil || *app.Enabled
		}
	}
	return false
}

func normalizeXimoAppKey(appKey string) string {
	appKey = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(appKey, "_", "-")))
	appKey = strings.Trim(appKey, "-.")
	return appKey
}

func isAllowedXimoAppKey(appKey string) bool {
	return ximoAppKeyPattern.MatchString(normalizeXimoAppKey(appKey))
}

func defaultXimoAppName(appKey string) string {
	switch normalizeXimoAppKey(appKey) {
	case defaultXimoAppKeyXimoDesk:
		return "XimoDesk"
	case defaultXimoAppKeyMobile:
		return "Ximo Mobile"
	default:
		parts := strings.Split(normalizeXimoAppKey(appKey), "-")
		for i, part := range parts {
			if part == "" {
				continue
			}
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
		return strings.Join(parts, " ")
	}
}

func normalizeXimoDeskChannel(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		return "stable"
	}
	return channel
}

func normalizeXimoDeskOS(osName string) string {
	switch strings.ToLower(strings.TrimSpace(osName)) {
	case "win", "win32", "windows":
		return "windows"
	case "darwin", "mac", "macos":
		return "macos"
	case "android":
		return "android"
	case "ios", "iphone", "ipad":
		return "ios"
	case "linux":
		return "linux"
	default:
		return strings.ToLower(strings.TrimSpace(osName))
	}
}

func normalizeXimoDeskArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "":
		return "universal"
	case "amd64", "x64", "x86_64":
		return "x86_64"
	case "arm64", "aarch64":
		return "aarch64"
	case "universal", "all", "any":
		return "universal"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

func normalizeXimoDeskLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(locale, "_", "-")))
	if locale == "" {
		return "all"
	}
	return locale
}

func normalizeXimoDeskPackageType(packageType string) string {
	packageType = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(packageType, ".")))
	return packageType
}

func normalizeXimoDeskReleaseID(release XimoDeskUpdateRelease) string {
	if strings.TrimSpace(release.ID) != "" {
		return sanitizeXimoDeskID(release.ID)
	}
	parts := []string{
		release.AppKey,
		release.Channel,
		release.OS,
		release.Arch,
		release.Locale,
		release.PackageType,
		release.Version,
	}
	return sanitizeXimoDeskID(strings.Join(parts, "-"))
}

func isAllowedXimoDeskChannel(channel string) bool {
	switch channel {
	case "stable", "beta":
		return true
	default:
		return false
	}
}

func isAllowedXimoDeskOS(osName string) bool {
	switch osName {
	case "windows", "macos", "linux", "android", "ios":
		return true
	default:
		return false
	}
}

func isAllowedXimoDeskArch(arch string) bool {
	switch arch {
	case "x86_64", "aarch64", "universal":
		return true
	default:
		return false
	}
}

func isAllowedXimoDeskLocale(locale string) bool {
	locale = normalizeXimoDeskLocale(locale)
	return locale == "all" || ximoDeskLocalePattern.MatchString(locale)
}

func isAllowedXimoDeskPackageType(packageType string) bool {
	switch normalizeXimoDeskPackageType(packageType) {
	case "msi", "zip", "exe", "dmg", "pkg", "apk", "aab", "ipa":
		return true
	default:
		return false
	}
}

func isXimoDeskReleaseEnabled(release XimoDeskUpdateRelease) bool {
	return release.Enabled == nil || *release.Enabled
}

func matchXimoDeskArch(releaseArch, requestArch string) bool {
	releaseArch = normalizeXimoDeskArch(releaseArch)
	requestArch = normalizeXimoDeskArch(requestArch)
	return releaseArch == requestArch || releaseArch == "universal"
}

func matchXimoDeskLocale(releaseLocale, requestLocale string) (int, bool) {
	releaseLocale = normalizeXimoDeskLocale(releaseLocale)
	requestLocale = normalizeXimoDeskLocale(requestLocale)
	if releaseLocale == requestLocale {
		return 0, true
	}
	if releaseLocale != "all" {
		releaseBase := strings.SplitN(releaseLocale, "-", 2)[0]
		requestBase := strings.SplitN(requestLocale, "-", 2)[0]
		if releaseBase == requestBase {
			return 1, true
		}
	}
	if releaseLocale == "all" {
		return 2, true
	}
	return 0, false
}

func compareXimoDeskVersions(a, b string) int {
	aParts := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	bParts := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")
	maxParts := len(aParts)
	if len(bParts) > maxParts {
		maxParts = len(bParts)
	}
	for i := 0; i < maxParts; i++ {
		aValue, bValue := 0, 0
		if i < len(aParts) {
			aValue = leadingInt(aParts[i])
		}
		if i < len(bParts) {
			bValue = leadingInt(bParts[i])
		}
		if aValue > bValue {
			return 1
		}
		if aValue < bValue {
			return -1
		}
	}
	return strings.Compare(a, b)
}

func leadingInt(value string) int {
	digits := strings.Builder{}
	for _, ch := range strings.TrimSpace(value) {
		if ch < '0' || ch > '9' {
			break
		}
		digits.WriteRune(ch)
	}
	if digits.Len() == 0 {
		return 0
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0
	}
	return n
}

func ximoDeskBoolPtr(value bool) *bool {
	return &value
}
