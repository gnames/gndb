package iofs

import (
	_ "embed"
	"os"

	"github.com/gnames/gndb/pkg/config"
)

//go:embed config.yaml
var ConfigYAML string

//go:embed sources.yaml
var SourcesYAML string

func EnsureDirs(homeDir string) error {
	dirs := []string{
		config.ConfigDir(homeDir),
		config.CacheDir(homeDir),
		config.LogDir(homeDir),
	}
	for _, v := range dirs {
		if err := touchDir(v); err != nil {
			return err
		}
	}
	return nil
}

func touchDir(dir string) error {
	info, err := os.Stat(dir)
	if err == nil && info.IsDir() {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return CreateDirError(dir, err)
	}

	return nil
}

func EnsureConfigFile(homeDir string) error {
	configPath := config.ConfigFilePath(homeDir)

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// Write embedded config.yaml to the config directory
	if err := os.WriteFile(configPath, []byte(ConfigYAML), 0644); err != nil {
		return CopyFileError(configPath, err)
	}

	return nil
}

func EnsureSourcesFile(homeDir string) error {
	sourcesPath := config.SourcesFilePath(homeDir)

	// Check if sources file already exists
	if _, err := os.Stat(sourcesPath); err == nil {
		return nil
	}

	// Write embedded sources.yaml to the config directory
	if err := os.WriteFile(sourcesPath, []byte(SourcesYAML), 0644); err != nil {
		return CopyFileError(sourcesPath, err)
	}

	return nil
}
