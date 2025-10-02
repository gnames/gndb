package cmd

import (
	"os"
	"testing"

	"github.com/gnames/gndb/internal/iofs"
	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	var err error
	tempHome := t.TempDir()

	t.Run("creates config and sources files", func(t *testing.T) {

		err = iofs.EnsureDirs(tempHome)
		require.NoError(t, err)

		err = iofs.EnsureConfigFile(tempHome)
		require.NoError(t, err)

		confFile := config.ConfigFilePath(tempHome)
		content, err := os.ReadFile(confFile)
		require.NoError(t, err)
		assert.Equal(t, iofs.ConfigYAML, string(content))

		var cfgViper *config.Config
		cfgViper, err = initConfig(tempHome)
		require.NoError(t, err)

		cfg = config.New()
		opts = cfgViper.ToOptions()
		cfg.Update(opts)

		// Set HomeDir (mimics bootstrap behavior)
		cfg.Update([]config.Option{config.OptHomeDir(tempHome)})

		// Verify HomeDir is set correctly
		assert.Equal(t, tempHome, cfg.HomeDir)

		err = os.RemoveAll(config.ConfigDir(tempHome))
		require.NoError(t, err)
	})
}
