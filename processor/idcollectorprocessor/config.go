// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor"

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

// Config specifies the set of attributes to be inserted, updated, upserted and
// deleted and the properties to include/exclude a span from being processed.
// This processor handles all forms of modifications to attributes within a span, log, or metric.
// Prior to any actions being applied, each span is compared against
// the include properties and then the exclude properties if they are specified.
// This determines if a span is to be processed or not.
// The list of actions is applied in order specified in the configuration.
type Config struct {
	Patterns         []string `mapstructure:"patterns"`
	NegativePatterns []string `mapstructure:"negative_patterns"`
	TargetAttribute  string   `mapstructure:"target_attribute"`
	ExcludeAttrs     []string `mapstructure:"exclude_attributes"`
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Patterns == nil {
		return errors.New("missing required field \"patterns\"")
	}
	if cfg.TargetAttribute == "" {
		return errors.New("missing required field \"target_attribute\"")
	}
	return nil
}
