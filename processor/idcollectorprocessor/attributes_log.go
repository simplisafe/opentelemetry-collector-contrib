// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor"

import (
	"context"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logidcollectorprocessor struct {
	logger                   *zap.Logger
	cfg                      *Config
	compiledPatterns         []*regexp.Regexp
	compiledNegativePatterns []*regexp.Regexp
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

	var compiledNegativePatterns []*regexp.Regexp
	if oCfg.NegativePatterns != nil {
		compiledNegativePatterns = make([]*regexp.Regexp, 0, len(oCfg.NegativePatterns))
		for _, pattern := range oCfg.NegativePatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				logger.Error("failed to compile pattern", zap.String("pattern", pattern), zap.Error(err))
				continue
			}
			compiledNegativePatterns = append(compiledNegativePatterns, re)
		}
	} else {
		compiledNegativePatterns = make([]*regexp.Regexp, 0)
	}

	return &logidcollectorprocessor{
		logger:                   logger,
		cfg:                      oCfg,
		compiledPatterns:         compiledPatterns,
		compiledNegativePatterns: compiledNegativePatterns,
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

				collectedIDs := make(map[string]struct{})

				// Recurse through all attributes
				var processAttributes func(attrs pcommon.Value)
				processAttributes = func(attrs pcommon.Value) {
					switch attrs.Type() {
					case pcommon.ValueTypeMap:
						attrs.Map().Range(func(k string, mv pcommon.Value) bool {
							processAttributes(mv)
							return true
						})
					case pcommon.ValueTypeSlice:
						for i := 0; i < attrs.Slice().Len(); i++ {
							processAttributes(attrs.Slice().At(i))
						}
					case pcommon.ValueTypeStr:
						strValue := attrs.Str()
						for _, pattern := range a.compiledPatterns {
							ids := pattern.FindAllString(strValue, -1)
							for i := 0; i < len(ids); i++ {
								collectedIDs[ids[i]] = struct{}{}
							}
						}
					}
				}

				// Process the raw Body
				processAttributes(lr.Body())

				// Process all attributes (recursively)
				topAttrs.Range(func(k string, v pcommon.Value) bool {
					processAttributes(v)
					return true // continue iterating
				})

				if len(collectedIDs) > 0 {
					// remove any collected IDs that match the negative patterns
					for _, pattern := range a.compiledNegativePatterns {
						for id := range collectedIDs {
							if pattern.MatchString(id) {
								delete(collectedIDs, id)
							}
						}
					}

					uniqueIDs := make([]string, 0, len(collectedIDs))
					for id := range collectedIDs {
						uniqueIDs = append(uniqueIDs, id)
					}

					topAttrs.PutStr(a.cfg.TargetAttribute, strings.Join(uniqueIDs, ","))
				}
			}
		}
	}
	return ld, nil
}
