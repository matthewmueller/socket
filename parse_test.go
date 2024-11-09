package socket_test

import (
	"testing"

	"github.com/matthewmueller/socket"
)

func equal(t testing.TB, input string, expect string) {
	t.Helper()
	u, err := socket.Parse(input)
	if err != nil {
		if err.Error() != expect {
			t.Errorf("expected %q, got %q", expect, err.Error())
		}
		return
	}
	if u.String() != expect {
		t.Errorf("expected %q, got %q", expect, u.String())
	}
}

func TestParse5000(t *testing.T) {
	equal(t, "5000", "http://127.0.0.1:5000")
}

func TestParseColon5000(t *testing.T) {
	equal(t, ":5000", "http://127.0.0.1:5000")
}

func TestParse0(t *testing.T) {
	equal(t, "0", "http://127.0.0.1:0")
}

func TestParse0000(t *testing.T) {
	equal(t, "0.0.0.0", "http://0.0.0.0:3000")
}

func TestParse127001(t *testing.T) {
	equal(t, "127.0.0.1", "http://127.0.0.1:3000")
}

func TestParse1270015000(t *testing.T) {
	equal(t, "127.0.0.1:5000", "http://127.0.0.1:5000")
}

func TestParseLocalhost(t *testing.T) {
	equal(t, "localhost", "http://localhost:3000")
}

func TestParseOtherhost(t *testing.T) {
	equal(t, "otherhost", "http://otherhost:3000")
}

func TestParseTmpSock(t *testing.T) {
	equal(t, "/tmp.sock", "unix:///tmp.sock")
}

func TestParseWhateverTmpSock(t *testing.T) {
	equal(t, "/whatever/tmp.sock", "unix:///whatever/tmp.sock")
}

func TestParseDotWhateverTmpSock(t *testing.T) {
	equal(t, "./whatever/tmp.sock", "unix://./whatever/tmp.sock")
}

func TestParseHttps(t *testing.T) {
	equal(t, "https:", "https://127.0.0.1:443")
}

func TestParseHttpsLocalhost8000(t *testing.T) {
	equal(t, "https://localhost:8000/a/b/c", "https://localhost:8000/a/b/c")
}

func TestParse80Ab(t *testing.T) {
	equal(t, "80.ab", `urlx: unable to parse "80.ab"`)
}

func TestParseHttp12700149341(t *testing.T) {
	equal(t, "http://127.0.0.1:49341", "http://127.0.0.1:49341")
}

func TestParseBracketColon50516(t *testing.T) {
	equal(t, "[::]:50516", "http://[::]:50516")
}

func TestParseBracketColon443(t *testing.T) {
	equal(t, "[::]:443", "https://[::]:443")
}

func TestParseBracketColon80(t *testing.T) {
	equal(t, "[::]:80", "http://[::]:80")
}

func TestParseUnixRel(t *testing.T) {
	equal(t, "unix://./some/path", "unix://./some/path")
}

func TestParseUnixAbs(t *testing.T) {
	equal(t, "unix:///some/path", "unix:///some/path")
}

func TestParseFd3(t *testing.T) {
	equal(t, "fd:3", "fd://3")
}

func TestParseFd20(t *testing.T) {
	equal(t, "fd:20", "fd://20")
}
