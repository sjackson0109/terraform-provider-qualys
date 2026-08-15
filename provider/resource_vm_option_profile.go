package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceVMOptionProfile() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys VM scan option profile: the scan behaviour (ports, detection " +
			"depth, performance, authentication) that scans and schedules reference.\n\n" +
			"Some option profile settings do not survive a round trip through the Qualys " +
			"API and are therefore not exposed here — see the notes on individual " +
			"attributes and the provider documentation.",

		CreateContext: resourceVMOptionProfileCreate,
		ReadContext:   resourceVMOptionProfileRead,
		UpdateContext: resourceVMOptionProfileUpdate,
		DeleteContext: resourceVMOptionProfileDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Name of the option profile.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"owner": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"global": {
				Description: "Whether the profile is available to all users in the " +
					"subscription. Setting `is_default` also makes the profile global, so " +
					"leaving this false alongside `is_default = true` will not hold.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"is_default": {
				Description: "Whether this is the default profile for new scans and maps.\n\n" +
					"Qualys makes a default profile global as a side effect, so setting this " +
					"to `true` will also set `global` to `true` whatever the configuration " +
					"says. Set both together to keep plans honest.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"offline_scanner": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			// Ports
			"scan_tcp_ports": {
				Description:  "TCP ports to scan.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "standard",
				ValidateFunc: validation.StringInSlice([]string{"none", "full", "standard", "light"}, false),
			},
			"scan_tcp_ports_additional": {
				Description: "Extra TCP ports to scan beyond `scan_tcp_ports`, comma separated.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"scan_udp_ports": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "standard",
				ValidateFunc: validation.StringInSlice([]string{"none", "full", "standard", "light"}, false),
			},
			"scan_udp_ports_additional": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"three_way_handshake": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			// Dead hosts
			"scan_dead_hosts": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"close_vuln_on_dead_hosts": {
				Description: "Mark vulnerabilities as fixed when the host is found dead.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"purge_host_data": {
				Description: "Purge host data when the host is found dead for the configured " +
					"number of consecutive scans. This deletes vulnerability history in Qualys.",
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			// Performance
			"scan_overall_performance": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "normal",
				ValidateFunc: validation.StringInSlice([]string{"high", "normal", "low", "custom"}, false),
			},
			"scan_intensity": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"normal", "medium", "low", "minimum"}, false),
			},
			"scan_packet_delay": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"minimum", "short", "medium", "long", "maximum"}, false),
			},
			"scan_parallel_scaling":   {Type: schema.TypeBool, Optional: true, Default: false},
			"scan_external_scanners":  {Type: schema.TypeString, Optional: true},
			"scan_scanner_appliances": {Type: schema.TypeString, Optional: true},
			"scan_total_process":      {Type: schema.TypeString, Optional: true},
			"scan_http_process":       {Type: schema.TypeString, Optional: true},

			// Detection
			"vulnerability_detection": {
				Description: "Which vulnerabilities to test for. `custom` requires " +
					"`custom_search_list_ids`; Qualys silently falls back to complete " +
					"detection if a custom selection cannot be resolved, so the provider " +
					"rejects that combination at plan time instead.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "complete",
				ValidateFunc: validation.StringInSlice([]string{"complete", "custom", "runtime"}, false),
			},
			"custom_search_list_ids": {
				Description: "Static or dynamic search list IDs defining the QIDs to test " +
					"when `vulnerability_detection` is `custom`.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"exclude_search_list_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"password_brute_forcing_system": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"minimal", "limited", "standard", "exhaustive"}, false),
			},
			"basic_host_information_checks": {Type: schema.TypeBool, Optional: true, Default: true},
			"oval_checks":                   {Type: schema.TypeBool, Optional: true, Default: false},
			"all_qrdi_checks":               {Type: schema.TypeBool, Optional: true, Default: false},

			// Behaviour
			"additional_certificate_detection": {Type: schema.TypeBool, Optional: true, Default: false},
			"load_balancer_detection":          {Type: schema.TypeBool, Optional: true, Default: false},
			"enable_dissolvable_agent":         {Type: schema.TypeBool, Optional: true, Default: false},
			"test_authentication":              {Type: schema.TypeBool, Optional: true, Default: false},

			// Authentication
			"authentication": {
				Description: "Authentication technologies to attempt during the scan, for " +
					"example `Windows` or `Unix`. This is a list of technologies, not a set " +
					"of per-technology switches.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"include_system_auth":          {Type: schema.TypeBool, Optional: true, Default: false},
			"use_system_auth_on_duplicate": {Type: schema.TypeBool, Optional: true, Default: false},
			"use_user_auth_on_duplicate":   {Type: schema.TypeBool, Optional: true, Default: false},

			"modified": {Type: schema.TypeString, Computed: true},
		},

		CustomizeDiff: resourceVMOptionProfileCustomizeDiff,
	}
}

func resourceVMOptionProfileCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	if d.Get("vulnerability_detection").(string) == "custom" {
		ids, _ := d.Get("custom_search_list_ids").(*schema.Set)
		if ids == nil || ids.Len() == 0 {
			return errors.New(`vulnerability_detection = "custom" requires custom_search_list_ids; ` +
				"Qualys silently falls back to complete detection when a custom selection " +
				"cannot be resolved, which would scan far more than intended")
		}
	}

	// Qualys makes a default profile global. Surfacing that here keeps the plan
	// truthful rather than letting the next read show an unexplained change.
	if d.Get("is_default").(bool) && !d.Get("global").(bool) {
		return errors.New("is_default = true also makes the profile global in Qualys; " +
			"set global = true as well so the configuration matches what will exist")
	}
	return nil
}

func vmOptionProfileInputFrom(d *schema.ResourceData) vmdr.VMOptionProfileInput {
	return vmdr.VMOptionProfileInput{
		Title:   d.Get("title").(string),
		Owner:   d.Get("owner").(string),
		Global:  d.Get("global").(bool),
		Default: d.Get("is_default").(bool),
		Offline: d.Get("offline_scanner").(bool),

		ScanTCPPorts:           d.Get("scan_tcp_ports").(string),
		ScanTCPPortsAdditional: d.Get("scan_tcp_ports_additional").(string),
		ScanUDPPorts:           d.Get("scan_udp_ports").(string),
		ScanUDPPortsAdditional: d.Get("scan_udp_ports_additional").(string),
		ThreeWayHandshake:      d.Get("three_way_handshake").(bool),

		ScanDeadHosts:        d.Get("scan_dead_hosts").(bool),
		CloseVulnOnDeadHosts: d.Get("close_vuln_on_dead_hosts").(bool),
		PurgeHostData:        d.Get("purge_host_data").(bool),

		ScanOverallPerformance: d.Get("scan_overall_performance").(string),
		ScanIntensity:          d.Get("scan_intensity").(string),
		ScanPacketDelay:        d.Get("scan_packet_delay").(string),
		ScanParallelScaling:    d.Get("scan_parallel_scaling").(bool),
		ScanExternalScanners:   d.Get("scan_external_scanners").(string),
		ScanScannerAppliances:  d.Get("scan_scanner_appliances").(string),
		ScanTotalProcess:       d.Get("scan_total_process").(string),
		ScanHTTPProcess:        d.Get("scan_http_process").(string),

		VulnerabilityDetection: d.Get("vulnerability_detection").(string),
		CustomSearchListIDs:    stringSet(d, "custom_search_list_ids"),
		ExcludeSearchListIDs:   stringSet(d, "exclude_search_list_ids"),

		PasswordBruteForcingSystem: d.Get("password_brute_forcing_system").(string),
		BasicHostInformationChecks: d.Get("basic_host_information_checks").(bool),
		OVALChecks:                 d.Get("oval_checks").(bool),
		AllQRDIChecks:              d.Get("all_qrdi_checks").(bool),

		AdditionalCertificateDetection: d.Get("additional_certificate_detection").(bool),
		LoadBalancer:                   d.Get("load_balancer_detection").(bool),
		EnableDissolvableAgent:         d.Get("enable_dissolvable_agent").(bool),
		TestAuthentication:             d.Get("test_authentication").(bool),

		Authentication:           stringSet(d, "authentication"),
		IncludeSystemAuth:        d.Get("include_system_auth").(bool),
		UseSystemAuthOnDuplicate: d.Get("use_system_auth_on_duplicate").(bool),
		UseUserAuthOnDuplicate:   d.Get("use_user_auth_on_duplicate").(bool),
	}
}

func resourceVMOptionProfileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := c.CreateVMOptionProfile(ctx, vmOptionProfileInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceVMOptionProfileRead(ctx, d, meta)
}

func resourceVMOptionProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	p, err := c.GetVMOptionProfile(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"title":           p.Title,
		"owner":           p.Owner,
		"global":          p.Global,
		"is_default":      p.Default,
		"offline_scanner": p.Offline,

		"scan_tcp_ports":            p.ScanTCPPorts,
		"scan_tcp_ports_additional": p.ScanTCPPortsAdditional,
		"scan_udp_ports":            p.ScanUDPPorts,
		"scan_udp_ports_additional": p.ScanUDPPortsAdditional,

		"scan_dead_hosts":          p.ScanDeadHosts,
		"close_vuln_on_dead_hosts": p.CloseVulnOnDeadHosts,
		"purge_host_data":          p.PurgeHostData,

		"scan_overall_performance": p.ScanOverallPerformance,
		"scan_packet_delay":        p.ScanPacketDelay,
		"scan_parallel_scaling":    p.ScanParallelScaling,

		"vulnerability_detection": p.VulnerabilityDetection,
		"custom_search_list_ids":  p.CustomSearchListIDs,

		"password_brute_forcing_system": p.PasswordBruteForcingSystem,
		"basic_host_information_checks": p.BasicHostInformationChecks,
		"oval_checks":                   p.OVALChecks,
		"all_qrdi_checks":               p.AllQRDIChecks,

		"additional_certificate_detection": p.AdditionalCertificateDetection,
		"load_balancer_detection":          p.LoadBalancer,
		"enable_dissolvable_agent":         p.EnableDissolvableAgent,
		"test_authentication":              p.TestAuthentication,

		"authentication": p.Authentication,
		"modified":       p.Modified,
	}))
}

func resourceVMOptionProfileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateVMOptionProfile(ctx, d.Id(), vmOptionProfileInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceVMOptionProfileRead(ctx, d, meta)
}

func resourceVMOptionProfileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteVMOptionProfile(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil
		}
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Failed to delete option profile",
			Detail: fmt.Sprintf("%v\n\nScans and schedules that reference an option profile "+
				"are not updated when it is deleted. If this profile is still referenced, "+
				"change those configurations first.", err),
		}}
	}
	return nil
}
