package config

import (
	"path/filepath"
)

var (
	// MinVersionSFGA determines the SFGA version which is still compatible
	// with GNdb. Versions higher than minimal are all supported.
	MinVersionSFGA = "v0.3.30"
	// AppName is used in generating file system paths.
	AppName = "gndb"
)

// ConfigDir returns the directory path for configuration files.
// Returns ~/.config/gndb by default.
func ConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".config", AppName)
}

// CacheDir returns the directory path for cache files.
// Returns ~/.cache/gndb by default.
func CacheDir(homeDir string) string {
	return filepath.Join(homeDir, ".cache", AppName)
}

// LogDir returns the directory path for log files.
// Returns ~/.local/share/gndb/logs by default.
func LogDir(homeDir string) string {
	return filepath.Join(homeDir, ".local", "share", AppName, "logs")
}

// ConfigFilePath returns the full path to the config.yaml file.
// Returns ~/.config/gndb/config.yaml by default.
func ConfigFilePath(homeDir string) string {
	return filepath.Join(ConfigDir(homeDir), "config.yaml")
}
