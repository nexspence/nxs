package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_SearchAssets_PrefixFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("repository") != "raw-hosted" {
			t.Errorf("repository not sent: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[
			{"id":"a1","repository":"raw-hosted","path":"dist/app.tar.gz","fileSize":10},
			{"id":"a2","repository":"raw-hosted","path":"other/x.txt","fileSize":5}
		],"continuationToken":null}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	assets, err := c.SearchAssets("raw-hosted", "dist/")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 1 || assets[0].Path != "dist/app.tar.gz" {
		t.Errorf("prefix filter failed: %+v", assets)
	}
}

func TestClient_SearchAssets_Pagination(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("continuationToken") == "" {
			page++
			w.Write([]byte(`{"items":[{"id":"a1","path":"p1"}],"continuationToken":"50"}`))
			return
		}
		w.Write([]byte(`{"items":[{"id":"a2","path":"p2"}],"continuationToken":null}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	assets, err := c.SearchAssets("raw-hosted", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets across pages, got %d", len(assets))
	}
}
