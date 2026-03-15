package iosources

import (
	"log/slog"
	"os"

	"github.com/gnames/gndb/pkg/config"
	"github.com/gnames/gndb/pkg/sources"
)

type iosources struct {
	cfg *config.Config
}

func New(cfg *config.Config) sources.Sources {
	res := iosources{cfg: cfg}
	return &res
}

func (s *iosources) Load() (*sources.SourcesConfig, error) {
	sourcesPath := config.SourcesFilePath(s.cfg.HomeDir)
	sourcesConfig, err := loadSourcesConfig(sourcesPath)
	if err != nil {
		return nil, SourcesConfigError(sourcesPath, err)
	}

	// Load custom sources if the file exists
	customPath := config.CustomSourcesFilePath(s.cfg.HomeDir)
	if _, err := os.Stat(customPath); err == nil {
		customConfig, err := loadCustomSourcesConfig(customPath)
		if err != nil {
			return nil, SourcesConfigError(customPath, err)
		}
		for _, ds := range customConfig.DataSources {
			if ds.ID < 1000 {
				slog.Warn("Ignoring custom source with reserved ID",
					"id", ds.ID,
					"title_short", ds.TitleShort,
					"file", customPath,
				)
				continue
			}
			sourcesConfig.DataSources = append(sourcesConfig.DataSources, ds)
		}
	}

	return sourcesConfig, nil
}
