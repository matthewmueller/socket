package socket

import (
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// listenFdsStart corresponds to `SD_LISTEN_FDS_START`.
const listenFdsStart = 3

// maybeActivate tries to create listeners using systemd's socket activation.
// This is where systemd passes a socket into the Go process, rather than the
// Go process creating the socket itself.
func maybeActivate() ([]net.Listener, error) {
	// LISTEN_PID must be set and match the current process ID or the parent's
	// process ID. Systemd will set the LISTEN_PID to current pid, but for our
	// tests this would be hard to simulate, so we also allow the parent pid.
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || (pid != os.Getpid() && pid != os.Getppid()) {
		return nil, nil
	}

	nfds, err := strconv.Atoi(os.Getenv("LISTEN_FDS"))
	if err != nil || nfds == 0 {
		return nil, nil
	}

	names := strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")

	files := make([]*os.File, 0, nfds)
	for fd := listenFdsStart; fd < listenFdsStart+nfds; fd++ {
		syscall.CloseOnExec(fd)
		name := "LISTEN_FD_" + strconv.Itoa(fd)
		offset := fd - listenFdsStart
		if offset < len(names) && len(names[offset]) > 0 {
			name = names[offset]
		}
		files = append(files, os.NewFile(uintptr(fd), name))
	}

	listeners := make([]net.Listener, 0, nfds)
	for _, file := range files {
		listener, err := net.FileListener(file)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}
