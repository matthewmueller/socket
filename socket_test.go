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
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/matthewmueller/signals"
	"github.com/matthewmueller/socket"
	"github.com/matthewmueller/testchild"
	"golang.org/x/sync/errgroup"
)

func TestLoadTCP(t *testing.T) {
	is := is.New(t)
	listener, err := socket.Listen(":0")
	is.NoErr(err)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(r.URL.Path))
		}),
	}
	go server.Serve(listener)
	transport, err := socket.Transport(listener.Addr().String())
	is.NoErr(err)
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}
	res, err := client.Get("http://" + listener.Addr().String() + "/hello")
	is.NoErr(err)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.Equal(string(body), "/hello")
	server.Shutdown(context.Background())
}

func TestLoadNumberOnly(t *testing.T) {
	is := is.New(t)
	listener, err := socket.Listen("0")
	is.NoErr(err)
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(r.URL.Path))
		}),
	}
	go server.Serve(listener)
	transport, err := socket.Transport(listener.Addr().String())
	is.NoErr(err)
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}
	res, err := client.Get("http://" + listener.Addr().String() + "/hello")
	is.NoErr(err)
	body, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.Equal(string(body), "/hello")
	server.Shutdown(context.Background())
}

func TestDial(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	listener, err := socket.Listen(":0")
	is.NoErr(err)
	defer listener.Close()
	msg := "hello world"
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			incoming := make([]byte, len(msg))
			if _, err := io.ReadFull(conn, incoming); err != nil {
				conn.Close()
				return
			}
			conn.Write([]byte(string(incoming)))
			conn.Write([]byte(string(incoming)))
			conn.Close()
		}
	}()
	conn, err := socket.Dial(ctx, listener.Addr().String())
	is.NoErr(err)
	defer conn.Close()
	conn.Write([]byte(msg))
	outgoing := make([]byte, len(msg)*2)
	_, err = io.ReadFull(conn, outgoing)
	is.NoErr(err)
	is.Equal(string(outgoing), msg+msg)
}

func TestUDSCleanup(t *testing.T) {
	is := is.New(t)
	listener, err := socket.Listen("./test.sock")
	is.NoErr(err)
	defer listener.Close()
	is.NoErr(listener.Close())
	stat, err := os.Stat("test.sock")
	is.True(errors.Is(err, os.ErrNotExist))
	is.Equal(stat, nil)
}

func TestListenPortTooHigh(t *testing.T) {
	is := is.New(t)
	ln0, err := socket.Listen(":65536")
	ae, ok := err.(*net.AddrError)
	is.True(ok)
	is.Equal(ae.Err, "invalid port")
	is.Equal(ln0, nil)
}

func TestServe(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := socket.Listen(":0")
	is.NoErr(err)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(205)
	})
	eg := new(errgroup.Group)
	eg.Go(func() error { return socket.Serve(ctx, listener, handler) })
	res, err := http.Get("http://" + listener.Addr().String())
	is.NoErr(err)
	is.Equal(res.StatusCode, 205)
	cancel()
	eg.Wait()
	res, err = http.Get("http://" + listener.Addr().String())
	is.True(err != nil)
	is.True(res == nil)
	is.True(strings.Contains(err.Error(), `connection refused`)) // should have stopped
}

func TestListenAndServe(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "/test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(205)
	})
	eg := new(errgroup.Group)
	eg.Go(func() error { return socket.ListenAndServe(ctx, socketPath, handler) })
	transport, err := socket.Transport(socketPath)
	is.NoErr(err)
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}
	attempts := 0
	for {
		if attempts > 5 {
			is.Fail() // should have connected by now
		}
		res, err := client.Get("http://localhost")
		if err != nil {
			attempts++
			time.Sleep(50 * time.Millisecond)
			continue
		}
		is.Equal(res.StatusCode, 205)
		break
	}
	cancel()
	eg.Wait()
	res, err := http.Get("http://localhost")
	is.True(err != nil)
	is.True(res == nil)
	is.True(strings.Contains(err.Error(), `connection refused`)) // should have stopped
}

func TestListenAndServeFd(t *testing.T) {
	parent := func(t testing.TB, cmd *exec.Cmd) {
		is := is.New(t)
		ln, err := socket.Listen(":0")
		is.NoErr(err)
		defer ln.Close()
		file, err := ln.(*net.TCPListener).File()
		is.NoErr(err)
		cmd.ExtraFiles = append(cmd.ExtraFiles, file)
		is.NoErr(cmd.Start())

		// Test the connection
		res, err := http.Get("http://" + ln.Addr().String())
		is.NoErr(err)
		is.Equal(res.StatusCode, 205)

		// Send an interrupt signal
		is.NoErr(cmd.Process.Signal(os.Interrupt))

		// Wait for the process to exit gracefully
		is.NoErr(cmd.Wait())

	}

	child := func(t testing.TB) {
		is := is.New(t)
		ctx := signals.Trap(context.Background(), os.Interrupt)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(205)
		})
		if err := socket.ListenAndServe(ctx, "fd:3", handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				is.NoErr(err)
			}
		}
	}

	testchild.Run(t, parent, child)
}

func TestListenEmpty(t *testing.T) {
	is := is.New(t)
	ln0, err := socket.Listen("")
	is.NoErr(err)
	is.NoErr(ln0.Close())
}
