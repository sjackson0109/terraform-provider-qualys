package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/form3tech-oss/terraform-provider-qualys/cloudview/gcp"
	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// clients bundles the per-API-family clients. The families differ in transport,
// envelope, pagination and error model, so they are separate clients rather than
// one client with mode switches.
type clients struct {
	vmdr *vmdr.Client
	gcp  *gcp.ConnectorService
}

func vmdrClient(meta interface{}) (*vmdr.Client, error) {
	c, ok := meta.(*clients)
	if !ok || c.vmdr == nil {
		return nil, fmt.Errorf("qualys: the VM/PA API client is not configured; " +
			"set `base_url`, `username` and `password` on the provider")
	}
	return c.vmdr, nil
}

func gcpService(meta interface{}) *gcp.ConnectorService {
	c, ok := meta.(*clients)
	if !ok {
		return nil
	}
	return c.gcp
}

// stringSet reads a TypeSet of strings from resource data.
func stringSet(d *schema.ResourceData, key string) []string {
	raw, ok := d.GetOk(key)
	if !ok {
		return nil
	}
	set, ok := raw.(*schema.Set)
	if !ok {
		return nil
	}
	out := make([]string, 0, set.Len())
	for _, v := range set.List() {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// stringList reads a TypeList of strings from resource data.
func stringList(d *schema.ResourceData, key string) []string {
	raw, ok := d.GetOk(key)
	if !ok {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

// combineErrors collects non-nil errors into diagnostics.
//
// The previous implementation tested `e == nil` before appending, so it appended
// only nil errors — and diag.FromErr(nil) yields nothing. The effect was that
// every d.Set failure it was given was discarded and the call always reported
// success. Callers use it to report attribute-set failures, which is precisely
// the case that was being lost.
func combineErrors(errs ...error) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, e := range errs {
		if e != nil {
			diags = append(diags, diag.FromErr(e)...)
		}
	}
	return diags
}

// setAll writes several attributes, returning the first failure.
//
// Set errors are reported rather than ignored: a silently dropped attribute
// produces a resource whose state does not match Qualys, which surfaces later as
// an unexplained permanent diff.
func setAll(d *schema.ResourceData, values map[string]interface{}) error {
	for k, v := range values {
		if err := d.Set(k, v); err != nil {
			return fmt.Errorf("setting %s: %w", k, err)
		}
	}
	return nil
}
