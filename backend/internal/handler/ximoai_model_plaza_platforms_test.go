package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestXimoAIModelPlazaPlatformForUsesTheUpstreamPlatformCatalog(t *testing.T) {
	for _, spec := range domain.Platforms() {
		platform, ok := ximoAIModelPlazaPlatformFor(spec.ID)
		require.Truef(t, ok, "upstream platform %q must be available to model plaza", spec.ID)
		require.Equal(t, spec.DisplayName, platform.DisplayName)
		require.NotEmpty(t, platform.Protocol)
	}
}

func TestXimoAIModelPlazaPlatformForRejectsMetadataAliasesAsPlatforms(t *testing.T) {
	for _, slug := range []string{"moonshot", "doubao", "qwen", "compatible-alias", "composite", ""} {
		_, ok := ximoAIModelPlazaPlatformFor(slug)
		require.Falsef(t, ok, "%q is metadata or grouping data, not a concrete platform", slug)
	}
}
