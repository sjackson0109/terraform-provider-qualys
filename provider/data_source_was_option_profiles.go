package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWASOptionProfiles() *schema.Resource {
	return &schema.Resource{
		Description: "Look up Qualys WAS scan option profiles, for referencing profiles " +
			"managed outside Terraform from scan schedules. This is the WAS counterpart to " +
			"`data.qualys_vm_option_profiles`.",

		ReadContext: dataSourceWASOptionProfilesRead,

		Schema: map[string]*schema.Schema{
			"name_contains": {
				Description: "Filter results to profiles whose name contains this substring " +
					"(case-insensitive). Applied client-side, after the API call — the search API " +
					"takes no server-side name filter.",
				Type:     schema.TypeString,
				Optional: true,
			},

			"option_profiles": {
				Description: "Matching option profiles, ordered by ID for stable output.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                          {Type: schema.TypeString, Computed: true},
						"name":                        {Type: schema.TypeString, Computed: true},
						"comments":                    {Type: schema.TypeString, Computed: true},
						"max_crawl_requests":          {Type: schema.TypeInt, Computed: true},
						"performance":                 {Type: schema.TypeString, Computed: true},
						"bruteforce_option":           {Type: schema.TypeString, Computed: true},
						"timeout_error_threshold":     {Type: schema.TypeInt, Computed: true},
						"unexpected_error_threshold":  {Type: schema.TypeInt, Computed: true},
						"detect_credit_card_numbers":  {Type: schema.TypeBool, Computed: true},
						"detect_social_security_nums": {Type: schema.TypeBool, Computed: true},
					},
				},
			},
		},
	}
}

func dataSourceWASOptionProfilesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c, err := qpsClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	profiles, err := c.SearchWASOptionProfiles(ctx, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if needle := strings.ToLower(d.Get("name_contains").(string)); needle != "" {
		filtered := profiles[:0]
		for _, p := range profiles {
			if strings.Contains(strings.ToLower(p.Name), needle) {
				filtered = append(filtered, p)
			}
		}
		profiles = filtered
	}

	sort.Slice(profiles, func(i, j int) bool { return numericLess(profiles[i].ID, profiles[j].ID) })

	out := make([]interface{}, 0, len(profiles))
	ids := make([]string, 0, len(profiles))
	for _, p := range profiles {
		ids = append(ids, p.ID)
		out = append(out, map[string]interface{}{
			"id":                          p.ID,
			"name":                        p.Name,
			"comments":                    p.Comments,
			"max_crawl_requests":          p.MaxCrawlRequests,
			"performance":                 p.Performance,
			"bruteforce_option":           p.BruteforceOption,
			"timeout_error_threshold":     p.TimeoutErrorThreshold,
			"unexpected_error_threshold":  p.UnexpectedErrorThreshold,
			"detect_credit_card_numbers":  p.DetectCreditCardNumbers,
			"detect_social_security_nums": p.DetectSocialSecurityNums,
		})
	}

	if err := d.Set("option_profiles", out); err != nil {
		return diag.FromErr(err)
	}
	d.SetId(digestID(ids))
	return nil
}
