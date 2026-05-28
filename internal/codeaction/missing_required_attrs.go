// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"fmt"

	"github.com/hashicorp/terraform-ls/internal/diagnostics"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// buildMissingAttrsAction constructs a QuickFix that inserts all missing
// required attributes (as `<attr> = null`) just before the closing `}` of the
// block body the diagnostic points at.
func buildMissingAttrsAction(extra diagnostics.MissingRequiredAttributesData, in Input, diag lsp.Diagnostic) (lsp.CodeAction, bool) {
	if len(extra.MissingAttributes) == 0 {
		return lsp.CodeAction{}, false
	}

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
		Diagnostics: []lsp.Diagnostic{diag},
		IsPreferred: true,
		Edit: lsp.WorkspaceEdit{
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{
				in.URI: edits,
			},
		},
	}, true
}
