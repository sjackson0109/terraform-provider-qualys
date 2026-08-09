package qps

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// WASBurpImport is the desired input for importing a Burp Suite scan report
// into WAS findings for a web application. Confirmed by a user-supplied
// "Gap Review" document quoting the official Qualys "Import Burp Report"
// reference: POST import/was/burp with a flat data.{webAppId, purgeResults,
// closeUnreportedIssues, fileName, burpXml} body — not the usual
// ServiceRequest.data.<Wrapper> object nesting this package's other WAS
// create calls use.
//
// This is deliberately not wrapped in a Terraform resource: like a scan or
// report launch, an import is a one-shot job against accumulating findings
// state, not an object with an update/delete lifecycle Terraform can own.
// See doc 06 in this provider's discovery notes for the same reasoning
// applied to VM/WAS scan launches.
type WASBurpImport struct {
	WebAppID string
	// PurgeResults, if true, discards previously imported Burp issues
	// before importing this report. If false (the default), previous
	// imports are retained alongside the new one.
	PurgeResults bool
	// CloseUnreportedIssues, if true, closes previously imported issues
	// that are absent from this report.
	CloseUnreportedIssues bool
	FileName              string
	// BurpXML is the raw Burp Suite XML report content.
	BurpXML string
}

type wasBurpImportWire struct {
	WebAppID              string `json:"webAppId,omitempty"`
	PurgeResults          bool   `json:"purgeResults"`
	CloseUnreportedIssues bool   `json:"closeUnreportedIssues"`
	FileName              string `json:"fileName,omitempty"`
	BurpXML               string `json:"burpXml,omitempty"`
}

// ImportWASBurp imports a Burp Suite scan report into WAS findings for a web
// application. Requires the "Import Burp Report" permission.
func (c *Client) ImportWASBurp(ctx context.Context, in WASBurpImport) error {
	if strings.TrimSpace(in.WebAppID) == "" {
		return fmt.Errorf("qualys qps: WAS Burp import requires a web application id")
	}
	if strings.TrimSpace(in.BurpXML) == "" {
		return fmt.Errorf("qualys qps: WAS Burp import requires the Burp XML report content")
	}
	return c.call(ctx, http.MethodPost, "/qps/rest/3.0/import/was/burp",
		&ServiceRequest{Data: wasBurpImportWire{
			WebAppID:              in.WebAppID,
			PurgeResults:          in.PurgeResults,
			CloseUnreportedIssues: in.CloseUnreportedIssues,
			FileName:              in.FileName,
			BurpXML:               in.BurpXML,
		}}, nil, true)
}
