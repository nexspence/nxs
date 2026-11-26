package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

// userServer answers the calls a user change makes: the user itself, the
// role list, the role assignment and the update.
func userServer(t *testing.T, gotUpdatePath *string, gotUpdateBody *map[string]any, gotRoleIDs *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/security/users/ci-bot":
			w.Write([]byte(`{"userId":"ci-bot","roles":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/service/rest/v1/security/roles":
			w.Write([]byte(`[
				{"id":"11111111-1111-1111-1111-111111111111","name":"nx-admin"},
				{"id":"22222222-2222-2222-2222-222222222222","name":"nx-deployer"}
			]`))
		case r.Method == http.MethodPut && r.URL.Path == "/service/rest/v1/security/users/ci-bot/roles":
			b, _ := io.ReadAll(r.Body)
			var body map[string][]string
			json.Unmarshal(b, &body)
			*gotRoleIDs = body["roleIds"]
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/service/rest/v1/security/users/ci-bot":
			*gotUpdatePath = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, gotUpdateBody)
			w.Write([]byte(`{"userId":"ci-bot","emailAddress":"ci-bot@example.com"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
}

// An update names only the fields it changes: the server keeps everything
// the payload leaves out, so sending the whole user would overwrite it.
func TestClient_UserUpdate_SendsChangedFieldsOnly(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	var gotRoleIDs []string
	srv := userServer(t, &gotPath, &gotBody, &gotRoleIDs)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserUpdate("ci-bot", "", "Bot", "ci-bot@example.com", nil); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/service/rest/v1/security/users/ci-bot" {
		t.Errorf("update went to %q", gotPath)
	}
	if gotBody["lastName"] != "Bot" {
		t.Errorf("lastName = %v, want Bot", gotBody["lastName"])
	}
	if gotBody["emailAddress"] != "ci-bot@example.com" {
		t.Errorf("emailAddress = %v", gotBody["emailAddress"])
	}
	if _, ok := gotBody["firstName"]; ok {
		t.Errorf("firstName was sent although it did not change: %v", gotBody)
	}
}

// The roles are given by name and reach the server as the ids it stores.
func TestClient_UserUpdate_ResolvesRoleNames(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	var gotRoleIDs []string
	srv := userServer(t, &gotPath, &gotBody, &gotRoleIDs)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserUpdate("ci-bot", "", "", "", []string{"nx-deployer"}); err != nil {
		t.Fatal(err)
	}
	if len(gotRoleIDs) != 1 || gotRoleIDs[0] != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("roleIds = %v, want the deployer id", gotRoleIDs)
	}
}

// A role the server does not know is an error, not a silent skip.
func TestClient_UserUpdate_UnknownRole(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	var gotRoleIDs []string
	srv := userServer(t, &gotPath, &gotBody, &gotRoleIDs)
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserUpdate("ci-bot", "", "", "", []string{"nx-nope"}); err == nil {
		t.Fatal("expected an error for an unknown role")
	}
}

func TestClient_UserDelete(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/service/rest/v1/security/users/ci-bot" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserDelete("ci-bot"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/service/rest/v1/security/users/ci-bot" {
		t.Errorf("delete went to %q", gotPath)
	}
}

// The password travels in the body under the name the handler binds, and
// nowhere else.
func TestClient_UserChangePassword(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.UserChangePassword("ci-bot", "Str0ng-Test-Pass"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/service/rest/v1/security/users/ci-bot/change-password" {
		t.Errorf("change-password went to %q", gotPath)
	}
	if gotBody["newPassword"] != "Str0ng-Test-Pass" {
		t.Errorf("newPassword = %v", gotBody["newPassword"])
	}
}
