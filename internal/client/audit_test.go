package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

func TestClient_AuditList_SendsTheQueryParameters(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"id":1,"eventTime":"2026-01-02T03:04:05Z","username":"admin","domain":"REPOSITORY","action":"CREATE","result":"success"}],"total":1}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	entries, err := c.AuditList(client.AuditParams{
		Limit:    25,
		Offset:   50,
		Domain:   "REPOSITORY",
		Action:   "CREATE",
		Username: "admin",
		From:     "2026-01-01",
		To:       "2026-01-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/service/rest/v1/audit" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	want := "action=CREATE&domain=REPOSITORY&from=2026-01-01&limit=25&offset=50&to=2026-01-31&username=admin"
	if gotQuery != want {
		t.Errorf("unexpected query:\n got: %s\nwant: %s", gotQuery, want)
	}
	if len(entries) != 1 || entries[0].Username != "admin" || entries[0].Result != "success" {
		t.Errorf("unexpected entries: %+v", entries)
	}
}

func TestClient_AuditList_SendsNoParameterThatWasNotSet(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[],"total":0}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL, "token")
	entries, err := c.AuditList(client.AuditParams{})
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "" {
		t.Errorf("unexpected query: %s", gotQuery)
	}
	if len(entries) != 0 {
		t.Errorf("unexpected entries: %+v", entries)
	}
}
