package cmd

import (
	"fmt"
	"strings"

	"github.com/nexspence/nxs/internal/client"
	"github.com/spf13/cobra"
)

// promoteResolver is the subset of *client.Client that resolveComponents needs,
// so the resolver can be unit-tested with a fake.
type promoteResolver interface {
	PromotionRules() ([]client.PromotionRule, error)
	Search(client.SearchParams) ([]client.Component, error)
}

// resolveComponents maps a rule name-or-id plus component references to a
// rule ID and concrete component IDs. coordRefs are "group:name:version"
// strings resolved against the rule's from_repo; rawIDs are passed through.
func resolveComponents(r promoteResolver, ruleNameOrID string, coordRefs, rawIDs []string) (string, []string, error) {
	rules, err := r.PromotionRules()
	if err != nil {
		return "", nil, err
	}
	var rule *client.PromotionRule
	for i := range rules {
		if rules[i].ID == ruleNameOrID || rules[i].Name == ruleNameOrID {
			rule = &rules[i]
			break
		}
	}
	if rule == nil {
		return "", nil, fmt.Errorf("promotion rule %q not found", ruleNameOrID)
	}

	ids := append([]string{}, rawIDs...)
	for _, ref := range coordRefs {
		parts := strings.SplitN(ref, ":", 3)
		if len(parts) != 3 {
			return "", nil, fmt.Errorf("invalid component %q: expected group:name:version", ref)
		}
		group, name, version := parts[0], parts[1], parts[2]
		comps, err := r.Search(client.SearchParams{Repo: rule.FromRepo, Query: name})
		if err != nil {
			return "", nil, err
		}
		var matched []string
		for _, c := range comps {
			if c.Group == group && c.Name == name && c.Version == version {
				matched = append(matched, c.ID)
			}
		}
		switch len(matched) {
		case 0:
			return "", nil, fmt.Errorf("no component %s in repo %q", ref, rule.FromRepo)
		case 1:
			ids = append(ids, matched[0])
		default:
			return "", nil, fmt.Errorf("ambiguous component %s: %d matches", ref, len(matched))
		}
	}
	return rule.ID, ids, nil
}

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote artifacts between repositories",
}

var promoteRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List promotion rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		rules, err := nxsClient.PromotionRules()
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(rules)
			return nil
		}
		rows := make([][]string, 0, len(rules))
		for _, r := range rules {
			gates := []string{}
			if r.RequireScanPass {
				gates = append(gates, "scan")
			}
			if r.RequireManualApproval {
				gates = append(gates, "approval")
			}
			rows = append(rows, []string{r.ID, r.Name, r.FromRepo + "→" + r.ToRepo, strings.Join(gates, ",")})
		}
		printer.Table([]string{"ID", "NAME", "FLOW", "GATES"}, rows)
		return nil
	},
}

var promoteRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Promote components via a rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		rule, _ := cmd.Flags().GetString("rule")
		coords, _ := cmd.Flags().GetStringSlice("component")
		rawIDs, _ := cmd.Flags().GetStringSlice("component-id")
		if rule == "" {
			return fmt.Errorf("--rule is required")
		}
		if len(coords) == 0 && len(rawIDs) == 0 {
			return fmt.Errorf("at least one --component or --component-id is required")
		}
		ruleID, ids, err := resolveComponents(nxsClient, rule, coords, rawIDs)
		if err != nil {
			return err
		}
		reqs, err := nxsClient.Promote(ruleID, ids)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(reqs)
			return nil
		}
		rows := make([][]string, 0, len(reqs))
		for _, rq := range reqs {
			rows = append(rows, []string{rq.ID, rq.ComponentID, rq.Status, rq.Error})
		}
		printer.Table([]string{"REQUEST", "COMPONENT", "STATUS", "ERROR"}, rows)
		return nil
	},
}

var promoteRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List promotion requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		reqs, err := nxsClient.PromotionRequests(status)
		if err != nil {
			return err
		}
		if flagJSON {
			printer.JSON(reqs)
			return nil
		}
		rows := make([][]string, 0, len(reqs))
		for _, rq := range reqs {
			rows = append(rows, []string{rq.ID, rq.ComponentID, rq.Status, rq.Error})
		}
		printer.Table([]string{"REQUEST", "COMPONENT", "STATUS", "ERROR"}, rows)
		return nil
	},
}

var promoteApproveCmd = &cobra.Command{
	Use:   "approve <request-id>",
	Short: "Approve a promotion request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		if err := nxsClient.PromotionApprove(args[0]); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Request %s approved", args[0]))
		return nil
	},
}

var promoteRejectCmd = &cobra.Command{
	Use:   "reject <request-id>",
	Short: "Reject a promotion request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireClient(); err != nil {
			return err
		}
		reason, _ := cmd.Flags().GetString("reason")
		if err := nxsClient.PromotionReject(args[0], reason); err != nil {
			return err
		}
		printer.Success(fmt.Sprintf("Request %s rejected", args[0]))
		return nil
	},
}

func init() {
	promoteRunCmd.Flags().String("rule", "", "Promotion rule name or ID (required)")
	promoteRunCmd.Flags().StringSlice("component", nil, "Component as group:name:version (repeatable)")
	promoteRunCmd.Flags().StringSlice("component-id", nil, "Raw component UUID (repeatable)")
	promoteRequestsCmd.Flags().String("status", "", "Filter by status (pending/approved/rejected/done)")
	promoteRejectCmd.Flags().String("reason", "", "Rejection reason")
	promoteCmd.AddCommand(promoteRulesCmd, promoteRunCmd, promoteRequestsCmd, promoteApproveCmd, promoteRejectCmd)
	rootCmd.AddCommand(promoteCmd)
}
