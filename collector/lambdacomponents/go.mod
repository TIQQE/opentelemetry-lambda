module github.com/open-telemetry/opentelemetry-lambda/collector/lambdacomponents

go 1.18

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.58.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor v0.58.0
	go.opentelemetry.io/collector v0.58.0
	go.uber.org/multierr v1.8.0
)

require (
	cloud.google.com/go/compute v1.6.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf v1.4.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension v0.58.0 // indirect
	go.opentelemetry.io/collector/pdata v0.58.0 // indirect
	go.opentelemetry.io/otel v1.9.0 // indirect
	go.opentelemetry.io/otel/metric v0.31.0 // indirect
	go.opentelemetry.io/otel/trace v1.9.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/zap v1.22.0 // indirect
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	golang.org/x/sys v0.0.0-20220627191245-f75cf1eec38b // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220628213854-d9e0b6570c03 // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
