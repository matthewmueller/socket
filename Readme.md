# Socket

[![Go Reference](https://pkg.go.dev/badge/github.com/matthewmueller/socket.svg)](https://pkg.go.dev/github.com/matthewmueller/socket)

Socket utilities for Go.

## Features

- Familiar API
- Listening and dial unix domain sockets
- Graceful server shutdown with `context.Context`
- [Systemd socket activation](https://0pointer.de/blog/projects/socket-activation.html)
- No third-party dependencies

## Install

```sh
go get github.com/matthewmueller/socket
```

## Example

```go
package main

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

func main() {
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
}
```

## Development

First, clone the repo:

```sh
git clone https://github.com/matthewmueller/socket
cd socket
```

Next, install dependencies:

```sh
go mod tidy
```

Finally, try running the tests:

```sh
go test ./...
```

## License

MIT
