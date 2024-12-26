//go:build windows

package socket

import (
	"fmt"
	"net"
	"net/url"
)

func listenFd(*url.URL) (net.Listener, error) {
	return nil, fmt.Errorf("socket: listening on a file descriptor is not supported on windows")
}
