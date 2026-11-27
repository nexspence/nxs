package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func tempFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "acme_1.0_amd64.deb")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestClient_Push_SendsQuery(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK) // a join answers 200, not 201
	}))
	defer srv.Close()

	c := client.New(srv.URL, "tok")
	err := c.Push("apt-acme", "pool/main/acme_1.0_amd64.deb", tempFile(t, "deb"),
		url.Values{"suite": {"acme-1.5", "acme-1.5.1"}}, nil)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if gotPath != "/repository/apt-acme/pool/main/acme_1.0_amd64.deb" {
		t.Errorf("path %q", gotPath)
	}
	if !reflect.DeepEqual(gotQuery["suite"], []string{"acme-1.5", "acme-1.5.1"}) {
		t.Errorf("suite %v", gotQuery["suite"])
	}
	if gotBody != "deb" {
		t.Errorf("body %q", gotBody)
	}
}

func TestClient_Push_NoQueryLeavesTheURLAlone(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := client.New(srv.URL, "tok").Push("raw", "a/b.bin", tempFile(t, "x"), nil, nil); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if gotRawQuery != "" {
		t.Errorf("unexpected query %q", gotRawQuery)
	}
}

// The server's reason reaches the user: for a conflict it names where the
// package is already stored.
func TestClient_Push_ReportsTheServersReason(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":"acme 1.0 (amd64) is already stored at /pool/main/acme_1.0_amd64.deb"}`)
	}))
	defer srv.Close()

	err := client.New(srv.URL, "tok").Push("apt-acme", "pool/other/acme.deb", tempFile(t, "x"), url.Values{"suite": {"s"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "409") || !strings.Contains(err.Error(), "already stored at /pool/main/acme_1.0_amd64.deb") {
		t.Fatalf("want the 409 and its reason, got %v", err)
	}
}

func TestClient_Push_NonJSONErrorKeepsTheStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "<html>bad gateway</html>")
	}))
	defer srv.Close()

	err := client.New(srv.URL, "tok").Push("raw", "a.bin", tempFile(t, "x"), nil, nil)
	if err == nil || err.Error() != "upload failed: server returned 502" {
		t.Fatalf("got %v", err)
	}
}
