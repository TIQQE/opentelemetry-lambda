package extension

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/open-telemetry/opentelemetry-lambda/collector/pkg/utility"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
)

var (
	HTTPClient *http.Client
)

type HTTPClientSettings struct {
	Connect          time.Duration
	ConnKeepAlive    time.Duration
	ExpectContinue   time.Duration
	IdleConn         time.Duration
	MaxAllIdleConns  int
	MaxHostIdleConns int
	ResponseHeader   time.Duration
	TLSHandshake     time.Duration
}

func NewHTTPClient(httpSettings HTTPClientSettings) (*http.Client, error) {
	var client http.Client
	tr := &http.Transport{
		ResponseHeaderTimeout: httpSettings.ResponseHeader,
		Proxy:                 http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			KeepAlive: httpSettings.ConnKeepAlive,
			DualStack: true,
			Timeout:   httpSettings.Connect,
		}).DialContext,
		MaxIdleConns:          httpSettings.MaxAllIdleConns,
		IdleConnTimeout:       httpSettings.IdleConn,
		TLSHandshakeTimeout:   httpSettings.TLSHandshake,
		MaxIdleConnsPerHost:   httpSettings.MaxHostIdleConns,
		ExpectContinueTimeout: httpSettings.ExpectContinue,
	}

	err := http2.ConfigureTransport(tr)
	if err != nil {
		return &client, fmt.Errorf("failed to create custom http client: %w", err)
	}

	return &http.Client{
		Transport: tr,
	}, nil
}

func GetHttpClient() (*http.Client, error) {
	if HTTPClient != nil {
		return HTTPClient, nil
	}

	c, err := NewHTTPClient(HTTPClientSettings{
		Connect:          6 * time.Second,
		ConnKeepAlive:    30 * time.Second,
		ExpectContinue:   1 * time.Second,
		IdleConn:         90 * time.Second,
		MaxAllIdleConns:  30,
		MaxHostIdleConns: 10,
		ResponseHeader:   0,
		TLSHandshake:     60 * time.Second,
	})

	if err != nil {
		err = errors.Wrap(err, "failed to create custom client")
		utility.LogError(err, "NewHTTPClientError", "Failed to create http client")

		return nil, err
	}

	HTTPClient = c
	return HTTPClient, nil
}
