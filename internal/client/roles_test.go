package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

// roleServer answers the three calls a role assignment makes: the user, the
// role list, and the assignment itself.
func roleServer(t *testing.T, userRoles []string, gotPath *string, gotBody *map[string][]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/security/users/ci-bot":
			json.NewEncoder(w).Encode(map[string]any{
				"userId": "ci-bot",
				"roles":  userRoles,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/security/roles":
			w.Write([]byte(`[
				{"id":"11111111-1111-1111-1111-111111111111","name":"nx-admin"},
				{"id":"22222222-2222-2222-2222-222222222222","name":"nx-deployer"}
			]`))
		case r.Method == http.MethodPut && r.URL.Path == "/service/rest/v1/security/users/ci-bot/roles":
			*gotPath = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, gotBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

// The server ignores a userId filter on the user list, so the assignment must
// read the user by its own path and must not take the first user of a list.
func TestClient_RoleAssign_KeepsExistingRoles(t *testing.T) {
	var gotPath string
	var gotBody map[string][]string
	srv := roleServer(t, []string{"nx-admin"}, &gotPath, &gotBody)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RoleAssign("ci-bot", []string{"nx-deployer"}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/service/rest/v1/security/users/ci-bot/roles" {
		t.Errorf("assignment went to %q", gotPath)
	}
	want := []string{"11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222"}
	got := gotBody["roleIds"]
	if len(got) != len(want) {
		t.Fatalf("roleIds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roleIds = %v, want %v", got, want)
		}
	}
}

// Assigning a role the user already has must not fail or duplicate it.
func TestClient_RoleAssign_IsIdempotent(t *testing.T) {
	var gotPath string
	var gotBody map[string][]string
	srv := roleServer(t, []string{"nx-deployer"}, &gotPath, &gotBody)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RoleAssign("ci-bot", []string{"nx-deployer"}); err != nil {
		t.Fatal(err)
	}
	if ids := gotBody["roleIds"]; len(ids) != 1 {
		t.Errorf("roleIds = %v, want one entry", ids)
	}
}

// A role name the server does not know is an error, not a silent skip.
func TestClient_RoleAssign_UnknownRole(t *testing.T) {
	var gotPath string
	var gotBody map[string][]string
	srv := roleServer(t, []string{}, &gotPath, &gotBody)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RoleAssign("ci-bot", []string{"nx-nope"}); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}

// Revoking keeps the roles that were not named.
func TestClient_RoleRevoke(t *testing.T) {
	var gotPath string
	var gotBody map[string][]string
	srv := roleServer(t, []string{"nx-admin", "nx-deployer"}, &gotPath, &gotBody)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.RoleRevoke("ci-bot", []string{"nx-admin"}); err != nil {
		t.Fatal(err)
	}
	ids := gotBody["roleIds"]
	if len(ids) != 1 || ids[0] != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("roleIds = %v, want the deployer id only", ids)
	}
}

// The create call takes role ids, not the names the user typed.
func TestClient_UserCreate_ResolvesRoleNames(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/security/roles":
			w.Write([]byte(`[{"id":"22222222-2222-2222-2222-222222222222","name":"nx-deployer"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/service/rest/v1/security/users":
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &gotBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"userId":"ci-bot"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserCreate("ci-bot", "", "", "ci-bot@example.com", "pw", []string{"nx-deployer"}); err != nil {
		t.Fatal(err)
	}
	roles, ok := gotBody["roles"].([]any)
	if !ok || len(roles) != 1 {
		t.Fatalf("roles = %v, want one entry", gotBody["roles"])
	}
	if roles[0] != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("roles = %v, want the deployer id", roles)
	}
}
