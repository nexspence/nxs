package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

// readBody decodes a request body into a map, so a test can state the body it
// expects without depending on the order the keys were written in.
func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}
	return body
}

func TestClient_RepoUpdate_SendsFormatConfigAndTopLevelFields(t *testing.T) {
	var (
		putPath string
		putBody map[string]any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/repositories/npm-hosted":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-hosted","format":"npm","type":"hosted","online":true}`))
		case r.Method == http.MethodPut:
			putPath = r.URL.Path
			putBody = readBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-hosted"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	online := false
	description := "internal packages"
	writePolicy := "deny"
	if err := c.RepoUpdate("npm-hosted", client.RepoUpdateOptions{
		Online:      &online,
		Description: &description,
		WritePolicy: &writePolicy,
	}); err != nil {
		t.Fatal(err)
	}

	if putPath != "/service/rest/v1/repositories/npm/hosted/npm-hosted" {
		t.Errorf("unexpected update path: %s", putPath)
	}
	want := map[string]any{
		"online":      false,
		"description": "internal packages",
		"formatConfig": map[string]any{
			"write_policy": "deny",
		},
	}
	if !reflect.DeepEqual(putBody, want) {
		t.Errorf("unexpected update body:\n got: %v\nwant: %v", putBody, want)
	}
}

func TestClient_RepoUpdate_SendsNoFieldThatWasNotSet(t *testing.T) {
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-hosted","format":"npm","type":"hosted"}`))
		case r.Method == http.MethodPut:
			putBody = readBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-hosted"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RepoUpdate("npm-hosted", client.RepoUpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(putBody) != 0 {
		t.Errorf("expected an empty body, got: %v", putBody)
	}
}

func TestClient_RepoUpdate_KeepsTheStoredFormatConfig(t *testing.T) {
	var putBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-group","format":"npm","type":"group","formatConfig":{"member_names":["npm-hosted","npm-proxy"]}}`))
		case r.Method == http.MethodPut:
			putBody = readBody(t, r)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"npm-group"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	writePolicy := "deny"
	if err := c.RepoUpdate("npm-group", client.RepoUpdateOptions{WritePolicy: &writePolicy}); err != nil {
		t.Fatal(err)
	}
	formatConfig, ok := putBody["formatConfig"].(map[string]any)
	if !ok {
		t.Fatalf("expected a formatConfig, got: %v", putBody)
	}
	if formatConfig["write_policy"] != "deny" {
		t.Errorf("write policy not sent: %v", formatConfig)
	}
	members, ok := formatConfig["member_names"].([]any)
	if !ok || len(members) != 2 {
		t.Errorf("the stored members were dropped: %v", formatConfig)
	}
}

func TestClient_ComponentDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.ComponentDelete("comp-1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/service/rest/v1/components/comp-1" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
}
