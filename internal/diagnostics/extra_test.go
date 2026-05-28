// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
)

// TestMissingRequiredAttributesData_jsonRoundTrip locks in the wire contract:
// the payload must survive a marshal/unmarshal cycle unchanged, since it
// crosses the LSP boundary as lsp.Diagnostic.Data and is decoded back by the
// code-action layer (and by other producers such as the hcl-lang validator).
func TestMissingRequiredAttributesData_jsonRoundTrip(t *testing.T) {
	want := MissingRequiredAttributesData{
		Kind:              MissingRequiredAttributesKind,
		MissingAttributes: []string{"ami", "instance_type"},
		InsertAfterRange: hcl.Range{
			Filename: "main.tf",
			Start:    hcl.Pos{Line: 5, Column: 1, Byte: 80},
			End:      hcl.Pos{Line: 5, Column: 1, Byte: 80},
		},
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}

	var got MissingRequiredAttributesData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %s", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("round-trip mismatch: %s", diff)
	}
}

func TestMissingRequiredAttributesData_kind(t *testing.T) {
	data, err := json.Marshal(MissingRequiredAttributesData{
		Kind:              MissingRequiredAttributesKind,
		MissingAttributes: []string{"ami"},
	})
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}

	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		t.Fatalf("unmarshal: %s", err)
	}
	if probe.Kind != MissingRequiredAttributesKind {
		t.Errorf("expected kind %q, got %q", MissingRequiredAttributesKind, probe.Kind)
	}
}
