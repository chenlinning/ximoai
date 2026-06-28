package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ximoAppPackageDirEnv          = "XIMOAPP_PACKAGE_DIR"
	ximoAppDownloadBaseURLEnv     = "XIMOAPP_DOWNLOAD_BASE_URL"
	defaultXimoAppDownloadBaseURL = "https://ximoai.cn/downloads/ximoapp"
)

var ximoDeskSafeNamePattern = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type XimoDeskPackageUpload struct {
	AppKey                  string
	Channel                 string
	OS                      string
	Arch                    string
	Locale                  string
	Version                 string
	MinSupportedVersion     string
	MinSupportedVersionCode int64
	PackageType             string
	OriginalName            string
	Notes                   string
	Force                   bool
	Enabled                 bool
	Reader                  io.Reader
}

func (s *SettingService) SaveXimoDeskPackageRelease(ctx context.Context, upload XimoDeskPackageUpload) (*XimoDeskUpdateRelease, *XimoDeskUpdateConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, nil, fmt.Errorf("setting service is not configured")
	}
	if upload.Reader == nil {
		return nil, nil, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "package file is required")
	}
	release, err := buildXimoDeskReleaseFromUpload(upload)
	if err != nil {
		return nil, nil, err
	}

	dir := XimoDeskPackageDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create XimoAPP package dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create XimoAPP package temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), upload.Reader)
	closeErr := tmp.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("write XimoAPP package: %w", err)
	}
	if closeErr != nil {
		return nil, nil, fmt.Errorf("close XimoAPP package temp file: %w", closeErr)
	}
	if written <= 0 {
		return nil, nil, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "package file is empty")
	}

	uploadedAt := time.Now().UTC()
	release.FileSize = written
	release.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	release.DownloadURL = XimoDeskPackageDownloadURL(release.FileName)
	release.UploadedAt = uploadedAt.Format(time.RFC3339)
	release.PublishedAt = release.UploadedAt
	if release.VersionCode == 0 {
		release.VersionCode = ximoAppVersionCodeFromTime(uploadedAt)
	}

	dest, err := XimoDeskPackagePath(release.FileName)
	if err != nil {
		return nil, nil, err
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmpName, dest); err != nil {
		return nil, nil, fmt.Errorf("store XimoAPP package: %w", err)
	}

	cfg, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	if cfg == nil {
		cfg = DefaultXimoDeskUpdateConfig()
	}
	cfg.Enabled = true
	cfg.Releases = upsertXimoDeskRelease(cfg.Releases, release)
	if err := s.SaveXimoDeskUpdateConfig(ctx, cfg); err != nil {
		return nil, nil, err
	}
	saved, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, savedRelease := range saved.Releases {
		if savedRelease.ID == release.ID {
			current := savedRelease
			return &current, saved, nil
		}
	}
	return &release, saved, nil
}

func (s *SettingService) DeleteXimoDeskUpdateRelease(ctx context.Context, id string) (*XimoDeskUpdateConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("setting service is not configured")
	}
	id = sanitizeXimoDeskID(id)
	if id == "" {
		return nil, infraerrors.BadRequest("INVALID_XIMODESK_RELEASE", "release id is required")
	}
	cfg, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = DefaultXimoDeskUpdateConfig()
	}
	next := cfg.Releases[:0]
	var removed *XimoDeskUpdateRelease
	for _, release := range cfg.Releases {
		if release.ID == id {
			r := release
			removed = &r
			continue
		}
		next = append(next, release)
	}
	if removed == nil {
		return nil, infraerrors.BadRequest("INVALID_XIMODESK_RELEASE", "release not found")
	}
	cfg.Releases = next
	if removed.FileName != "" {
		fileName := sanitizeXimoDeskFileName(removed.FileName)
		if fileName != "" {
			_ = os.Remove(filepath.Join(XimoDeskPackageDir(), fileName))
		}
	}
	if err := s.SaveXimoDeskUpdateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return s.GetXimoDeskUpdateConfig(ctx)
}

func (s *SettingService) DeleteXimoAppUpdateApp(ctx context.Context, appKey string) (*XimoDeskUpdateConfig, error) {
	if s == nil || s.settingRepo == nil {
		return nil, fmt.Errorf("setting service is not configured")
	}
	appKey = normalizeXimoAppKey(appKey)
	if appKey == "" {
		return nil, infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_APP", "app key is required")
	}
	cfg, err := s.GetXimoDeskUpdateConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = DefaultXimoDeskUpdateConfig()
	}

	nextApps := cfg.Apps[:0]
	removed := false
	for _, app := range cfg.Apps {
		if normalizeXimoAppKey(app.Key) == appKey {
			removed = true
			continue
		}
		nextApps = append(nextApps, app)
	}
	if !removed {
		return nil, infraerrors.BadRequest("INVALID_XIMOAPP_UPDATE_APP", "app not found")
	}

	nextReleases := cfg.Releases[:0]
	for _, release := range cfg.Releases {
		if normalizeXimoAppKey(release.AppKey) == appKey {
			if fileName := sanitizeXimoDeskFileName(release.FileName); fileName != "" {
				_ = os.Remove(filepath.Join(XimoDeskPackageDir(), fileName))
			}
			continue
		}
		nextReleases = append(nextReleases, release)
	}
	cfg.Apps = nextApps
	cfg.Releases = nextReleases
	if err := s.SaveXimoDeskUpdateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return s.GetXimoDeskUpdateConfig(ctx)
}

func XimoDeskPackageDir() string {
	if dir := strings.TrimSpace(os.Getenv(ximoAppPackageDirEnv)); dir != "" {
		return dir
	}
	if dataDir := strings.TrimSpace(os.Getenv("DATA_DIR")); dataDir != "" {
		return filepath.Join(dataDir, "downloads", "ximoapp")
	}
	return filepath.Join("data", "downloads", "ximoapp")
}

func XimoDeskPackagePath(fileName string) (string, error) {
	fileName = sanitizeXimoDeskFileName(fileName)
	if fileName == "" {
		return "", infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "file name is required")
	}
	return filepath.Join(XimoDeskPackageDir(), fileName), nil
}

func XimoDeskPackageLookupPath(fileName string) (string, error) {
	fileName = sanitizeXimoDeskFileName(fileName)
	if fileName == "" {
		return "", infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "file name is required")
	}
	return filepath.Join(XimoDeskPackageDir(), fileName), nil
}

func XimoDeskPackageDownloadURL(fileName string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(ximoAppDownloadBaseURLEnv)), "/")
	if baseURL == "" {
		baseURL = defaultXimoAppDownloadBaseURL
	}
	return baseURL + "/" + url.PathEscape(sanitizeXimoDeskFileName(fileName))
}

func buildXimoDeskReleaseFromUpload(upload XimoDeskPackageUpload) (XimoDeskUpdateRelease, error) {
	packageType := normalizeXimoDeskUploadPackageType(upload.PackageType, upload.OriginalName)
	if !isAllowedXimoDeskPackageType(packageType) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "package type is not supported")
	}
	release := XimoDeskUpdateRelease{
		AppKey:                  normalizeXimoAppKey(upload.AppKey),
		Enabled:                 ximoDeskBoolPtr(upload.Enabled),
		Channel:                 normalizeXimoDeskChannel(upload.Channel),
		OS:                      normalizeXimoDeskOS(upload.OS),
		Arch:                    normalizeXimoDeskArch(upload.Arch),
		Locale:                  normalizeXimoDeskLocale(upload.Locale),
		PackageType:             packageType,
		Version:                 strings.TrimSpace(upload.Version),
		MinSupportedVersion:     strings.TrimSpace(upload.MinSupportedVersion),
		MinSupportedVersionCode: upload.MinSupportedVersionCode,
		Notes:                   strings.TrimSpace(upload.Notes),
		Force:                   upload.Force,
	}
	if release.AppKey == "" {
		release.AppKey = defaultXimoAppKeyXimoDesk
	}
	if !isAllowedXimoAppKey(release.AppKey) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "app key is not supported")
	}
	if release.Version == "" {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "version is required")
	}
	if !isAllowedXimoDeskChannel(release.Channel) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "channel is not supported")
	}
	if !isAllowedXimoDeskOS(release.OS) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "os is not supported")
	}
	if !isAllowedXimoDeskArch(release.Arch) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "arch is not supported")
	}
	if !isAllowedXimoDeskLocale(release.Locale) {
		return XimoDeskUpdateRelease{}, infraerrors.BadRequest("INVALID_XIMOAPP_PACKAGE", "locale is not supported")
	}
	release.FileName = buildXimoDeskPackageFileName(release)
	release.ID = normalizeXimoDeskReleaseID(release)
	return release, nil
}

func normalizeXimoDeskUploadPackageType(packageType, originalName string) string {
	packageType = normalizeXimoDeskPackageType(packageType)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(originalName)), ".")
	if packageType == "" {
		return ext
	}
	if ext != "" && ext != packageType {
		return ""
	}
	return packageType
}

func buildXimoDeskPackageFileName(release XimoDeskUpdateRelease) string {
	name := fmt.Sprintf("XimoAPP-%s-%s-%s-%s-%s.%s", release.AppKey, release.Version, release.OS, release.Arch, release.Locale, release.PackageType)
	return sanitizeXimoDeskFileName(name)
}

func ximoAppVersionCodeFromTime(value time.Time) int64 {
	code, err := strconv.ParseInt(value.UTC().Format("20060102150405"), 10, 64)
	if err != nil {
		return value.UTC().Unix()
	}
	return code
}

func upsertXimoDeskRelease(releases []XimoDeskUpdateRelease, release XimoDeskUpdateRelease) []XimoDeskUpdateRelease {
	out := make([]XimoDeskUpdateRelease, 0, len(releases)+1)
	replaced := false
	for _, existing := range releases {
		sameSlot := existing.ID == release.ID ||
			(normalizeXimoAppKey(existing.AppKey) == normalizeXimoAppKey(release.AppKey) &&
				existing.Channel == release.Channel &&
				existing.OS == release.OS &&
				existing.Arch == release.Arch &&
				existing.Locale == release.Locale &&
				existing.PackageType == release.PackageType &&
				existing.Version == release.Version)
		if sameSlot {
			if existing.FileName != "" && existing.FileName != release.FileName {
				if filePath, err := XimoDeskPackagePath(existing.FileName); err == nil {
					_ = os.Remove(filePath)
				}
			}
			out = append(out, release)
			replaced = true
			continue
		}
		out = append(out, existing)
	}
	if !replaced {
		out = append(out, release)
	}
	return out
}

func sanitizeXimoDeskID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = ximoDeskSafeNamePattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-.")
}

func sanitizeXimoDeskFileName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "." || value == string(filepath.Separator) {
		return ""
	}
	value = ximoDeskSafeNamePattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-.")
}
