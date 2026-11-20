package authsetup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenStateStoreRelativePath(t *testing.T) {
	dir := t.TempDir()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	t.Chdir(dir)

	defer func() {
		_ = os.Chdir(cwd)
	}()

	store, err := openStateStore(filepath.Join("subdir", "auth-state.db"))
	require.NoError(t, err)
	defer store.close()

	_, err = store.authVersion()
	require.NoError(t, err)
}

func TestOpenStateStoreAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth-state.db")

	store, err := openStateStore(path)
	require.NoError(t, err)
	defer store.close()

	_, err = store.authVersion()
	require.NoError(t, err)
}
