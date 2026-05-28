// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/diagnostics"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func TestBuildMissingAttrsAction(t *testing.T) {
	extra := diagnostics.MissingRequiredAttributesData{
		Kind:              diagnostics.MissingRequiredAttributesKind,
		MissingAttributes: []string{"ami", "instance_type"},
		InsertAfterRange: hcl.Range{
			Filename: "main.tf",
			Start:    hcl.Pos{Line: 4, Column: 1, Byte: 60},
			End:      hcl.Pos{Line: 4, Column: 1, Byte: 60},
		},
	}

	uri := lsp.DocumentURI("file:///main.tf")
	in := Input{URI: uri}
	origDiag := lsp.Diagnostic{
		Message: `Required attributes not specified: "ami", "instance_type"`,
	}

	action, ok := buildMissingAttrsAction(extra, in, origDiag)
	if !ok {
		t.Fatal("expected an action to be built")
	}

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
		extra := diagnostics.MissingRequiredAttributesData{
			Kind:              diagnostics.MissingRequiredAttributesKind,
			MissingAttributes: tc.attrs,
		}
		action, ok := buildMissingAttrsAction(extra, Input{URI: "file:///x.tf"}, lsp.Diagnostic{})
		if !ok {
			t.Fatalf("attrs %v: expected an action", tc.attrs)
		}
		if action.Title != tc.title {
			t.Errorf("attrs %v: want title %q, got %q", tc.attrs, tc.title, action.Title)
		}
	}
}

func TestBuildMissingAttrsAction_noAttributes(t *testing.T) {
	_, ok := buildMissingAttrsAction(diagnostics.MissingRequiredAttributesData{}, Input{}, lsp.Diagnostic{})
	if ok {
		t.Error("expected no action when there are no missing attributes")
	}
}
