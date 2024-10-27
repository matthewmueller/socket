package socket_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/matthewmueller/socket"
	"golang.org/x/sync/errgroup"
)

func ExampleListenAndServe() {
	// Create a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	})

	// Create the socket path
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	socketPath := filepath.Join(tmpdir, "unix.sock")

	// Listen on the unix domain socket
	ln, err := socket.Listen(socketPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	// Start the server
	eg := new(errgroup.Group)
	eg.Go(func() error {
		return socket.Serve(ctx, ln, handler)
	})

	// Create the transport over the unix domain socket
	transport, err := socket.Transport(socketPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create the client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}

	// Make a request
	res, err := client.Get("http://localhost")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read the response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Cancel the context
	cancel()

	// Wait for the server to shutdown
	if err := eg.Wait(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
			return
		}
	}

	// Output the response
	fmt.Println(string(body))
	// Output: hello world!
}
