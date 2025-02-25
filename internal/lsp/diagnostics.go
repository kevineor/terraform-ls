// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

import (
	"log"

	"github.com/hashicorp/hcl/v2"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

func HCLSeverityToLSP(severity hcl.DiagnosticSeverity) lsp.DiagnosticSeverity {
	var sev lsp.DiagnosticSeverity
	switch severity {
	case hcl.DiagError:
		sev = lsp.SeverityError
	case hcl.DiagWarning:
		sev = lsp.SeverityWarning
	case hcl.DiagInvalid:
		panic("invalid diagnostic")
	}
	return sev
}

func HCLDiagsToLSP(hclDiags hcl.Diagnostics, source string) []lsp.Diagnostic {
	diags := []lsp.Diagnostic{}

	for _, hclDiag := range hclDiags {
		msg := hclDiag.Summary
		if hclDiag.Detail != "" {
			msg += ": " + hclDiag.Detail
		}
		var rnge lsp.Range
		if hclDiag.Subject != nil {
			rnge = HCLRangeToLSP(*hclDiag.Subject)
		}
		codeActions := HCLCodeActionsToLSP(hclDiag.CodeActions, source)

		diags = append(diags, lsp.Diagnostic{
			Range:    rnge,
			Severity: HCLSeverityToLSP(hclDiag.Severity),
			Source:   source,
			Message:  msg,
			Data:     codeActions,
		})

		log.Printf("CONVERTED %v", diags)

	}
	return diags
}

func HCLCodeActionsToLSP(codeActions []hcl.CodeAction, source string) []lsp.CodeAction {
	lspCodeActions := []lsp.CodeAction{}
	for _, codeAction := range codeActions {
		lspCodeActions = append(lspCodeActions, HCLCodeActionToLSP(codeAction, source))
	}
	return lspCodeActions
}

func HCLCodeActionToLSP(codeAction hcl.CodeAction, source string) lsp.CodeAction {
	var edits map[lsp.DocumentURI][]lsp.TextEdit
	for _, edit := range codeAction.Edits {
		if edits == nil {
			edits = map[lsp.DocumentURI][]lsp.TextEdit{}
		}
		edits[lsp.DocumentURI(source)] = append(edits[lsp.DocumentURI(edit.Range.Filename)], lsp.TextEdit{
			Range:   HCLRangeToLSP(edit.Range),
			NewText: edit.NewText,
		})
	}

	return lsp.CodeAction{
		Title: codeAction.Message,
		Kind:  lsp.CodeActionKind(codeAction.Kind),
		Edit: lsp.WorkspaceEdit{
			Changes: edits,
		},
	}
}
