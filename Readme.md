# Socket

[![Go Reference](https://pkg.go.dev/badge/github.com/matthewmueller/socket.svg)](https://pkg.go.dev/github.com/matthewmueller/socket)

Socket utilities for Go. This library is a superset of `http.ListenAndServe`.

## Features

- Familiar API
- Listen on and connect to Unix Domain Sockets
- Graceful server shutdown with `context.Context`
- Support for socket activation (see below)
- No third-party dependencies

## Install

```sh
go get github.com/matthewmueller/socket
```

## Examples

### Listen on a port

```go
socket.ListenAndServe(ctx, ":3000", handler)
```

### Listen on a specific host and port

```go
socket.ListenAndServe(ctx, "0.0.0.0:3000", handler)
```

### Listen on a Unix Domain Socket

```go
socket.ListenAndServe(ctx, "/some/unix.socket", handler)
```

### Create a client that can talk through a Unix Domain Socket

```go
client := &http.Client{
  Transport: socket.Transport("/some/unix.socket")
}
res, err := client.Get("http://localhost")
```

### Parse a Unix Domain Socket into a URL

```go
url, err := socket.Parse("./some/unix.socket")
url.String() // unix://./some/unix.socket
```

### Listen on a file descriptor (Socket Activation)

Tools like Systemd support passing a socket to the processes that it manages. This allows Systemd to manage the lifecycle of socket, not your server.

This greatly improves restart time and allows connections to hang (rather than be refused) until the server is online again. There's a test in [socket_test.go](./socket_test.go) with an example of how to do this in pure Go without Systemd.

```go
socket.ListenAndServe(ctx, "fd:3", handler)
```

There are some other benefits like lazily starting processes (think Lambda function cold starts) that are covered in more detail in https://0pointer.de/blog/projects/socket-activation.html.

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
