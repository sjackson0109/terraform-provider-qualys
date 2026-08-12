package vmdr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestCreateVaultSendsTitleAndTypeAndParameters(t *testing.T) {
	var gotAction string
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		gotForm = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT>
			<ITEM_LIST><ITEM><KEY>ID</KEY><VALUE>77</VALUE></ITEM></ITEM_LIST>
			</RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	id, err := c.CreateVault(context.Background(), VaultInput{
		Title:      "Prod CyberArk",
		Type:       "CyberArk AIM",
		Parameters: map[string]string{"endpoint_url": "https://vault.example.com"},
	})
	if err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if id != "77" {
		t.Errorf("id = %q, want 77", id)
	}
	if gotAction != "create" {
		t.Errorf("action = %q, want create", gotAction)
	}
	if gotForm.Get("title") != "Prod CyberArk" || gotForm.Get("type") != "CyberArk AIM" ||
		gotForm.Get("endpoint_url") != "https://vault.example.com" {
		t.Errorf("form = %v", gotForm)
	}
}

func TestCreateVaultRequiresTitleAndType(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request should be sent without title and type")
	}))
	defer srv.Close()

	if _, err := c.CreateVault(context.Background(), VaultInput{Type: "HashiCorp"}); err == nil {
		t.Fatal("expected an error with no title")
	}
	if _, err := c.CreateVault(context.Background(), VaultInput{Title: "x"}); err == nil {
		t.Fatal("expected an error with no type")
	}
}

func TestUpdateVaultSendsID(t *testing.T) {
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	err := c.UpdateVault(context.Background(), "77", VaultInput{Title: "Renamed", Type: "HashiCorp"})
	if err != nil {
		t.Fatalf("UpdateVault: %v", err)
	}
	if gotForm.Get("id") != "77" || gotForm.Get("title") != "Renamed" {
		t.Errorf("form = %v", gotForm)
	}
}

func TestDeleteVaultSendsID(t *testing.T) {
	var gotAction string
	var gotForm url.Values
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotAction = r.Form.Get("action")
		gotForm = r.Form
		fmt.Fprint(w, `<SIMPLE_RETURN><RESPONSE><TEXT>OK</TEXT></RESPONSE></SIMPLE_RETURN>`)
	}))
	defer srv.Close()

	if err := c.DeleteVault(context.Background(), "77"); err != nil {
		t.Fatalf("DeleteVault: %v", err)
	}
	if gotAction != "delete" || gotForm.Get("id") != "77" {
		t.Errorf("action = %q form = %v", gotAction, gotForm)
	}
}

func TestGetVaultNotFound(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<VAULT_LIST_OUTPUT><RESPONSE></RESPONSE></VAULT_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	_, err := c.GetVault(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListVaultsParsesEntries(t *testing.T) {
	c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<VAULT_LIST_OUTPUT><RESPONSE><VAULT_LIST>
			<VAULT><ID>1</ID><TITLE>Prod</TITLE><VAULT_TYPE>HashiCorp</VAULT_TYPE></VAULT>
			</VAULT_LIST></RESPONSE></VAULT_LIST_OUTPUT>`)
	}))
	defer srv.Close()

	vaults, err := c.ListVaults(context.Background())
	if err != nil {
		t.Fatalf("ListVaults: %v", err)
	}
	if len(vaults) != 1 || vaults[0].ID != "1" || vaults[0].Type != "HashiCorp" {
		t.Errorf("vaults = %+v", vaults)
	}
}
