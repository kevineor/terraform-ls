// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl-lang/schemacontext"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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

		if diag := missingRequiredAttributesDiagnostic(nodeType, bodySchema); diag != nil {
			diags = append(diags, diag)
		}
	}

	return ctx, diags
}

// missingRequiredAttributesDiagnostic returns a single diagnostic listing all
// required attributes absent from body, or nil if none are missing. The
// diagnostic carries a MissingRequiredAttributesDiagnosticExtra so a language
// server can offer a quickfix code action that inserts them.
func missingRequiredAttributesDiagnostic(body *hclsyntax.Body, bodySchema *schema.BodySchema) *hcl.Diagnostic {
	var missing []string
	for name, attr := range bodySchema.Attributes {
		if attr.IsRequired {
			if _, ok := body.Attributes[name]; !ok {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)

	quoted := make([]string, len(missing))
	for i, name := range missing {
		quoted[i] = fmt.Sprintf("%q", name)
	}

	var summary, detail string
	if len(missing) == 1 {
		summary = fmt.Sprintf("Required attribute %s not specified", quoted[0])
		detail = fmt.Sprintf("An attribute named %s is required here", quoted[0])
	} else {
		summary = fmt.Sprintf("Required attributes not specified: %s", strings.Join(quoted, ", "))
		detail = fmt.Sprintf("The following attributes are required here: %s", strings.Join(quoted, ", "))
	}

	return &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   detail,
		Subject:  body.SrcRange.Ptr(),
		Extra: lang.MissingRequiredAttributesDiagnosticExtra{
			Kind:              "missingRequiredAttributes",
			MissingAttributes: missing,
			InsertAfterRange:  body.EndRange,
		},
	}
}

type unknownRequiredAttrsCtxKey struct{}

func HasUnknownRequiredAttributes(ctx context.Context) bool {
	_, ok := ctx.Value(unknownRequiredAttrsCtxKey{}).(bool)
	return ok
}

func WithUnknownRequiredAttributes(ctx context.Context) context.Context {
	return context.WithValue(ctx, unknownRequiredAttrsCtxKey{}, true)
}
