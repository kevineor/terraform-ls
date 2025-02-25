// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

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

func TestHCLCodeActionToLSP(t *testing.T) {
	type args struct {
		codeActions hcl.CodeAction
	}
	tests := []struct {
		name string
		args args
		want lsp.CodeAction
	}{
		{
			name: "empty",
			args: args{
				codeActions: hcl.CodeAction{
					Message: "test",
					Kind:    "quickfix",
				},
			},
			want: lsp.CodeAction{
				Title: "test",
				Kind:  "quickfix",
			},
		},
		{
			name: "with edits",
			args: args{
				codeActions: hcl.CodeAction{
					Message: "test with edits",
					Kind:    "quickfix",
					Edits: []hcl.TextEdit{
						{
							Range: hcl.Range{
								Filename: "testfile",
								Start: hcl.Pos{
									Line:   1,
									Column: 2,
									Byte:   0,
								},
								End: hcl.Pos{
									Line:   1,
									Column: 2,
									Byte:   0,
								},
							},
							NewText: "Append this text",
						},
					},
				},
			},
			want: lsp.CodeAction{
				Title: "test with edits",
				Kind:  "quickfix",
				Edit: lsp.WorkspaceEdit{
					Changes: map[lsp.DocumentURI][]lsp.TextEdit{
						"testfile": {
							{
								Range: lsp.Range{
									Start: lsp.Position{Line: 0, Character: 1},
									End:   lsp.Position{Line: 0, Character: 1},
								},
								NewText: "Append this text",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, HCLCodeActionToLSP(tt.args.codeActions)); diff != "" {
				t.Errorf("HCLCodeActionToLSP() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
