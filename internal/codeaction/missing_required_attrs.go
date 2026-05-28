// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"fmt"

	"github.com/hashicorp/hcl-lang/lang"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// BuildMissingAttrsAction constructs a QuickFix CodeAction that inserts all
// missing required attributes just before the closing `}` of the block body.
func BuildMissingAttrsAction(
	extra lang.MissingRequiredAttributesDiagnosticExtra,
	uri lsp.DocumentURI,
	origDiag lsp.Diagnostic,
) lsp.CodeAction {
	insertPos := ilsp.HCLPosToLSP(extra.InsertAfterRange.Start)
	insertRange := lsp.Range{Start: insertPos, End: insertPos}

	edits := make([]lsp.TextEdit, 0, len(extra.MissingAttributes))
	for _, attr := range extra.MissingAttributes {
		edits = append(edits, lsp.TextEdit{
			Range:   insertRange,
			NewText: fmt.Sprintf("  %s = null\n", attr),
		})
	}

	return lsp.CodeAction{
		Title:       fmt.Sprintf("Add missing required attributes (%d)", len(extra.MissingAttributes)),
		Kind:        lsp.QuickFix,
		Diagnostics: []lsp.Diagnostic{origDiag},
		IsPreferred: true,
		Edit: lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				uri: edits,
			},
		},
	}
}
