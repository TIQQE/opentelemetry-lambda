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

package telemetryapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-collections/go-datastructures/queue"
	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
	"github.com/tiqqe/go-logger"
)

const (
	initialQueueSize    = 5
	minBatchSize        = 10
	defaultListenerPort = "4323"
)

// Listener is used to listen to the Telemetry API
type Listener struct {
	httpServer *http.Server
	// queue is a synchronous queue and is used to put the received log events to be dispatched later
	queue *queue.Queue
}

// NewListener returns a Lambda Telemetry API listener.
func NewListener() *Listener {
	return &Listener{
		httpServer: nil,
		queue:      queue.New(initialQueueSize),
	}
}

func listenOnAddress() string {
	var addr string
	envAwsLocal, ok := os.LookupEnv("AWS_SAM_LOCAL")

	if ok && envAwsLocal == "true" {
		addr = ":" + defaultListenerPort
	} else {
		addr = "sandbox:" + defaultListenerPort
	}

	return addr
}

// Start the server in a goroutine where the log events will be sent. It handles incoming
// requests from the Telemetry API.
func (s *Listener) Start() (string, error) {
	address := listenOnAddress()

	s.httpServer = &http.Server{Addr: address}
	http.HandleFunc("/", s.httpHandler)

	go func() {
		// Listen and handle incoming requests
		err := s.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			utility.LogError(err, "Start", "Unexpected stop on HTTP Server")
			s.Shutdown()

		} else {
			logger.InfoStringf("HTTP Server closed: %v", err.Error())
		}
	}()

	return fmt.Sprintf("http://%s/", address), nil
}

// httpHandler handles the requests coming from the Telemetry API.
// Everytime Telemetry API sends log events, this function will read
// them from the response body and put into a synchronous queue to be
// dispatched later. Logging or printing besides the error cases below
// is not recommended if you have subscribed to receive extension logs.
// Otherwise, logging here will cause Telemetry API to send new logs for
// the printed lines which may create an infinite loop.
func (s *Listener) httpHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utility.LogError(err, "httpHandler", "Failed reading body")
		return
	}

	// Parse and put the log messages into the queue
	var slice []Event
	_ = json.Unmarshal(body, &slice)

	for _, el := range slice {
		s.queue.Put(el)
	}

	slice = nil
}

// Shutdown the HTTP server listening for logs
func (s *Listener) Shutdown() {
	if s.httpServer != nil {
		ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

		err := s.httpServer.Shutdown(ctx)
		if err != nil {
			utility.LogError(err, "Shutdown", "Failed to shutdown HTTP server gracefully.")

		} else {
			s.httpServer = nil
		}
	}
}

func (s *Listener) Wait(ctx context.Context, requestId string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			items, err := s.queue.Get(minBatchSize)
			if err != nil {
				return fmt.Errorf("unable to get telemetry events from queue: %w", err)
			}

			for _, item := range items {
				i, ok := item.(Event)
				if !ok {
					logger.WarnStringf("Non-Event found in queue. Item: %v", item)
					continue
				}

				if i.Type == PLATFORM_LOGS_DROPPED {
					err := errors.New("failed to process event")
					utility.LogError(err, "TelemetryAPIWait", "Can't process one or more events", utility.KeyValue{K: "event", V: i})

					continue
				}

				if i.Type != "platform.runtimeDone" {
					continue
				}

				if i.Record["requestId"] == requestId {
					return nil
				}
			}
		}
	}
}
