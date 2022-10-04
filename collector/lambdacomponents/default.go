// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lambdacomponents

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.uber.org/multierr"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oidcauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor"
)

func Components() (component.Factories, error) {
	var errs []error

	receivers, err := component.MakeReceiverFactoryMap(
		otlpreceiver.NewFactory(),
	)

	if err != nil {
		errs = append(errs, err)
	}

	exporters, err := component.MakeExporterFactoryMap(
		otlpexporter.NewFactory(),
		loggingexporter.NewFactory(),
		awsxrayexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
		prometheusremotewriteexporter.NewFactory(),
	)

	if err != nil {
		errs = append(errs, err)
	}

	processors, err := component.MakeProcessorFactoryMap(
		spanprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		filterprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		attributesprocessor.NewFactory(),
		groupbytraceprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		probabilisticsamplerprocessor.NewFactory(),
	)

	if err != nil {
		errs = append(errs, err)
	}

	extensions, err := component.MakeExtensionFactoryMap(
		oidcauthextension.NewFactory(),
		basicauthextension.NewFactory(),
		sigv4authextension.NewFactory(),
		oauth2clientauthextension.NewFactory(),
	)

	if err != nil {
		errs = append(errs, err)
	}

	factories := component.Factories{
		Receivers:  receivers,
		Exporters:  exporters,
		Processors: processors,
		Extensions: extensions,
	}

	return factories, multierr.Combine(errs...)
}
