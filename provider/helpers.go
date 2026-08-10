package provider

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/form3tech-oss/terraform-provider-qualys/cloudview/gcp"
	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// clients bundles the per-API-family clients. The families differ in transport,
// envelope, pagination and error model, so they are separate clients rather than
// one client with mode switches.
type clients struct {
	vmdr *vmdr.Client
	qps  *qps.Client
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

// stringSetFromInterface converts the raw interface{} d.GetChange returns
// for a TypeSet attribute (a *schema.Set) into a plain string slice. Unlike
// stringSet, this takes the value directly rather than reading it from
// ResourceData, since GetChange's old/new values aren't otherwise
// accessible through GetOk.
func stringSetFromInterface(raw interface{}) []string {
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

// hexColour matches a six-digit hex colour, the form the tagging API documents.
var hexColour = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func qpsClient(meta interface{}) (*qps.Client, error) {
	c, ok := meta.(*clients)
	if !ok || c.qps == nil {
		return nil, fmt.Errorf("qualys: the portal (qps) API client is not configured; " +
			"set `base_url`, `username` and `password` on the provider")
	}
	return c.qps, nil
}

// ipSetSchema builds a set-of-IPs attribute whose membership is compared by
// canonical address range rather than by the literal string.
//
// The Qualys API returns hyphenated ranges, while practitioners may write CIDR.
// Without a custom hash, a config of "10.0.0.0/24" would never match the
// "10.0.0.0-10.0.0.255" that Read stores, and the resource would diff forever.
// Hashing the normalised form makes the two the same set element.
func ipSetSchema(description string, required bool) *schema.Schema {
	return &schema.Schema{
		Description: description,
		Type:        schema.TypeSet,
		Optional:    !required,
		Required:    required,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Set:         hashNormalizedIP,
	}
}

// hashNormalizedIP hashes an IP entry by its canonical range, so that CIDR and
// the equivalent hyphenated range collide deliberately.
func hashNormalizedIP(v interface{}) int {
	s, _ := v.(string)
	return schema.HashString(vmdr.NormalizeIPEntry(s))
}

// noWireDelimiters rejects values containing the characters the appliance API
// uses as field and entry separators. A VLAN or route field containing ',' or
// '|' would split into extra fields on the wire and misconfigure the
// appliance's networking; the client validates too, but failing at plan time is
// kinder than at apply.
func noWireDelimiters(v interface{}, key string) ([]string, []error) {
	s, ok := v.(string)
	if !ok {
		return nil, []error{fmt.Errorf("%s must be a string", key)}
	}
	if strings.ContainsAny(s, ",|") {
		return nil, []error{fmt.Errorf(
			"%s must not contain ',' or '|'; the Qualys API uses them as separators", key)}
	}
	return nil, nil
}
