package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStaticSearchLists() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys static QID search lists, for referencing a list " +
			"managed outside Terraform from a VM option profile's " +
			"`custom_search_list_ids`/`exclude_search_list_ids`, by title instead of a " +
			"hardcoded ID.",

		ReadContext: dataSourceStaticSearchListsRead,

		Schema: map[string]*schema.Schema{
			"title_contains": {
				Description: "Filter results to lists whose title contains this substring " +
					"(case-insensitive). Applied client-side, after the API call — the list " +
					"API takes no server-side title filter.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"search_lists": {
				Description: "Matching search lists, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":     {Type: schema.TypeString, Computed: true},
						"title":  {Type: schema.TypeString, Computed: true},
						"global": {Type: schema.TypeBool, Computed: true},
						"qids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"comments": {Type: schema.TypeString, Computed: true},
						"created":  {Type: schema.TypeString, Computed: true},
						"modified": {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceStaticSearchListsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := vmdrClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	lists, err := c.ListStaticSearchLists(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("title_contains").(string)); needle != "" {
		filtered := lists[:0]
		for _, l := range lists {
			if strings.Contains(strings.ToLower(l.Title), needle) {
				filtered = append(filtered, l)
			}
		}
		lists = filtered
	}

	sort.Slice(lists, func(i, j int) bool { return numericLess(lists[i].ID, lists[j].ID) })

	out := make([]interface{}, 0, len(lists))
	ids := make([]string, 0, len(lists))
	for _, l := range lists {
		ids = append(ids, l.ID)
		out = append(out, map[string]interface{}{
			"id":       l.ID,
			"title":    l.Title,
			"global":   l.Global,
			"qids":     l.QIDs,
			"comments": l.Comments,
			"created":  l.Created,
			"modified": l.Modified,
		})
	}

	if err := d.Set("search_lists", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
