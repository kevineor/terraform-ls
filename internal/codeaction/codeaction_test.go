// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform-ls/internal/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// stubProvider is a Provider whose output is fully controlled by the test.
type stubProvider struct {
	kinds   []lsp.CodeActionKind
	actions []lsp.CodeAction
	err     error
	called  bool
}

func (p *stubProvider) Kinds() []lsp.CodeActionKind { return p.kinds }
func (p *stubProvider) CodeActions(ctx context.Context, in Input) ([]lsp.CodeAction, error) {
	p.called = true
	return p.actions, p.err
}

func TestRegistry_Kinds_unionSortedDeduped(t *testing.T) {
	reg := &Registry{providers: []Provider{
		&stubProvider{kinds: []lsp.CodeActionKind{lsp.QuickFix}},
		&stubProvider{kinds: []lsp.CodeActionKind{ilsp.SourceFormatAllTerraform, lsp.QuickFix}},
	}}

	got := reg.Kinds()
	want := []lsp.CodeActionKind{lsp.QuickFix, ilsp.SourceFormatAllTerraform}
	if len(got) != len(want) {
		t.Fatalf("expected %d kinds, got %v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("kind %d: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestRegistry_CodeActions_filtersByRequestedKinds(t *testing.T) {
	quickFix := &stubProvider{
		kinds:   []lsp.CodeActionKind{lsp.QuickFix},
		actions: []lsp.CodeAction{{Title: "qf"}},
	}
	format := &stubProvider{
		kinds:   []lsp.CodeActionKind{ilsp.SourceFormatAllTerraform},
		actions: []lsp.CodeAction{{Title: "fmt"}},
	}
	reg := &Registry{providers: []Provider{quickFix, format}}

	tests := []struct {
		name       string
		only       []lsp.CodeActionKind
		wantTitles []string
	}{
		{"empty only yields nothing", nil, nil},
		{"exact quickfix", []lsp.CodeActionKind{lsp.QuickFix}, []string{"qf"}},
		{"exact format", []lsp.CodeActionKind{ilsp.SourceFormatAllTerraform}, []string{"fmt"}},
		// "source" is a parent kind of "source.formatAll.terraform".
		{"parent kind matches sub-kind", []lsp.CodeActionKind{"source"}, []string{"fmt"}},
		// A sibling kind must not match.
		{"sibling kind does not match", []lsp.CodeActionKind{"source.fixAll"}, nil},
		{"both", []lsp.CodeActionKind{lsp.QuickFix, "source"}, []string{"qf", "fmt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reg.CodeActions(context.Background(), Input{}, tt.only)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(got) != len(tt.wantTitles) {
				t.Fatalf("want %v, got %v", tt.wantTitles, got)
			}
			for i, title := range tt.wantTitles {
				if got[i].Title != title {
					t.Errorf("action %d: want %q, got %q", i, title, got[i].Title)
				}
			}
		})
	}
}

func TestRegistry_CodeActions_providerErrorDoesNotSinkOthers(t *testing.T) {
	failing := &stubProvider{
		kinds: []lsp.CodeActionKind{lsp.QuickFix},
		err:   errors.New("boom"),
	}
	ok := &stubProvider{
		kinds:   []lsp.CodeActionKind{ilsp.SourceFormatAllTerraform},
		actions: []lsp.CodeAction{{Title: "fmt"}},
	}
	reg := &Registry{providers: []Provider{failing, ok}}

	got, err := reg.CodeActions(context.Background(), Input{},
		[]lsp.CodeActionKind{lsp.QuickFix, ilsp.SourceFormatAllTerraform})

	if err == nil {
		t.Error("expected the failing provider's error to be reported")
	}
	if len(got) != 1 || got[0].Title != "fmt" {
		t.Fatalf("expected the healthy provider to still contribute, got %v", got)
	}
}

func TestQuickFixProvider_dispatchByKind(t *testing.T) {
	provider := newQuickFixProvider()
	RegisterQuickFix(provider, diagnostics.MissingRequiredAttributesKind, buildMissingAttrsAction)

	uri := lsp.DocumentURI("file:///main.tf")
	data, err := json.Marshal(diagnostics.MissingRequiredAttributesData{
		Kind:              diagnostics.MissingRequiredAttributesKind,
		MissingAttributes: []string{"ami"},
		InsertAfterRange: hcl.Range{
			Start: hcl.Pos{Line: 2, Column: 1, Byte: 33},
			End:   hcl.Pos{Line: 2, Column: 1, Byte: 33},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	in := Input{
		URI: uri,
		Diagnostics: []lsp.Diagnostic{
			// A diagnostic without Data must be ignored.
			{Message: "no data"},
			// A diagnostic with an unregistered kind must be ignored.
			{Data: json.RawMessage(`{"kind":"somethingElse"}`)},
			// The one we handle, supplied as a decoded map (as the LSP layer
			// would deliver it) to exercise rawData's map path.
			{Data: asMap(t, data)},
		},
	}

	actions, err := provider.CodeActions(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected exactly 1 action, got %d", len(actions))
	}
	if actions[0].Kind != lsp.QuickFix {
		t.Errorf("expected quickfix, got %q", actions[0].Kind)
	}
	edits := actions[0].Edit.Changes[uri]
	if len(edits) != 1 || edits[0].NewText != "  ami = null\n" {
		t.Errorf("unexpected edits: %#v", edits)
	}
}

func asMap(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
