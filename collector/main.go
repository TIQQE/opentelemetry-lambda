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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/open-telemetry/opentelemetry-lambda/collector/extension"
	"github.com/open-telemetry/opentelemetry-lambda/collector/internal/extension/lambdaextension"
	"github.com/open-telemetry/opentelemetry-lambda/collector/lambdacomponents"
	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
)

var (
	sp              = newSpanProcessor()
	extensionName   = filepath.Base(os.Args[0]) // extension name has to match the filename
	extensionClient = extension.NewClient(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
)

func main() {
	// Setup default lambda components
	factories, err := lambdacomponents.Components()
	if err != nil {
		utility.LogError(err, "LambdaComponents", "Failed to make factories")
		return
	}

	// Add Lambda Extension
	lambdaFactory := lambdaextension.NewFactory(sp)
	factories.Extensions[lambdaFactory.Type()] = lambdaFactory

	collector, err := NewCollector(factories)
	if err != nil {
		utility.LogError(err, "Collector", "Failed to initialize new collector")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = collector.Start(ctx)
	if err != nil {
		utility.LogError(err, "Collector", "Failed to start the lambda layer collector extension")
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		cancel()
	}()

	_, err = extensionClient.Register(ctx, extensionName)
	if err != nil {
		utility.LogError(err, "Register", "Failed to register extension")
		return
	}

	// Will block until shutdown event is received or cancelled via the context.
	processEvents(ctx, collector)
}

func processEvents(ctx context.Context, collector *Collector) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			response, err := extensionClient.NextEvent(ctx)
			if err != nil {
				utility.LogError(err, "processEvents", "Failed to process event")
				return
			}

			// Exit if we receive a SHUTDOWN event
			if response.EventType == extension.Shutdown {
				collector.Stop() // TODO: handle return values
				return
			}

			select {
			case <-sp.waitCh:
			case <-time.After(1000 * time.Millisecond):
			}

			for count := sp.activeSpanCount(); count > 0; count = sp.activeSpanCount() {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}
