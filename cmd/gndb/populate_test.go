package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOverrideFlags(t *testing.T) {
	tests := []struct {
		name              string
		sourceCount       int
		hasReleaseVersion bool
		hasReleaseDate    bool
		expectError       bool
		errorContains     string
	}{
		{
			name:              "no sources, no overrides - OK",
			sourceCount:       0,
			hasReleaseVersion: false,
			hasReleaseDate:    false,
			expectError:       false,
		},
		{
			name:              "single source, no overrides - OK",
			sourceCount:       1,
			hasReleaseVersion: false,
			hasReleaseDate:    false,
			expectError:       false,
		},
		{
			name:              "single source, release-version override - OK",
			sourceCount:       1,
			hasReleaseVersion: true,
			hasReleaseDate:    false,
			expectError:       false,
		},
		{
			name:              "single source, release-date override - OK",
			sourceCount:       1,
			hasReleaseVersion: false,
			hasReleaseDate:    true,
			expectError:       false,
		},
		{
			name:              "single source, both overrides - OK",
			sourceCount:       1,
			hasReleaseVersion: true,
			hasReleaseDate:    true,
			expectError:       false,
		},
		{
			name:              "multiple sources, no overrides - OK",
			sourceCount:       2,
			hasReleaseVersion: false,
			hasReleaseDate:    false,
			expectError:       false,
		},
		{
			name:              "multiple sources, release-version override - ERROR",
			sourceCount:       3,
			hasReleaseVersion: true,
			hasReleaseDate:    false,
			expectError:       true,
			errorContains:     "cannot override release version with multiple sources (3 sources selected)",
		},
		{
			name:              "multiple sources, release-date override - ERROR",
			sourceCount:       2,
			hasReleaseVersion: false,
			hasReleaseDate:    true,
			expectError:       true,
			errorContains:     "cannot override release date with multiple sources (2 sources selected)",
		},
		{
			name:              "multiple sources, both overrides - ERROR (release-version checked first)",
			sourceCount:       2,
			hasReleaseVersion: true,
			hasReleaseDate:    true,
			expectError:       true,
			errorContains:     "cannot override release version with multiple sources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOverrideFlags(tt.sourceCount, tt.hasReleaseVersion, tt.hasReleaseDate)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
