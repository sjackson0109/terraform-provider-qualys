package vmdr

import (
	"fmt"
	"math/big"
	"net/netip"
	"sort"
	"strings"
)

// IP targets in the VM/PA API are written as single addresses and hyphenated
// ranges, comma separated:
//
//	10.10.10.1,10.10.10.10-10.10.10.20
//
// Whether the `ips` parameter also accepts CIDR is not stated either way in any
// Qualys documentation obtained so far — every documented example uses hyphens, and
// the one endpoint that explicitly documents CIDR (Restricted IPs) is a different
// API. Rather than depend on an unverified behaviour, the client accepts CIDR from
// practitioners and converts it to a hyphenated range on the wire. That is correct
// under either answer, so no tenant probe is needed before shipping.
//
// Responses come back as separate IP and IP_RANGE elements, so the same conversion
// runs in reverse when comparing desired against actual state; otherwise a config
// written as 10.0.0.0/24 would diff forever against a response of
// 10.0.0.0-10.0.0.255.

// ipRange is a closed interval of addresses. A single address is a range whose
// bounds are equal.
type ipRange struct {
	lo, hi netip.Addr
}

// parseIPEntry accepts a single address, a hyphenated range, or CIDR.
func parseIPEntry(s string) (ipRange, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ipRange{}, fmt.Errorf("empty IP entry")
	}

	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return ipRange{}, fmt.Errorf("parsing CIDR %q: %w", s, err)
		}
		p = p.Masked()
		lo := p.Addr()
		hi := lastAddr(p)
		return ipRange{lo: lo, hi: hi}, nil
	}

	if i := strings.Index(s, "-"); i >= 0 {
		loS, hiS := strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
		lo, err := netip.ParseAddr(loS)
		if err != nil {
			return ipRange{}, fmt.Errorf("parsing range start %q: %w", loS, err)
		}
		hi, err := netip.ParseAddr(hiS)
		if err != nil {
			return ipRange{}, fmt.Errorf("parsing range end %q: %w", hiS, err)
		}
		if lo.Is4() != hi.Is4() {
			return ipRange{}, fmt.Errorf("range %q mixes IPv4 and IPv6", s)
		}
		if hi.Less(lo) {
			return ipRange{}, fmt.Errorf("range %q is inverted", s)
		}
		return ipRange{lo: lo, hi: hi}, nil
	}

	a, err := netip.ParseAddr(s)
	if err != nil {
		return ipRange{}, fmt.Errorf("parsing IP %q: %w", s, err)
	}
	return ipRange{lo: a, hi: a}, nil
}

// lastAddr returns the highest address in a prefix.
func lastAddr(p netip.Prefix) netip.Addr {
	addr := p.Addr()
	bits := addr.BitLen()
	hostBits := bits - p.Bits()
	if hostBits == 0 {
		return addr
	}

	b := addr.AsSlice()
	n := new(big.Int).SetBytes(b)
	// Set all host bits.
	mask := new(big.Int).Lsh(big.NewInt(1), uint(hostBits))
	mask.Sub(mask, big.NewInt(1))
	n.Or(n, mask)

	out := n.Bytes()
	// Left-pad to the original width; big.Int drops leading zero bytes.
	if len(out) < len(b) {
		pad := make([]byte, len(b)-len(out))
		out = append(pad, out...)
	}
	res, ok := netip.AddrFromSlice(out)
	if !ok {
		return addr
	}
	return res
}

// String renders a range the way the API expects it.
func (r ipRange) String() string {
	if r.lo == r.hi {
		return r.lo.String()
	}
	return r.lo.String() + "-" + r.hi.String()
}

// FormatIPSet converts practitioner-supplied entries into the comma-separated
// wire form, expanding CIDR to hyphenated ranges.
//
// Entries are normalised and de-duplicated but not merged: adjacent ranges are
// left distinct, because merging them would make the value the provider sends
// differ from what the practitioner wrote in a way they cannot see.
func FormatIPSet(entries []string) (string, error) {
	seen := make(map[string]struct{}, len(entries))
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.TrimSpace(e) == "" {
			continue
		}
		r, err := parseIPEntry(e)
		if err != nil {
			return "", err
		}
		s := r.String()
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return strings.Join(out, ","), nil
}

// NormalizeIPSet canonicalises entries for comparison, so that a config written
// as CIDR compares equal to the hyphenated ranges the API returns.
//
// The result is sorted, making it order-insensitive: Qualys does not guarantee
// the order of IP_SET members, and an ordering difference is not a real change.
func NormalizeIPSet(entries []string) ([]string, error) {
	seen := make(map[string]struct{}, len(entries))
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.TrimSpace(e) == "" {
			continue
		}
		r, err := parseIPEntry(e)
		if err != nil {
			return nil, err
		}
		s := r.String()
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, _ := parseIPEntry(out[i])
		rj, _ := parseIPEntry(out[j])
		if ri.lo == rj.lo {
			return ri.hi.Less(rj.hi)
		}
		return ri.lo.Less(rj.lo)
	})
	return out, nil
}

// IPSetsEqual reports whether two IP sets describe the same addresses, ignoring
// notation and ordering.
func IPSetsEqual(a, b []string) bool {
	na, err := NormalizeIPSet(a)
	if err != nil {
		return false
	}
	nb, err := NormalizeIPSet(b)
	if err != nil {
		return false
	}
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

// ipSet is the IP_SET element shared by several list responses.
type ipSet struct {
	IPs    []string `xml:"IP"`
	Ranges []string `xml:"IP_RANGE"`
}

// entries flattens an IP_SET into normalised entries.
func (s *ipSet) entries() []string {
	if s == nil {
		return nil
	}
	all := make([]string, 0, len(s.IPs)+len(s.Ranges))
	all = append(all, s.IPs...)
	all = append(all, s.Ranges...)
	out, err := NormalizeIPSet(all)
	if err != nil {
		// A value the API produced that we cannot parse is passed through rather
		// than dropped: losing it silently would understate the group's contents.
		return all
	}
	return out
}
