package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceTenantCapabilities() *schema.Resource {
	return &schema.Resource{
		Description: "Probes the subscription's installed platform and per-module " +
			"versions, for feature-gating configuration on tenants that do not have every " +
			"module (WAS, VM, CA, FIM, ...) enabled. Calls `GET /qps/rest/portal/version` " +
			"(doc 11, \"Useful for the tenant capability probe\").\n\n" +
			"This session found no confirmed field-level schema for the response — not " +
			"even a class or DTD name, unlike every other object this provider models — so " +
			"`raw` is a generic passthrough of the response's top-level keys rather than a " +
			"typed schema this provider has no evidence for. Nested values are re-encoded " +
			"as their own JSON text. Inspect `raw`'s keys against your own tenant before " +
			"depending on any particular one.",

		ReadContext: dataSourceTenantCapabilitiesRead,

		Schema: map[string]*schema.Schema{
			"raw": {
				Description: "Flattened top-level key/value pairs from the portal version " +
					"response.",
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceTenantCapabilitiesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pv, err := c.GetPortalVersion(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	raw := make(map[string]interface{}, len(pv.Raw))
	for k, v := range pv.Raw {
		raw[k] = v
	}

	if err := d.Set("raw", raw); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("tenant-capabilities")
	return nil
}
