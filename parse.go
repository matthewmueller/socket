// Package socket is a parser for accepting more server addresses including unix
// domain sockets during development. Eventually, we'll want to switch off of
// peg for better error messages. For now, it's quicker to iterate on the
// grammar using a peg parser.
package socket

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

//go:generate go tool peg -strict -switch -inline parse.peg

var ErrParsing = errors.New("urlx: unable to parse")

var defaultHost = "127.0.0.1"
var defaultPort = "3000"

func Parse(input string) (*url.URL, error) {
	parser := &parser{Buffer: input}
	parser.Init()
	err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("%w %q", ErrParsing, input)
	}
	parser.Execute()

	// Fallback to regular url parsing
	if parser.url.Raw != "" && !strings.Contains(parser.url.Port, "x") {
		u, err := url.Parse(input)
		if err != nil {
			return nil, err
		}
		return u, nil
	}

	// Create a new url
	u := new(url.URL)

	// Handle the scheme
	if parser.url.Scheme != "" {
		u.Scheme = parser.url.Scheme
	} else if parser.url.Port == "443" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}

	hasHost := parser.url.Host != ""
	hasPort := parser.url.Port != ""
	hasPath := parser.url.Path != ""

	// Handle the path if there is one
	u.Path = parser.url.Path

	// Handle the port
	port := defaultPort
	if !hasPort && u.Scheme == "https" {
		port = "443"
	}

	if hasHost && u.Scheme == "fd" {
		u.Host = parser.url.Host
	}

	if u.Scheme == "unix" || u.Scheme == "fd" {
		return u, nil
	}

	// Handle the host and port
	if hasHost && hasPort {
		u.Host = parser.url.Host + ":" + parser.url.Port
	} else if hasPort {
		u.Host = defaultHost + ":" + parser.url.Port // default to 127.0.0.1
	} else if hasHost {
		u.Host = parser.url.Host + ":" + port
	} else if !hasPath { // Only append default host:port if there's no path
		u.Host = defaultHost + ":" + port
	}

	return u, nil
}

// Used in the parser
type uri struct {
	Port   string
	Scheme string
	Host   string
	Path   string
	Raw    string
}
