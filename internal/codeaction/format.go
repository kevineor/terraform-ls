// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"context"

	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// formatProvider offers the "Format Document" source action backed by
// `terraform fmt`.
type formatProvider struct {
	format func(ctx context.Context, in Input) ([]lsp.TextEdit, error)
}

func (p formatProvider) Kinds() []lsp.CodeActionKind {
	return []lsp.CodeActionKind{ilsp.SourceFormatAllTerraform}
}

func (p formatProvider) CodeActions(ctx context.Context, in Input) ([]lsp.CodeAction, error) {
	edits, err := p.format(ctx, in)
	if err != nil {
		return nil, err
	}

	return []lsp.CodeAction{
		{
			Title: "Format Document",
			Kind:  ilsp.SourceFormatAllTerraform,
			Edit: lsp.WorkspaceEdit{
				Changes: map[lsp.DocumentURI][]lsp.TextEdit{
					in.URI: edits,
				},
			},
		},
	}, nil
}
