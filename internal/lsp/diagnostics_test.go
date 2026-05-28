// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/diagnostics"
)

func TestHCLDiagsToLSP_ExtraSerializedToData(t *testing.T) {
	extra := diagnostics.MissingRequiredAttributesData{
		Kind:              diagnostics.MissingRequiredAttributesKind,
		MissingAttributes: []string{"ami"},
		InsertAfterRange: hcl.Range{
			Filename: "main.tf",
			Start:    hcl.Pos{Line: 3, Column: 1, Byte: 30},
			End:      hcl.Pos{Line: 3, Column: 1, Byte: 30},
		},
	}
	hclDiags := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Missing required attribute \"ami\"",
			Detail:   "An attribute named \"ami\" is required here",
			Subject:  &hcl.Range{Filename: "main.tf"},
			Extra:    extra,
		},
	}

	lspDiags := HCLDiagsToLSP(hclDiags, "terraform-ls")

	if len(lspDiags) != 1 {
		t.Fatalf("expected 1 LSP diagnostic, got %d", len(lspDiags))
	}
	if lspDiags[0].Data == nil {
		t.Fatal("expected Data to be non-nil")
	}

	raw, ok := lspDiags[0].Data.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T", lspDiags[0].Data)
	}

	var decoded diagnostics.MissingRequiredAttributesData
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	if decoded.Kind != diagnostics.MissingRequiredAttributesKind {
		t.Errorf("unexpected kind: %q", decoded.Kind)
	}
	if len(decoded.MissingAttributes) != 1 || decoded.MissingAttributes[0] != "ami" {
		t.Errorf("unexpected MissingAttributes: %v", decoded.MissingAttributes)
	}
}

func TestHCLDiagsToLSP_NilExtraProducesNilData(t *testing.T) {
	hclDiags := hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "some error",
			Subject:  &hcl.Range{Filename: "main.tf"},
		},
	}
	lspDiags := HCLDiagsToLSP(hclDiags, "terraform-ls")
	if len(lspDiags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(lspDiags))
	}
	if lspDiags[0].Data != nil {
		t.Errorf("expected nil Data for diagnostic with no Extra, got %v", lspDiags[0].Data)
	}
}

func TestHCLDiagsToLSP_NeverReturnsNil(t *testing.T) {
	diags := HCLDiagsToLSP(nil, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{}, "test")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}

	diags = HCLDiagsToLSP(hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
		},
	}, "source")
	if diags == nil {
		t.Fatal("diags should not be nil")
	}
}
