package vmdr

import (
	"errors"
	"testing"
)

// "invalid id" is a parameter-validation message, not proof the object is gone.
// Classifying it as not-found would drop a live resource from Terraform state
// over a malformed request, and the next apply would create a duplicate.
func TestValidationMessagesAreNotNotFound(t *testing.T) {
	for _, text := range []string{
		"Bad Request - Parameter ids has invalid id value",
		"unknown id in parameter list",
	} {
		e := &Error{Status: 400, Code: 1905, Text: text}
		if errors.Is(e, ErrNotFound) {
			t.Errorf("%q must not classify as ErrNotFound", text)
		}
	}
}

func TestAbsenceMessagesAreNotFound(t *testing.T) {
	for _, text := range []string{
		"Asset Group Id not found",
		"object does not exist",
		"No such schedule",
	} {
		e := &Error{Status: 400, Text: text}
		if !errors.Is(e, ErrNotFound) {
			t.Errorf("%q should classify as ErrNotFound", text)
		}
	}
}
