package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// qpsNotFoundOnRead implements this provider's policy for a QPS
// OBJECT_NOT_FOUND response (or an equivalent empty-result search) during
// Read.
//
// The portal (qps) API returns the same OBJECT_NOT_FOUND code both when an
// object has genuinely been deleted and when it still exists but has fallen
// outside the caller's current scope — a role, scope-tag or business unit
// change (see qps.ErrNotFound's doc comment). A scoped search silently
// omitting an object it can no longer see is the same ambiguity by another
// shape. The two underlying cases demand opposite handling: a deleted
// object should be dropped from state so the next apply recreates it; an
// out-of-scope object must not be, because dropping it is exactly what
// causes Terraform to plan an unattended recreate of a resource that still
// exists. Since the response alone cannot distinguish the two, the safe
// default is to leave state untouched and surface an error explaining why
// — recreation must never happen silently.
//
// kind is a short, human-readable noun for the object ("asset tag", "WAS
// application", ...), used only in the diagnostic text. Callers must not
// call d.SetId("") alongside this — that is precisely the behaviour this
// function exists to avoid.
//
// d is narrowed to Id() — the one method this policy needs — both to make
// the no-mutation contract obvious at the call site and so tests can supply
// a minimal stand-in instead of a full SDK ResourceData.
func qpsNotFoundOnRead(d interface{ Id() string }, kind string) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("%s not found or no longer visible", kind),
		Detail: fmt.Sprintf(
			"Qualys reported OBJECT_NOT_FOUND for this %s (id %s). That code covers two "+
				"situations Qualys does not distinguish in the response: the object was "+
				"deleted, or it still exists but is now outside the caller's scope (for "+
				"example after a role, scope-tag or business unit change). Recreating a "+
				"still-existing object would be the wrong outcome, so this provider leaves "+
				"it in state rather than guessing which one happened. Terraform state is "+
				"unchanged by this error.\n\n"+
				"Work through this in order — do not skip to the last step:\n\n"+
				"  1. Check scope first. This is the more common cause. Confirm the role, "+
				"scope tags and business unit on the credentials configured for this "+
				"provider still match what this %s was created under. If they have "+
				"changed, restoring scope (or reconfiguring the provider with credentials "+
				"that have it) resolves this without touching state at all.\n\n"+
				"  2. If scope is unchanged, independently verify against Qualys — the UI, "+
				"or an API call using credentials known to have full visibility — whether "+
				"this %s genuinely still exists. Do not rely on this provider's own "+
				"OBJECT_NOT_FOUND response as that confirmation; that is the ambiguous "+
				"signal this whole check exists because of.\n\n"+
				"  3. Only once deletion is positively established (not merely \"this "+
				"provider can't see it\"), remove it from state explicitly:\n\n"+
				"         terraform state rm <resource address>\n\n"+
				"     A later apply will then recreate it. Removing it from state before "+
				"deletion is confirmed risks Terraform creating a duplicate object "+
				"alongside one that still exists — exactly what this check exists to "+
				"prevent, so treat step 3 as a last resort, never a default response to "+
				"this error.",
			kind, d.Id(), kind, kind),
	}}
}
