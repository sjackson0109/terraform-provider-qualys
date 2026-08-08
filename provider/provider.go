package provider

import (
	"context"
	"log"
	"os"

	"github.com/form3tech-oss/terraform-provider-qualys/cloudview/gcp"
	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("QUALYS_USERNAME", ""),
				Description: "Username for the Qualys API.",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("QUALYS_PASSWORD", ""),
				Description: "Password for the Qualys API.",
			},
			"base_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("QUALYS_URL", ""),
				Description: "Qualys API server URL for your subscription's platform, for " +
					"example `https://qualysapi.qualys.com`. Qualys hosts subscriptions on " +
					"several platforms with different hostnames; find yours under Help > About " +
					"in the Qualys UI, or at https://www.qualys.com/platform-identification/. " +
					"This must be set explicitly — there is no safe default, and pointing at " +
					"the wrong platform sends your credentials to it.",
			},
			"concurrency": {
				Type:     schema.TypeInt,
				Optional: true,
				Description: "Maximum concurrent API calls. Defaults to 2, matching the " +
					"documented per-subscription default. Raise this only if Qualys has " +
					"raised the limit for your subscription; exceeding it causes HTTP 409 " +
					"responses rather than faster runs.",
			},
			"max_retries": {
				Type:     schema.TypeInt,
				Optional: true,
				Description: "How many times to retry a call blocked by an API rate or " +
					"concurrency limit. Defaults to 4. Destructive operations are never " +
					"retried automatically regardless of this setting.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"qualys_gcp_connector":      dataSourceGCPConnector(),
			"qualys_asset_groups":       dataSourceAssetGroups(),
			"qualys_asset_tags":         dataSourceAssetTags(),
			"qualys_networks":           dataSourceNetworks(),
			"qualys_host_assets":        dataSourceHostAssets(),
			"qualys_scanner_appliances": dataSourceScannerAppliances(),
			"qualys_vaults":             dataSourceVaults(),
			"qualys_web_applications":   dataSourceWebApplications(),
			"qualys_scan_schedules":     dataSourceScanSchedules(),
			"qualys_option_profiles":    dataSourceOptionProfiles(),
			"qualys_host_detections":    dataSourceHostDetections(),
			"qualys_report_templates":   dataSourceReportTemplates(),
			"qualys_domains":            dataSourceDomains(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"qualys_gcp_connector":       resourceGCPConnector(),
			"qualys_asset_group":         resourceAssetGroup(),
			"qualys_asset_tag":           resourceAssetTag(),
			"qualys_static_search_list":  resourceStaticSearchList(),
			"qualys_vm_option_profile":   resourceVMOptionProfile(),
			"qualys_network":             resourceNetwork(),
			"qualys_ip_registration":     resourceIPRegistration(),
			"qualys_scan_schedule":       resourceScanSchedule(),
			"qualys_virtual_scanner":     resourceVirtualScanner(),
			"qualys_auth_record_windows": resourceWindowsAuthRecord(),
			"qualys_auth_record_unix":    resourceUnixAuthRecord(),
			"qualys_web_application":     resourceWebApplication(),
			"qualys_was_option_profile":  resourceWASOptionProfile(),
			"qualys_was_auth_record":     resourceWASAuthRecord(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	baseURL := d.Get("base_url").(string)

	vmdrClient, err := vmdr.NewClient(vmdr.Config{
		BaseURL:     baseURL,
		Username:    username,
		Password:    password,
		UserAgent:   "terraform-provider-qualys",
		Concurrency: d.Get("concurrency").(int),
		MaxRetries:  d.Get("max_retries").(int),
		// Warnings about deprecated API versions and limit waits go to the
		// Terraform log so an operator sees them in TF_LOG output.
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	qpsClient, err := qps.NewClient(qps.Config{
		BaseURL:     baseURL,
		Username:    username,
		Password:    password,
		UserAgent:   "terraform-provider-qualys",
		Concurrency: d.Get("concurrency").(int),
		MaxRetries:  d.Get("max_retries").(int),
		Logger:      log.New(os.Stderr, "", log.LstdFlags),
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return &clients{
		vmdr: vmdrClient,
		qps:  qpsClient,
		// The CloudView client is retained for the existing GCP connector
		// resource until it is migrated to the Connector v3 API.
		gcp: gcp.NewService(baseURL, username, password),
	}, nil
}
