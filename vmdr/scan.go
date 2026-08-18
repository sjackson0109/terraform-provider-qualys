package vmdr

import (
	"context"
	"net/url"
	"strings"
)

// Scan is one VM scan job, as returned by the scan list endpoint.
//
// Launch/cancel/pause/resume/delete are deliberately not modelled by this
// package: doc 06 classifies VM scan lifecycle actions as imperative
// operations with no stable update/delete semantics, not Terraform
// resources — a decision this file does not revisit. This type exists only
// to support a read-only qualys_vm_scans data source (e.g. to look up a
// running scan's status, or find recent scans of a target for reporting).
//
// Evidence tier: Corroborated, not Confirmed. Doc 03 §9 confirms the list
// action, its request filters (state, processed, type, target, user_login,
// launched_after/before_datetime, show_ags/op/status/last) and the response
// DTD name (scan_list_output.dtd), but this session could not fetch the DTD
// itself (docs.qualys.com is blocked in this sandbox) to confirm exact
// element names. The fields below use the same names as the confirmed
// request filters they mirror (REF/TARGET/TYPE/TITLE/duration and a nested
// STATUS>STATE, following the STATUS>STATE shape this codebase already
// confirmed for report schedules) — a reasonable inference from Qualys's
// consistent request/response field-naming convention seen throughout this
// codebase, not a fabricated guess, but unverified against a tenant.
type Scan struct {
	Ref            string
	Title          string
	Type           string
	UserLogin      string
	LaunchDatetime string
	Duration       string
	Target         string
	Processed      bool
	Status         string
}

type scanListOutput struct {
	Response struct {
		Scans []struct {
			Ref            string `xml:"REF"`
			Title          string `xml:"TITLE"`
			Type           string `xml:"TYPE"`
			UserLogin      string `xml:"USER_LOGIN"`
			LaunchDatetime string `xml:"LAUNCH_DATETIME"`
			Duration       string `xml:"DURATION"`
			Target         string `xml:"TARGET"`
			Processed      string `xml:"PROCESSED"`
			Status         struct {
				State string `xml:"STATE"`
			} `xml:"STATUS"`
		} `xml:"SCAN_LIST>SCAN"`
		Warning *warning `xml:"WARNING"`
	} `xml:"RESPONSE"`
}

func (o *scanListOutput) warning() *warning { return o.Response.Warning }

func (o *scanListOutput) scans() []*Scan {
	out := make([]*Scan, 0, len(o.Response.Scans))
	for _, s := range o.Response.Scans {
		out = append(out, &Scan{
			Ref:            strings.TrimSpace(s.Ref),
			Title:          s.Title,
			Type:           s.Type,
			UserLogin:      s.UserLogin,
			LaunchDatetime: s.LaunchDatetime,
			Duration:       s.Duration,
			Target:         s.Target,
			Processed:      boolFromFlag(s.Processed),
			Status:         s.Status.State,
		})
	}
	return out
}

// ScanFilter narrows a scan list request. Every field maps to a Confirmed
// request parameter (doc 03 §9).
type ScanFilter struct {
	State          string
	Processed      *bool
	Type           string
	Target         string
	UserLogin      string
	LaunchedAfter  string
	LaunchedBefore string
}

// ListScans returns VM scan jobs matching filter, following truncation.
func (c *Client) ListScans(ctx context.Context, filter ScanFilter) ([]*Scan, error) {
	params := url.Values{}
	setIfNotEmpty(params, "state", filter.State)
	setIfNotEmpty(params, "type", filter.Type)
	setIfNotEmpty(params, "target", filter.Target)
	setIfNotEmpty(params, "user_login", filter.UserLogin)
	setIfNotEmpty(params, "launched_after_datetime", filter.LaunchedAfter)
	setIfNotEmpty(params, "launched_before_datetime", filter.LaunchedBefore)
	if filter.Processed != nil {
		if *filter.Processed {
			params.Set("processed", "1")
		} else {
			params.Set("processed", "0")
		}
	}

	var all []*Scan
	err := c.listAll(ctx, request{
		capability: capScan,
		path:       "scan/",
		action:     "list",
		params:     params,
		method:     "GET",
	},
		func() paginated { return new(scanListOutput) },
		func(p paginated) error {
			all = append(all, p.(*scanListOutput).scans()...)
			return nil
		}, 0)
	if err != nil {
		return nil, err
	}
	return all, nil
}
