package client

import (
	"encoding/json"
	"time"
)

type PromotionRule struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	FromRepo              string `json:"from_repo"`
	ToRepo                string `json:"to_repo"`
	PathFilter            string `json:"path_filter,omitempty"`
	RequireScanPass       bool   `json:"require_scan_pass"`
	RequireManualApproval bool   `json:"require_manual_approval"`
}

type PromotionRequest struct {
	ID          string    `json:"id"`
	RuleID      string    `json:"rule_id"`
	ComponentID string    `json:"component_id"`
	Status      string    `json:"status"`
	RequestedBy string    `json:"requested_by"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *Client) PromotionRules() ([]PromotionRule, error) {
	resp, err := c.r.R().Get("/api/v1/promotion/rules")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var rules []PromotionRule
	return rules, json.Unmarshal(resp.Body(), &rules)
}

func (c *Client) Promote(ruleID string, componentIDs []string) ([]PromotionRequest, error) {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"rule_id": ruleID, "component_ids": componentIDs}).
		Post("/api/v1/promotion/promote")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var out struct {
		Requests []PromotionRequest `json:"requests"`
	}
	return out.Requests, json.Unmarshal(resp.Body(), &out)
}

func (c *Client) PromotionRequests(status string) ([]PromotionRequest, error) {
	req := c.r.R()
	if status != "" {
		req = req.SetQueryParam("status", status)
	}
	resp, err := req.Get("/api/v1/promotion/requests")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var reqs []PromotionRequest
	return reqs, json.Unmarshal(resp.Body(), &reqs)
}

func (c *Client) PromotionApprove(id string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		Post("/api/v1/promotion/requests/" + id + "/approve")
	if err != nil {
		return err
	}
	return checkErr(resp)
}

func (c *Client) PromotionReject(id, reason string) error {
	resp, err := c.r.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]any{"reason": reason}).
		Post("/api/v1/promotion/requests/" + id + "/reject")
	if err != nil {
		return err
	}
	return checkErr(resp)
}
