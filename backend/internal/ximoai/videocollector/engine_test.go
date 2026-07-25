package videocollector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindOutputFileUsesActualDirectoryEntry(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "output.mp4.part"), []byte("partial"), 0o600))
	expected := filepath.Join(directory, "output.mp4")
	require.NoError(t, os.WriteFile(expected, []byte("complete"), 0o600))

	require.Equal(t, expected, findOutputFile(directory))
	require.False(t, isRegularOutputFile(directory, filepath.Join(directory, "..", "outside.mp4")))
}
