package cmd

import (
	"testing"

	"github.com/nexspence/nxs/internal/client"
)

// fakeResolver satisfies the lookups resolveComponents needs.
type fakeResolver struct {
	rules  []client.PromotionRule
	search []client.Component
}

func (f fakeResolver) PromotionRules() ([]client.PromotionRule, error) { return f.rules, nil }
func (f fakeResolver) Search(p client.SearchParams) ([]client.Component, error) {
	return f.search, nil
}

func TestResolveComponents_ByCoordinates(t *testing.T) {
	r := fakeResolver{
		rules:  []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}},
		search: []client.Component{{ID: "c1", Group: "com.acme", Name: "app", Version: "1.0"}},
	}
	ruleID, ids, err := resolveComponents(r, "to-prod", []string{"com.acme:app:1.0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ruleID != "r1" || len(ids) != 1 || ids[0] != "c1" {
		t.Errorf("unexpected: rule=%s ids=%v", ruleID, ids)
	}
}

func TestResolveComponents_NoMatch(t *testing.T) {
	r := fakeResolver{
		rules:  []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}},
		search: []client.Component{{ID: "c1", Group: "com.acme", Name: "app", Version: "2.0"}},
	}
	_, _, err := resolveComponents(r, "to-prod", []string{"com.acme:app:1.0"}, nil)
	if err == nil {
		t.Fatal("expected no-match error")
	}
}

func TestResolveComponents_RuleNotFound(t *testing.T) {
	r := fakeResolver{rules: []client.PromotionRule{{ID: "r1", Name: "to-prod"}}}
	_, _, err := resolveComponents(r, "nope", nil, []string{"c9"})
	if err == nil {
		t.Fatal("expected rule-not-found error")
	}
}

func TestResolveComponents_RawIDBypass(t *testing.T) {
	r := fakeResolver{rules: []client.PromotionRule{{ID: "r1", Name: "to-prod", FromRepo: "staging"}}}
	ruleID, ids, err := resolveComponents(r, "r1", nil, []string{"raw-uuid"})
	if err != nil {
		t.Fatal(err)
	}
	if ruleID != "r1" || len(ids) != 1 || ids[0] != "raw-uuid" {
		t.Errorf("unexpected: rule=%s ids=%v", ruleID, ids)
	}
}
