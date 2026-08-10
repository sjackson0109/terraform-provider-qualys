package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceWASFindings() *schema.Resource {
	return &schema.Resource{
		Description: "Look up individual Qualys WAS scan findings: vulnerabilities, " +
			"sensitive-content exposures and information-gathered items reported against web " +
			"applications. Each returned object is exactly one independent Qualys finding — " +
			"never aggregated by QID, web application or URL. Read-only: it cannot create, " +
			"modify, ignore, reopen, retest or override the severity of a finding. To manage a " +
			"finding's ignored state, see `qualys_was_finding_ignore`.\n\n" +
			"This is the WAS counterpart to `data.qualys_vm_findings`; the two share " +
			"terminology and structure (`qid`, `severity`, `status`, `cvss_v3_base`, " +
			"`finding_key`, `enrich_with_knowledgebase`) wherever the underlying Qualys concepts " +
			"line up, without forcing identical shapes where they do not (WAS findings already " +
			"carry a stable Qualys-assigned `id`, so `finding_key` is simply that ID, not a " +
			"constructed composite key the way VM's `host:qid:protocol:port` is).\n\n" +
			"**Terraform state:** every field returned here, including asset/application " +
			"identifiers (`url`, `web_app_name`) and vulnerability detail, is persisted in " +
			"Terraform state once this data source is read. Protect state accordingly. This " +
			"initial implementation deliberately favours finding metadata over raw HTTP " +
			"request/response evidence: fields such as payload/request/response/parameter/" +
			"authentication/access_path have no confirmed representation on the Finding object " +
			"in this session's evidence and are not implemented — see the schema-level " +
			"provenance note below rather than the resource carrying an unverified guess.\n\n" +
			"**Evidence tier:** `id`, `uniqueId`, `qid`, `name`, `type`, `severity`, `status`, " +
			"`findingType`, `url`, `webApp.id`/`webApp.name`, `firstDetectedDate`, " +
			"`lastDetectedDate`, `isIgnored` and `cvssV3.base` are Confirmed (this provider's " +
			"discovery notes, doc 11 §9, and the existing `qualys_was_finding_ignore`/retest " +
			"work already established these). The `id`/`qid`/date-range query filters are " +
			"Corroborated: doc 11 §9 lists `id`, `qid`, `firstDetectedDate` and " +
			"`lastDetectedDate` as valid Finding filter fields, and documents GREATER/LESSER as " +
			"generally valid qps operators (this same package's own pagination cursor already " +
			"depends on GREATER working) — but the specific field/operator combination for the " +
			"date bounds is not independently verified. A CVSS v2 score, and the richer fields " +
			"a requirements document asked for (category, description, impact, solution, " +
			"payload/request/response, parameter, authentication, ssl, access_path, external " +
			"reference, OWASP/WASC category, CWE), have no confirmed representation on the " +
			"Finding object itself in this session's evidence and are deliberately not " +
			"implemented rather than guessed — CVSS v2 and several of the prose fields remain " +
			"available indirectly through `enrich_with_knowledgebase`, since that data " +
			"genuinely lives in Qualys's KnowledgeBase, not on the Finding object, for the " +
			"fields this session could confirm. Verify against a tenant before relying on this " +
			"in production.\n\n" +
			"**Breaking change from earlier versions of this data source:** `severity`, " +
			"`status`, `type` and `finding_type` changed type (`severity` from string to " +
			"number; `status`/`type`/`finding_type` from a single string to a set of strings) " +
			"to support the range/multi-value filtering this revision adds. Existing " +
			"configurations using the old single-string form need updating.",

		ReadContext: dataSourceWASFindingsRead,

		Schema: map[string]*schema.Schema{
			"web_app_id": {
				Description:   "Restrict to findings on this web application. Conflicts with `web_app_ids` and `finding_ids`.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"web_app_ids", "finding_ids"},
			},
			"web_app_ids": {
				Description: "Restrict to findings on these web applications. Implemented as one " +
					"search per web application ID (each using the Confirmed single-value " +
					"`webApp.id` filter), merged and deduplicated by finding ID — this session " +
					"found no confirmed multi-value equivalent for this field specifically. " +
					"Conflicts with `web_app_id` and `finding_ids`.",
				Type:          schema.TypeSet,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"web_app_id", "finding_ids"},
			},
			"finding_ids": {
				Description: "Retrieve exactly these findings by their Qualys finding ID, " +
					"bypassing every other search filter (every other filter in this schema " +
					"still applies afterward, narrowing this fixed set). An ID no longer " +
					"visible to the caller is simply omitted from the result, not an error. " +
					"Conflicts with `web_app_id` and `web_app_ids`.",
				Type:          schema.TypeSet,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"web_app_id", "web_app_ids"},
			},

			"qids": {
				Description: "Restrict to these QIDs. A single QID uses the Confirmed " +
					"server-side `qid` filter; two or more are applied client-side after " +
					"retrieval instead, since this session found no confirmed multi-value QID " +
					"query parameter.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"severity": {
				Description: "Restrict to exactly this severity (1-5). Conflicts in effect " +
					"with `minimum_severity`/`maximum_severity` are not prevented at the schema " +
					"level, but combining them narrows to their intersection.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 5),
			},
			"minimum_severity": {
				Description:  "Restrict to findings at or above this severity (1-5). Applied client-side.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 5),
			},
			"maximum_severity": {
				Description:  "Restrict to findings at or below this severity (1-5). Applied client-side.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(1, 5),
			},
			"status": {
				Description: "Restrict to these statuses: `NEW`, `ACTIVE`, `REOPENED`, `FIXED`, " +
					"`PROTECTED`, returned by Qualys unchanged — this data source does not " +
					"translate them into any downstream remediation-workflow vocabulary. A " +
					"single status uses the Confirmed server-side filter; two or more are " +
					"applied client-side after a broader retrieval.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"type": {
				Description: "Restrict to these finding types: `VULNERABILITY`, " +
					"`SENSITIVE_CONTENT`, `INFORMATION_GATHERED`. Same single-value-server, " +
					"multi-value-client-side handling as `status`.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"finding_type": {
				Description: "Restrict to these sources: `QUALYS`, `MANUAL`, `BURP`. Same " +
					"single-value-server, multi-value-client-side handling as `status`.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"is_ignored": {
				Description: "Restrict to findings with this ignored state. Always a server-side filter.",
				Type:        schema.TypeBool,
				Optional:    true,
			},

			"first_detected_after":  wasDateFilterSchema("Restrict to findings first detected after this time."),
			"first_detected_before": wasDateFilterSchema("Restrict to findings first detected before this time."),
			"last_detected_after":   wasDateFilterSchema("Restrict to findings last detected after this time."),
			"last_detected_before":  wasDateFilterSchema("Restrict to findings last detected before this time."),

			"enrich_with_knowledgebase": {
				Description: "Join each finding's QID against the Qualys KnowledgeBase (the " +
					"same subscription-wide catalog `data.qualys_vm_findings` enriches from) to " +
					"populate `title`/`category`/`cvss_v2_base`/`pci_flag`/`patchable`/" +
					"`diagnosis`/`consequence`/`solution`. Unique QIDs are collected first and " +
					"fetched in one batched request, never one per finding. Defaults to `false`.",
				Type:     schema.TypeBool,
				Optional: true,
			},
			"max_results": {
				Description: "Return at most this many findings, taken from the front of the " +
					"deterministic ID ordering, after every other filter has been applied. " +
					"Every matching record is still retrieved first. Leave unset to return everything.",
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},

			"findings": {
				Description: "Matching findings, ordered by numeric finding ID for stable " +
					"output. Empty when nothing matches — this is not an error.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"finding_key": {
							Description: "Stable identity, equal to `id`. The Qualys finding ID " +
								"is already globally unique, so no composite key is constructed.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"id": {
							Description: "The Qualys finding ID — the primary identity of this " +
								"finding, used exactly as Qualys reports it.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"unique_id":           {Type: schema.TypeString, Computed: true},
						"qid":                 {Type: schema.TypeString, Computed: true},
						"name":                {Type: schema.TypeString, Computed: true},
						"type":                {Type: schema.TypeString, Computed: true},
						"severity":            {Type: schema.TypeInt, Computed: true},
						"status":              {Type: schema.TypeString, Computed: true},
						"finding_type":        {Type: schema.TypeString, Computed: true},
						"web_app_id":          {Type: schema.TypeString, Computed: true},
						"web_app_name":        {Type: schema.TypeString, Computed: true},
						"url":                 {Type: schema.TypeString, Computed: true},
						"first_detected_date": {Type: schema.TypeString, Computed: true},
						"last_detected_date":  {Type: schema.TypeString, Computed: true},
						"is_ignored":          {Type: schema.TypeBool, Computed: true},
						"cvss_v3_base":        {Type: schema.TypeFloat, Computed: true},

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
							Description: "Populated only when `enrich_with_knowledgebase = true`. " +
								"Not available directly on the Finding object — see the " +
								"resource-level evidence-tier note.",
							Type:     schema.TypeFloat,
							Computed: true,
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
							Description: "KnowledgeBase description of the vulnerability. " +
								"Populated only when `enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"consequence": {
							Description: "KnowledgeBase description of impact. Populated only " +
								"when `enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
						"solution": {
							Description: "KnowledgeBase remediation guidance. Populated only " +
								"when `enrich_with_knowledgebase = true`.",
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func wasDateFilterSchema(description string) *schema.Schema {
	return &schema.Schema{
		Description: description + " ISO-8601/RFC3339, e.g. `2026-08-01T00:00:00Z`. Sent as a " +
			"server-side GREATER/LESSER filter and re-checked client-side — see the " +
			"resource-level evidence-tier note.",
		Type:         schema.TypeString,
		Optional:     true,
		ValidateFunc: validation.IsRFC3339Time,
	}
}

func dataSourceWASFindingsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	minSeverity := d.Get("minimum_severity").(int)
	maxSeverity := d.Get("maximum_severity").(int)
	if minSeverity > 0 && maxSeverity > 0 && minSeverity > maxSeverity {
		return diag.Errorf("minimum_severity (%d) must not be greater than maximum_severity (%d)",
			minSeverity, maxSeverity)
	}

	webAppID := d.Get("web_app_id").(string)
	webAppIDs := stringSet(d, "web_app_ids")
	findingIDs := stringSet(d, "finding_ids")
	qids := stringSet(d, "qids")
	statuses := stringSet(d, "status")
	types := stringSet(d, "type")
	findingTypes := stringSet(d, "finding_type")

	var severity string
	if v, ok := d.GetOk("severity"); ok {
		severity = strconv.Itoa(v.(int))
	}
	var isIgnored *bool
	if v, ok := d.GetOkExists("is_ignored"); ok {
		b := v.(bool)
		isIgnored = &b
	}

	base := qps.WASFindingFilter{
		Severity:            severity,
		IsIgnored:           isIgnored,
		FirstDetectedAfter:  d.Get("first_detected_after").(string),
		FirstDetectedBefore: d.Get("first_detected_before").(string),
		LastDetectedAfter:   d.Get("last_detected_after").(string),
		LastDetectedBefore:  d.Get("last_detected_before").(string),
	}
	if len(qids) == 1 {
		base.QID = qids[0]
	}
	if len(statuses) == 1 {
		base.Status = statuses[0]
	}
	if len(types) == 1 {
		base.Type = types[0]
	}
	if len(findingTypes) == 1 {
		base.FindingType = findingTypes[0]
	}

	var findings []*qps.WASFinding
	var diags diag.Diagnostics

	switch {
	case len(findingIDs) > 0:
		for _, id := range findingIDs {
			f, err := c.GetWASFinding(ctx, id)
			if err != nil {
				if errors.Is(err, qps.ErrNotFound) {
					continue
				}
				return diag.FromErr(err)
			}
			findings = append(findings, f)
		}
	case len(webAppIDs) > 0:
		seen := make(map[string]*qps.WASFinding)
		var crossAppConflicts []qps.WASFindingConflict
		for _, id := range webAppIDs {
			f := base
			f.WebAppID = id
			found, conflicts, err := c.SearchWASFindings(ctx, f)
			if err != nil {
				return diag.FromErr(err)
			}
			diags = append(diags, wasFindingConflictDiagnostics(conflicts)...)
			for _, finding := range found {
				if existing, dup := seen[finding.ID]; dup {
					if *existing != *finding {
						crossAppConflicts = append(crossAppConflicts,
							qps.WASFindingConflict{ID: finding.ID, First: existing, Other: finding})
					}
					continue
				}
				seen[finding.ID] = finding
				findings = append(findings, finding)
			}
		}
		diags = append(diags, wasFindingConflictDiagnostics(crossAppConflicts)...)
	default:
		f := base
		f.WebAppID = webAppID
		found, conflicts, err := c.SearchWASFindings(ctx, f)
		if err != nil {
			return diag.FromErr(err)
		}
		diags = append(diags, wasFindingConflictDiagnostics(conflicts)...)
		findings = found
	}

	findings, err = filterWASFindings(findings, wasFindingFilters{
		QIDs:                qids,
		Statuses:            statuses,
		Types:               types,
		FindingTypes:        findingTypes,
		MinimumSeverity:     minSeverity,
		MaximumSeverity:     maxSeverity,
		FirstDetectedAfter:  d.Get("first_detected_after").(string),
		FirstDetectedBefore: d.Get("first_detected_before").(string),
		LastDetectedAfter:   d.Get("last_detected_after").(string),
		LastDetectedBefore:  d.Get("last_detected_before").(string),
	})
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	sortWASFindings(findings)

	if max := d.Get("max_results").(int); max > 0 && len(findings) > max {
		findings = findings[:max]
	}

	var kb map[string]*vmdr.KBEntry
	if d.Get("enrich_with_knowledgebase").(bool) {
		vc, err := vmdrClient(meta)
		if err != nil {
			return append(diags, diag.FromErr(err)...)
		}
		qidsForKB := make([]string, 0, len(findings))
		for _, f := range findings {
			qidsForKB = append(qidsForKB, f.QID)
		}
		kb, err = fetchKnowledgeBase(ctx, vc, qidsForKB)
		if err != nil {
			return append(diags, diag.FromErr(fmt.Errorf("enriching with the KnowledgeBase: %w", err))...)
		}
	}

	out := make([]interface{}, 0, len(findings))
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
		row := map[string]interface{}{
			"finding_key":         f.ID,
			"id":                  f.ID,
			"unique_id":           f.UniqueID,
			"qid":                 f.QID,
			"name":                f.Name,
			"type":                f.Type,
			"severity":            f.Severity,
			"status":              f.Status,
			"finding_type":        f.FindingType,
			"web_app_id":          f.WebAppID,
			"web_app_name":        f.WebAppName,
			"url":                 f.URL,
			"first_detected_date": f.FirstDetectedDate,
			"last_detected_date":  f.LastDetectedDate,
			"is_ignored":          f.IsIgnored,
			"cvss_v3_base":        f.CVSSV3Base,
		}
		if entry, ok := kb[f.QID]; ok {
			row["title"] = entry.Title
			row["category"] = entry.Category
			row["cvss_v2_base"] = entry.CVSSV2Base
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
	d.SetId(digestID(ids))
	return diags
}

func wasFindingConflictDiagnostics(conflicts []qps.WASFindingConflict) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, c := range conflicts {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Conflicting findings share id %q", c.ID),
			Detail: fmt.Sprintf("Two decoded findings shared the same Qualys finding ID but "+
				"disagree on other fields (status %q vs %q). The first one seen was kept.",
				c.First.Status, c.Other.Status),
		})
	}
	return diags
}

// wasFindingFilters are the finding-scope filters applied client-side,
// either as a defensive re-check (fields already sent server-side) or as
// the sole enforcement (multi-value QID/status/type/finding_type, the
// severity range) — see WASFindingFilter's and the schema's doc comments
// for why these specifically aren't always sent as query parameters.
type wasFindingFilters struct {
	QIDs            []string
	Statuses        []string
	Types           []string
	FindingTypes    []string
	MinimumSeverity int
	MaximumSeverity int

	FirstDetectedAfter  string
	FirstDetectedBefore string
	LastDetectedAfter   string
	LastDetectedBefore  string
}

func filterWASFindings(findings []*qps.WASFinding, f wasFindingFilters) ([]*qps.WASFinding, error) {
	qidSet := toStringSet(f.QIDs)
	statusSet := toStringSet(f.Statuses)
	typeSet := toStringSet(f.Types)
	findingTypeSet := toStringSet(f.FindingTypes)

	out := make([]*qps.WASFinding, 0, len(findings))
	for _, finding := range findings {
		if qidSet != nil && !qidSet[finding.QID] {
			continue
		}
		if statusSet != nil && !statusSet[finding.Status] {
			continue
		}
		if typeSet != nil && !typeSet[finding.Type] {
			continue
		}
		if findingTypeSet != nil && !findingTypeSet[finding.FindingType] {
			continue
		}
		if f.MinimumSeverity > 0 && finding.Severity < f.MinimumSeverity {
			continue
		}
		if f.MaximumSeverity > 0 && finding.Severity > f.MaximumSeverity {
			continue
		}

		ok, err := withinWASDateFilter(finding.FirstDetectedDate, f.FirstDetectedAfter, f.FirstDetectedBefore)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		ok, err = withinWASDateFilter(finding.LastDetectedDate, f.LastDetectedAfter, f.LastDetectedBefore)
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

func toStringSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, v := range values {
		out[v] = true
	}
	return out
}

// withinWASDateFilter mirrors data_source_vm_findings.go's withinDateFilter,
// with one difference: it parses with time.RFC3339Nano rather than
// time.RFC3339, since this session has no confirmed evidence for whether
// WAS finding timestamps ever carry fractional seconds and RFC3339Nano
// parses both forms. See withinDateFilter's doc comment for the shared
// reasoning on excluding a finding with a missing value, and on treating a
// value this provider itself decoded as unparseable as a surfaced error.
func withinWASDateFilter(value, after, before string) (bool, error) {
	if after == "" && before == "" {
		return true, nil
	}
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return false, fmt.Errorf("qualys: finding datetime %q is not RFC3339; "+
			"cannot evaluate a date filter against it", value)
	}
	if after != "" {
		afterT, err := time.Parse(time.RFC3339Nano, after)
		if err != nil {
			return false, fmt.Errorf("qualys: parsing date filter %q: %w", after, err)
		}
		if t.Before(afterT) {
			return false, nil
		}
	}
	if before != "" {
		beforeT, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return false, fmt.Errorf("qualys: parsing date filter %q: %w", before, err)
		}
		if t.After(beforeT) {
			return false, nil
		}
	}
	return true, nil
}

// sortWASFindings orders findings by numeric Qualys finding ID — already a
// globally unique, consistently sortable identity, so no secondary sort
// key (e.g. web_app_id) is needed.
func sortWASFindings(findings []*qps.WASFinding) {
	sort.Slice(findings, func(i, j int) bool {
		return numericLess(findings[i].ID, findings[j].ID)
	})
}
