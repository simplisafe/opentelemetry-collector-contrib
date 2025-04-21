// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id       component.ID
		expected component.Config
	}{
		{
			id: component.NewIDWithName(metadata.Type, "insert"),
			expected: &Config{
				Patterns: PatternsArray{
					"\\b[a-zA-Z0-9]{32}\\b",
					"\\b[a-zA-Z0-9]{32}\\b",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
			require.NoError(t, err)

			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			sub, err := cm.Sub(tt.id.String())
			require.NoError(t, err)
			require.NoError(t, sub.Unmarshal(cfg))

			// assert.NoError(t, xconfmap.Validate(cfg))
			// assert.Equal(t, tt.expected, cfg)
		})
	}
}
