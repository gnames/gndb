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

//go:embed custom_sources.yaml
var CustomSourcesYAML string

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

// EnsureSourcesFile always overwrites sources.yaml with the embedded version.
// Users should not edit this file; use custom_sources.yaml for custom sources.
// This file is overwritten at every run to ensure users have the latest
// modifications for 'official' GNverifier datasets.
func EnsureSourcesFile(homeDir string) error {
	sourcesPath := config.SourcesFilePath(homeDir)

	if err := os.WriteFile(sourcesPath, []byte(SourcesYAML), 0644); err != nil {
		return CopyFileError(sourcesPath, err)
	}

	return nil
}

// EnsureCustomSourcesFile writes custom_sources.yaml only on first run.
// This file is owned by the user and will never be overwritten.
func EnsureCustomSourcesFile(homeDir string) error {
	customPath := config.CustomSourcesFilePath(homeDir)

	if _, err := os.Stat(customPath); err == nil {
		return nil
	}

	if err := os.WriteFile(customPath, []byte(CustomSourcesYAML), 0644); err != nil {
		return CopyFileError(customPath, err)
	}

	return nil
}
