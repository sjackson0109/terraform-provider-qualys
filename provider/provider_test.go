package provider

import (
	"testing"
)

// accProviders (acctest_helpers_test.go) is this package's
// ProviderFactories for resource.Test-based acceptance tests; requireRealTenant
// there is the real-tenant precheck. Both superseded this file's older,
// now-removed testAccProviders/testAccPreCheck, which predated any test
// actually driving the provider through resource.Test and had no callers.

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}

}

func TestProviderImpl(t *testing.T) {
	var _ = Provider()
}
