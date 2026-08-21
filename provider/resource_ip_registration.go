package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
)

func resourceIPRegistration() *schema.Resource {
	return &schema.Resource{
		Description: "Registers IP addresses with a Qualys subscription and enables them for " +
			"scanning.\n\n" +
			"~> **Destroying this resource does not remove the addresses from Qualys.** The " +
			"Qualys API provides no action to delete IPs from a subscription. Set " +
			"`purge_on_destroy` to purge their vulnerability data instead, or remove the " +
			"addresses from the Qualys UI.",

		CreateContext: resourceIPRegistrationCreate,
		ReadContext:   resourceIPRegistrationRead,
		UpdateContext: resourceIPRegistrationUpdate,
		DeleteContext: resourceIPRegistrationDelete,

		Schema: map[string]*schema.Schema{
			"ips": {
				Description: "Addresses to register. Accepts single addresses, hyphenated " +
					"ranges and CIDR blocks; CIDR is converted to the ranges the Qualys API " +
					"expects. Changing this forces replacement, because the API cannot " +
					"un-register an address.",
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      hashNormalizedIP,
			},
			"network_id": {
				Description: "Qualys network these addresses belong to. Declare this " +
					"explicitly rather than relying on the default.",
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"enable_vm": {
				Description: "Enable these addresses for Vulnerability Management scanning.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
			},
			"enable_pc": {
				Description: "Enable these addresses for Policy Compliance scanning.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
			},

			"tracking_method": {
				Description: "How Qualys identifies these hosts. `EC2` and `AGENT` cannot be " +
					"set through this API, and a host already tracked by either cannot be " +
					"changed away from it.",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"IP", "DNS", "NETBIOS"}, false),
			},

			"owner":   {Type: schema.TypeString, Optional: true},
			"ud1":     {Type: schema.TypeString, Optional: true},
			"ud2":     {Type: schema.TypeString, Optional: true},
			"ud3":     {Type: schema.TypeString, Optional: true},
			"comment": {Type: schema.TypeString, Optional: true},

			"asset_group_title": {
				Description: "Add the addresses to this existing asset group as they are " +
					"registered. Applied at creation only.",
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"purge_on_destroy": {
				Description: "When destroying, purge the hosts' vulnerability data.\n\n" +
					"This is destructive and irreversible: it deletes vulnerability and " +
					"compliance data, tickets and exceptions, and removes the assets from " +
					"AssetView. It does not un-register the addresses, which the API cannot " +
					"do. Purging is asynchronous, so the data may persist briefly after " +
					"apply returns. Defaults to `false`.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceIPRegistrationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	in := vmdr.AddIPsInput{
		IPs:             stringSet(d, "ips"),
		EnableVM:        d.Get("enable_vm").(bool),
		EnablePC:        d.Get("enable_pc").(bool),
		TrackingMethod:  d.Get("tracking_method").(string),
		Owner:           d.Get("owner").(string),
		UD1:             d.Get("ud1").(string),
		UD2:             d.Get("ud2").(string),
		UD3:             d.Get("ud3").(string),
		Comment:         d.Get("comment").(string),
		AssetGroupTitle: d.Get("asset_group_title").(string),
	}
	if err := c.AddIPs(ctx, in); err != nil {
		return diag.FromErr(err)
	}

	// The add action returns no identifier, so the ID is derived from what the
	// resource manages: the normalised address set within its network.
	normalised, err := vmdr.NormalizeIPSet(in.IPs)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(append([]string{d.Get("network_id").(string)}, normalised...)))

	return resourceIPRegistrationRead(ctx, d, meta)
}

func resourceIPRegistrationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ips := stringSet(d, "ips")
	if len(ips) == 0 {
		d.SetId("")
		return nil
	}

	hosts, err := c.ListHosts(ctx, vmdr.HostFilter{
		IPs:       ips,
		NetworkID: d.Get("network_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	if len(hosts) == 0 {
		// None of the managed addresses are registered any more. Since the API
		// cannot un-register, this usually means they were removed in the UI.
		d.SetId("")
		return nil
	}

	// Host attributes are per host, while this resource manages a set of them.
	// Report the first host's values rather than inventing an aggregate: they are
	// applied uniformly by Update, so they agree unless changed outside Terraform.
	h := hosts[0]
	return diag.FromErr(setAll(d, map[string]interface{}{
		"tracking_method": h.TrackingMethod,
		"owner":           h.Owner,
		"comment":         h.Comments,
	}))
}

func resourceIPRegistrationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = c.UpdateIPs(ctx, stringSet(d, "ips"), d.Get("network_id").(string), vmdr.HostAssetUpdate{
		TrackingMethod: d.Get("tracking_method").(string),
		Owner:          d.Get("owner").(string),
		UD1:            d.Get("ud1").(string),
		UD2:            d.Get("ud2").(string),
		UD3:            d.Get("ud3").(string),
		Comment:        d.Get("comment").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceIPRegistrationRead(ctx, d, meta)
}

// resourceIPRegistrationDelete cannot un-register addresses: the API has no such
// action. It optionally purges their data, and always explains what remains.
func resourceIPRegistrationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	ips := stringSet(d, "ips")

	if !d.Get("purge_on_destroy").(bool) {
		d.SetId("")
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  "IP addresses removed from state but still registered in Qualys",
			Detail: fmt.Sprintf("The Qualys API provides no action to remove IP addresses "+
				"from a subscription, so %s remain registered. Set purge_on_destroy to "+
				"delete their vulnerability data, or remove them from the Qualys UI.",
				strings.Join(ips, ", ")),
		}}
	}

	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.PurgeHosts(ctx, vmdr.PurgeInput{
		IPs:        ips,
		NetworkIDs: nonEmpty(d.Get("network_id").(string)),
	}); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Host data queued for purging; addresses remain registered",
		Detail: "Vulnerability data for the managed addresses has been queued for purging. " +
			"Purging is asynchronous, so the data may persist for a short time. The " +
			"addresses themselves remain registered with the subscription — the Qualys " +
			"API cannot un-register them.",
	}}
}

func nonEmpty(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return []string{s}
}
