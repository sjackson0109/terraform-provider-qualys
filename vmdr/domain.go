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

	// Response is a pointer because the one confirmed sample of this
	// endpoint's output (the official "List Domain" documentation page) has
	// DOMAIN directly under the DOMAIN_LIST root, with no enclosing RESPONSE
	// element at all — unlike every other asset/* list output in this client
	// (HOST_LIST_OUTPUT, ASSET_GROUP_LIST_OUTPUT, ...), which wraps its list
	// and WARNING truncation marker in RESPONSE. Since that is the dominant
	// convention across the rest of this exact API family, and the sample
	// may simply be a documentation example that omits it, this field is
	// decoded defensively: present, warning() reports it and listAll follows
	// the truncation; absent (matching the confirmed sample), it decodes to
	// nil and warning() reports no truncation, which is a no-op change from
	// the single-request behaviour this had before.
	Response *struct {
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *domainListOutput) warning() *warning {
	if o.Response == nil {
		return nil
	}
	return o.Response.Warning
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

// ListDomains returns every registered domain, following truncation if the
// response reports it. No filter parameters are confirmed for this
// endpoint, so none are sent.
func (c *Client) ListDomains(ctx context.Context) ([]*Domain, error) {
	var all []*Domain
	err := c.listAll(ctx, request{
		capability: capAssetDomain,
		path:       "asset/domain/",
		action:     "list",
		method:     "GET",
	},
		func() paginated { return new(domainListOutput) },
		func(p paginated) error {
			all = append(all, p.(*domainListOutput).domains()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
