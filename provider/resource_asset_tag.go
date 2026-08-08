package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/qps"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAssetTag() *schema.Resource {
	return &schema.Resource{
		Description: "A Qualys asset tag.\n\n" +
			"Tags live in a single subscription-wide tree managed by AssetView/CSAM, and " +
			"the same tag is referenced by VM scan and report targeting and by WAS web " +
			"applications. One tag resource therefore serves all three modules.\n\n" +
			"A tag with no `rule_type` is static: membership is assigned explicitly. A tag " +
			"with a rule type is dynamic: Qualys evaluates `rule_text` to decide membership, " +
			"and the members are not managed by Terraform.",

		CreateContext: resourceAssetTagCreate,
		ReadContext:   resourceAssetTagRead,
		UpdateContext: resourceAssetTagUpdate,
		DeleteContext: resourceAssetTagDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Tag name. Must be unique among its siblings.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"parent_tag_id": {
				Description: "ID of the parent tag. Omit for a tag at the root of the tree.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"rule_type": {
				Description: "Evaluation rule for a dynamic tag. Omit for a static tag.",
				Type:        schema.TypeString,
				Optional:    true,
				ValidateFunc: validation.StringInSlice([]string{
					qps.RuleTypeStatic,
					qps.RuleTypeGroovy,
					qps.RuleTypeOSRegex,
					qps.RuleTypeNetworkRange,
					qps.RuleTypeNameContains,
					qps.RuleTypeInstalledSoftware,
					qps.RuleTypeOpenPorts,
					qps.RuleTypeVulnExist,
					qps.RuleTypeAssetSearch,
					qps.RuleTypeGlobalAssetView,
				}, false),
			},
			"rule_text": {
				Description: "Expression evaluated for a dynamic tag. Required when " +
					"`rule_type` selects a dynamic rule; its syntax depends on the rule type.",
				Type:     schema.TypeString,
				Optional: true,
			},
			"color": {
				Description:  "Display colour as a hex triplet, for example `#FF0000`.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringMatch(hexColour, "must be a hex colour such as #FF0000"),
			},

			"created":  {Type: schema.TypeString, Computed: true},
			"modified": {Type: schema.TypeString, Computed: true},
			"children": {
				Description: "Child tags, as reported by Qualys. Manage a child by declaring " +
					"it as its own resource with `parent_tag_id` set, not by editing this list.",
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":   {Type: schema.TypeString, Computed: true},
						"name": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},

		CustomizeDiff: resourceAssetTagCustomizeDiff,
	}
}

// resourceAssetTagCustomizeDiff catches rule/rule_text mismatches at plan time
// rather than letting the API reject them at apply time.
func resourceAssetTagCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	ruleType := d.Get("rule_type").(string)
	ruleText := d.Get("rule_text").(string)

	dynamic := ruleType != "" && ruleType != qps.RuleTypeStatic
	if dynamic && ruleText == "" {
		return errors.New("rule_text is required when rule_type selects a dynamic rule; " +
			"a dynamic tag with no expression would match nothing")
	}
	if !dynamic && ruleText != "" {
		return errors.New("rule_text is only meaningful with a dynamic rule_type; " +
			"a static tag's membership is assigned explicitly, not evaluated")
	}
	return nil
}

func tagInputFrom(d *schema.ResourceData) qps.TagInput {
	return qps.TagInput{
		Name:     d.Get("name").(string),
		Parent:   d.Get("parent_tag_id").(string),
		RuleType: d.Get("rule_type").(string),
		RuleText: d.Get("rule_text").(string),
		Color:    d.Get("color").(string),
	}
}

func resourceAssetTagCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	tag, err := c.CreateTag(ctx, tagInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(tag.ID)
	return resourceAssetTagRead(ctx, d, meta)
}

func resourceAssetTagRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	tag, err := c.GetTag(ctx, d.Id())
	if err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			// OBJECT_NOT_FOUND also covers "outside your scope", so warn rather
			// than silently dropping: if this is a permissions change, recreating
			// the tag would be the wrong response.
			d.SetId("")
			return diag.Diagnostics{{
				Severity: diag.Warning,
				Summary:  "Asset tag no longer visible; removing it from state",
				Detail: "Qualys reported OBJECT_NOT_FOUND, which it also returns for objects " +
					"outside the caller's scope. If the tag still exists and the credentials' " +
					"scope changed, restore the scope before applying, or Terraform will " +
					"create a duplicate tag.",
			}}
		}
		return diag.FromErr(err)
	}

	children := make([]interface{}, 0, len(tag.Children))
	for _, ch := range tag.Children {
		children = append(children, map[string]interface{}{"id": ch.ID, "name": ch.Name})
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"name":          tag.Name,
		"parent_tag_id": tag.Parent,
		"rule_type":     tag.RuleType,
		"rule_text":     tag.RuleText,
		"color":         tag.Color,
		"created":       tag.Created,
		"modified":      tag.Modified,
		"children":      children,
	}))
}

func resourceAssetTagUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateTag(ctx, d.Id(), tagInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceAssetTagRead(ctx, d, meta)
}

func resourceAssetTagDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteTag(ctx, d.Id()); err != nil {
		if errors.Is(err, qps.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
