package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVirtualScanner() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys virtual scanner appliance.\n\n" +
			"This resource covers the Qualys side of provisioning: it registers the " +
			"appliance and exports the `activation_code` needed to personalise the " +
			"appliance VM. Deploying that VM is a separate job for whichever " +
			"infrastructure provider hosts it (vSphere, AWS, Azure, GCP, …), which " +
			"consumes the code exported here.\n\n" +
			"Only virtual appliances can be created and deleted through the API. " +
			"Physical appliances are registered out of band.",

		CreateContext: resourceVirtualScannerCreate,
		ReadContext:   resourceVirtualScannerRead,
		UpdateContext: resourceVirtualScannerUpdate,
		DeleteContext: resourceVirtualScannerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the scanner appliance.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"polling_interval": {
				Description: "How often the appliance polls Qualys, in seconds (60–360). " +
					"Defaults to Qualys' own default of 180.",
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(60, 360),
			},
			"asset_group_id": {
				Description: "Asset group to create the appliance in.\n\n" +
					"Required for Unit Manager and Scanner accounts. **Must be omitted for " +
					"Manager accounts** — Qualys rejects the request if a Manager supplies it.",
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"comments": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network_id": {
				Description: "Qualys network to assign the appliance to. An appliance belongs " +
					"to exactly one network, and a network with no appliance cannot be " +
					"scanned. Requires Manager privileges.",
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vlan": {
				Description: "VLANs configured on the appliance.\n\n" +
					"Declaring this block makes Terraform authoritative over the appliance's " +
					"VLANs: the Qualys API replaces the whole list on every update, so any " +
					"VLAN not listed here is removed. Omit the block entirely to leave " +
					"VLANs managed elsewhere untouched.",
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vlan_id": {Type: schema.TypeString, Required: true},
						"ip":      {Type: schema.TypeString, Required: true},
						"netmask": {Type: schema.TypeString, Required: true},
						"name":    {Type: schema.TypeString, Required: true},
					},
				},
			},
			"static_route": {
				Description: "Static routes configured on the appliance. Authoritative in the " +
					"same way as `vlan`.",
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip":      {Type: schema.TypeString, Required: true},
						"netmask": {Type: schema.TypeString, Required: true},
						"gateway": {Type: schema.TypeString, Required: true},
						"name":    {Type: schema.TypeString, Required: true},
					},
				},
			},

			"wait_for_online": {
				Description: "Wait for the appliance to come online before completing.\n\n" +
					"Only useful when the appliance VM is deployed in the same apply. Qualys " +
					"checks appliance health every four hours, so a newly activated appliance " +
					"can take a while to report Online — size `wait_for_online_timeout` " +
					"accordingly rather than expecting seconds.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"wait_for_online_timeout": {
				Description: "How long to wait for the appliance to come online, as a Go " +
					"duration. Defaults to `30m`.",
				Type:     schema.TypeString,
				Optional: true,
				Default:  "30m",
			},

			"activation_code": {
				Description: "Personalisation code for deploying the appliance VM.\n\n" +
					"Qualys returns this only until the appliance activates; afterwards the " +
					"value is no longer available from the API. Terraform keeps the code it " +
					"received at creation rather than clearing it, so it stays usable for " +
					"reference — but it cannot be recovered if lost from state before the " +
					"appliance is deployed.",
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},

			"status":             {Type: schema.TypeString, Computed: true},
			"uuid":               {Type: schema.TypeString, Computed: true},
			"software_version":   {Type: schema.TypeString, Computed: true},
			"vulnsigs_version":   {Type: schema.TypeString, Computed: true},
			"heartbeats_missed":  {Type: schema.TypeString, Computed: true},
			"up_to_date":         {Type: schema.TypeBool, Computed: true},
			"running_scan_count": {Type: schema.TypeInt, Computed: true},
		},
	}
}

func resourceVirtualScannerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	created, err := c.CreateVirtualScanner(ctx, vmdr.VirtualScannerInput{
		Name:            d.Get("name").(string),
		PollingInterval: d.Get("polling_interval").(int),
		AssetGroupID:    d.Get("asset_group_id").(string),
		Comments:        d.Get("comments").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(created.ID)

	// Store the activation code now. Qualys stops returning it once the appliance
	// activates, and it cannot be recovered afterwards.
	if err := d.Set("activation_code", created.ActivationCode); err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	if networkID := d.Get("network_id").(string); networkID != "" {
		// Network assignment is a separate action, not a create parameter.
		if err := c.AssignApplianceToNetwork(ctx, created.ID, networkID); err != nil {
			// The appliance exists; failing here would strand it outside state.
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Scanner created but network assignment failed",
				Detail: fmt.Sprintf("%v\n\nThe appliance exists and is under management. "+
					"Assigning an appliance to a network requires Manager privileges; "+
					"re-apply once the account has them, or assign it in the Qualys UI.", err),
			})
		}
	}

	if diags2 := applyApplianceConfig(ctx, c, d); diags2.HasError() {
		return append(diags, diags2...)
	} else {
		diags = append(diags, diags2...)
	}

	if d.Get("wait_for_online").(bool) {
		if err := waitForApplianceOnline(ctx, c, d); err != nil {
			return append(diags, diag.FromErr(err)...)
		}
	}

	return append(diags, resourceVirtualScannerRead(ctx, d, meta)...)
}

// applyApplianceConfig sends VLANs and routes, but only those the configuration
// actually declares.
func applyApplianceConfig(ctx context.Context, c *vmdr.Client, d *schema.ResourceData) diag.Diagnostics {
	vlansRaw, hasVLANs := d.GetOk("vlan")
	routesRaw, hasRoutes := d.GetOk("static_route")

	up := vmdr.ApplianceUpdate{
		Name:            d.Get("name").(string),
		Comments:        d.Get("comments").(string),
		PollingInterval: d.Get("polling_interval").(int),
	}

	// Only claim authority over a list the configuration declares. Sending
	// set_vlans unconditionally would clear VLANs configured outside Terraform.
	if hasVLANs {
		up.SetVLANs = true
		for _, raw := range vlansRaw.([]interface{}) {
			m := raw.(map[string]interface{})
			up.VLANs = append(up.VLANs, vmdr.VLAN{
				ID:      m["vlan_id"].(string),
				IP:      m["ip"].(string),
				Netmask: m["netmask"].(string),
				Name:    m["name"].(string),
			})
		}
	}
	if hasRoutes {
		up.SetRoutes = true
		for _, raw := range routesRaw.([]interface{}) {
			m := raw.(map[string]interface{})
			up.Routes = append(up.Routes, vmdr.StaticRoute{
				IP:      m["ip"].(string),
				Netmask: m["netmask"].(string),
				Gateway: m["gateway"].(string),
				Name:    m["name"].(string),
			})
		}
	}

	if err := c.UpdateAppliance(ctx, d.Id(), up); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func waitForApplianceOnline(ctx context.Context, c *vmdr.Client, d *schema.ResourceData) error {
	timeout, err := time.ParseDuration(d.Get("wait_for_online_timeout").(string))
	if err != nil {
		return fmt.Errorf("wait_for_online_timeout is not a valid duration: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Qualys checks appliance health every four hours, so polling faster than
	// about a minute only burns API quota without learning anything sooner.
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		a, err := c.GetAppliance(waitCtx, d.Id())
		if err == nil && a.Online() {
			return nil
		}
		if err != nil && !errors.Is(err, vmdr.ErrNotFound) {
			return err
		}

		select {
		case <-ticker.C:
		case <-waitCtx.Done():
			return fmt.Errorf("scanner appliance %s did not come online within %s. "+
				"The appliance VM must be deployed and personalised with the activation "+
				"code before Qualys will see it, and Qualys checks appliance health every "+
				"four hours, so this may simply need longer", d.Id(), timeout)
		}
	}
}

func resourceVirtualScannerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	a, err := c.GetAppliance(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	values := map[string]interface{}{
		"name":               a.Name,
		"status":             a.Status,
		"uuid":               a.UUID,
		"software_version":   a.Version,
		"vulnsigs_version":   a.VulnSigsVersion,
		"heartbeats_missed":  a.HeartbeatsMissed,
		"up_to_date":         a.UpToDate(),
		"running_scan_count": a.RunningScanCount,
		"comments":           a.Comments,
	}
	if a.PollingInterval > 0 {
		values["polling_interval"] = a.PollingInterval
	}
	if a.NetworkID != "" {
		values["network_id"] = a.NetworkID
	}

	// Only update the activation code when Qualys still returns one. It stops
	// returning the code once the appliance activates, and overwriting the stored
	// value with an empty string would erase it from state on the first apply
	// after activation.
	if a.ActivationCode != "" {
		values["activation_code"] = a.ActivationCode
	}

	return diag.FromErr(setAll(d, values))
}

func resourceVirtualScannerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if diags := applyApplianceConfig(ctx, c, d); diags.HasError() {
		return diags
	}

	if d.HasChange("network_id") {
		if networkID := d.Get("network_id").(string); networkID != "" {
			if err := c.AssignApplianceToNetwork(ctx, d.Id(), networkID); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return resourceVirtualScannerRead(ctx, d, meta)
}

func resourceVirtualScannerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteVirtualScanner(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil
		}
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Failed to delete virtual scanner",
			Detail: fmt.Sprintf("%v\n\nQualys refuses to delete an appliance while it is "+
				"running scans. If a scan is in progress, wait for it to finish or cancel "+
				"it, then apply again.", err),
		}}
	}

	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "Virtual scanner deleted; dependent configuration was changed by Qualys",
		Detail: "Deleting a scanner appliance removes it from any asset groups it belonged " +
			"to and deactivates scan schedules that referenced it. Qualys applies those " +
			"changes itself, so affected schedules may now show as inactive.",
	}}
}
