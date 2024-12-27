package socket_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/matthewmueller/socket"
)

func formatEq(t testing.TB, input string, expect string) {
	t.Helper()
	ln, err := socket.Listen(input)
	if err != nil {
		if err.Error() != expect {
			t.Errorf("expected %q, got %q", expect, err.Error())
		}
		return
	}
	defer ln.Close()
	actual := socket.Format(ln)
	if actual != expect {
		t.Errorf("expected %q, got %q", expect, actual)
	}
}

func formatContains(t testing.TB, input string, expect string) {
	t.Helper()
	ln, err := socket.Listen(input)
	if err != nil {
		if strings.Contains(err.Error(), expect) {
			t.Errorf("expected %q to contain %q", err.Error(), expect)
		}
		return
	}
	defer ln.Close()
	actual := socket.Format(ln)
	if !strings.Contains(actual, expect) {
		t.Errorf("expected %q to contain %q", actual, expect)
	}
}

func TestFormat(t *testing.T) {
	formatEq(t, "5000", "http://127.0.0.1:5000")
	formatEq(t, ":5000", "http://127.0.0.1:5000")

	formatEq(t, "0.0.0.0:4000", "http://0.0.0.0:4000")

	// Random
	formatContains(t, "", "http://127.0.0.1:")
	formatContains(t, "0", "http://127.0.0.1:")
	formatContains(t, ":0", "http://127.0.0.1:")

	// Socket
	socketPath := filepath.Join(t.TempDir(), "/test.sock")
	formatEq(t, socketPath, socketPath)
}
