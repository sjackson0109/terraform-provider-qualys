package provider

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sjackson0109/terraform-provider-qualys/qps"
	"github.com/sjackson0109/terraform-provider-qualys/vmdr"
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
			"qualys_gcp_connector":        dataSourceGCPConnector(),
			"qualys_aws_connector":        dataSourceAWSConnector(),
			"qualys_azure_connector":      dataSourceAzureConnector(),
			"qualys_asset_groups":         dataSourceAssetGroups(),
			"qualys_asset_tags":           dataSourceAssetTags(),
			"qualys_networks":             dataSourceNetworks(),
			"qualys_host_assets":          dataSourceHostAssets(),
			"qualys_scanner_appliances":   dataSourceScannerAppliances(),
			"qualys_vaults":               dataSourceVaults(),
			"qualys_was_applications":     dataSourceWASApplications(),
			"qualys_vm_scan_schedules":    dataSourceVMScanSchedules(),
			"qualys_vm_option_profiles":   dataSourceVMOptionProfiles(),
			"qualys_host_detections":      dataSourceHostDetections(),
			"qualys_vm_report_templates":  dataSourceVMReportTemplates(),
			"qualys_domains":              dataSourceDomains(),
			"qualys_vm_findings":          dataSourceVMFindings(),
			"qualys_was_findings":         dataSourceWASFindings(),
			"qualys_was_report_templates": dataSourceWASReportTemplates(),
			"qualys_was_option_profiles":  dataSourceWASOptionProfiles(),
			"qualys_was_auth_records":     dataSourceWASAuthRecords(),
			"qualys_was_dns_overrides":    dataSourceWASDNSOverrides(),
			"qualys_static_search_lists":  dataSourceStaticSearchLists(),

			"qualys_tagged_assets":       dataSourceTaggedAssets(),
			"qualys_vm_auth_records":     dataSourceVMAuthRecords(),
			"qualys_vm_scans":            dataSourceScans(),
			"qualys_reports":             dataSourceReports(),
			"qualys_vm_report_schedules": dataSourceVMReportSchedules(),
			"qualys_tenant_capabilities": dataSourceTenantCapabilities(),
			"qualys_users":               dataSourceUsers(),
			"qualys_am_users":            dataSourceAMUsers(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"qualys_gcp_connector":       resourceGCPConnector(),
			"qualys_aws_connector":       resourceAWSConnector(),
			"qualys_azure_connector":     resourceAzureConnector(),
			"qualys_asset_group":         resourceAssetGroup(),
			"qualys_asset_tag":           resourceAssetTag(),
			"qualys_static_search_list":  resourceStaticSearchList(),
			"qualys_vm_option_profile":   resourceVMOptionProfile(),
			"qualys_network":             resourceNetwork(),
			"qualys_ip_registration":     resourceIPRegistration(),
			"qualys_vm_scan_schedule":    resourceVMScanSchedule(),
			"qualys_virtual_scanner":     resourceVirtualScanner(),
			"qualys_auth_record_windows": resourceWindowsAuthRecord(),
			"qualys_auth_record_unix":    resourceUnixAuthRecord(),

			// The remaining confirmed authentication record types (doc 03 §11)
			// share the identical generic CRUD shape Windows and Unix already
			// use, so they are registered directly against the factory rather
			// than each getting its own named wrapper function.
			"qualys_auth_record_network_ssh":     resourceAuthRecord(vmdr.AuthNetworkSSH, "Network (SSH)", nil),
			"qualys_auth_record_oracle":          resourceAuthRecord(vmdr.AuthOracle, "Oracle", nil),
			"qualys_auth_record_oracle_listener": resourceAuthRecord(vmdr.AuthOracleListener, "Oracle Listener", nil),
			"qualys_auth_record_snmp":            resourceAuthRecord(vmdr.AuthSNMP, "SNMP", nil),
			"qualys_auth_record_mssql":           resourceAuthRecord(vmdr.AuthMSSQL, "Microsoft SQL Server", nil),
			"qualys_auth_record_mysql":           resourceAuthRecord(vmdr.AuthMySQL, "MySQL", nil),
			"qualys_auth_record_postgresql":      resourceAuthRecord(vmdr.AuthPostgre, "PostgreSQL", nil),
			"qualys_auth_record_sybase":          resourceAuthRecord(vmdr.AuthSybase, "Sybase", nil),
			"qualys_auth_record_ibm_db2":         resourceAuthRecord(vmdr.AuthIBMDB2, "IBM DB2", nil),
			"qualys_auth_record_informix":        resourceAuthRecord(vmdr.AuthInformix, "Informix", nil),
			"qualys_auth_record_mariadb":         resourceAuthRecord(vmdr.AuthMariaDB, "MariaDB", nil),
			"qualys_auth_record_mongodb":         resourceAuthRecord(vmdr.AuthMongoDB, "MongoDB", nil),
			"qualys_auth_record_neo4j":           resourceAuthRecord(vmdr.AuthNeo4j, "Neo4j", nil),
			"qualys_auth_record_sapiq":           resourceAuthRecord(vmdr.AuthSAPIQ, "SAP IQ", nil),
			"qualys_auth_record_sap_hana":        resourceAuthRecord(vmdr.AuthSAPHANA, "SAP HANA", nil),
			"qualys_auth_record_vmware":          resourceAuthRecord(vmdr.AuthVMware, "VMware ESX/ESXi", nil),
			"qualys_auth_record_vcenter":         resourceAuthRecord(vmdr.AuthVCenter, "VMware vCenter", nil),
			"qualys_auth_record_http":            resourceAuthRecord(vmdr.AuthHTTP, "HTTP", nil),
			"qualys_auth_record_apache":          resourceAuthRecord(vmdr.AuthApache, "Apache", nil),
			"qualys_auth_record_nginx":           resourceAuthRecord(vmdr.AuthNginx, "Nginx", nil),
			"qualys_auth_record_ms_iis":          resourceAuthRecord(vmdr.AuthMSIIS, "Microsoft IIS", nil),
			"qualys_auth_record_ibm_websphere":   resourceAuthRecord(vmdr.AuthIBMWebSphere, "IBM WebSphere", nil),
			"qualys_auth_record_tomcat":          resourceAuthRecord(vmdr.AuthTomcat, "Apache Tomcat", nil),
			"qualys_auth_record_oracle_weblogic": resourceAuthRecord(vmdr.AuthOracleWebLogic, "Oracle WebLogic", nil),
			"qualys_auth_record_jboss":           resourceAuthRecord(vmdr.AuthJBoss, "JBoss", nil),
			"qualys_auth_record_kubernetes":      resourceAuthRecord(vmdr.AuthKubernetes, "Kubernetes", nil),
			"qualys_auth_record_docker":          resourceAuthRecord(vmdr.AuthDocker, "Docker", nil),
			"qualys_auth_record_palo_alto":       resourceAuthRecord(vmdr.AuthPaloAlto, "Palo Alto Firewall", nil),

			"qualys_vault":                 resourceVault(),
			"qualys_excluded_ips":          resourceExcludedIPs(),
			"qualys_host_tag_assignment":   resourceHostTagAssignment(),
			"qualys_user_scope_assignment": resourceUserScopeAssignment(),

			"qualys_was_application":     resourceWASApplication(),
			"qualys_was_option_profile":  resourceWASOptionProfile(),
			"qualys_was_auth_record":     resourceWASAuthRecord(),
			"qualys_was_scan_schedule":   resourceWASScanSchedule(),
			"qualys_was_report_schedule": resourceWASReportSchedule(),
			"qualys_was_finding_ignore":  resourceWASFindingIgnore(),
			"qualys_was_dns_override":    resourceWASDNSOverride(),
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
	}, nil
}
