// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl-lang/schemacontext"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-ls/internal/lsp"
	"github.com/zclconf/go-cty/cty"
)

type MissingRequiredAttribute struct{}

func (mra MissingRequiredAttribute) Visit(ctx context.Context, node hclsyntax.Node, nodeSchema schema.Schema) (context.Context, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	if HasUnknownRequiredAttributes(ctx) {
		return ctx, diags
	}

	switch nodeType := node.(type) {
	case *hclsyntax.Block:
		// Providers are excluded from the validation for the time being
		// due to complexity around required attributes with dynamic defaults
		// See https://github.com/hashicorp/vscode-terraform/issues/1616
		nestingLvl, nestingOk := schemacontext.BlockNestingLevel(ctx)
		if nodeType.Type == "provider" && (nestingOk && nestingLvl == 0) {
			ctx = WithUnknownRequiredAttributes(ctx)
		}
	case *hclsyntax.Body:
		if nodeSchema == nil {
			return ctx, diags
		}

		bodySchema := nodeSchema.(*schema.BodySchema)
		if bodySchema.Attributes == nil {
			return ctx, diags
		}

		for name, attr := range bodySchema.Attributes {
			if attr.IsRequired {
				_, ok := nodeType.Attributes[name]
				if !ok {
					diag := hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Required attribute %q not specified", name),
						Detail:   fmt.Sprintf("An attribute named %q is required here", name),
						Subject:  nodeType.SrcRange.Ptr(),
					}
					nodeType.Attributes[name] = &hclsyntax.Attribute{
						Name: name,
						Expr: &hclsyntax.LiteralValueExpr{
							Val: cty.NullVal(cty.Type{}),
						},
					}
					diag.CodeActions = append(diag.CodeActions, mra.buildMissingRequiredAttributeCodeAction(diag, name))
					log.Printf("PRODUCED_DIAAAG : %v", diag)
					diags = append(diags, &diag)

				}
			}
		}
	}

	return ctx, diags
}

func (mra MissingRequiredAttribute) buildMissingRequiredAttributeCodeAction(diag hcl.Diagnostic, missingAttribute string) hcl.CodeAction {
	var edits []hcl.TextEdit
	var edit hcl.TextEdit

	edit.Range = *diag.Subject
	edit.Range.Start = edit.Range.End // To Append after the diagnostic
	edit.Range.Start.Column = 1
	edit.Range.End.Column = 1
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	body.SetAttributeValue(missingAttribute, cty.NullVal(cty.String))
	var tokens hclwrite.Tokens
	tokens = body.BuildTokens(tokens)

	edit.NewText = fmt.Sprintf("%s", tokens.Bytes())
	edits = append(edits, edit)

	return hcl.CodeAction{
		Message: "Add missing attribute",
		Kind:    hcl.CodeActionKind(lsp.QuickfixAddMissingAttributes),
		Edits:   edits,
	}
}

type CodeActionMissingRequiredAttribute struct {
	wrapped          interface{}
	MissingAttribute string
}

func (w CodeActionMissingRequiredAttribute) UnwrapDiagnosticExtra() interface{} {
	return w.wrapped
}

type unknownRequiredAttrsCtxKey struct{}

func HasUnknownRequiredAttributes(ctx context.Context) bool {
	_, ok := ctx.Value(unknownRequiredAttrsCtxKey{}).(bool)
	return ok
}

func WithUnknownRequiredAttributes(ctx context.Context) context.Context {
	return context.WithValue(ctx, unknownRequiredAttrsCtxKey{}, true)
}
