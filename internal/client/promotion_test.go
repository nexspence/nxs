package client_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_PromotionRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/promotion/rules" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"r1","name":"to-prod","from_repo":"staging","to_repo":"prod","require_manual_approval":true}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	rules, err := c.PromotionRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].FromRepo != "staging" || !rules[0].RequireManualApproval {
		t.Errorf("unexpected: %+v", rules)
	}
}

func TestClient_Promote(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/promotion/promote" || r.Method != http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"requests":[{"id":"pr1","rule_id":"r1","component_id":"c1","status":"pending"}]}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	reqs, err := c.Promote("r1", []string{"c1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].ID != "pr1" {
		t.Errorf("unexpected: %+v", reqs)
	}
	if gotBody["rule_id"] != "r1" {
		t.Errorf("rule_id not sent: %+v", gotBody)
	}
	ids := gotBody["component_ids"].([]any)
	if len(ids) != 1 || ids[0] != "c1" {
		t.Errorf("component_ids not sent: %+v", gotBody)
	}
}

func TestClient_PromotionRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "pending" {
			t.Errorf("status filter not sent, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"pr1","status":"pending","component_id":"c1"}]`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	reqs, err := c.PromotionRequests("pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 1 || reqs[0].ID != "pr1" {
		t.Errorf("unexpected: %+v", reqs)
	}
}

func TestClient_PromotionApproveReject(t *testing.T) {
	var paths []string
	var rejectBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/v1/promotion/requests/pr1/reject" {
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &rejectBody)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	if err := c.PromotionApprove("pr1"); err != nil {
		t.Fatal(err)
	}
	if err := c.PromotionReject("pr1", "bad scan"); err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/api/v1/promotion/requests/pr1/approve" {
		t.Errorf("approve path wrong: %v", paths)
	}
	if rejectBody["reason"] != "bad scan" {
		t.Errorf("reason not sent: %+v", rejectBody)
	}
}
