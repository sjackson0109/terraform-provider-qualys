package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Networks let the same IP range exist more than once in a subscription, each
// occurrence scanned by its own appliances. They require the Network Support
// feature to be enabled on the subscription.
//
// The API exposes list, create and update — and no delete. Networks can be
// deleted from the Qualys UI, but there is no documented API action for it, so a
// Terraform destroy cannot remove one. DeleteNetwork therefore does not exist on
// this client; the resource reports the situation rather than pretending.

// Network is a Qualys network.
type Network struct {
	ID   string
	Name string
	// ApplianceIDs are the scanner appliances assigned to this network. An
	// appliance belongs to exactly one network, and a network with no appliance
	// cannot be scanned.
	ApplianceIDs []string
}

// GlobalDefaultNetworkID is the network every subscription has. Scan targets
// with no explicit network belong to it.
const GlobalDefaultNetworkID = "0"

type networkListOutput struct {
	Response struct {
		Networks []struct {
			ID        string `xml:"ID"`
			Name      string `xml:"NAME"`
			Appliance []struct {
				ID string `xml:"ID"`
			} `xml:"SCANNER_APPLIANCE_LIST>SCANNER_APPLIANCE"`
		} `xml:"NETWORK_LIST>NETWORK"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *networkListOutput) warning() *warning { return o.Response.Warning }

func (o *networkListOutput) networks() []*Network {
	out := make([]*Network, 0, len(o.Response.Networks))
	for i := range o.Response.Networks {
		n := &o.Response.Networks[i]
		net := &Network{ID: strings.TrimSpace(n.ID), Name: n.Name}
		for _, a := range n.Appliance {
			if id := strings.TrimSpace(a.ID); id != "" {
				net.ApplianceIDs = append(net.ApplianceIDs, id)
			}
		}
		out = append(out, net)
	}
	return out
}

// CreateNetwork creates a custom network and returns its ID.
func (c *Client) CreateNetwork(ctx context.Context, name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("qualys vmdr: network name is required")
	}

	params := url.Values{}
	params.Set("name", name)

	var out simpleReturn
	if err := c.do(ctx, request{
		capability: capNetwork,
		path:       "network/",
		action:     "create",
		params:     params,
	}, &out); err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("qualys vmdr: network was created but the API returned no ID "+
			"(response: %q); import it to bring it under management", out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateNetwork renames a network. Name is the only mutable attribute.
func (c *Client) UpdateNetwork(ctx context.Context, id, name string) error {
	params := url.Values{}
	params.Set("id", id)
	params.Set("name", name)
	return c.do(ctx, request{
		capability: capNetwork,
		path:       "network/",
		action:     "update",
		params:     params,
	}, nil)
}

// GetNetwork returns one network, or ErrNotFound.
func (c *Client) GetNetwork(ctx context.Context, id string) (*Network, error) {
	nets, err := c.ListNetworks(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	for _, n := range nets {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: network %s: %w", id, ErrNotFound)
}

// ListNetworks returns networks, optionally filtered by ID.
func (c *Client) ListNetworks(ctx context.Context, ids []string) ([]*Network, error) {
	params := url.Values{}
	setListIfNotEmpty(params, "ids", ids)

	var all []*Network
	err := c.listAll(ctx, request{
		capability: capNetwork,
		path:       "network/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(networkListOutput) },
		func(p paginated) error {
			all = append(all, p.(*networkListOutput).networks()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
