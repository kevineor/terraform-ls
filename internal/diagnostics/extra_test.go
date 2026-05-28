// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl/v2"
)

func TestDeserializeExtra_missingRequiredAttributes(t *testing.T) {
	extra := lang.MissingRequiredAttributesDiagnosticExtra{
		Kind:              "missingRequiredAttributes",
		MissingAttributes: []string{"ami", "instance_type"},
		InsertAfterRange: hcl.Range{
			Filename: "main.tf",
			Start:    hcl.Pos{Line: 5, Column: 1, Byte: 80},
			End:      hcl.Pos{Line: 5, Column: 1, Byte: 80},
		},
	}

	data, err := json.Marshal(extra)
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}

	got, err := DeserializeExtra(json.RawMessage(data))
	if err != nil {
		t.Fatalf("DeserializeExtra: %s", err)
	}

	result, ok := got.(lang.MissingRequiredAttributesDiagnosticExtra)
	if !ok {
		t.Fatalf("expected MissingRequiredAttributesDiagnosticExtra, got %T", got)
	}

	if len(result.MissingAttributes) != 2 {
		t.Fatalf("expected 2 missing attributes, got %d", len(result.MissingAttributes))
	}
	if result.MissingAttributes[0] != "ami" || result.MissingAttributes[1] != "instance_type" {
		t.Errorf("unexpected attributes: %v", result.MissingAttributes)
	}
	if result.InsertAfterRange.Filename != "main.tf" {
		t.Errorf("unexpected filename: %q", result.InsertAfterRange.Filename)
	}
	if result.InsertAfterRange.Start.Line != 5 {
		t.Errorf("unexpected line: %d", result.InsertAfterRange.Start.Line)
	}
}

func TestDeserializeExtra_unknownKind(t *testing.T) {
	data := json.RawMessage(`{"kind":"nonexistent"}`)

	_, err := DeserializeExtra(data)
	if err == nil {
		t.Fatal("expected error for unknown kind, got nil")
	}
}

func TestDeserializeExtra_invalidJSON(t *testing.T) {
	_, err := DeserializeExtra(json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
