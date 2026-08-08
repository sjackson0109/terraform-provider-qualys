package provider

import (
	"context"
	"errors"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceStaticSearchList() *schema.Resource {
	return &schema.Resource{
		Description: "A static search list: an explicit set of vulnerability QIDs.\n\n" +
			"Option profiles reference search lists by ID to define custom vulnerability " +
			"detection, so a profile with `vulnerability_detection = \"custom\"` depends on " +
			"one of these.\n\n" +
			"Membership is managed authoritatively: removing a QID from the configuration " +
			"removes it from the list.",

		CreateContext: resourceStaticSearchListCreate,
		ReadContext:   resourceStaticSearchListRead,
		UpdateContext: resourceStaticSearchListUpdate,
		DeleteContext: resourceStaticSearchListDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"title": {
				Description: "Name of the search list.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"qids": {
				Description: "Qualys vulnerability IDs in this list.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"global": {
				Description: "Whether the list is available to all users in the subscription.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"comments": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"created":  {Type: schema.TypeString, Computed: true},
			"modified": {Type: schema.TypeString, Computed: true},
		},
	}
}

func staticSearchListInputFrom(d *schema.ResourceData) vmdr.StaticSearchListInput {
	return vmdr.StaticSearchListInput{
		Title:    d.Get("title").(string),
		Global:   d.Get("global").(bool),
		Comments: d.Get("comments").(string),
		QIDs:     stringSet(d, "qids"),
	}
}

func resourceStaticSearchListCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	id, err := c.CreateStaticSearchList(ctx, staticSearchListInputFrom(d))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)
	return resourceStaticSearchListRead(ctx, d, meta)
}

func resourceStaticSearchListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	l, err := c.GetStaticSearchList(ctx, d.Id())
	if err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	return diag.FromErr(setAll(d, map[string]interface{}{
		"title":    l.Title,
		"qids":     l.QIDs,
		"global":   l.Global,
		"comments": l.Comments,
		"created":  l.Created,
		"modified": l.Modified,
	}))
}

func resourceStaticSearchListUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.UpdateStaticSearchList(ctx, d.Id(), staticSearchListInputFrom(d)); err != nil {
		return diag.FromErr(err)
	}
	return resourceStaticSearchListRead(ctx, d, meta)
}

func resourceStaticSearchListDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := c.DeleteStaticSearchList(ctx, d.Id()); err != nil {
		if errors.Is(err, vmdr.ErrNotFound) {
			return nil
		}
		return diag.FromErr(err)
	}
	return nil
}
