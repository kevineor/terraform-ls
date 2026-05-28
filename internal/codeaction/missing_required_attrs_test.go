// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"testing"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TestBuildMissingAttrsAction(t *testing.T) {
	extra := lang.MissingRequiredAttributesDiagnosticExtra{
		Kind:              "missingRequiredAttributes",
		MissingAttributes: []string{"ami", "instance_type"},
		InsertAfterRange: hcl.Range{
			Filename: "main.tf",
			Start:    hcl.Pos{Line: 4, Column: 1, Byte: 60},
			End:      hcl.Pos{Line: 4, Column: 1, Byte: 60},
		},
	}

	uri := lsp.DocumentURI("file:///main.tf")
	origDiag := lsp.Diagnostic{
		Message: "Missing required attributes: \"ami\", \"instance_type\"",
	}

	action := BuildMissingAttrsAction(extra, uri, origDiag)

	if action.Kind != lsp.QuickFix {
		t.Errorf("expected QuickFix kind, got %q", action.Kind)
	}
	if !action.IsPreferred {
		t.Error("expected IsPreferred to be true")
	}
	if len(action.Diagnostics) != 1 {
		t.Errorf("expected 1 diagnostic, got %d", len(action.Diagnostics))
	}

	edits, ok := action.Edit.Changes[uri]
	if !ok {
		t.Fatal("expected changes for URI")
	}
	if len(edits) != 2 {
		t.Fatalf("expected 2 text edits, got %d", len(edits))
	}

	// Both edits insert at the same position (InsertAfterRange.Start, LSP-adjusted).
	// hcl.Pos{Line:4, Col:1} → LSP {Line:3, Character:0}
	wantLine := uint32(3)
	wantChar := uint32(0)
	for i, e := range edits {
		if e.Range.Start.Line != wantLine || e.Range.Start.Character != wantChar {
			t.Errorf("edit %d: unexpected insert position %v", i, e.Range.Start)
		}
	}
	if edits[0].NewText != "  ami = null\n" {
		t.Errorf("edit 0: unexpected text %q", edits[0].NewText)
	}
	if edits[1].NewText != "  instance_type = null\n" {
		t.Errorf("edit 1: unexpected text %q", edits[1].NewText)
	}
}

func TestBuildMissingAttrsAction_title(t *testing.T) {
	cases := []struct {
		attrs []string
		title string
	}{
		{[]string{"a"}, "Add missing required attributes (1)"},
		{[]string{"a", "b", "c"}, "Add missing required attributes (3)"},
	}

	for _, tc := range cases {
		extra := lang.MissingRequiredAttributesDiagnosticExtra{
			Kind:             "missingRequiredAttributes",
			MissingAttributes: tc.attrs,
		}
		action := BuildMissingAttrsAction(extra, "file:///x.tf", lsp.Diagnostic{})
		if action.Title != tc.title {
			t.Errorf("attrs %v: want title %q, got %q", tc.attrs, tc.title, action.Title)
		}
	}
}
