package vmdr

import (
	"context"
	"strings"
)

// Domain is a registered asset domain and its netblocks
// (/api/2.0/fo/asset/domain/).
//
// This client is deliberately read-only. The family also supports
// add/update/delete, and doc 03 confirms the family exists (Basic-auth
// only, Manager role required, bulk delete since platform 10.25), but no
// source obtained during discovery names the actual add/update/delete
// request parameters — only the list output shape, which a fresh web
// search confirmed against the official "List Domain" documentation page
// (sample XML: DOMAIN_LIST/DOMAIN with DOMAIN_NAME, DOMAIN_ID, NETWORK,
// NETBLOCK/RANGE). See doc 08. A qualys_domain resource is not built on a
// guessed write schema.
type Domain struct {
	ID        string
	Name      string
	NetworkID string
	Netblocks []string
}

type domainListOutput struct {
	Domains []struct {
		Name    string `xml:"DOMAIN_NAME"`
		ID      string `xml:"DOMAIN_ID"`
		Network struct {
			ID string `xml:"NETWORK_ID"`
		} `xml:"NETWORK"`
		Netblocks []struct {
			Range struct {
				Start string `xml:"START"`
				End   string `xml:"END"`
			} `xml:"RANGE"`
		} `xml:"NETBLOCK"`
	} `xml:"DOMAIN"`
}

func (o *domainListOutput) domains() []*Domain {
	out := make([]*Domain, 0, len(o.Domains))
	for _, d := range o.Domains {
		dom := &Domain{
			ID:        strings.TrimSpace(d.ID),
			Name:      d.Name,
			NetworkID: strings.TrimSpace(d.Network.ID),
		}
		for _, nb := range d.Netblocks {
			start, end := strings.TrimSpace(nb.Range.Start), strings.TrimSpace(nb.Range.End)
			if start == "" {
				continue
			}
			if end == "" || end == start {
				dom.Netblocks = append(dom.Netblocks, start)
				continue
			}
			dom.Netblocks = append(dom.Netblocks, start+"-"+end)
		}
		out = append(out, dom)
	}
	return out
}

// ListDomains returns every registered domain. No filter parameters are
// confirmed for this endpoint, so none are sent.
func (c *Client) ListDomains(ctx context.Context) ([]*Domain, error) {
	var out domainListOutput
	if err := c.do(ctx, request{
		capability: capAssetDomain,
		path:       "asset/domain/",
		action:     "list",
		method:     "GET",
	}, &out); err != nil {
		return nil, err
	}
	return out.domains(), nil
}
