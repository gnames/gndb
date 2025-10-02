package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gnames/gn"
)

// Update applies a slice of Option functions to the Config.
// This is the only way to modify a Config after creation.
// Invalid options are rejected with warnings - config remains in valid state.
func (c *Config) Update(opts []Option) {
	for _, opt := range opts {
		opt(c)
	}
}

// ToOptions converts the Config to a slice of Option functions.
// Only includes persistent fields appropriate for config.yaml.
// Excludes runtime-only fields (HomeDir, SourceIDs, ReleaseVersion/Date, WithFlatClassification).
// Used for round-tripping config.yaml ↔ Config conversions.
func (c *Config) ToOptions() []Option {
	var res []Option
	var s string
	var i int
	s = c.Database.Host
	if s != "" {
		res = append(res, OptDatabaseHost(s))
	}
	i = c.Database.Port
	if i > 0 {
		res = append(res, OptDatabasePort(i))
	}
	s = c.Database.User
	if s != "" {
		res = append(res, OptDatabaseUser(s))
	}
	s = c.Database.Password
	if s != "" {
		res = append(res, OptDatabasePassword(s))
	}
	s = c.Database.Database
	if s != "" {
		res = append(res, OptDatabaseDatabase(s))
	}
	s = c.Database.SSLMode
	if s != "" {
		res = append(res, OptDatabaseSSLMode(s))
	}
	i = c.Database.BatchSize
	if i > 0 {
		res = append(res, OptDatabaseBatchSize(i))
	}

	s = c.Log.Format
	if s != "" {
		res = append(res, OptLogFormat(s))
	}
	s = c.Log.Level
	if s != "" {
		res = append(res, OptLogLevel(s))
	}
	s = c.Log.Destination
	if s != "" {
		res = append(res, OptLogDestination(s))
	}

	i = c.JobsNumber
	if i > 0 {
		res = append(res, OptJobsNumber(i))
	}
	return res
}

func isValidString(name, s string) bool {
	res := s != ""
	if !res {
		gn.Warn("<em>%s</em> cannot be empty, ignoring", name)
	}
	return res
}

func isValidInt(name string, i int) bool {
	res := i > 0
	if !res {
		gn.Warn("<em>%s</em> has to be positive number, ignoring %d", name, i)
	}
	return res
}

func isValidEnum(name, val string) bool {
	s := struct{}{}
	data := map[string]map[string]struct{}{
		"Database.SSLMode": {"disable": s, "require": s,
			"verify-ca": s, "verify-full": s},
		"Log.Level":       {"debug": s, "info": s, "warn": s, "error": s},
		"Log.Format":      {"json": s, "text": s, "tint": s},
		"Log.Destination": {"file": s, "stdin": s, "stdout": s},
	}
	vals := slices.Sorted(maps.Keys(data[name]))
	var lines []string
	for _, v := range vals {
		line := fmt.Sprintf("  * %s", v)
		lines = append(lines, line)
	}
	if _, ok := data[name][val]; ok {
		return true
	} else {
		gn.Warn(
			"<em>%s</em> does not support '%s' as a value. "+
				"Valid values are: \n%s\nIgnoring...",
			[]string{name, val, strings.Join(lines, "\n")},
		)
		return false
	}
}
