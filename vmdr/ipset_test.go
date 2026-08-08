package vmdr

import "testing"

func TestFormatIPSetExpandsCIDRToHyphenatedRange(t *testing.T) {
	cases := []struct{ in, want string }{
		{"10.10.10.1", "10.10.10.1"},
		{"10.10.10.10-10.10.10.20", "10.10.10.10-10.10.10.20"},
		{"10.0.0.0/24", "10.0.0.0-10.0.0.255"},
		{"192.168.1.0/30", "192.168.1.0-192.168.1.3"},
		{"10.1.2.3/32", "10.1.2.3"},
		// Unmasked CIDR is masked first, so a sloppy config still sends a sane range.
		{"10.0.0.7/24", "10.0.0.0-10.0.0.255"},
	}
	for _, c := range cases {
		got, err := FormatIPSet([]string{c.in})
		if err != nil {
			t.Errorf("FormatIPSet(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("FormatIPSet(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatIPSetJoinsAndDeduplicates(t *testing.T) {
	got, err := FormatIPSet([]string{"10.0.0.1", "10.0.0.1", " ", "10.0.0.0/31"})
	if err != nil {
		t.Fatalf("FormatIPSet: %v", err)
	}
	if want := "10.0.0.1,10.0.0.0-10.0.0.1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatIPSetRejectsBadInput(t *testing.T) {
	for _, bad := range []string{"not-an-ip", "10.0.0.5-10.0.0.1", "10.0.0.1-::1", "10.0.0.0/99"} {
		if _, err := FormatIPSet([]string{bad}); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}

// A config written as CIDR must compare equal to the hyphenated range the API
// returns, or the resource diffs forever.
func TestIPSetsEqualAcrossNotations(t *testing.T) {
	if !IPSetsEqual([]string{"10.0.0.0/24"}, []string{"10.0.0.0-10.0.0.255"}) {
		t.Error("CIDR and its equivalent range compared unequal")
	}
	if !IPSetsEqual([]string{"10.0.0.2", "10.0.0.1"}, []string{"10.0.0.1", "10.0.0.2"}) {
		t.Error("ordering was treated as a difference")
	}
	if IPSetsEqual([]string{"10.0.0.0/24"}, []string{"10.0.0.0-10.0.0.254"}) {
		t.Error("genuinely different sets compared equal")
	}
}

func TestIPv6Supported(t *testing.T) {
	got, err := FormatIPSet([]string{"2001:db8::/126"})
	if err != nil {
		t.Fatalf("FormatIPSet: %v", err)
	}
	if want := "2001:db8::-2001:db8::3"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIPSetEntriesFlattensResponse(t *testing.T) {
	s := &ipSet{IPs: []string{"10.0.0.5"}, Ranges: []string{"10.0.0.1-10.0.0.3"}}
	got := s.entries()
	want := []string{"10.0.0.1-10.0.0.3", "10.0.0.5"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
