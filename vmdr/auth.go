package vmdr

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Authentication records let scans log in to targets, which finds materially
// more than an unauthenticated scan.
//
// Credentials are write-only here. Qualys may return a password in a list
// response when the calling account holds the "View Password in Authentication
// Record" permission, but relying on that would mean the provider's behaviour
// changed with the account's permissions, and it would put the secret in state
// and in plan output. Passwords are therefore sent and never read back, and
// drift in the credential itself is not detected.
//
// Where the subscription has a vault configured, referencing a vault record is
// better still: the secret never passes through Terraform at all.

// AuthRecordType identifies an authentication record endpoint.
//
// The slugs are not always what the technology name suggests: Cisco IOS and
// Checkpoint Firewall records are created through the `unix` endpoint rather
// than having their own.
type AuthRecordType string

const (
	AuthWindows AuthRecordType = "windows"
	AuthUnix    AuthRecordType = "unix"
	AuthOracle  AuthRecordType = "oracle"
	AuthMSSQL   AuthRecordType = "ms_sql"
	AuthMySQL   AuthRecordType = "mysql"
	AuthPostgre AuthRecordType = "postgresql"
	AuthSNMP    AuthRecordType = "snmp"
	AuthVMware  AuthRecordType = "vmware"
	AuthHTTP    AuthRecordType = "http"
	AuthDocker  AuthRecordType = "docker"
)

// AuthRecord is an authentication record.
//
// It deliberately carries no password field: the client never reads one back.
type AuthRecord struct {
	ID       string
	Title    string
	Type     AuthRecordType
	Username string
	Comments string
	IPs      []string
	Created  string
	Modified string

	// VaultID and VaultType are set when the record takes its secret from a
	// vault rather than storing one in Qualys.
	VaultID   string
	VaultType string
}

// AuthRecordInput is the desired state of an authentication record.
type AuthRecordInput struct {
	Title    string
	Username string
	Comments string
	IPs      []string

	// Password is write-only. Leave it empty when using a vault.
	Password string

	// VaultID and VaultType reference a configured vault instead of a password.
	VaultID   string
	VaultType string

	// WindowsDomain and WindowsDomainType apply to Windows records. The domain
	// type cannot be changed after creation.
	WindowsDomain     string
	WindowsDomainType string
}

// authRecordGeneric parses whichever per-type list element a response carries.
//
// Each record type has its own wrapper element (AUTH_WINDOWS_LIST, AUTH_UNIX_LIST
// and so on), so rather than a struct per type this decodes the common fields
// from any of them.
type authRecordGeneric struct {
	Response struct {
		Lists []struct {
			Records []struct {
				ID       string `xml:"ID"`
				Title    string `xml:"TITLE"`
				Username string `xml:"USERNAME"`
				Comments string `xml:"COMMENTS"`
				Created  string `xml:"CREATED>DATETIME"`
				Modified string `xml:"LAST_MODIFIED>DATETIME"`
				IPSet    *ipSet `xml:"IP_SET"`
				Vault    struct {
					ID   string `xml:"VAULT_ID"`
					Type string `xml:"VAULT_TYPE"`
				} `xml:"VAULT"`
			} `xml:",any"`
		} `xml:",any"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *authRecordGeneric) warning() *warning { return o.Response.Warning }

func (o *authRecordGeneric) records(t AuthRecordType) []*AuthRecord {
	var out []*AuthRecord
	for _, list := range o.Response.Lists {
		for i := range list.Records {
			r := &list.Records[i]
			if strings.TrimSpace(r.ID) == "" {
				continue
			}
			out = append(out, &AuthRecord{
				ID:        strings.TrimSpace(r.ID),
				Title:     r.Title,
				Type:      t,
				Username:  r.Username,
				Comments:  r.Comments,
				IPs:       r.IPSet.entries(),
				Created:   r.Created,
				Modified:  r.Modified,
				VaultID:   strings.TrimSpace(r.Vault.ID),
				VaultType: strings.TrimSpace(r.Vault.Type),
			})
		}
	}
	return out
}

func authPath(t AuthRecordType) string {
	return "auth/" + string(t) + "/"
}

// authParams encodes the fields shared by create and update.
func authParams(in AuthRecordInput, t AuthRecordType) (url.Values, error) {
	p := url.Values{}
	setIfNotEmpty(p, "title", in.Title)
	setIfNotEmpty(p, "username", in.Username)
	setIfNotEmpty(p, "comments", in.Comments)

	if in.Password != "" && in.VaultID != "" {
		return nil, fmt.Errorf("qualys vmdr: an authentication record takes either a password " +
			"or a vault reference, not both")
	}
	if in.Password != "" {
		p.Set("password", in.Password)
	}
	if in.VaultID != "" {
		p.Set("vault_id", in.VaultID)
		setIfNotEmpty(p, "vault_type", in.VaultType)
	}

	if t == AuthWindows {
		setIfNotEmpty(p, "domain", in.WindowsDomain)
		setIfNotEmpty(p, "domain_type", in.WindowsDomainType)
	}

	return p, nil
}

// CreateAuthRecord creates a record and returns its ID.
func (c *Client) CreateAuthRecord(ctx context.Context, t AuthRecordType, in AuthRecordInput) (string, error) {
	if strings.TrimSpace(in.Title) == "" {
		return "", fmt.Errorf("qualys vmdr: authentication record title is required")
	}

	params, err := authParams(in, t)
	if err != nil {
		return "", err
	}
	if len(in.IPs) > 0 {
		ips, err := FormatIPSet(in.IPs)
		if err != nil {
			return "", err
		}
		params.Set("ips", ips)
	}

	var out simpleReturn
	if err := c.do(ctx, request{
		capability: capAuth,
		path:       authPath(t),
		action:     "create",
		params:     params,
	}, &out); err != nil {
		return "", err
	}

	id, ok := out.itemValue("ID")
	if !ok || strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("qualys vmdr: authentication record was created but the API "+
			"returned no ID (response: %q); import it to bring it under management",
			out.Response.Text)
	}
	return strings.TrimSpace(id), nil
}

// UpdateAuthRecord applies desired state.
//
// The IP list is sent via set_ips so Terraform stays authoritative over which
// hosts the record applies to.
func (c *Client) UpdateAuthRecord(ctx context.Context, t AuthRecordType, id string, in AuthRecordInput) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("qualys vmdr: authentication record id is required for update")
	}

	params, err := authParams(in, t)
	if err != nil {
		return err
	}
	params.Set("ids", id)

	ips, err := FormatIPSet(in.IPs)
	if err != nil {
		return err
	}
	params.Set("set_ips", ips)

	return c.do(ctx, request{
		capability: capAuth,
		path:       authPath(t),
		action:     "update",
		params:     params,
	}, nil)
}

// DeleteAuthRecord removes a record. Scans lose credentialed access to the hosts
// it covered, which is a silent reduction in scan depth rather than an error.
func (c *Client) DeleteAuthRecord(ctx context.Context, t AuthRecordType, id string) error {
	params := url.Values{}
	params.Set("ids", id)
	return c.do(ctx, request{
		capability:  capAuth,
		path:        authPath(t),
		action:      "delete",
		params:      params,
		destructive: true,
	}, nil)
}

// GetAuthRecord returns one record, or ErrNotFound.
//
// The returned record never carries a password: credentials are write-only.
func (c *Client) GetAuthRecord(ctx context.Context, t AuthRecordType, id string) (*AuthRecord, error) {
	records, err := c.ListAuthRecords(ctx, t, []string{id})
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("qualys vmdr: %s authentication record %s: %w", t, id, ErrNotFound)
}

// ListAuthRecords returns records of one type, optionally filtered by ID.
func (c *Client) ListAuthRecords(ctx context.Context, t AuthRecordType, ids []string) ([]*AuthRecord, error) {
	params := url.Values{}
	setListIfNotEmpty(params, "ids", ids)

	var all []*AuthRecord
	err := c.listAll(ctx, request{
		capability: capAuth,
		path:       authPath(t),
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(authRecordGeneric) },
		func(p paginated) error {
			all = append(all, p.(*authRecordGeneric).records(t)...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}

// Vault is a configured credential vault.
type Vault struct {
	ID    string
	Title string
	Type  string
}

type vaultListOutput struct {
	Response struct {
		Vaults []struct {
			ID    string `xml:"ID"`
			Title string `xml:"TITLE"`
			Type  string `xml:"VAULT_TYPE"`
		} `xml:"VAULT_LIST>VAULT"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *vaultListOutput) warning() *warning { return o.Response.Warning }

// ListVaults returns the vaults configured for the subscription.
func (c *Client) ListVaults(ctx context.Context) ([]*Vault, error) {
	var all []*Vault
	err := c.listAll(ctx, request{
		capability: capVault,
		path:       "vault/",
		action:     "list",
		method:     "GET",
	},
		func() paginated { return new(vaultListOutput) },
		func(p paginated) error {
			o := p.(*vaultListOutput)
			for i := range o.Response.Vaults {
				v := &o.Response.Vaults[i]
				all = append(all, &Vault{
					ID:    strings.TrimSpace(v.ID),
					Title: v.Title,
					Type:  v.Type,
				})
			}
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
