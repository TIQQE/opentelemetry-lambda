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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/service"
)

var (
	// Version variable will be replaced at link time after `make` has been run.
	Version = "latest"

	// GitHash variable will be replaced at link time after `make` has been run.
	GitHash = "<NOT PROPERLY GENERATED>"
)

const (
	OtelTestEnv = "test"
	OtelProdEnv = "prod"
)

type Config struct {
	Extensions struct {
		Oauth2client struct {
			ClientId     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			TokenUrl     string `yaml:"token_url"`

			EndpointParams struct {
				GrantType string `yaml:"grant_type"`
				Scope     string `yaml:"scope"`
			} `yaml:"endpoint_params"`
		} `yaml:"oauth2client"`
	} `yaml:"extensions"`

	Receivers struct {
		Otlp struct {
			Protocols struct {
				Grpc struct {
					Endpoint string `yaml:"endpoint"`
				} `yaml:"grpc"`
				Http struct {
					Endpoint string `yaml:"endpoint"`
				} `yaml:"http"`
			} `yaml:"protocols"`
		} `yaml:"otlp"`
	} `yaml:"receivers"`

	Processors struct {
		MemoryLimiter struct {
			CheckInterval        string `yaml:"check_interval"`
			LimitPercentage      int    `yaml:"limit_percentage"`
			SpikeLimitPercentage int    `yaml:"spike_limit_percentage"`
		} `yaml:"memory_limiter"`
		GroupByTrace struct {
			WaitDuration string `yaml:"wait_duration"`
			NumTraces    int    `yaml:"num_traces"`
		} `yaml:"groupbytrace,omitempty"`
		Batch struct {
			Timeout string `yaml:"timeout"`
		} `yaml:"batch,omitempty"`
	} `yaml:"processors"`

	Exporters struct {
		Otlp struct {
			Endpoint       string `yaml:"endpoint"`
			RetryOnFailure struct {
				Enabled         bool   `yaml:"enabled"`
				InitialInterval string `yaml:"initial_interval"`
			} `yaml:"retry_on_failure"`
			Timeout string `yaml:"timeout"`
			Tls     struct {
				CaFile   string `yaml:"ca_file"`
				CertFile string `yaml:"cert_file"`
				KeyFile  string `yaml:"key_file"`
			} `yaml:"tls"`
			Auth struct {
				Authenticator string `yaml:"authenticator"`
			} `yaml:"auth"`
		} `yaml:"otlp"`
	} `yaml:"exporters"`

	Service struct {
		Extensions []string `yaml:"extensions"`
		Pipelines  struct {
			Traces struct {
				Receivers  []string `yaml:"receivers"`
				Processors []string `yaml:"processors"`
				Exporters  []string `yaml:"exporters"`
			} `yaml:"traces"`
			Metrics struct {
				Receivers  []string `yaml:"receivers"`
				Processors []string `yaml:"processors"`
				Exporters  []string `yaml:"exporters"`
			} `yaml:"metrics"`
		} `yaml:"pipelines"`
	} `yaml:"service"`
}

// Collector implements the OtelcolRunner interfaces running a single otelcol as a go routine within the
// same process as the test executor.
type Collector struct {
	factories      component.Factories
	configProvider service.ConfigProvider
	svc            *service.Collector
	appDone        chan struct{}
	stopped        bool
}

func updateConfig() {
	clientName, ex := os.LookupEnv("OTEL_CLIENT_NAME")
	if !ex {
		utility.LogError(nil, "updateConfigError", "require OTEL_CLIENT_NAME")
		return
	}
	env, ex := os.LookupEnv("OTEL_ENV")
	if !ex {
		utility.LogError(nil, "updateConfigError", "require OTEL_ENV")
		return
	}
	clientId, ex := os.LookupEnv("OTEL_CLIENT_ID")
	if !ex {
		utility.LogError(nil, "updateConfigError", "require OTEL_CLIENT_ID")
		return
	}
	clientSecret, ex := os.LookupEnv("OTEL_CLIENT_SECRET")
	if !ex {
		utility.LogError(nil, "updateConfigError", "require OTEL_CLIENT_SECRET")
		return
	}
	enableGroupBatch, ex := os.LookupEnv("OTEL_ENABLE_GROUP_BATCH")
	if !ex {
		utility.LogError(nil, "updateConfigError", "require OTEL_ENABLE_GROUP_BATCH")
		return
	}

	file := Config{}
	var yamlFile []byte
	var err error
	if enableGroupBatch == "true" {
		yamlFile, err = ioutil.ReadFile("/opt/collector-config/" + clientName + "/" + env + "/config-with-group-batch.yaml")
	} else {
		yamlFile, err = ioutil.ReadFile("/opt/collector-config/" + clientName + "/" + env + "/config.yaml")
	}

	err = yaml.Unmarshal(yamlFile, &file)
	if err != nil {
		utility.LogError(err, "updateConfigError", "failed to unmarshal config file")
		return
	}

	file.Extensions.Oauth2client.ClientId = clientId
	file.Extensions.Oauth2client.ClientSecret = clientSecret

	d, err := yaml.Marshal(&file)
	if err != nil {
		utility.LogError(err, "updateConfigError", "failed to marshal config file")
		return
	}

	err = ioutil.WriteFile("/tmp/config.yaml", d, 0755)
	if err != nil {
		utility.LogError(err, "updateConfigError", "failed to write config file")
		return
	}
}

func DisplayConfig(file string) string {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		utility.LogError(err, "DisplayConfigError", "Failed reading data", utility.KeyValue{K: "Filename", V: file})
		return ""
	}

	return string(data)
}

func getConfig() string {
	val, ex := os.LookupEnv("OPENTELEMETRY_COLLECTOR_CONFIG_FILE")
	if !ex {
		updateConfig()
		// 👉 Prints your collector configuration
		// logger.InfoString(DisplayConfig("/tmp/config.yaml"))

		return "/tmp/config.yaml"
	}

	// 👉 Prints your collector configuration
	// logger.InfoString(DisplayConfig(val))
	return val
}

func NewCollector(factories component.Factories) *Collector {
	providers := []confmap.Provider{fileprovider.New(), envprovider.New(), yamlprovider.New()}
	mapProvider := make(map[string]confmap.Provider, len(providers))

	for _, provider := range providers {
		mapProvider[provider.Scheme()] = provider
	}

	cfgSet := service.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:       []string{getConfig()},
			Providers:  mapProvider,
			Converters: []confmap.Converter{expandconverter.New()},
		},
	}

	cfgProvider, err := service.NewConfigProvider(cfgSet)
	if err != nil {
		utility.LogError(err, "ConfigProviderError", "Failed on creating config provider", utility.KeyValue{K: "ConfigProviderSettings", V: cfgSet})
		return nil
	}

	col := &Collector{
		factories:      factories,
		configProvider: cfgProvider,
	}

	return col
}

func (c *Collector) Start(ctx context.Context) error {
	params := service.CollectorSettings{
		BuildInfo: component.BuildInfo{
			Command:     "otelcol",
			Description: "Lambda Collector",
			Version:     Version,
		},
		ConfigProvider: c.configProvider,
		Factories:      c.factories,
		LoggingOptions: utility.NopCoreLogger(),
	}

	var err error
	c.svc, err = service.New(params)
	if err != nil {
		return err
	}

	c.appDone = make(chan struct{})

	go func() {
		defer close(c.appDone)
		appErr := c.svc.Run(ctx)
		if appErr != nil {
			err = appErr
		}
	}()

	for {
		state := c.svc.GetState()

		// While waiting for collector start, an error was found. Most likely
		// an invalid custom collector configuration file.
		if err != nil {
			return err
		}

		switch state {
		case service.Starting:
			// NoOp
		case service.Running:
			return nil
		default:
			err = fmt.Errorf("unable to start, otelcol state is %d", state)
		}
	}
}

func (c *Collector) Stop() error {
	if !c.stopped {
		c.stopped = true
		c.svc.Shutdown()
	}
	<-c.appDone

	return nil
}
