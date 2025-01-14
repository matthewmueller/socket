package socket

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Dial creates a connection to an address
func Dial(ctx context.Context, address string) (net.Conn, error) {
	url, err := Parse(address)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	// Empty host means the path is a unix domain socket
	if url.Host == "" {
		return dialer.DialContext(ctx, "unix", address)
	}
	return dialer.DialContext(ctx, "tcp", url.Host)
}

// Transport creates a RoundTripper for an HTTP Client
func Transport(path string) (*http.Transport, error) {
	url, err := Parse(path)
	if err != nil {
		return nil, err
	}
	// Empty host means the path is a unix domain socket
	if url.Host == "" {
		dialer := new(net.Dialer)
		return &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", path)
			},
		}, nil
	}
	return httpTransport(url.Host), nil
}

// httpTransport is a modified from http.DefaultTransport
func httpTransport(host string) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, host)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
