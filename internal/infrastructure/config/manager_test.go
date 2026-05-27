package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxy/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_LoadAndCurrent(t *testing.T) {
	isolateConfigEnv(t)

	path := writeConfigFile(t, "config.yaml", `
server:
  address: ":9000"
ip_access:
  default_policy: allow
`)
	manager := NewManager(path, 10*time.Millisecond, NewFileLoader(), &testutils.MockLogger{})

	require.NoError(t, manager.Load(context.Background()))

	cfg, ok := manager.Current()
	assert.True(t, ok)
	assert.Equal(t, ":9000", cfg.Server.Address)
	assert.Equal(t, int64(1), manager.Version())
	assert.False(t, manager.LoadedAt().IsZero())
}

func TestManager_StartReloadsChangedFile(t *testing.T) {
	isolateConfigEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  address: ":9001"
ip_access:
  default_policy: allow
`), 0o600))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(path, 10*time.Millisecond, NewFileLoader(), &testutils.MockLogger{})
	changes := make(chan Change, 4)
	unsubscribe := manager.Subscribe(func(change Change) {
		changes <- change
	})
	defer unsubscribe()

	require.NoError(t, manager.Start(ctx))

	initial := waitForChange(t, changes)
	assert.Equal(t, int64(1), initial.Version)
	assert.Equal(t, ":9001", initial.Current.Server.Address)

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  address: ":9002"
ip_access:
  default_policy: deny
`), 0o600))

	reloaded := waitForChange(t, changes)
	assert.Equal(t, int64(2), reloaded.Version)
	assert.NotNil(t, reloaded.Previous)
	assert.Equal(t, ":9001", reloaded.Previous.Server.Address)
	assert.Equal(t, ":9002", reloaded.Current.Server.Address)
	assert.Equal(t, "deny", reloaded.Current.IPAccess.DefaultPolicy)
}

func TestManager_KeepsPreviousConfigOnInvalidReload(t *testing.T) {
	isolateConfigEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
server:
  address: ":9101"
ip_access:
  default_policy: allow
`), 0o600))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager := NewManager(path, 10*time.Millisecond, NewFileLoader(), &testutils.MockLogger{})
	require.NoError(t, manager.Start(ctx))
	_, ok := manager.Current()
	require.True(t, ok)

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, os.WriteFile(path, []byte(`
ip_access:
  default_policy: captcha
`), 0o600))

	time.Sleep(50 * time.Millisecond)

	cfg, ok := manager.Current()
	require.True(t, ok)
	assert.Equal(t, ":9101", cfg.Server.Address)
	assert.Equal(t, int64(1), manager.Version())
}

func waitForChange(t *testing.T, changes <-chan Change) Change {
	t.Helper()

	select {
	case change := <-changes:
		return change
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for config change")
		return Change{}
	}
}
