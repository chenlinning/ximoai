package migrations

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXimoAIMembershipCleanupMigration(t *testing.T) {
	sql, err := os.ReadFile("920_ximoai_membership_cleanup.sql")
	require.NoError(t, err)
	content := string(sql)

	require.Contains(t, content, "UPDATE user_memberships")
	require.Contains(t, content, "DELETE FROM membership_levels")
	require.Contains(t, content, "WHERE code = 'bronze'")
	require.Contains(t, content, "is_default = TRUE")
	require.Contains(t, content, "UPDATE api_keys ak")
	require.Contains(t, content, "DELETE FROM membership_managed_keys")
	require.Contains(t, content, "Membership Key - ")
}
