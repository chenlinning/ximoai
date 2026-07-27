package modelcatalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRegistryIsAuditedAndLoads(t *testing.T) {
	registry, err := RegistryForTest()
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Models) == 0 || len(registry.Profiles) == 0 {
		t.Fatal("model registry is empty")
	}
	for _, record := range registry.Models {
		if !isVerifiedRecord(record) {
			t.Fatalf("record %s/%s is not fully audited", record.Platform, record.Model)
		}
	}
}

func TestLookupCustomPlatformUsesUniqueUpstreamModelID(t *testing.T) {
	record, ok := Lookup("custom-openai-compatible", "gpt-5")
	if !ok {
		t.Fatal("custom platform did not match gpt-5")
	}
	if record.Platform != "openai" || record.Brand != "OpenAI" {
		t.Fatalf("unexpected match: %#v", record)
	}
}

func TestLookupBuiltInPlatformDoesNotCrossMatchAnotherProvider(t *testing.T) {
	_, ok := Lookup("anthropic", "gpt-5")
	if ok {
		t.Fatal("built-in platform cross-matched gpt-5")
	}
}

func TestVolcengineAgentPlanRegistryIncludesProviderModelIDs(t *testing.T) {
	tests := []struct {
		model     string
		modelType string
		mode      string
	}{
		{model: "doubao-seed-tts-2.0", modelType: "tts", mode: "bidirectional"},
		{model: "doubao-seed-asr-2.0", modelType: "asr", mode: "stream"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			record, ok := Lookup("volcengine-agent-plan", tt.model)
			if !ok {
				t.Fatalf("model registry did not match %s", tt.model)
			}
			if record.Brand != "Doubao" || !containsString(record.Types, tt.modelType) || !containsString(record.InvocationModes, tt.mode) {
				t.Fatalf("unexpected metadata for %s: %#v", tt.model, record)
			}
		})
	}
}

func TestPublicMetadataDoesNotExposeInternalAccessProfile(t *testing.T) {
	metadata, ok := PublicMetadataFor("custom-openai-compatible", "gpt-5")
	if !ok {
		t.Fatal("custom platform did not match gpt-5")
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	serialized := strings.ToLower(string(raw))
	for _, forbidden := range []string{"endpoint", "protocol", "field", "upstream"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("public metadata contains %q: %s", forbidden, serialized)
		}
	}
}

func TestPublicMetadataSeparatesReasoningLevelsFromThinkingToggle(t *testing.T) {
	metadata, ok := PublicMetadataFor("custom-openai-compatible", "kimi-k2.6")
	if !ok {
		t.Fatal("custom platform did not match kimi-k2.6")
	}
	if len(metadata.ReasoningLevels) != 0 || !metadata.ThinkingSupported {
		t.Fatalf("kimi-k2.6 should expose only thinking support: %#v", metadata)
	}

	metadata, ok = PublicMetadataFor("custom-openai-compatible", "kimi-k3")
	if !ok {
		t.Fatal("custom platform did not match kimi-k3")
	}
	if len(metadata.ReasoningLevels) != 3 || metadata.ThinkingSupported {
		t.Fatalf("kimi-k3 should expose reasoning levels only: %#v", metadata)
	}
}
