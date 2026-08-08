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
