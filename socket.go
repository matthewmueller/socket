package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

// Listen creates a new listener based on the addr
func Listen(addr string) (net.Listener, error) {
	// Otherwise create a new listener based on the addr
	url, err := Parse(addr)
	if err != nil {
		return nil, err
	}

	// Handle unix, tcp, and fd schemes
	switch url.Scheme {
	case "unix":
		// Unix domain socket paths can't be more than 103 characters long
		if len(addr) > 103 {
			return nil, fmt.Errorf("socket: unix path too long %q", addr)
		}
		addr, err := net.ResolveUnixAddr("unix", addr)
		if err != nil {
			return nil, err
		}
		ln, err := net.ListenUnix("unix", addr)
		if err != nil {
			return nil, err
		}
		return ln, nil

	case "fd":
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

		// Otherwise, bind to a TCP port
	default:
		addr, err := net.ResolveTCPAddr("tcp", url.Host)
		if err != nil {
			return nil, err
		}
		ln, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return nil, err
		}
		return ln, nil
	}
}

// Serve the handler at address. // When the context is canceled, the server
// will be gracefully shutdown.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	// Create the HTTP server
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: handler,
	}
	// Make the server shutdownable
	shutdownCh := shutdown(ctx, server)
	// Serve requests
	if err := server.Serve(listener); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	// Handle any errors that occurred while shutting down
	if err := <-shutdownCh; err != nil {
		if !errors.Is(err, context.Canceled) {
			return err
		}
	}
	return nil
}

// ListenAndServe is a convenience function that combines Listen and Serve.
// When the context is canceled, the server will be gracefully shutdown.
func ListenAndServe(ctx context.Context, addr string, handler http.Handler) error {
	ln, err := Listen(addr)
	if err != nil {
		return err
	}
	return Serve(ctx, ln, handler)
}

// Shutdown the server when the context is canceled
func shutdown(ctx context.Context, server *http.Server) <-chan error {
	shutdown := make(chan error, 1)
	go func() {
		<-ctx.Done()
		// Wait for one more interrupt to force an immediate shutdown, otherwise
		// take as much time as needed to finish ongoing requests
		forceCtx := trap(context.Background(), os.Interrupt)
		if err := server.Shutdown(forceCtx); err != nil {
			shutdown <- err
		}
		close(shutdown)
	}()
	return shutdown
}

// Trap cancels the context based on a signal
func trap(ctx context.Context, signals ...os.Signal) context.Context {
	ret, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, len(signals))
	go func() {
		<-ch
		signal.Stop(ch)
		cancel()
	}()
	signal.Notify(ch, signals...)
	return ret
}
