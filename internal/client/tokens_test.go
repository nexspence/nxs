package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_TokenList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tokens" || r.Method != http.MethodGet {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"t1","name":"ci","scopes":["read"]}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	toks, err := c.TokenList()
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 1 || toks[0].Name != "ci" {
		t.Errorf("unexpected: %+v", toks)
	}
}

func TestClient_TokenCreate(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tokens" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"t2","name":"deploy","token":"nxs_secret"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	days := 30
	tok, err := c.TokenCreate("deploy", []string{"write"}, &days)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "nxs_secret" {
		t.Errorf("expected plaintext token, got %q", tok.Token)
	}
	if gotBody["name"] != "deploy" {
		t.Errorf("name not sent: %+v", gotBody)
	}
	if gotBody["expiresInDays"].(float64) != 30 {
		t.Errorf("expiresInDays not sent: %+v", gotBody)
	}
}

func TestClient_TokenDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.TokenDelete("t2"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/tokens/t2" {
		t.Errorf("unexpected %s %s", gotMethod, gotPath)
	}
}
