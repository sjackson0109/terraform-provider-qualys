package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAssetGroup() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys asset group: a named set of scan targets referenced by scans, " +
			"schedules and reports.\n\n" +
			"Member lists are managed authoritatively. Removing an entry from the " +
			"configuration removes it from the group, and emptying a list empties it in " +
			"Qualys.",

		CreateContext: resourceAssetGroupCreate,
		ReadContext:   resourceAssetGroupRead,
		UpdateContext: resourceAssetGroupUpdate,
		DeleteContext: resourceAssetGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Name of the asset group. Must be unique within the subscription.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"network_id": {
				Description: "ID of the Qualys network this group belongs to. Defaults to the " +
					"Global Default Network (`0`) when omitted.\n\n" +
					"Declare the intended network explicitly rather than relying on the " +
					"default. Changing it forces replacement: the Qualys edit action provides " +
					"no way to move an existing group between networks.",
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"comments": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"ips": ipSetSchema("IP addresses, hyphenated ranges, or CIDR blocks in this group. "+
				"CIDR is accepted for convenience and converted to the hyphenated ranges "+
				"the Qualys API expects. Set membership is compared by address range, so "+
				"`10.0.0.0/24` and `10.0.0.0-10.0.0.255` are the same entry and neither "+
				"produces a permanent diff.", false),
			"domains": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"dns_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"netbios_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"appliance_ids": {
				Description: "Scanner appliance IDs assigned to this group.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"default_appliance_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"division": {Type: schema.TypeString, Optional: true},
			"function": {Type: schema.TypeString, Optional: true},
			"location": {Type: schema.TypeString, Optional: true},

			"business_impact": {
				Description: "Business impact rating, applied to every host in the group.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ValidateFunc: validation.StringInSlice(
					[]string{"critical", "high", "medium", "low", "none"}, true),
			},

			"cvss_enviro_cdp": {
				Description:  "CVSS environmental collateral damage potential.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"high", "medium-high", "low-medium", "low", "none"}, true),
			},
			"cvss_enviro_td": {
				Description:  "CVSS environmental target distribution.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"high", "medium", "low", "none"}, true),
			},
			"cvss_enviro_cr": {
				Description:  "CVSS environmental confidentiality requirement.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"high", "medium", "low"}, true),
			},
			"cvss_enviro_ir": {
				Description:  "CVSS environmental integrity requirement.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"high", "medium", "low"}, true),
			},
			"cvss_enviro_ar": {
				Description:  "CVSS environmental availability requirement.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"high", "medium", "low"}, true),
			},

			"owner_id":    {Type: schema.TypeString, Computed: true},
			"unit_id":     {Type: schema.TypeString, Computed: true},
			"last_update": {Type: schema.TypeString, Computed: true},
		},
	}
}

func assetGroupInputFrom(d *schema.ResourceData) vmdr.AssetGroupInput {
	return vmdr.AssetGroupInput{
		Title:              d.Get("title").(string),
		NetworkID:          d.Get("network_id").(string),
		Comments:           d.Get("comments").(string),
		IPs:                stringSet(d, "ips"),
		Domains:            stringSet(d, "domains"),
		DNSNames:           stringSet(d, "dns_names"),
		NetBIOSNames:       stringSet(d, "netbios_names"),
		ApplianceIDs:       stringSet(d, "appliance_ids"),
		DefaultApplianceID: d.Get("default_appliance_id").(string),
		Division:           d.Get("division").(string),
		Function:           d.Get("function").(string),
		Location:           d.Get("location").(string),
		BusinessImpact:     d.Get("business_impact").(string),
		CVSSEnviroCDP:      d.Get("cvss_enviro_cdp").(string),
		CVSSEnviroTD:       d.Get("cvss_enviro_td").(string),
		CVSSEnviroCR:       d.Get("cvss_enviro_cr").(string),
		CVSSEnviroIR:       d.Get("cvss_enviro_ir").(string),
		CVSSEnviroAR:       d.Get("cvss_enviro_ar").(string),
	}
}

func resourceAssetGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := c.CreateAssetGroup(ctx, assetGroupInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceAssetGroupRead(ctx, d, meta)
}

func resourceAssetGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	g, err := c.GetAssetGroup(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			// Gone from Qualys; drop it from state so the next plan recreates it.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"title":                g.Title,
		"network_id":           g.NetworkID,
		"comments":             g.Comments,
		"ips":                  g.IPs,
		"domains":              g.Domains,
		"dns_names":            g.DNSNames,
		"netbios_names":        g.NetBIOSNames,
		"appliance_ids":        g.ApplianceIDs,
		"default_appliance_id": g.DefaultApplianceID,
		"division":             g.Division,
		"function":             g.Function,
		"location":             g.Location,
		"business_impact":      g.BusinessImpact,
		"cvss_enviro_cdp":      g.CVSSEnviroCDP,
		"cvss_enviro_td":       g.CVSSEnviroTD,
		"cvss_enviro_cr":       g.CVSSEnviroCR,
		"cvss_enviro_ir":       g.CVSSEnviroIR,
		"cvss_enviro_ar":       g.CVSSEnviroAR,
		"owner_id":             g.OwnerID,
		"unit_id":              g.UnitID,
		"last_update":          g.LastUpdate,
	}))
}

func resourceAssetGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateAssetGroup(ctx, d.Id(), assetGroupInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceAssetGroupRead(ctx, d, meta)
}

func resourceAssetGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteAssetGroup(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil // already gone
		}
		return diag.FromErr(err)
	}
	return nil
}
