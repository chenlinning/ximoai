package modelcatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed model-registry.json
var registryBytes []byte

type Registry struct {
	SchemaVersion int                      `json:"schema_version"`
	CheckedAt     string                   `json:"checked_at"`
	Profiles      map[string]AccessProfile `json:"profiles"`
	Models        []ModelRecord            `json:"models"`
}

type ModelRecord struct {
	Platform        string   `json:"platform"`
	Model           string   `json:"model"`
	Brand           string   `json:"brand"`
	Types           []string `json:"types"`
	InvocationModes []string `json:"invocation_modes"`
	ProfileIDs      []string `json:"profile_ids"`
	Audit           Audit    `json:"audit"`
}

type Audit struct {
	Status      string `json:"status"`
	ModelSource string `json:"model_source"`
	Protocol    string `json:"protocol"`
	Reasoning   string `json:"reasoning"`
	VerifiedAt  string `json:"verified_at"`
}

// AccessProfile is internal metadata. It is deliberately not part of any
// public model-plaza DTO.
type AccessProfile struct {
	Protocol  string     `json:"protocol"`
	Transport string     `json:"transport"`
	Endpoint  string     `json:"endpoint"`
	Mode      string     `json:"mode"`
	Reasoning *Reasoning `json:"reasoning,omitempty"`
}

type Reasoning struct {
	Kind             string   `json:"kind"`
	Field            string   `json:"field,omitempty"`
	Levels           []string `json:"levels,omitempty"`
	Default          string   `json:"default,omitempty"`
	DisableSupported bool     `json:"disable_supported"`
}

type PublicMetadata struct {
	Brand             string   `json:"brand"`
	Types             []string `json:"types"`
	InvocationModes   []string `json:"invocation_modes"`
	ReasoningLevels   []string `json:"reasoning_levels,omitempty"`
	ThinkingSupported bool     `json:"thinking_supported,omitempty"`
}

var builtInRegistryPlatforms = map[string]struct{}{
	"openai":      {},
	"anthropic":   {},
	"gemini":      {},
	"antigravity": {},
	"grok":        {},
}

var (
	loadOnce sync.Once
	registry Registry
	loadErr  error
)

func load() (Registry, error) {
	loadOnce.Do(func() {
		if err := json.Unmarshal(registryBytes, &registry); err != nil {
			loadErr = fmt.Errorf("decode model registry: %w", err)
			return
		}
		if err := validate(registry); err != nil {
			loadErr = err
		}
	})
	return registry, loadErr
}

func Lookup(platform, model string) (ModelRecord, bool) {
	registry, err := load()
	if err != nil {
		return ModelRecord{}, false
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	model = strings.ToLower(strings.TrimSpace(model))
	// Built-in platforms require a platform-qualified match. This prevents an
	// ordinary model ID from changing metadata across built-in protocol paths.
	for _, record := range registry.Models {
		if strings.ToLower(record.Platform) != platform || strings.ToLower(record.Model) != model {
			continue
		}
		if !isVerifiedRecord(record) {
			return ModelRecord{}, false
		}
		return cloneRecord(record), true
	}
	if _, ok := builtInRegistryPlatforms[platform]; ok {
		return ModelRecord{}, false
	}

	// Other fixed-compatible platform aliases may reuse a verified official
	// model by exact upstream ID. Ambiguous IDs are rejected instead of guessed.
	var match *ModelRecord
	for _, record := range registry.Models {
		if strings.ToLower(record.Model) != model || !isVerifiedRecord(record) {
			continue
		}
		if match != nil {
			return ModelRecord{}, false
		}
		candidate := cloneRecord(record)
		match = &candidate
	}
	if match != nil {
		return *match, true
	}
	return ModelRecord{}, false
}

func PublicMetadataFor(platform, model string) (PublicMetadata, bool) {
	registry, err := load()
	if err != nil {
		return PublicMetadata{}, false
	}
	record, ok := Lookup(platform, model)
	if !ok {
		return PublicMetadata{}, false
	}
	return PublicMetadata{
		Brand:             record.Brand,
		Types:             append([]string(nil), record.Types...),
		InvocationModes:   append([]string(nil), record.InvocationModes...),
		ReasoningLevels:   publicReasoningLevels(registry, record),
		ThinkingSupported: publicThinkingSupported(registry, record),
	}, true
}

func RegistryForTest() (Registry, error) {
	return load()
}

func validate(registry Registry) error {
	if registry.SchemaVersion != 1 {
		return fmt.Errorf("unsupported model registry schema version %d", registry.SchemaVersion)
	}
	if strings.TrimSpace(registry.CheckedAt) == "" {
		return fmt.Errorf("model registry checked_at is required")
	}
	seen := make(map[string]struct{}, len(registry.Models))
	for index, record := range registry.Models {
		platform := strings.ToLower(strings.TrimSpace(record.Platform))
		model := strings.ToLower(strings.TrimSpace(record.Model))
		if platform == "" || model == "" {
			return fmt.Errorf("model registry record %d has empty platform or model", index)
		}
		key := platform + "\x00" + model
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate model registry key %q", key)
		}
		seen[key] = struct{}{}
		if !isVerifiedRecord(record) {
			return fmt.Errorf("model registry record %q is not fully audited", key)
		}
		if record.Brand == "" || len(record.Types) == 0 || len(record.InvocationModes) == 0 {
			return fmt.Errorf("model registry record %q has incomplete public metadata", key)
		}
		for _, profileID := range record.ProfileIDs {
			if _, ok := registry.Profiles[profileID]; !ok {
				return fmt.Errorf("model registry record %q references missing profile %q", key, profileID)
			}
		}
	}
	return nil
}

func isVerifiedRecord(record ModelRecord) bool {
	if record.Audit.Status != "verified" ||
		strings.TrimSpace(record.Audit.ModelSource) == "" ||
		record.Audit.Protocol != "verified" ||
		record.Audit.VerifiedAt == "" {
		return false
	}
	switch record.Audit.Reasoning {
	case "verified", "not_applicable", "not_available":
		return true
	default:
		return false
	}
}

func cloneRecord(record ModelRecord) ModelRecord {
	record.Types = append([]string(nil), record.Types...)
	record.InvocationModes = append([]string(nil), record.InvocationModes...)
	record.ProfileIDs = append([]string(nil), record.ProfileIDs...)
	return record
}

func publicReasoningLevels(registry Registry, record ModelRecord) []string {
	levels := make([]string, 0)
	for _, profileID := range record.ProfileIDs {
		profile, ok := registry.Profiles[profileID]
		if !ok || profile.Reasoning == nil {
			continue
		}
		switch profile.Reasoning.Kind {
		case "effort", "level_or_budget", "effort_and_toggle":
		default:
			continue
		}
		for _, level := range profile.Reasoning.Levels {
			if !containsString(levels, level) {
				levels = append(levels, level)
			}
		}
	}
	return levels
}

func publicThinkingSupported(registry Registry, record ModelRecord) bool {
	for _, profileID := range record.ProfileIDs {
		profile, ok := registry.Profiles[profileID]
		if !ok || profile.Reasoning == nil {
			continue
		}
		switch profile.Reasoning.Kind {
		case "toggle", "effort_and_toggle":
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
