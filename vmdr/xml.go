package vmdr

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// maxResponseBytes caps how much of a response body we will decode. VMDR list
// endpoints are paginated (see pagination.go), so a single response should never
// approach this; a body that does is treated as hostile or broken rather than
// streamed into memory.
const maxResponseBytes = 256 << 20 // 256 MiB

// decodeXML parses a VMDR XML response into v.
//
// VMDR responses carry a DOCTYPE pointing at a DTD hosted on the platform, e.g.
//
//	<!DOCTYPE SIMPLE_RETURN SYSTEM "https://qualysapi.qualys.com/api/2.0/simple_return.dtd">
//
// Those DTDs are schema *documentation*. They are never fetched at runtime: doing
// so would leak request timing to a third party, add a hard dependency on a remote
// host in the parse path, and hand control of parsing to whatever that URL returns.
//
// Go's encoding/xml already declines to resolve external entities or fetch external
// DTDs, so the parser is safe by construction. We do not rely on that alone: the
// entity map below is fixed to the five predefined XML entities, so a document that
// declares its own internal entities fails rather than expanding them. That closes
// billion-laughs style expansion, which is otherwise reachable without any network
// access at all.
func decodeXML(r io.Reader, v interface{}) error {
	dec := xml.NewDecoder(io.LimitReader(r, maxResponseBytes))

	// Reject anything that is not plain, well-formed XML.
	dec.Strict = true

	// Only the five predefined entities resolve. Any <!ENTITY ...> the document
	// declares is absent from this map, so referencing it is an error instead of
	// an expansion. Assigning a non-nil map also prevents the decoder from
	// populating one from the document's internal subset.
	dec.Entity = map[string]string{
		"lt":   "<",
		"gt":   ">",
		"amp":  "&",
		"apos": "'",
		"quot": `"`,
	}

	// CharsetReader is left nil deliberately: Qualys documents responses as UTF-8,
	// and a nil reader makes a non-UTF-8 declaration a decode error rather than an
	// attempt to load an arbitrary charset converter.
	dec.CharsetReader = nil

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decoding XML response: %w", err)
	}
	return nil
}

// decodeXMLBytes is decodeXML over an in-memory body. Response bodies are read
// whole (rather than streamed) because the error model needs to inspect a failed
// body after a typed decode has already been attempted — see classifyResponse.
func decodeXMLBytes(b []byte, v interface{}) error {
	return decodeXML(bytes.NewReader(b), v)
}
