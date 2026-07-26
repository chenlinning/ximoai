package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration918BackfillsLegacyRuntimeWithoutOverwritingGroupPermission(t *testing.T) {
	content, err := FS.ReadFile("918_ximoai_backfill_volcengine_agent_plan_runtime.sql")
	require.NoError(t, err)

	sql := strings.ToLower(string(content))
	require.Contains(t, sql, "platform.kind = 'volcengine_agent_plan'")
	require.Contains(t, sql, "platform.protocol = 'native'")
	require.Contains(t, sql, "https://ark.cn-beijing.volces.com/api/plan/v3")
	require.NotContains(t, sql, "set allow_image_generation")
}
