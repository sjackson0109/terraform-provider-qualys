package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys network, allowing the same IP range to exist more than once " +
			"in a subscription with its own scanners.\n\n" +
			"Requires the Network Support feature to be enabled on the subscription.\n\n" +
			"~> **Destroying this resource does not delete the network.** The Qualys API " +
			"provides no delete action for networks — they can only be removed from the " +
			"Qualys UI. Terraform will remove the network from state and leave it in place.",

		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		UpdateContext: resourceNetworkUpdate,
		DeleteContext: resourceNetworkDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the network.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"appliance_ids": {
				Description: "Scanner appliances assigned to this network. Assignment is made " +
					"from the appliance, so this attribute is read-only here. A network with " +
					"no appliance cannot be scanned.",
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := c.CreateNetwork(ctx, d.Get("name").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	n, err := c.GetNetwork(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":          n.Name,
		"appliance_ids": n.ApplianceIDs,
	}))
}

func resourceNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateNetwork(ctx, d.Id(), d.Get("name").(string)); err != nil {
		return diag.FromErr(err)
	}
	return resourceNetworkRead(ctx, d, meta)
}

// resourceNetworkDelete removes the network from state without deleting it.
//
// The Qualys API has no delete action for networks. Failing the destroy would
// leave the resource permanently un-destroyable; silently succeeding would imply
// the network is gone. Warning and forgetting is the honest option, and the
// warning names what is left behind so it can be cleaned up in the UI.
func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Network removed from state but not deleted in Qualys",
		Detail: "The Qualys API provides no action to delete a network, so " +
			"network " + name + " still exists in your subscription. Delete it from the " +
			"Qualys UI if it is no longer wanted. Assets and scanner appliances assigned " +
			"to it are unaffected by this operation.",
	}}
}
