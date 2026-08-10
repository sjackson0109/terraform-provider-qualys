package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestWASAuthRecordRequiresARecordType(t *testing.T) {
	err := diffFor(t, resourceWASAuthRecord(), map[string]interface{}{
		"name": "storefront-login",
	})
	if err == nil {
		t.Fatal("expected a plan-time error when neither form_record nor server_record is set")
	}
}

func TestWASAuthRecordAcceptsFormRecord(t *testing.T) {
	err := diffFor(t, resourceWASAuthRecord(), map[string]interface{}{
		"name": "storefront-login",
		"form_record": []interface{}{
			map[string]interface{}{
				"field": []interface{}{
					map[string]interface{}{"name": "username", "value": "scanner"},
					map[string]interface{}{"name": "password", "value": "s3cret", "secured": true},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("form-based record rejected: %v", err)
	}
}

func TestWASAuthRecordAcceptsStandardFormRecord(t *testing.T) {
	err := diffFor(t, resourceWASAuthRecord(), map[string]interface{}{
		"name": "storefront-login",
		"form_record": []interface{}{
			map[string]interface{}{
				"sub_type":  "STANDARD",
				"login_url": "https://shop.example.com/login",
				"username":  "scanner",
				"password":  "s3cret",
			},
		},
	})
	if err != nil {
		t.Errorf("STANDARD form record rejected: %v", err)
	}
}

func TestWASAuthRecordRejectsEmptyFormRecord(t *testing.T) {
	err := diffFor(t, resourceWASAuthRecord(), map[string]interface{}{
		"name":        "storefront-login",
		"form_record": []interface{}{map[string]interface{}{"sub_type": "STANDARD"}},
	})
	if err == nil {
		t.Fatal("expected a plan-time error: a form_record with neither username/password nor field has no credential")
	}
}

func TestWASAuthRecordAcceptsServerRecord(t *testing.T) {
	err := diffFor(t, resourceWASAuthRecord(), map[string]interface{}{
		"name": "storefront-basic-auth",
		"server_record": []interface{}{
			map[string]interface{}{"username": "scanner", "password": "s3cret"},
		},
	})
	if err != nil {
		t.Errorf("server-based record rejected: %v", err)
	}
}

func TestWASAuthRecordFormAndServerConflict(t *testing.T) {
	s := resourceWASAuthRecord().Schema
	if len(s["form_record"].ConflictsWith) == 0 {
		t.Error("form_record must conflict with server_record")
	}
	if len(s["server_record"].ConflictsWith) == 0 {
		t.Error("server_record must conflict with form_record")
	}
}

func TestWASAuthRecordPasswordAndFieldValueAreSensitive(t *testing.T) {
	s := resourceWASAuthRecord().Schema
	serverPassword := s["server_record"].Elem.(*schema.Resource).Schema["password"]
	if serverPassword == nil || !serverPassword.Sensitive {
		t.Error("server_record.password must be Sensitive")
	}
	fieldValue := s["form_record"].Elem.(*schema.Resource).Schema["field"].Elem.(*schema.Resource).Schema["value"]
	if fieldValue == nil || !fieldValue.Sensitive {
		t.Error("form_record.field.value must be Sensitive")
	}
}
