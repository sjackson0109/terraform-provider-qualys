package provider

import "testing"

func TestWASOptionProfilePerformanceValidation(t *testing.T) {
	v := resourceWASOptionProfile().Schema["performance"].ValidateFunc
	for _, ok := range []string{"LOW", "MEDIUM", "HIGH"} {
		if _, errs := v(ok, "performance"); len(errs) != 0 {
			t.Errorf("valid performance level %q rejected: %v", ok, errs)
		}
	}
	if _, errs := v("EXTREME", "performance"); len(errs) == 0 {
		t.Error("expected an unrecognised performance level to be rejected")
	}
}

func TestWASOptionProfileBruteforceOptionIsUnvalidated(t *testing.T) {
	// bruteforce_option has no confirmed enum in official documentation, so it
	// must not carry a ValidateFunc that would reject a value the API accepts.
	if resourceWASOptionProfile().Schema["bruteforce_option"].ValidateFunc != nil {
		t.Error("bruteforce_option should not be client-validated until its accepted values are confirmed")
	}
}

// The create encoder omits these fields when left at their Go zero value, so
// Qualys can apply its own server-side default. Without Computed, Terraform
// would expect the post-apply state to match the config-implied zero value
// and fail every create that omits one of them with "Provider produced
// inconsistent result after apply".
func TestWASOptionProfileOptionalFieldsAreComputed(t *testing.T) {
	s := resourceWASOptionProfile().Schema
	for _, field := range []string{
		"max_crawl_requests", "performance", "bruteforce_option",
		"timeout_error_threshold", "unexpected_error_threshold",
		"detect_credit_card_numbers", "detect_social_security_numbers",
	} {
		if !s[field].Computed {
			t.Errorf("%s must be Computed: a create that omits it lets Qualys assign a "+
				"default, and without Computed that mismatches the planned zero value", field)
		}
	}
}
