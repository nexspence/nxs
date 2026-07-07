package client_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_BlobStoreCompact(t *testing.T) {
	var gotPath, gotQuery, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"store":"default","scannedBlobs":5,"orphans":2,"freedBytes":2048,"dryRun":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	res, err := c.BlobStoreCompact("default", true, "24h")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/blobstores/default/compact" {
		t.Errorf("unexpected %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(gotQuery, "dry_run=true") || !strings.Contains(gotQuery, "min_age=24h") {
		t.Errorf("unexpected query %q", gotQuery)
	}
	if res.Orphans != 2 || res.FreedBytes != 2048 || !res.DryRun || res.Store != "default" {
		t.Errorf("unexpected result %+v", res)
	}
}
