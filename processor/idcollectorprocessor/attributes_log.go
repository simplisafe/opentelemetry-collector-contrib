// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor"

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logidcollectorprocessor struct {
	logger           *zap.Logger
	cfg              *Config
	compiledPatterns []*regexp.Regexp
}

// newLogidcollectorprocessor returns a processor that modifies attributes of a
// log record. To construct the idcollector processors, the use of the factory
// methods are required in order to validate the inputs.
func newLogidcollectorprocessor(logger *zap.Logger, oCfg *Config) *logidcollectorprocessor {
	compiledPatterns := make([]*regexp.Regexp, 0, len(oCfg.Patterns))
	for _, pattern := range oCfg.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			logger.Error("failed to compile pattern", zap.String("pattern", pattern), zap.Error(err))
			continue
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	return &logidcollectorprocessor{
		logger:           logger,
		cfg:              oCfg,
		compiledPatterns: compiledPatterns,
	}
}

func (a *logidcollectorprocessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rs := rls.At(i)
		ilss := rs.ScopeLogs()

		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)

				topAttrs := lr.Attributes()

				var collectedIDs []string

				// recurse through all attributes
				var processAttributes func(attrs pcommon.Map)
				processAttributes = func(attrs pcommon.Map) {
					attrs.Range(func(key string, value pcommon.Value) bool {
						if value.Type() == pcommon.ValueTypeMap {
							// If the value is a nested map, recurse into it
							processAttributes(value.Map())
						} else if value.Type() == pcommon.ValueTypeStr {
							// If the value is a string, extract alphanumeric IDs
							strValue := value.Str()
							for _, pattern := range a.compiledPatterns {
								ids := pattern.FindAllString(strValue, -1)
								if len(ids) > 0 {
									collectedIDs = append(collectedIDs, ids...)
								}
							}
						}
						return true // Continue iteration
					})
				}

				processAttributes(topAttrs)

				if len(collectedIDs) > 0 {
					sort.Strings(collectedIDs) // Sort IDs alphanumerically
					topAttrs.PutStr(a.cfg.TargetAttribute, strings.Join(collectedIDs, ","))
				}
			}
		}
	}
	return ld, nil
}
