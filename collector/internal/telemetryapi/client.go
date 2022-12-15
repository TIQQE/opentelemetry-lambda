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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
)

const (
	SchemaVersion20220701          = "2022-07-01"
	SchemaVersionLatest            = SchemaVersion20220701
	lambdaAgentIdentifierHeaderKey = "Lambda-Extension-Identifier"
)

// Client is used for subscribing to the Telemetry API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a Lambda Telemetry API client.
//  Reference: https://github.com/awsdocs/aws-lambda-developer-guide/blob/main/doc_source/telemetry-api-reference.md
func NewClient() *Client {
	baseURL := fmt.Sprintf("http://%s/%s/telemetry", os.Getenv("AWS_LAMBDA_RUNTIME_API"), SchemaVersionLatest)

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Subscribe sends subscription request to the Telemetry API Client.
//  PUT http://${AWS_LAMBDA_RUNTIME_API}/2022-07-01/telemetry
//  Reference:
//   https://github.com/awsdocs/aws-lambda-developer-guide/blob/main/doc_source/telemetry-api-reference.md#subscribe
//   https://github.com/awsdocs/aws-lambda-developer-guide/blob/main/doc_source/telemetry-api.md#sending-a-subscription-request-to-the-telemetry-api
func (c *Client) Subscribe(ctx context.Context, extensionID string, listenerURI string) (string, error) {
	eventTypes := []EventType{Platform}

	bufferingConfig := BufferingCfg{
		TimeoutMS: 100,
		MaxItems:  1000,
		MaxBytes:  256 * 1024,
	}

	destination := Destination{
		Protocol:   HTTProto,
		HTTPMethod: HTTPPost,
		Encoding:   JSON,
		URI:        URI(listenerURI),
	}

	request := &SubscribeRequest{
		SchemaVersion: SchemaVersionLatest,
		EventTypes:    eventTypes,
		BufferingCfg:  bufferingConfig,
		Destination:   destination,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal SubscribeRequest: %w", err)
	}

	// Before sending a subscription request, you must have an
	// extension ID (Lambda-Extension-Identifier)
	headers := make(map[string]string)
	headers[lambdaAgentIdentifierHeaderKey] = extensionID

	// Send a Subscribe API request
	response, err := httpPutWithHeaders(ctx, c.httpClient, c.baseURL, data, headers)
	if err != nil {
		utility.LogError(err, "Subscribe", "Subscription failed")
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusAccepted {
		utility.LogError(err, "Subscribe", "Subscription failed. Logs API is not supported! Is this extension running in a local sandbox?", utility.KeyValue{K: "status_code", V: response.StatusCode})

	} else if response.StatusCode != http.StatusOK {
		utility.LogError(err, "Subscribe", "Subscription failed.", utility.KeyValue{K: "baseURL", V: c.baseURL}, utility.KeyValue{K: "status_code", V: response.StatusCode})

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", fmt.Errorf("request to %s failed: %d[%s]: %w", c.baseURL, response.StatusCode, response.Status, err)
		}

		return "", fmt.Errorf("request to %s failed: %d[%s] %s", c.baseURL, response.StatusCode, response.Status, string(body))
	}

	body, _ := io.ReadAll(response.Body)

	return string(body), nil
}

// httpPutWithHeaders sends request to Telemetry API Client
// with HTTP Put method.
func httpPutWithHeaders(ctx context.Context, client *http.Client, url string, data []byte, headers map[string]string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	contentType := "application/json"
	request.Header.Set("Content-Type", contentType)

	if headers != nil {
		for k, v := range headers {
			request.Header.Set(k, v)
		}
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
