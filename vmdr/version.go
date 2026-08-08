package vmdr

// API versions are chosen per capability, never globally. Qualys versions each
// resource independently, and runs an explicit deprecation programme: a superseded
// path reaches End of Support six months after its replacement ships, then End of
// Life six months after that, at which point it is decommissioned.
//
// Migrations documented as in flight at the time of writing:
//
//	Scan API                      -> v3.0
//	Schedule Scan API             -> v5.0
//	Option Profile APIs           -> v4.0-v6.0 (varies by sub-type)
//	Asset Host / VM Detection API -> v5.0
//	Report Template Scan API      -> v7.0
//
// The versions below are the ones whose request and response schemas are actually
// documented in the material gathered so far. Several are therefore pinned to a
// generation that has a known expiry. That is a deliberate, recorded trade-off
// rather than an oversight: shipping against an undocumented schema would be
// guesswork. Each constant carries the migration target so the upgrade is a
// one-line change once the newer schema is confirmed.
//
// See docs/discovery/07-api-version-selection.md.
type apiVersion string

const (
	// v2_0 is the generation most VM/PA resources are documented against.
	v2_0 apiVersion = "2.0"
)

// capability identifies an API surface for version selection and EOS reporting.
type capability string

const (
	capAssetIP        capability = "asset/ip"
	capAssetHost      capability = "asset/host"
	capAssetGroup     capability = "asset/group"
	capAssetExcluded  capability = "asset/excluded_ip"
	capAssetDomain    capability = "asset/domain"
	capHostDetection  capability = "asset/host/vm/detection"
	capNetwork        capability = "network"
	capAppliance      capability = "appliance"
	capOptionProfile  capability = "subscription/option_profile"
	capSearchList     capability = "qid/search_list"
	capScan           capability = "scan"
	capScheduleScan   capability = "schedule/scan"
	capReport         capability = "report"
	capScheduleReport capability = "schedule/report"
	capAuth           capability = "auth"
	capVault          capability = "vault"
	capSession        capability = "session"
)

// versionFor returns the API version this client targets for a capability.
func versionFor(c capability) apiVersion {
	if v, ok := pinnedVersions[c]; ok {
		return v
	}
	return v2_0
}

var pinnedVersions = map[capability]apiVersion{
	// Every capability currently sits on 2.0. Entries are listed explicitly, rather
	// than relying on the default, so that bumping one is a visible edit here and
	// the set of things needing migration is greppable.
	capAssetIP:        v2_0,
	capAssetHost:      v2_0, // migrating -> 5.0 (schema not yet documented)
	capAssetGroup:     v2_0,
	capAssetExcluded:  v2_0,
	capAssetDomain:    v2_0,
	capHostDetection:  v2_0, // migrating -> 5.0 (schema not yet documented)
	capNetwork:        v2_0,
	capAppliance:      v2_0,
	capOptionProfile:  v2_0, // migrating -> 4.0-6.0 (schema not yet documented)
	capSearchList:     v2_0,
	capScan:           v2_0, // migrating -> 3.0 (schema not yet documented)
	capScheduleScan:   v2_0, // migrating -> 5.0 (schema not yet documented)
	capReport:         v2_0, // a 3.0 fetch endpoint exists; scope undocumented
	capScheduleReport: v2_0,
	capAuth:           v2_0,
	capVault:          v2_0,
	capSession:        v2_0,
}

// deprecatedCapabilities are those known to have a newer generation available. The
// client warns once per capability when one is used, so an operator learns about
// the migration from their own logs rather than from a decommissioning outage.
var deprecatedCapabilities = map[capability]string{
	capAssetHost:     "5.0",
	capHostDetection: "5.0",
	capOptionProfile: "4.0-6.0",
	capScan:          "3.0",
	capScheduleScan:  "5.0",
}
