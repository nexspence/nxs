package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_CleanupCreate_SendsThePolicyFields(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"p1","name":"nxs-cli-test","format":"npm"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	err := c.CleanupCreate(client.CleanupPolicy{
		Name:         "nxs-cli-test",
		Format:       "npm",
		Criteria:     map[string]any{"artifactAgeDays": 30},
		ScheduleCron: "0 3 * * *",
		DryRun:       true,
		Scope:        client.CleanupScope{RepositoryName: "npm-hosted"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/service/rest/v1/cleanup-policies" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
	want := map[string]any{
		"name":            "nxs-cli-test",
		"format":          "npm",
		"criteria":        map[string]any{"artifactAgeDays": float64(30)},
		"scheduleCron":    "0 3 * * *",
		"enabled":         false,
		"dryRun":          true,
		"retainNVersions": float64(0),
		"scope": map[string]any{
			"repositoryName": "npm-hosted",
		},
	}
	if !reflect.DeepEqual(gotBody, want) {
		t.Errorf("unexpected create body:\n got: %v\nwant: %v", gotBody, want)
	}
}

func TestClient_CleanupUpdate_SendsThePolicyFieldsToThePolicy(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotBody = readBody(t, r)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"p1","name":"nxs-cli-test"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	err := c.CleanupUpdate("p1", client.CleanupPolicy{
		Name:     "nxs-cli-test",
		Format:   "*",
		Criteria: map[string]any{},
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPut || gotPath != "/service/rest/v1/cleanup-policies/p1" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotBody["name"] != "nxs-cli-test" || gotBody["enabled"] != true {
		t.Errorf("unexpected update body: %v", gotBody)
	}
	if _, ok := gotBody["id"]; ok {
		t.Errorf("the body carries the id the path already names: %v", gotBody)
	}
}

func TestClient_CleanupDelete(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.CleanupDelete("p1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/service/rest/v1/cleanup-policies/p1" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
}

func TestClient_CleanupPreview(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"assets":[{"path":"dist/app.tgz","repository":"npm-hosted","sizeBytes":10,"reason":"age 30d"}],"totalCount":1,"totalBytes":10}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	preview, err := c.CleanupPreview("p1")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/cleanup-policies/p1/preview" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if preview.TotalCount != 1 || preview.TotalBytes != 10 {
		t.Errorf("unexpected preview totals: %+v", preview)
	}
	if len(preview.Assets) != 1 || preview.Assets[0].Path != "dist/app.tgz" {
		t.Errorf("unexpected preview assets: %+v", preview.Assets)
	}
}

func TestClient_CleanupList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/service/rest/v1/cleanup-policies" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"p1","name":"nxs-cli-test","format":"npm","criteria":{"artifactAgeDays":30},"scope":{}}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	policies, err := c.CleanupList()
	if err != nil {
		t.Fatal(err)
	}
	if len(policies) != 1 || policies[0].Name != "nxs-cli-test" {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	if policies[0].Criteria["artifactAgeDays"] != float64(30) {
		t.Errorf("the criteria were not decoded: %+v", policies[0].Criteria)
	}
}

// The server answers with the policy's fields as its own JSON, so the client
// has to decode the same names.
func TestClient_CleanupPolicy_JsonNames(t *testing.T) {
	var policy client.CleanupPolicy
	body := `{"id":"p1","name":"nxs-cli-test","description":"d","format":"npm","criteria":{"artifactAgeDays":30},"scheduleCron":"0 3 * * *","enabled":true,"dryRun":false,"retainNVersions":2,"scope":{"repositoryName":"npm-hosted","pathPrefix":"/tmp/"},"lastRunAt":"2026-01-02T03:04:05Z","lastRunFreedBytes":22,"lastRunCount":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-02T00:00:00Z"}`
	if err := json.Unmarshal([]byte(body), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.ID != "p1" || policy.ScheduleCron != "0 3 * * *" || policy.RetainNVersions != 2 {
		t.Errorf("unexpected policy: %+v", policy)
	}
	if policy.Scope.RepositoryName != "npm-hosted" || policy.Scope.PathPrefix != "/tmp/" {
		t.Errorf("unexpected scope: %+v", policy.Scope)
	}
	if policy.LastRunFreed != 22 || policy.LastRunCount != 1 || policy.LastRunAt == nil {
		t.Errorf("unexpected run statistics: %+v", policy)
	}
}
