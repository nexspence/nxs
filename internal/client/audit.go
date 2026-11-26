package client

import (
	"encoding/json"
	"strconv"
	"time"
)

// AuditEntry is one recorded action in the audit log.
type AuditEntry struct {
	ID         int64          `json:"id"`
	EventTime  time.Time      `json:"eventTime"`
	UserID     *string        `json:"userId,omitempty"`
	Username   string         `json:"username"`
	RemoteIP   string         `json:"remoteIp,omitempty"`
	UserAgent  string         `json:"userAgent,omitempty"`
	Domain     string         `json:"domain"`
	Action     string         `json:"action"`
	EntityType string         `json:"entityType,omitempty"`
	EntityID   string         `json:"entityId,omitempty"`
	EntityName string         `json:"entityName,omitempty"`
	Context    map[string]any `json:"context,omitempty"`
	Result     string         `json:"result"`
}

// AuditParams narrows an audit-log query. An empty field is not sent, so the
// server applies its own default for it.
//
// The audit endpoint pages by offset, not by a continuation token: it answers
// {"items": [...], "total": N} and takes an "offset" to resume from.
type AuditParams struct {
	Limit    int
	Offset   int
	Domain   string
	Action   string
	Username string
	From     string
	To       string
}

// auditResponse is the envelope the audit endpoint answers with.
type auditResponse struct {
	Items []AuditEntry `json:"items"`
	Total int          `json:"total"`
}

// AuditList returns the audit entries the server holds for the given filters.
func (c *Client) AuditList(params AuditParams) ([]AuditEntry, error) {
	req := c.r.R()
	if params.Limit > 0 {
		req = req.SetQueryParam("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		req = req.SetQueryParam("offset", strconv.Itoa(params.Offset))
	}
	if params.Domain != "" {
		req = req.SetQueryParam("domain", params.Domain)
	}
	if params.Action != "" {
		req = req.SetQueryParam("action", params.Action)
	}
	if params.Username != "" {
		req = req.SetQueryParam("username", params.Username)
	}
	if params.From != "" {
		req = req.SetQueryParam("from", params.From)
	}
	if params.To != "" {
		req = req.SetQueryParam("to", params.To)
	}

	resp, err := req.Get("/service/rest/v1/audit")
	if err != nil {
		return nil, err
	}
	if err := checkErr(resp); err != nil {
		return nil, err
	}
	var page auditResponse
	if err := json.Unmarshal(resp.Body(), &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}
