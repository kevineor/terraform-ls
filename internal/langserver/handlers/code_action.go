// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package handlers

import (
	"context"

	"github.com/hashicorp/terraform-ls/internal/codeaction"
	"github.com/hashicorp/terraform-ls/internal/langserver/errors"
	ilsp "github.com/hashicorp/terraform-ls/internal/lsp"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
	"github.com/hashicorp/terraform-ls/internal/terraform/module"
)

func (svc *service) TextDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) []lsp.CodeAction {
	ca, err := svc.textDocumentCodeAction(ctx, params)
	if err != nil {
		svc.logger.Printf("code action failed: %s", err)
	}

	return ca
}

func (svc *service) textDocumentCodeAction(ctx context.Context, params lsp.CodeActionParams) ([]lsp.CodeAction, error) {
	// Code actions must be explicitly requested. Exit early when the client
	// asks for nothing, so we never compute (or format) without being asked.
	if len(params.Context.Only) == 0 {
		svc.logger.Printf("No code action requested, exiting")
		return nil, nil
	}

	dh := ilsp.HandleFromDocumentURI(params.TextDocument.URI)
	doc, err := svc.stateStore.DocumentStore.GetDocument(dh)
	if err != nil {
		return nil, err
	}

	in := codeaction.Input{
		Handle:      dh,
		URI:         lsp.DocumentURI(dh.FullURI()),
		Text:        doc.Text,
		Range:       params.Range,
		Diagnostics: params.Context.Diagnostics,
	}

	return svc.codeActionRegistry().CodeActions(ctx, in, params.Context.Only)
}

// codeActionRegistry lazily builds the per-session code-action registry,
// wiring the providers to the services they need.
func (svc *service) codeActionRegistry() *codeaction.Registry {
	svc.codeActionsOnce.Do(func() {
		svc.codeActions = codeaction.NewRegistry(codeaction.Deps{
			Format: func(ctx context.Context, in codeaction.Input) ([]lsp.TextEdit, error) {
				tfExec, err := module.TerraformExecutorForModule(ctx, in.Handle.Dir.Path())
				if err != nil {
					return nil, errors.EnrichTfExecError(err)
				}
				return svc.formatDocument(ctx, tfExec, in.Text, in.Handle)
			},
		})
	})
	return svc.codeActions
}
