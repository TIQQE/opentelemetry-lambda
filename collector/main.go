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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/open-telemetry/opentelemetry-lambda/collector/internal/extensionapi"
	"github.com/open-telemetry/opentelemetry-lambda/collector/internal/telemetryapi"
	"github.com/open-telemetry/opentelemetry-lambda/collector/lambdacomponents"
	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
)

var (
	extensionName = filepath.Base(os.Args[0]) // extension name has to match the filename
)

type lifecycleManager struct {
	collector       *Collector
	extensionClient *extensionapi.Client
	listener        *telemetryapi.Listener
}

func main() {
	ctx, lm := newLifecycleManager(context.Background())

	// Will block until shutdown event is received or cancelled via the context.
	if lm != nil {
		lm.processEvents(ctx)
	}
}

func newLifecycleManager(ctx context.Context) (context.Context, *lifecycleManager) {
	ctx, cancel := context.WithCancel(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		cancel()
	}()

	// Step 1: Register the Lambda Extension API
	extensionClient := extensionapi.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	response, err := extensionClient.Register(ctx, extensionName)
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Cannot register extension.")
		return ctx, nil
	}

	// Step 2: Start the local HTTP listener which will receive data from Telemetry API
	listener := telemetryapi.NewListener()
	addrress, err := listener.Start()
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Cannot start Telemetry API Listener.")
		return ctx, nil
	}

	// Step 3: Subscribe the listener to Telemetry API
	telemetryClient := telemetryapi.NewClient()
	_, err = telemetryClient.Subscribe(ctx, response.ExtensionID, addrress)
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Cannot register Telemetry API client.")
		return ctx, nil
	}

	factories, err := lambdacomponents.Components()
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Failed to initialize lambda components")
		return ctx, nil
	}

	collector, err := NewCollector(factories)
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Failed to initialize new collector")
		return ctx, nil
	}

	err = collector.Start(ctx)
	if err != nil {
		utility.LogError(err, "LifecycleManager", "Failed to start the lambda layer collector extension")
		extensionClient.InitError(ctx, fmt.Sprintf("failed to start the collector: %v", err))
		return ctx, nil
	}

	return ctx, &lifecycleManager{
		listener:        listener,
		collector:       collector,
		extensionClient: extensionClient,
	}
}

func (lm *lifecycleManager) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			// This is a blocking action
			response, err := lm.extensionClient.NextEvent(ctx)
			if err != nil {
				utility.LogError(err, "processEvents", "Error waiting for extension event")
				lm.extensionClient.ExitError(ctx, fmt.Sprintf("error waiting for extension event: %v", err))

				return
			}

			// Exit if we receive a SHUTDOWN event
			if response.EventType == extensionapi.Shutdown {
				lm.listener.Shutdown()
				err = lm.collector.Stop()
				if err != nil {
					utility.LogError(err, "processEvents", "Failed stopping the collector", utility.KeyValue{K: "request_id", V: response.RequestID})
					lm.extensionClient.ExitError(ctx, fmt.Sprintf("error stopping collector: %v", err))
				}

				return
			}

			err = lm.listener.Wait(ctx, response.RequestID)
			if err != nil {
				utility.LogError(err, "processEvents", "Problem waiting for platform.runtimeDone event", utility.KeyValue{K: "request_id", V: response.RequestID})
			}
		}
	}
}
