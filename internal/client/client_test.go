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
