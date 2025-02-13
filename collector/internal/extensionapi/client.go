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

package extensionapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RegisterResponse is the body of the response for /register
type RegisterResponse struct {
	FunctionName    string `json:"functionName"`
	FunctionVersion string `json:"functionVersion"`
	Handler         string `json:"handler"`
	ExtensionID     string
}

// NextEventResponse is the response for /event/next
type NextEventResponse struct {
	EventType          EventType `json:"eventType"`
	DeadlineMs         int64     `json:"deadlineMs"`
	RequestID          string    `json:"requestId"`
	InvokedFunctionArn string    `json:"invokedFunctionArn"`
	Tracing            Tracing   `json:"tracing"`
}

// Tracing is part of the response for /event/next
type Tracing struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// StatusResponse is the body of the response for /init/error and /exit/error
type StatusResponse struct {
	Status string `json:"status"`
}

// EventType represents the type of events recieved from /event/next
type EventType string

const (
	SchemaVersion20200101 = "2020-01-01"
	SchemaVersionLatest   = SchemaVersion20200101

	// Invoke is a lambda INVOKE event
	Invoke EventType = "INVOKE"

	// Shutdown is a SHUTDOWN event for the environment
	Shutdown EventType = "SHUTDOWN"

	ExtensionNameHeader      = "Lambda-Extension-Name"
	ExtensionIdentiferHeader = "Lambda-Extension-Identifier"
	ExtensionErrorType       = "Lambda-Extension-Function-Error-Type"
)

// Client is a simple client for the Lambda Extensions API.
type Client struct {
	baseURL     string
	extensionID string
	httpClient  *http.Client
}

// NewClient returns a Lambda Extensions API client.
//  POST http://${AWS_RUNTIME_API}/2020-01-01/extension
func NewClient(awsLambdaRuntimeAPI string) *Client {
	baseURL := fmt.Sprintf("http://%s/%s/extension", awsLambdaRuntimeAPI, SchemaVersionLatest)

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Register will register the extension with the Extensions API.
// Each API call must include the Lambda-Extension-Name header.
//  Reference: https://github.com/awsdocs/aws-lambda-developer-guide/blob/main/doc_source/telemetry-api.md#telemetry-api-registration
func (e *Client) Register(ctx context.Context, extensionName string) (*RegisterResponse, error) {
	const action = "/register"
	url := e.baseURL + action

	requestBody, err := json.Marshal(map[string]interface{}{
		"events": []EventType{Invoke, Shutdown},
	})

	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	request.Header.Set(ExtensionNameHeader, extensionName)

	var registerResponse RegisterResponse
	response, err := e.doRequest(request, &registerResponse)
	if err != nil {
		return nil, err
	}

	e.extensionID = response.Header.Get(ExtensionIdentiferHeader)
	registerResponse.ExtensionID = e.extensionID

	return &registerResponse, nil
}

// NextEvent blocks while long polling for the next lambda invoke or shutdown.
func (e *Client) NextEvent(ctx context.Context) (*NextEventResponse, error) {
	const action = "/event/next"
	url := e.baseURL + action

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set(ExtensionIdentiferHeader, e.extensionID)

	var nextEventResponse NextEventResponse
	_, err = e.doRequest(request, &nextEventResponse)
	if err != nil {
		return nil, err
	}

	return &nextEventResponse, nil
}

// InitError reports an initialization error to the platform.
// Call it when you registered but failed to initialize.
func (e *Client) InitError(ctx context.Context, errorType string) (*StatusResponse, error) {
	const action = "/init/error"
	url := e.baseURL + action

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set(ExtensionErrorType, errorType)
	request.Header.Set(ExtensionIdentiferHeader, e.extensionID)

	var statusResponse StatusResponse
	_, err = e.doRequest(request, &statusResponse)
	if err != nil {
		return nil, err
	}

	return &statusResponse, nil
}

// ExitError reports an error to the platform before exiting.
// Call it when you encounter an unexpected failure.
func (e *Client) ExitError(ctx context.Context, errorType string) (*StatusResponse, error) {
	const action = "/exit/error"
	url := e.baseURL + action

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set(ExtensionErrorType, errorType)
	request.Header.Set(ExtensionIdentiferHeader, e.extensionID)

	var statusResponse StatusResponse
	_, err = e.doRequest(request, &statusResponse)
	if err != nil {
		return nil, err
	}

	return &statusResponse, nil
}

// doRequest sends an HTTP request and returns an HTTP response.
func (e *Client) doRequest(request *http.Request, out interface{}) (*http.Response, error) {
	response, err := e.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 {
		err := fmt.Errorf("request failed with status %s", response.Status)
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, out)
	if err != nil {
		return response, err
	}

	return response, nil
}
