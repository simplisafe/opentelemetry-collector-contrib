// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package idcollectorprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/idcollectorprocessor/internal/metadata"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// NewFactory returns a new factory for the Attributes processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

// Note: This isn't a valid configuration because the processor would do no work.
func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		newLogidcollectorprocessor(set.Logger, oCfg).processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}
