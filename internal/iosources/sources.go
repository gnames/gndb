package iosources

import (
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
	return sourcesConfig, nil
}
