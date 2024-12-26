//go:build !windows

package socket

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"syscall"
)

func listenFd(url *url.URL) (net.Listener, error) {
	// File descriptors are passed in through systemd
	fd, err := strconv.Atoi(url.Host)
	if err != nil {
		return nil, err
	}
	syscall.CloseOnExec(fd)
	file := os.NewFile(uintptr(fd), url.Host)
	ln, err := net.FileListener(file)
	if err != nil {
		return nil, err
	}
	return ln, nil
}
