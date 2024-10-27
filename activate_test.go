package socket_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/matthewmueller/socket"
	"golang.org/x/sync/errgroup"
)

func listen(addr string) (net.Listener, *http.Client, error) {
	listener, err := socket.Listen(addr)
	if err != nil {
		return nil, nil, err
	}
	transport, err := socket.Transport(listener.Addr().String())
	if err != nil {
		return nil, nil, err
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return listener, client, nil
}

func prepareEnv(prefix string, files ...*os.File) []string {
	if len(files) == 0 {
		return nil
	}
	return []string{
		prefix + "_FDS=" + strconv.Itoa(len(files)),
	}
}

func inject(extras *[]*os.File, env *[]string, prefix string, files ...*os.File) {
	if len(files) == 0 {
		return
	}
	environ := prepareEnv(prefix, files...)
	*extras = append(*extras, files...)
	*env = append(*env, environ...)
}

func TestUnixPassthrough(t *testing.T) {
	// Parent process
	parent := func(t testing.TB) {
		is := is.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dir := t.TempDir()
		appSocket, appClient, err := listen(filepath.Join(dir, "app.sock"))
		is.NoErr(err)
		defer appSocket.Close()
		appFile, err := appSocket.(*net.UnixListener).File()
		is.NoErr(err)
		// Ignore -test.count otherwise this will continue recursively
		var args []string
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-test.count=") {
				continue
			}
			args = append(args, arg)
		}
		cmd := exec.CommandContext(ctx, os.Args[0], append(args, "-test.v=true", "-test.run=^"+t.Name()+"$")...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "CHILD=1")
		cmd.Env = append(cmd.Env, "LISTEN_PID="+strconv.Itoa(os.Getpid()))
		inject(&cmd.ExtraFiles, &cmd.Env, "LISTEN", appFile)
		is.NoErr(cmd.Start())

		// Test app socket
		res, err := appClient.Get("http://unix/ping")
		is.NoErr(err)
		is.Equal(res.StatusCode, 200)
		body, err := io.ReadAll(res.Body)
		is.NoErr(err)
		is.Equal(string(body), "app pong")
		res, err = appClient.Get("http://unix/close")
		is.NoErr(err)
		is.Equal(res.StatusCode, 200)

		is.NoErr(cmd.Wait())
	}

	// Child process
	child := func(t testing.TB) {
		is := is.New(t)

		appListener, err := socket.Listen("anything")
		is.NoErr(err)

		// Manually flush response so we can shutdown the server in the handler
		flush := func(w http.ResponseWriter) {
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}

		// Serve app
		appServer := &http.Server{
			Addr: appListener.Addr().String(),
		}
		appServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/ping":
				w.Write([]byte("app pong"))
			case "/close":
				flush(w)
				is.NoErr(appServer.Shutdown(context.Background()))
			default:
				w.WriteHeader(404)
			}
		})

		serve := func(server *http.Server, listener net.Listener) error {
			if err := server.Serve(listener); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
			return nil
		}

		eg := new(errgroup.Group)
		eg.Go(func() error { return serve(appServer, appListener) })
		is.NoErr(eg.Wait())
	}

	if value := os.Getenv("CHILD"); value != "" {
		child(t)
	} else {
		parent(t)
	}
}
