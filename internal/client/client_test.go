package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_HandlesUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "bad-token")
	err := c.HealthCheck()
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "invalid token or session expired" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_BearerHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "nxs_test_token")
	_ = c.HealthCheck()
	if gotAuth != "Bearer nxs_test_token" {
		t.Errorf("expected Bearer header, got %q", gotAuth)
	}
}

func TestClient_RepoList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/service/rest/v1/repositories" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"maven-releases","format":"maven2","type":"hosted","url":"http://x/repository/maven-releases","online":true}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	repos, err := c.RepoList("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "maven-releases" {
		t.Errorf("unexpected repos: %+v", repos)
	}
}

func TestClient_RepoDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RepoDelete("maven-releases"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if gotPath != "/service/rest/v1/repositories/maven-releases" {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

func TestClient_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"id":"abc","repository":"maven-releases","format":"maven2","name":"app","version":"1.0"}],"continuationToken":null}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	results, err := c.Search(client.SearchParams{Repo: "maven-releases"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "app" {
		t.Errorf("unexpected results: %+v", results)
	}
}
