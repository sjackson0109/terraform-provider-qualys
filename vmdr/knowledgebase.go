package vmdr

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// KBEntry is one Qualys KnowledgeBase vulnerability record: the
// human-readable metadata behind a QID (title, category, CVSS scores,
// compliance/patch flags). It enriches a VMFinding, which carries only the
// detection instance, not the vulnerability's static description.
//
// Evidence tier: Corroborated, not Confirmed — the same caveat as
// VMFinding's DETECTION_LIST schema applies here. This session could not
// fetch docs.qualys.com (blocked in this sandbox) or verify against a
// captured tenant response; this reflects Qualys's long-published
// KnowledgeBase API v2 schema (knowledge_base_vuln_list_output.dtd). This
// provider's discovery notes (doc 08) already record the KnowledgeBase API
// as existing but subscription-gated (an add-on); this file assumes that
// gate is open for a caller that uses it. Verify against a tenant before
// relying on this in production.
type KBEntry struct {
	QID        string
	Title      string
	Category   string
	CVSSV2Base float64
	CVSSV3Base float64
	PCIFlag    bool
	Patchable  bool
}

type kbEntryWire struct {
	QID      string `xml:"QID"`
	Title    string `xml:"TITLE"`
	Category string `xml:"CATEGORY"`
	CVSS     struct {
		Base string `xml:"BASE"`
	} `xml:"CVSS"`
	CVSSV3 struct {
		Base string `xml:"BASE"`
	} `xml:"CVSS_V3"`
	PCIFlag   string `xml:"PCI_FLAG"`
	Patchable string `xml:"PATCHABLE"`
}

type kbListOutput struct {
	Response struct {
		Vulns   []kbEntryWire `xml:"VULN_LIST>VULN"`
		Warning *warning      `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *kbListOutput) warning() *warning { return o.Response.Warning }

func atofSafe(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func (o *kbListOutput) entries() []*KBEntry {
	out := make([]*KBEntry, 0, len(o.Response.Vulns))
	for _, v := range o.Response.Vulns {
		out = append(out, &KBEntry{
			QID:        strings.TrimSpace(v.QID),
			Title:      strings.TrimSpace(v.Title),
			Category:   strings.TrimSpace(v.Category),
			CVSSV2Base: atofSafe(v.CVSS.Base),
			CVSSV3Base: atofSafe(v.CVSSV3.Base),
			PCIFlag:    boolFromFlag(v.PCIFlag),
			Patchable:  boolFromFlag(v.Patchable),
		})
	}
	return out
}

// GetKnowledgeBaseEntries returns KB metadata for the given QIDs in one
// batched request (following truncation, if the response is large enough to
// truncate) — never one request per QID. A caller enriching N findings
// should first collect their unique QIDs and call this once, then join the
// result against findings in memory; see the qualys_vm_findings data
// source for the reference implementation. Unknown or de-listed QIDs are
// simply absent from the result, not an error.
func (c *Client) GetKnowledgeBaseEntries(ctx context.Context, qids []string) ([]*KBEntry, error) {
	if len(qids) == 0 {
		return nil, nil
	}

	params := url.Values{}
	setListIfNotEmpty(params, "ids", qids)

	var all []*KBEntry
	err := c.listAll(ctx, request{
		capability: capKnowledgeBase,
		path:       "knowledge_base/vuln/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(kbListOutput) },
		func(p paginated) error {
			all = append(all, p.(*kbListOutput).entries()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
