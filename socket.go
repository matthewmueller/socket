package socket

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

// Listen creates a new listener based on the addr
func Listen(addr string) (net.Listener, error) {
	// If the addr is empty, listen on a random port
	if addr == "" {
		addr = ":0"
	}

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
		// Listen on a file descriptor
		return listenFd(url)

	default:
		// Otherwise, bind to a TCP port
		return listenTcp(url)
	}
}

// listenTcp listens on a TCP port
func listenTcp(url *url.URL) (net.Listener, error) {
	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		return nil, fmt.Errorf("socket: invalid address %q", url.Host)
	}
	start, end, err := portRange(port)
	if err != nil {
		return nil, err
	}
	for port := start; port <= end; port++ {
		addr, err := net.ResolveTCPAddr("tcp", host+":"+strconv.Itoa(port))
		if err != nil {
			return nil, err
		}
		ln, err := net.ListenTCP("tcp", addr)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) && !strings.Contains(err.Error(), "address already in use") {
				return nil, err
			}
			continue
		}
		return ln, nil
	}
	return nil, fmt.Errorf("socket: no available ports in range %d-%d", start, end)
}

func portRange(port string) (start int, end int, err error) {
	s := strings.ReplaceAll(port, "x", "0")
	e := strings.ReplaceAll(port, "x", "9")
	start, err = strconv.Atoi(s)
	if err != nil {
		return 0, 0, err
	}
	end, err = strconv.Atoi(e)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
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
