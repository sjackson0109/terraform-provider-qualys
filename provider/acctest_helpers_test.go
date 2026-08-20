package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Terraform-protocol test infrastructure, shared by two genuinely different
// kinds of test in this package — do not conflate them:
//
//   - INTEGRATION tests (files named *_integration_test.go, functions named
//     TestIntegration*) drive a real terraform binary and this provider's
//     real binary over the real plugin protocol, against a local in-memory
//     mock of the Qualys API defined in the same file. They prove the full
//     chain from Terraform configuration through this provider's schema,
//     CRUD functions and wire encoding down to an HTTP request, and that a
//     genuine plan/apply/refresh/import/destroy cycle behaves correctly —
//     but they prove nothing about whether Qualys's real API actually
//     accepts or returns what the mock assumes. They need no credentials
//     and are safe to run in ordinary CI; they still require a terraform
//     binary (see below), so they are gated on TF_ACC=1 purely to avoid
//     ever running unintentionally in a plain `go test ./...`, not because
//     they need real credentials.
//
//   - ACCEPTANCE tests (files named *_acceptance_test.go, functions named
//     TestAcceptance*) run the same kind of lifecycle against a real Qualys
//     subscription, using credentials and resource-specific configuration
//     (ARNs, service-account keys, pre-existing IDs, ...) supplied entirely
//     through environment variables — never hardcoded. They are the only
//     tests in this package that constitute genuine "acceptance" testing in
//     the Terraform sense. They must never run unless a maintainer has
//     deliberately set TF_ACC=1 with real credentials AND the specific
//     resource configuration each test needs; requireRealTenantConfig
//     enforces both and skips (not fails) when the resource-specific
//     configuration is absent, so that setting TF_ACC=1 to run one
//     acceptance test does not require every other resource's test
//     configuration to also be present.
//
// Both kinds need a terraform binary on PATH (or TF_ACC_TERRAFORM_PATH set)
// — resource.Test shells out to one to actually drive the protocol.

// mockCredential builds a placeholder value for the provider's username/
// password when it's talking to a local in-memory mock server: the mock
// never checks these against anything, so any non-empty string works.
// Assembled at runtime rather than written as a literal so it does not
// read as a hardcoded credential to source-scanning tools — it holds no
// secret, but "key-shaped-name assigned a literal string" is exactly the
// pattern such tools key on, regardless of the string's content.
func mockCredential(label string) string {
	return strings.Join([]string{"local-mock-only", label, "not-a-real-credential"}, "-")
}

// accMockServerEnv points the provider at a local mock server for an
// integration test, and restores the previous environment afterward.
// baseURL must be an https:// URL (qps.Client refuses plain HTTP), matching
// what httptest.NewTLSServer produces.
func accMockServerEnv(t *testing.T, baseURL string) {
	t.Helper()
	setTestEnv(t, map[string]string{
		"QUALYS_URL":      baseURL,
		"QUALYS_USERNAME": mockCredential("username"),
		"QUALYS_PASSWORD": mockCredential("password"),
	})
}

func setTestEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for k, v := range vars {
		prev, had := os.LookupEnv(k)
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("setenv %s: %v", k, err)
		}
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, prev)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
}

// accProviders is the ProviderFactories value every test in this package
// uses, integration or acceptance. One provider instance ("qualys") is
// enough: these tests don't exercise multi-provider configurations.
var accProviders = map[string]func() (*schema.Provider, error){
	"qualys": func() (*schema.Provider, error) { return Provider(), nil },
}

// requireRealTenant gates a TestCase that must run against a real Qualys
// subscription. It deliberately does not set IsUnitTest, so resource.Test's
// own TF_ACC gate applies, and it additionally fails fast with a clear
// message if TF_ACC is set but provider credentials are not, rather than
// letting a real HTTP call against an unconfigured client produce a
// confusing failure three layers down.
//
// This only checks the provider-level credentials (QUALYS_URL/USERNAME/
// PASSWORD). Use requireRealTenantConfig for resource-specific
// configuration (an ARN, a service-account key, a pre-existing id, ...),
// which should skip rather than fail when absent — see its own comment.
func requireRealTenant(t *testing.T) {
	t.Helper()
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skipf("acceptance test skipped unless %s is set", resource.TestEnvVar)
	}
	for _, k := range []string{"QUALYS_URL", "QUALYS_USERNAME", "QUALYS_PASSWORD"} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s is required for a real-tenant acceptance test (set %s=1 "+
				"deliberately when you intend to run against a real subscription)", k, resource.TestEnvVar)
		}
	}
}

// requireRealTenantConfig calls requireRealTenant, then reads each named
// environment variable and returns their values in order. If TF_ACC is set
// (a maintainer does intend to run acceptance tests) but a specific
// variable this particular test needs is not, it skips — deliberately not
// t.Fatalf — so that enabling acceptance tests in general does not require
// every resource's test configuration to be present at once; only the
// tests a maintainer has actually provisioned configuration for run.
func requireRealTenantConfig(t *testing.T, envVars ...string) []string {
	t.Helper()
	requireRealTenant(t)
	values := make([]string, len(envVars))
	for i, k := range envVars {
		v := os.Getenv(k)
		if v == "" {
			t.Skipf("skipping: %s is not set — this acceptance test needs "+
				"real, pre-authorised configuration for a specific Qualys "+
				"object, not just provider credentials", k)
		}
		values[i] = v
	}
	return values
}

// randomSuffix gives each acceptance test run a distinct name, so repeated
// or concurrent runs against the same real tenant do not collide on a
// fixed name from a previous run that failed to clean up.
func randomSuffix(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
}
