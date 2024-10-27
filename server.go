package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
)

// Listen creates a new listener based on the path
func Listen(path string) (net.Listener, error) {
	// Try activating the socket first
	lns, err := maybeActivate()
	if err != nil {
		return nil, err
	} else if len(lns) > 0 {
		return lns[0], nil
	}
	// Otherwise create a new listener based on the path
	url, err := Parse(path)
	if err != nil {
		return nil, err
	}
	// Empty host means the path is a unix domain socket
	if url.Host == "" {
		// Unix domain socket path can't be more than 103 characters long
		if len(path) > 103 {
			return nil, fmt.Errorf("socket: unix path too long %q", path)
		}
		addr, err := net.ResolveUnixAddr("unix", path)
		if err != nil {
			return nil, err
		}
		unix, err := net.ListenUnix("unix", addr)
		if err != nil {
			return nil, err
		}
		return unix, nil
	}
	// Otherwise, we listen on a TCP port
	addr, err := net.ResolveTCPAddr("tcp", url.Host)
	if err != nil {
		return nil, err
	}
	tcp, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	return tcp, nil
}

// Serve the handler at address
func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	// Create the HTTP server
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: handler,
	}
	// Make the server shutdownable
	shutdown := shutdown(ctx, server)
	// Serve requests
	if err := server.Serve(listener); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	// Handle any errors that occurred while shutting down
	if err := <-shutdown; err != nil {
		if !errors.Is(err, context.Canceled) {
			return err
		}
	}
	return nil
}

// ListenAndServe is a convenience function that combines Listen and Serve
func ListenAndServe(ctx context.Context, path string, handler http.Handler) error {
	ln, err := Listen(path)
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
		// Wait for one more interrupt to force an immediate shutdown
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
