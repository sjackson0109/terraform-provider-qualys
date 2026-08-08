package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// diffFor runs CustomizeDiff over a proposed configuration without contacting an
// API, so plan-time validation can be tested directly.
func diffFor(t *testing.T, r *schema.Resource, raw map[string]interface{}) error {
	t.Helper()
	cfg := terraform.NewResourceConfigRaw(raw)
	_, err := r.Diff(context.Background(), &terraform.InstanceState{}, cfg, nil)
	return err
}

func TestAssetTagRejectsDynamicRuleWithoutExpression(t *testing.T) {
	r := resourceAssetTag()
	err := diffFor(t, r, map[string]interface{}{
		"name":      "linux-hosts",
		"rule_type": "OS_REGEX",
	})
	if err == nil {
		t.Fatal("expected a plan-time error: a dynamic tag with no rule_text matches nothing")
	}
	if !strings.Contains(err.Error(), "rule_text is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAssetTagRejectsExpressionOnStaticTag(t *testing.T) {
	r := resourceAssetTag()
	err := diffFor(t, r, map[string]interface{}{
		"name":      "manual",
		"rule_text": "os:linux",
	})
	if err == nil {
		t.Fatal("expected a plan-time error: rule_text is meaningless without a dynamic rule_type")
	}
	if !strings.Contains(err.Error(), "only meaningful") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAssetTagAcceptsValidConfigurations(t *testing.T) {
	r := resourceAssetTag()

	if err := diffFor(t, r, map[string]interface{}{"name": "prod"}); err != nil {
		t.Errorf("a plain static tag should be valid: %v", err)
	}
	if err := diffFor(t, r, map[string]interface{}{
		"name": "linux", "rule_type": "OS_REGEX", "rule_text": ".*Linux.*",
	}); err != nil {
		t.Errorf("a dynamic tag with an expression should be valid: %v", err)
	}
	if err := diffFor(t, r, map[string]interface{}{
		"name": "manual", "rule_type": "STATIC",
	}); err != nil {
		t.Errorf("an explicit STATIC rule type should be valid: %v", err)
	}
}

func TestAssetTagColourValidation(t *testing.T) {
	v := resourceAssetTag().Schema["color"].ValidateFunc
	if _, errs := v("#FF0000", "color"); len(errs) != 0 {
		t.Errorf("valid colour rejected: %v", errs)
	}
	if _, errs := v("red", "color"); len(errs) == 0 {
		t.Error("expected a non-hex colour to be rejected")
	}
}

// rule_text fed from a computed attribute is unknown at plan time; validating
// it as empty would fail a config that becomes valid at apply.
func TestAssetTagAllowsUnknownRuleTextAtPlanTime(t *testing.T) {
	err := diffFor(t, resourceAssetTag(), map[string]interface{}{
		"name":      "linux",
		"rule_type": "OS_REGEX",
		"rule_text": unknownValue,
	})
	if err != nil {
		t.Fatalf("an unknown rule_text must not fail the plan: %v", err)
	}
}
