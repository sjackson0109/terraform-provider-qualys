package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVMFindings() *schema.Resource {
	return &schema.Resource{
		Description: "Look up individual Qualys VM vulnerability findings: one affected host, " +
			"one QID, one detection instance — never aggregated by host or by QID. Where Qualys " +
			"reports the same QID on two ports of the same host as distinct detections, this " +
			"data source returns two findings. Read-only: it cannot create, modify, suppress, " +
			"acknowledge, purge or otherwise change a finding.\n\n" +
			"**Terraform state:** every field returned here, including `findings[*].results` (raw " +
			"technical detail from the scanned target) and asset identifiers (`ip`, `dns`, " +
			"`operating_system`), is persisted in Terraform state once this data source is read. " +
			"Protect state accordingly — it is not encrypted by Terraform itself. `results` is " +
			"marked sensitive so `terraform plan`/`apply` output does not print it, but it is " +
			"still stored in state in full.\n\n" +
			"**Evidence tier:** the per-detection fields this data source exposes (severity, " +
			"port, protocol, status, dates, results, and similar) are Corroborated against " +
			"Qualys's long-published Host List Detection API schema, not Confirmed via a fetched " +
			"or user-supplied source in this session (docs.qualys.com is blocked in this " +
			"sandbox) — see `vmdr/vmfinding.go`'s doc comment for the full provenance note. " +
			"Verify against a tenant, or a captured API response, before relying on this in " +
			"production.",

		ReadContext: dataSourceVMFindingsRead,

		Schema: map[string]*schema.Schema{
			"host_ids": {
				Description: "Restrict to these host IDs.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"ips": {
				Description: "Restrict to these IPs, hyphenated ranges or CIDR blocks.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"asset_group_ids": {
				Description: "Restrict to hosts in these asset groups (see `qualys_asset_group`). " +
					"Resolved by looking up each group's statically-configured IPs and merging " +
					"them into the `ips` filter — this endpoint has no confirmed asset-group-ID " +
					"query parameter of its own (see the resource-level evidence-tier note), so " +
					"a group whose membership is defined purely by DNS/NetBIOS name or by a " +
					"dynamic rule, rather than a static IP list, will not be fully resolved this " +
					"way. Pass `host_ids`/`ips` directly for those.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"qids": {
				Description: "Restrict to these QIDs. Applied client-side after retrieval " +
					"(see the resource-level evidence-tier note), so this does not reduce API " +
					"traffic, only the returned result.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"status": {
				Description: "Restrict to these Qualys detection statuses, e.g. `New`, " +
					"`Active`, `Fixed`, `Re-Opened` — sent to the API verbatim, exactly as " +
					"Qualys reports them; this data source does not translate them into any " +
					"downstream remediation-workflow vocabulary.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"minimum_severity": {
				Description:  "Restrict to findings at or above this severity (1-5).",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 5),
			},
			"maximum_severity": {
				Description:  "Restrict to findings at or below this severity (1-5).",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 5),
			},

			"first_found_after":  isoDateFilterSchema("Restrict to findings first found after this time."),
			"first_found_before": isoDateFilterSchema("Restrict to findings first found before this time."),
			"last_found_after":   isoDateFilterSchema("Restrict to findings last found after this time."),
			"last_found_before":  isoDateFilterSchema("Restrict to findings last found before this time."),
			"last_test_after":    isoDateFilterSchema("Restrict to findings last tested after this time."),
			"last_test_before":   isoDateFilterSchema("Restrict to findings last tested before this time."),

			"enrich_with_knowledgebase": {
				Description: "Join each finding's QID against the Qualys KnowledgeBase to " +
					"populate `title`/`category`/`cvss_v2_base`/`cvss_v3_base`/`pci_flag`/" +
					"`patchable`. Unique QIDs are collected first and fetched in one batched " +
					"request (or a small, paginated number of them), never one request per " +
					"finding. The KnowledgeBase API is a separately-authorized Qualys add-on; " +
					"if your subscription lacks it, leave this false — the core finding fields " +
					"remain fully populated either way. Defaults to `false`.",
				Type:     schema.TypeBool,
				Optional: true,
			},
			"max_results": {
				Description: "Return at most this many findings, taken from the front of the " +
					"deterministic `host_id`/`qid`/`port`/`protocol` ordering, after every other " +
					"filter has been applied. Every matching record is still retrieved from the " +
					"API first — this only truncates the returned collection, it does not stop " +
					"pagination early. Leave unset to return everything.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"findings": {
				Description: "Matching findings, ordered by host_id, qid, port, protocol for " +
					"stable output. Empty when nothing matches — this is not an error.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"finding_key": {
							Description: "Stable identity: `<host_id>:<qid>:<protocol>:<port>`, " +
								"or `<host_id>:<qid>` when Qualys reports no port for this " +
								"detection. Deterministic across reads of unchanged data.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_id":          {Type: schema.TypeString, Computed: true},
						"ip":               {Type: schema.TypeString, Computed: true},
						"dns":              {Type: schema.TypeString, Computed: true},
						"netbios":          {Type: schema.TypeString, Computed: true},
						"operating_system": {Type: schema.TypeString, Computed: true},
						"tracking_method":  {Type: schema.TypeString, Computed: true},
						"qid":              {Type: schema.TypeString, Computed: true},
						"port":             {Type: schema.TypeInt, Computed: true},
						"protocol":         {Type: schema.TypeString, Computed: true},
						"severity":         {Type: schema.TypeInt, Computed: true},
						"status":           {Type: schema.TypeString, Computed: true},
						"type":             {Type: schema.TypeString, Computed: true},
						"ssl":              {Type: schema.TypeBool, Computed: true},
						"results": {
							Description: "Raw technical detail Qualys captured for this " +
								"detection (banners, response fragments). May contain " +
								"sensitive information — see the resource-level state warning.",
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"first_found_datetime": {Type: schema.TypeString, Computed: true},
						"last_found_datetime":  {Type: schema.TypeString, Computed: true},
						"last_test_datetime":   {Type: schema.TypeString, Computed: true},
						"last_update_datetime": {Type: schema.TypeString, Computed: true},
						"last_fixed_datetime":  {Type: schema.TypeString, Computed: true},
						"times_found":          {Type: schema.TypeInt, Computed: true},
						"is_ignored":           {Type: schema.TypeBool, Computed: true},
						"is_disabled":          {Type: schema.TypeBool, Computed: true},

						"title": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"category": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"cvss_v2_base": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeFloat,
							Computed:    true,
						},
						"cvss_v3_base": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeFloat,
							Computed:    true,
						},
						"pci_flag": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"patchable": {
							Description: "Populated only when `enrich_with_knowledgebase = true`.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"diagnosis": {
							Description: "KnowledgeBase description of the vulnerability. Populated " +
								"only when `enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"consequence": {
							Description: "KnowledgeBase description of impact. Populated only when " +
								"`enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"solution": {
							Description: "KnowledgeBase remediation guidance. Populated only when " +
								"`enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func isoDateFilterSchema(description string) *schema.Schema {
	return &schema.Schema{
		Description: description + " ISO-8601/RFC3339, e.g. `2026-08-01T00:00:00Z`. Applied " +
			"client-side after retrieval — see the resource-level evidence-tier note.",
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.IsRFC3339Time,
	}
}

func dataSourceVMFindingsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	minSeverity := d.Get("minimum_severity").(int)
	maxSeverity := d.Get("maximum_severity").(int)
	if minSeverity > 0 && maxSeverity > 0 && minSeverity > maxSeverity {
		return diag.Errorf("minimum_severity (%d) must not be greater than maximum_severity (%d)",
			minSeverity, maxSeverity)
	}

	ips := stringSet(d, "ips")
	for _, groupID := range stringSet(d, "asset_group_ids") {
		group, err := c.GetAssetGroup(ctx, groupID)
		if err != nil {
			return diag.FromErr(fmt.Errorf("resolving asset_group_ids: %w", err))
		}
		ips = append(ips, group.IPs...)
	}

	findings, conflicts, err := c.ListVMFindings(ctx, vmdr.VMFindingFilter{
		HostIDs: stringSet(d, "host_ids"),
		IPs:     ips,
		Status:  stringSet(d, "status"),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, conflict := range conflicts {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Conflicting findings share the identity %q", conflict.Key),
			Detail: fmt.Sprintf("Two decoded findings produced the same finding_key but "+
				"disagree on other fields (status %q vs %q). The first one seen was kept; "+
				"the identity scheme (host_id:qid:protocol:port) may need to account for "+
				"another distinguishing field Qualys reports for this detection.",
				conflict.First.Status, conflict.Other.Status),
		})
	}

	findings, err = filterVMFindings(findings, vmFindingFilters{
		QIDs:             stringSet(d, "qids"),
		MinimumSeverity:  minSeverity,
		MaximumSeverity:  maxSeverity,
		FirstFoundAfter:  d.Get("first_found_after").(string),
		FirstFoundBefore: d.Get("first_found_before").(string),
		LastFoundAfter:   d.Get("last_found_after").(string),
		LastFoundBefore:  d.Get("last_found_before").(string),
		LastTestAfter:    d.Get("last_test_after").(string),
		LastTestBefore:   d.Get("last_test_before").(string),
	})
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	sortVMFindings(findings)

	if max := d.Get("max_results").(int); max > 0 && len(findings) > max {
		findings = findings[:max]
	}

	var kb map[string]*vmdr.KBEntry
	if d.Get("enrich_with_knowledgebase").(bool) {
		qids := make([]string, 0, len(findings))
		for _, f := range findings {
			qids = append(qids, f.QID)
		}
		kb, err = fetchKnowledgeBase(ctx, c, qids)
		if err != nil {
			return append(diags, diag.FromErr(fmt.Errorf("enriching with the KnowledgeBase: %w", err))...)
		}
	}

	out := make([]interface{}, 0, len(findings))
	keys := make([]string, 0, len(findings))
	for _, f := range findings {
		key := f.FindingKey()
		keys = append(keys, key)

		row := map[string]interface{}{
			"finding_key":          key,
			"host_id":              f.HostID,
			"ip":                   f.IP,
			"dns":                  f.DNS,
			"netbios":              f.NetBIOS,
			"operating_system":     f.OS,
			"tracking_method":      f.TrackingMethod,
			"qid":                  f.QID,
			"port":                 f.Port,
			"protocol":             f.Protocol,
			"severity":             f.Severity,
			"status":               f.Status,
			"type":                 f.Type,
			"ssl":                  f.SSL,
			"results":              f.Results,
			"first_found_datetime": f.FirstFoundDatetime,
			"last_found_datetime":  f.LastFoundDatetime,
			"last_test_datetime":   f.LastTestDatetime,
			"last_update_datetime": f.LastUpdateDatetime,
			"last_fixed_datetime":  f.LastFixedDatetime,
			"times_found":          f.TimesFound,
			"is_ignored":           f.IsIgnored,
			"is_disabled":          f.IsDisabled,
		}
		if entry, ok := kb[f.QID]; ok {
			row["title"] = entry.Title
			row["category"] = entry.Category
			row["cvss_v2_base"] = entry.CVSSV2Base
			row["cvss_v3_base"] = entry.CVSSV3Base
			row["pci_flag"] = entry.PCIFlag
			row["patchable"] = entry.Patchable
			row["diagnosis"] = entry.Diagnosis
			row["consequence"] = entry.Consequence
			row["solution"] = entry.Solution
		}
		out = append(out, row)
	}

	if err := d.Set("findings", out); err != nil {
		return append(diags, diag.FromErr(err)...)
	}
	d.SetId(digestID(keys))
	return diags
}

// vmFindingFilters are the finding-scope filters applied client-side, after
// ListVMFindings returns — see VMFindingFilter's doc comment for why these
// specifically aren't sent as query parameters.
type vmFindingFilters struct {
	QIDs            []string
	MinimumSeverity int
	MaximumSeverity int

	FirstFoundAfter  string
	FirstFoundBefore string
	LastFoundAfter   string
	LastFoundBefore  string
	LastTestAfter    string
	LastTestBefore   string
}

func filterVMFindings(findings []*vmdr.VMFinding, f vmFindingFilters) ([]*vmdr.VMFinding, error) {
	var qidSet map[string]bool
	if len(f.QIDs) > 0 {
		qidSet = make(map[string]bool, len(f.QIDs))
		for _, q := range f.QIDs {
			qidSet[q] = true
		}
	}

	out := make([]*vmdr.VMFinding, 0, len(findings))
	for _, finding := range findings {
		if qidSet != nil && !qidSet[finding.QID] {
			continue
		}
		if f.MinimumSeverity > 0 && finding.Severity < f.MinimumSeverity {
			continue
		}
		if f.MaximumSeverity > 0 && finding.Severity > f.MaximumSeverity {
			continue
		}
		ok, err := withinDateFilter(finding.FirstFoundDatetime, f.FirstFoundAfter, f.FirstFoundBefore)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ok, err = withinDateFilter(finding.LastFoundDatetime, f.LastFoundAfter, f.LastFoundBefore)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ok, err = withinDateFilter(finding.LastTestDatetime, f.LastTestAfter, f.LastTestBefore)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out = append(out, finding)
	}
	return out, nil
}

// withinDateFilter reports whether value (a Qualys RFC3339 datetime, or
// empty if Qualys did not report one for this finding) falls within
// [after, before]. Either bound may be empty to leave that side open. A
// finding whose value is empty cannot be shown to satisfy a bound that was
// actually configured, so it is excluded rather than assumed to pass —
// deterministic and conservative, rather than a silent pass-through.
func withinDateFilter(value, after, before string) (bool, error) {
	if after == "" && before == "" {
		return true, nil
	}
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		// A value this provider itself decoded failing to parse indicates a
		// format this data source doesn't yet handle, not a user input
		// mistake — surfaced rather than silently excluding the finding, so
		// it isn't mistaken for "didn't match."
		return false, fmt.Errorf("qualys: finding datetime %q is not RFC3339; "+
			"cannot evaluate a date filter against it", value)
	}
	if after != "" {
		afterT, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return false, fmt.Errorf("qualys: parsing date filter %q: %w", after, err)
		}
		if t.Before(afterT) {
			return false, nil
		}
	}
	if before != "" {
		beforeT, err := time.Parse(time.RFC3339, before)
		if err != nil {
			return false, fmt.Errorf("qualys: parsing date filter %q: %w", before, err)
		}
		if t.After(beforeT) {
			return false, nil
		}
	}
	return true, nil
}

// sortVMFindings orders findings deterministically by host_id, qid, port,
// protocol — numeric comparison for the numeric-looking identity fields, so
// host 10 sorts after host 9 rather than before it lexically.
func sortVMFindings(findings []*vmdr.VMFinding) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.HostID != b.HostID {
			return numericLess(a.HostID, b.HostID)
		}
		if a.QID != b.QID {
			return numericLess(a.QID, b.QID)
		}
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		return a.Protocol < b.Protocol
	})
}

// numericLess compares two Qualys ID strings numerically when both parse as
// integers, falling back to a lexical comparison otherwise — so "10" sorts
// after "9" instead of before it.
func numericLess(a, b string) bool {
	an, aerr := strconv.Atoi(a)
	bn, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return an < bn
	}
	return a < b
}

// fetchKnowledgeBase fetches KnowledgeBase entries for the given QIDs
// (which may repeat and include empties — this de-duplicates them) in one
// batched call, never one call per finding. Shared by both
// data.qualys_vm_findings and data.qualys_was_findings: Qualys maintains
// one subscription-wide KnowledgeBase keyed by QID across VM, PC and WAS,
// so a single client (vmdr.Client.GetKnowledgeBaseEntries) and this one
// join helper serve both data sources' enrichment. Callers collect QIDs
// from their own finding slice — see collectQIDs — since the finding type
// differs (vmdr.VMFinding vs qps.WASFinding) but the QID string does not.
func fetchKnowledgeBase(ctx context.Context, c *vmdr.Client, qids []string) (map[string]*vmdr.KBEntry, error) {
	unique := uniqueNonEmpty(qids)
	if len(unique) == 0 {
		return nil, nil
	}

	entries, err := c.GetKnowledgeBaseEntries(ctx, unique)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*vmdr.KBEntry, len(entries))
	for _, e := range entries {
		out[e.QID] = e
	}
	return out, nil
}

// uniqueNonEmpty returns values with empties dropped and duplicates
// collapsed, preserving first-seen order.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
