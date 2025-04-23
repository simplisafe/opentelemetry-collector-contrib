// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor

import (
	"context"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/testdata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor/internal/metadata"
)

// Common structure for all the Tests
type logTestCase struct {
	name               string
	inputBody          string
	inputTraceID       string
	inputSpanID        string
	inputAttributes    map[string]any
	expectedAttributes map[string]any
}

// runIndividualLogTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualLogTestCase(t *testing.T, tt logTestCase, tp processor.Logs) {

	t.Run(tt.name, func(t *testing.T) {
		ld := generateLogData(tt.name, tt.inputBody, tt.inputTraceID, tt.inputSpanID, tt.inputAttributes)
		assert.NoError(t, tp.ConsumeLogs(context.Background(), ld))

		// if the result log has an attribute "extracted_ids", parse it as a csv, sort the strings, rejoin them, and write them back to "extracted_ids".
		// This ensures the order of IDs is consistent across runs.
		extractedIDsField, found := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().Get("extracted_ids")
		if found {

			idList := strings.Split(extractedIDsField.AsString(), ",")
			sort.Strings(idList)
			extractedIDsField.SetStr(strings.Join(idList, ","))
		}

		assert.NoError(t, plogtest.CompareLogs(generateLogData(tt.name, tt.inputBody, tt.inputTraceID, tt.inputSpanID, tt.expectedAttributes), ld))
	})
}

func generateLogData(resourceName string, body string, traceID string, spanID string, attrs map[string]any) plog.Logs {
	td := plog.NewLogs()
	res := td.ResourceLogs().AppendEmpty()
	res.Resource().Attributes().PutStr("name", resourceName)
	sl := res.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr(body)
	traceIDParsed, err := ParseTraceID(traceID)
	if err == nil {
		lr.SetTraceID(pcommon.TraceID{})
	}
	spanIDParsed, err := ParseSpanID(spanID)
	if err == nil {
		lr.SetSpanID(pcommon.SpanID{})
	}
	lr.SetTraceID(traceIDParsed)
	lr.SetSpanID(spanIDParsed)
	lr.SetTimestamp(0)
	//nolint:errcheck
	lr.Attributes().FromRaw(attrs)
	return td
}

// TestLogProcessor_Values tests all possible value types.
func TestLogProcessor_NilEmptyData(t *testing.T) {
	type nilEmptyTestCase struct {
		name   string
		input  plog.Logs
		output plog.Logs
	}
	testCases := []nilEmptyTestCase{
		{
			name:   "empty",
			input:  plog.NewLogs(),
			output: plog.NewLogs(),
		},
		{
			name:   "one-empty-resource-logs",
			input:  testdata.GenerateLogsOneEmptyResourceLogs(),
			output: testdata.GenerateLogsOneEmptyResourceLogs(),
		},
		{
			name:   "no-libraries",
			input:  testdata.GenerateLogsOneEmptyResourceLogs(),
			output: testdata.GenerateLogsOneEmptyResourceLogs(),
		},
	}
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)

	tp, err := factory.CreateLogs(
		context.Background(), processortest.NewNopSettings(metadata.Type), oCfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, tp.ConsumeLogs(context.Background(), tt.input))
			assert.Equal(t, tt.output, tt.input)
		})
	}
}

func TestIDCollector_FindsOneOrMultipleUUID(t *testing.T) {
	testCases := []logTestCase{
		{
			name:         "single and multiple matches",
			inputBody:    "some log 33333333333333333333333333333333 here",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": "not an ID",
				"attr3": "11111111111111111111111111111111 22222222222222222222222222222222",
			},
			expectedAttributes: map[string]any{
				"attr1":         "00000000000000000000000000000000",
				"attr2":         "not an ID",
				"attr3":         "11111111111111111111111111111111 22222222222222222222222222222222",
				"extracted_ids": "00000000000000000000000000000000,11111111111111111111111111111111,22222222222222222222222222222222,33333333333333333333333333333333",
			},
		},
		{
			name:         "single and multiple matches with dupes",
			inputBody:    "some log 33333333333333333333333333333333 here",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": "not an ID",
				"attr3": "11111111111111111111111111111111 22222222222222222222222222222222 11111111111111111111111111111111",
			},
			expectedAttributes: map[string]any{
				"attr1":         "00000000000000000000000000000000",
				"attr2":         "not an ID",
				"attr3":         "11111111111111111111111111111111 22222222222222222222222222222222 11111111111111111111111111111111",
				"extracted_ids": "00000000000000000000000000000000,11111111111111111111111111111111,22222222222222222222222222222222,33333333333333333333333333333333",
			},
		},
		{
			name:         "finds nested matches",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222222222222222222222222222",
				},
			},
			expectedAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222222222222222222222222222",
				},
				"extracted_ids": "00000000000000000000000000000000,11111111111111111111111111111111,22222222222222222222222222222222",
			},
		},
		{
			name:         "finds array matches",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": []any{
					map[string]any{
						"attr1": "11111111111111111111111111111111",
						"attr2": "something something 22222222222222222222222222222222 something",
					},
				},
			},
			expectedAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": []any{
					map[string]any{
						"attr1": "11111111111111111111111111111111",
						"attr2": "something something 22222222222222222222222222222222 something",
					},
				},
				"extracted_ids": "00000000000000000000000000000000,11111111111111111111111111111111,22222222222222222222222222222222",
			},
		},
		{
			name:         "excludes traceID",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"actual_trace_id": "07aa8ca1835d3fd6b0c9e26828d50236",
				"actual_span_id":  "682bb4b582fcbb58",
				"attr2": []any{
					map[string]any{
						"attr1": "11111111111111111111111111111111",
					},
				},
			},
			expectedAttributes: map[string]any{
				"actual_trace_id": "07aa8ca1835d3fd6b0c9e26828d50236",
				"actual_span_id":  "682bb4b582fcbb58",
				"attr2": []any{
					map[string]any{
						"attr1": "11111111111111111111111111111111",
					},
				},
				"extracted_ids": "11111111111111111111111111111111",
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).TargetAttribute = "extracted_ids"
	cfg.(*Config).Patterns = []string{
		"\\b[a-zA-Z0-9]{32}\\b",
	}

	tp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestIDCollector_FindsMultipleDifferentLengthIDs(t *testing.T) {
	testCases := []logTestCase{
		{
			name:         "finds nested matches",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222",
				},
			},
			expectedAttributes: map[string]any{
				"attr1": "00000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222",
				},
				"extracted_ids": "00000000,11111111111111111111111111111111,22222222",
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).TargetAttribute = "extracted_ids"
	cfg.(*Config).Patterns = []string{
		"\\b[a-zA-Z0-9]{32}\\b",
		"\\b[a-zA-Z0-9]{8}\\b",
	}

	tp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestIDCollector_DoesntMatchLongerIDs(t *testing.T) {
	testCases := []logTestCase{
		{
			name:         "finds nested matches",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "000000000000000000000000000000000", // 33 characters
			},
			expectedAttributes: map[string]any{
				"attr1": "000000000000000000000000000000000",
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).TargetAttribute = "extracted_ids"
	cfg.(*Config).Patterns = []string{
		"\\b[a-zA-Z0-9]{32}\\b",
	}

	tp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestIDCollector_ExcludesNegativePatterns(t *testing.T) {
	testCases := []logTestCase{
		{
			name:         "excludes negative pattern matches",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222",
				},
			},
			expectedAttributes: map[string]any{
				"attr1": "00000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111 22222222",
				},
				"extracted_ids": "00000000,22222222",
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).TargetAttribute = "extracted_ids"
	cfg.(*Config).Patterns = []string{
		"\\b[a-zA-Z0-9]{32}\\b",
		"\\b[a-zA-Z0-9]{8}\\b",
	}
	cfg.(*Config).NegativePatterns = []string{
		"\\b[1]{32}\\b",
	}

	tp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func TestIDCollector_ExcludesAttrs(t *testing.T) {
	testCases := []logTestCase{
		{
			name:         "excludes excluded attributes",
			inputBody:    "some log",
			inputTraceID: "07aa8ca1835d3fd6b0c9e26828d50236",
			inputSpanID:  "682bb4b582fcbb58",
			inputAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111",
				},
				"exclude_me": "22222222222222222222222222222222",
			},
			expectedAttributes: map[string]any{
				"attr1": "00000000000000000000000000000000",
				"attr2": map[string]any{
					"attr1": "11111111111111111111111111111111",
				},
				"exclude_me":    "22222222222222222222222222222222",
				"extracted_ids": "00000000000000000000000000000000,11111111111111111111111111111111",
			},
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).TargetAttribute = "extracted_ids"
	cfg.(*Config).Patterns = []string{
		"\\b[a-zA-Z0-9]{32}\\b",
	}
	cfg.(*Config).ExcludeAttrs = []string{
		"exclude_me",
	}

	tp, err := factory.CreateLogs(context.Background(), processortest.NewNopSettings(metadata.Type), cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)

	for _, tt := range testCases {
		runIndividualLogTestCase(t, tt, tp)
	}
}

func ParseTraceID(traceIDStr string) (pcommon.TraceID, error) {
	var id pcommon.TraceID
	if hex.DecodedLen(len(traceIDStr)) != len(id) {
		return pcommon.TraceID{}, errors.New("trace ids must be 32 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(traceIDStr))
	if err != nil {
		return pcommon.TraceID{}, err
	}
	return id, nil
}

func ParseSpanID(spanIDStr string) (pcommon.SpanID, error) {
	var id pcommon.SpanID
	if hex.DecodedLen(len(spanIDStr)) != len(id) {
		return pcommon.SpanID{}, errors.New("span ids must be 16 hex characters")
	}
	_, err := hex.Decode(id[:], []byte(spanIDStr))
	if err != nil {
		return pcommon.SpanID{}, err
	}
	return id, nil
}
