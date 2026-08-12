package provider

import (
	"context"
	"sort"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// authRecordTypeSlugs are the Confirmed type slugs from doc 03 §11.
var authRecordTypeSlugs = []string{
	"windows", "unix", "network_ssh", "oracle", "oracle_listener", "snmp",
	"ms_sql", "mysql", "postgresql", "sybase", "ibm_db2", "informixdb",
	"mariadb", "mongodb", "neo4j", "sapiq", "sap_hana", "vmware", "vcenter",
	"http", "apache", "nginx", "ms_iis", "ibm_websphere", "tomcat",
	"oracle_weblogic", "jboss", "kubernetes", "docker", "palo_alto_firewall",
}

func dataSourceAuthRecords() *schema.Resource {
	return &schema.Resource{
		Description: "Look up authentication records of one type. The VM/PA authentication " +
			"API is per-type (`/api/2.0/fo/auth/<type>/`, doc 03 §11), so `type` selects " +
			"which endpoint this reads from — there is no cross-type list.",

		ReadContext: dataSourceAuthRecordsRead,

		Schema: map[string]*schema.Schema{
			"type": {
				Description:  "Authentication record type slug, e.g. `windows` or `oracle`.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(authRecordTypeSlugs, false),
			},
			"ids": {
				Description: "Return only records with these IDs. Empty returns every " +
					"record of the given type.",
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"auth_records": {
				Description: "Matching authentication records, ordered by ID for stable " +
					"output. Never includes a password: credentials are write-only.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":         {Type: schema.TypeString, Computed: true},
						"title":      {Type: schema.TypeString, Computed: true},
						"username":   {Type: schema.TypeString, Computed: true},
						"comments":   {Type: schema.TypeString, Computed: true},
						"ips":        {Type: schema.TypeList, Computed: true, Elem: &schema.Schema{Type: schema.TypeString}},
						"vault_id":   {Type: schema.TypeString, Computed: true},
						"vault_type": {Type: schema.TypeString, Computed: true},
						"created":    {Type: schema.TypeString, Computed: true},
						"modified":   {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceAuthRecordsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	t := vmdr.AuthRecordType(d.Get("type").(string))
	records, err := c.ListAuthRecords(ctx, t, stringSet(d, "ids"))
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })

	out := make([]interface{}, 0, len(records))
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.ID)
		out = append(out, map[string]interface{}{
			"id":         r.ID,
			"title":      r.Title,
			"username":   r.Username,
			"comments":   r.Comments,
			"ips":        r.IPs,
			"vault_id":   r.VaultID,
			"vault_type": r.VaultType,
			"created":    r.Created,
			"modified":   r.Modified,
		})
	}

	if err := d.Set("auth_records", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(append([]string{string(t)}, ids...)))
	return nil
}
