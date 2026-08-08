package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/form3tech-oss/terraform-provider-qualys/vmdr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// resourceAuthRecord builds a record resource for one authentication type.
//
// The types share a create/update/delete shape and differ mainly in which extra
// parameters they accept, so one factory serves them all rather than a
// near-identical file per technology.
func resourceAuthRecord(recordType vmdr.AuthRecordType, label string, extra map[string]*schema.Schema) *schema.Resource {
	s := map[string]*schema.Schema{
		"title": {
			Description: "Name of the authentication record.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"username": {
			Description: "Account the scanner authenticates as.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"password": {
			Description: "Password for the account.\n\n" +
				"This is **write-only**: it is sent to Qualys and never read back, so a " +
				"password changed outside Terraform is not detected as drift. Prefer " +
				"`vault_id` where the subscription has a vault configured — that way the " +
				"secret never passes through Terraform state at all.",
			Type:          schema.TypeString,
			Optional:      true,
			Sensitive:     true,
			ConflictsWith: []string{"vault_id"},
		},
		"vault_id": {
			Description: "ID of a configured Qualys vault to take the credential from, " +
				"instead of storing a password. Preferred over `password`.",
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{"password"},
		},
		"vault_type": {
			Description:  "Type of the referenced vault, for example `CyberArk AIM`.",
			Type:         schema.TypeString,
			Optional:     true,
			RequiredWith: []string{"vault_id"},
		},
		"ips": ipSetSchema("Hosts this record applies to. Accepts addresses, ranges and "+
			"CIDR blocks; membership is compared by address range, so notation does not "+
			"cause a diff. Managed authoritatively: removing an entry stops the record "+
			"applying to it.", false),
		"comments": {
			Type:     schema.TypeString,
			Optional: true,
		},

		"created":  {Type: schema.TypeString, Computed: true},
		"modified": {Type: schema.TypeString, Computed: true},
	}

	for k, v := range extra {
		s[k] = v
	}

	return &schema.Resource{
		Description: fmt.Sprintf("A Qualys %s authentication record.\n\n", label) +
			"Authenticated scans find materially more than unauthenticated ones. " +
			"Credentials supplied here are write-only — see `password`.",

		CreateContext: resourceAuthRecordCreate(recordType),
		ReadContext:   resourceAuthRecordRead(recordType),
		UpdateContext: resourceAuthRecordUpdate(recordType),
		DeleteContext: resourceAuthRecordDelete(recordType),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: s,

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
			// A value fed from another resource (password = random_password.x.result
			// is the canonical case) is unknown at plan time and reads as "". The
			// check must wait for apply in that case, or a perfectly valid first
			// plan fails.
			if !d.NewValueKnown("password") || !d.NewValueKnown("vault_id") {
				return nil
			}
			// A record with neither a password nor a vault cannot authenticate.
			// The API accepts it, and the scan then silently falls back to
			// unauthenticated — worse than an error, because the scan still
			// "succeeds" with far less coverage.
			if d.Get("password").(string) == "" && d.Get("vault_id").(string) == "" {
				return errors.New("a credential is required: set either password or vault_id. " +
					"Qualys accepts a record with neither, but scans using it fall back to " +
					"unauthenticated scanning and report far less")
			}
			return nil
		},
	}
}

func resourceWindowsAuthRecord() *schema.Resource {
	return resourceAuthRecord(vmdr.AuthWindows, "Windows", map[string]*schema.Schema{
		"domain": {
			Description: "Active Directory or NetBIOS domain to authenticate against.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		"domain_type": {
			Description: "How the domain is interpreted.\n\n" +
				"Qualys does not allow this to change after the record is created, so " +
				"changing it forces replacement.",
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			ValidateFunc: validation.StringInSlice([]string{
				"ad_domain", "netbios_user", "netbios_service",
			}, false),
		},
	})
}

func resourceUnixAuthRecord() *schema.Resource {
	return resourceAuthRecord(vmdr.AuthUnix, "Unix/Linux (SSH)", nil)
}

func authRecordInputFrom(d *schema.ResourceData) vmdr.AuthRecordInput {
	return vmdr.AuthRecordInput{
		Title:             d.Get("title").(string),
		Username:          d.Get("username").(string),
		Password:          d.Get("password").(string),
		VaultID:           d.Get("vault_id").(string),
		VaultType:         d.Get("vault_type").(string),
		Comments:          d.Get("comments").(string),
		IPs:               stringSet(d, "ips"),
		WindowsDomain:     getString(d, "domain"),
		WindowsDomainType: getString(d, "domain_type"),
	}
}

// getString reads an attribute that only some record types define.
func getString(d *schema.ResourceData, key string) string {
	v, ok := d.GetOk(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func resourceAuthRecordCreate(t vmdr.AuthRecordType) schema.CreateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		c, err := vmdrClient(meta)
		if err != nil {
			return diag.FromErr(err)
		}
		id, err := c.CreateAuthRecord(ctx, t, authRecordInputFrom(d))
		if err != nil {
			return diag.FromErr(err)
		}
		d.SetId(id)
		return resourceAuthRecordRead(t)(ctx, d, meta)
	}
}

func resourceAuthRecordRead(t vmdr.AuthRecordType) schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		c, err := vmdrClient(meta)
		if err != nil {
			return diag.FromErr(err)
		}

		r, err := c.GetAuthRecord(ctx, t, d.Id())
		if err != nil {
			if errors.Is(err, vmdr.ErrNotFound) {
				d.SetId("")
				return nil
			}
			return diag.FromErr(err)
		}

		// The password is deliberately absent: it is never read back, so drift in
		// the credential itself is not detected. Everything else is refreshed.
		return diag.FromErr(setAll(d, map[string]interface{}{
			"title":    r.Title,
			"username": r.Username,
			"comments": r.Comments,
			"ips":      r.IPs,
			"vault_id": r.VaultID,
			"created":  r.Created,
			"modified": r.Modified,
		}))
	}
}

func resourceAuthRecordUpdate(t vmdr.AuthRecordType) schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		c, err := vmdrClient(meta)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := c.UpdateAuthRecord(ctx, t, d.Id(), authRecordInputFrom(d)); err != nil {
			return diag.FromErr(err)
		}
		return resourceAuthRecordRead(t)(ctx, d, meta)
	}
}

func resourceAuthRecordDelete(t vmdr.AuthRecordType) schema.DeleteContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		c, err := vmdrClient(meta)
		if err != nil {
			return diag.FromErr(err)
		}
		if err := c.DeleteAuthRecord(ctx, t, d.Id()); err != nil {
			if errors.Is(err, vmdr.ErrNotFound) {
				return nil
			}
			return diag.FromErr(err)
		}
		return diag.Diagnostics{{
			Severity: diag.Warning,
			Summary:  "Authentication record deleted; scans of these hosts lose credentialed access",
			Detail: "Scans covering the hosts this record applied to will now run " +
				"unauthenticated. They will still succeed, but report considerably less.",
		}}
	}
}
