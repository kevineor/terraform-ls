// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package codeaction provides the abstraction terraform-ls uses to assemble
// textDocument/codeAction responses.
//
// Adding a new code action means writing a Provider (or, for the common
// diagnostic-driven case, a single RegisterQuickFix call) and registering it
// in NewRegistry. The langserver handler stays unchanged: it builds an Input
// and asks the Registry for the actions the client requested.
package codeaction

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-ls/internal/document"
	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// Input is the resolved context of a single textDocument/codeAction request,
// shared by every Provider.
type Input struct {
	// Handle identifies the document the request targets.
	Handle document.Handle
	// URI is the full document URI, ready to key a WorkspaceEdit.
	URI lsp.DocumentURI
	// Text is the current document body.
	Text []byte
	// Range is the selection/cursor range the client asked about.
	Range lsp.Range
	// Diagnostics are the diagnostics the client attached to the request
	// (typically the ones overlapping Range). Diagnostic-driven quickfixes
	// read their payload from these.
	Diagnostics []lsp.Diagnostic
}

// Deps is the narrow set of server services a Provider may need. Keeping it
// small (rather than handing providers the whole langserver service) keeps
// providers unit-testable and their dependencies explicit.
type Deps struct {
	// Format returns the text edits that format the input document, or an
	// error if formatting is unavailable (e.g. no Terraform executor).
	Format func(ctx context.Context, in Input) ([]lsp.TextEdit, error)
}

// Provider produces code actions of one or more kinds.
type Provider interface {
	// Kinds reports the code action kinds this provider can emit. It is used
	// both to advertise server capabilities and to skip the provider when the
	// client did not request any of its kinds.
	Kinds() []lsp.CodeActionKind

	// CodeActions returns the actions applicable to in. It is only called when
	// the client requested at least one of Kinds().
	CodeActions(ctx context.Context, in Input) ([]lsp.CodeAction, error)
}

// Registry holds the ordered set of providers serving code actions.
type Registry struct {
	providers []Provider
}

// Kinds returns the union of all provider kinds, sorted and de-duplicated,
// for advertising in the server capabilities.
func (r *Registry) Kinds() []lsp.CodeActionKind {
	seen := make(map[lsp.CodeActionKind]struct{})
	kinds := make([]lsp.CodeActionKind, 0)
	for _, p := range r.providers {
		for _, k := range p.Kinds() {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			kinds = append(kinds, k)
		}
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

// CodeActions runs every provider whose kinds the client requested and
// returns their concatenated actions. A provider error is collected but does
// not prevent the other providers from contributing.
func (r *Registry) CodeActions(ctx context.Context, in Input, only []lsp.CodeActionKind) ([]lsp.CodeAction, error) {
	// Code actions must be explicitly requested; an empty Only means the
	// client wants nothing in particular, so we return nothing.
	if len(only) == 0 {
		return nil, nil
	}

	var actions []lsp.CodeAction
	var errs error
	for _, p := range r.providers {
		if !kindsRequested(only, p.Kinds()) {
			continue
		}
		got, err := p.CodeActions(ctx, in)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		actions = append(actions, got...)
	}
	return actions, errs
}

// kindsRequested reports whether any provided kind was asked for in only.
// Code action kinds are hierarchical ("source.formatAll.terraform" is a
// sub-kind of "source"), so a provided kind matches a requested kind when it
// is equal to it or nested beneath it.
func kindsRequested(only []lsp.CodeActionKind, provided []lsp.CodeActionKind) bool {
	for _, p := range provided {
		for _, req := range only {
			if p == req || strings.HasPrefix(string(p), string(req)+".") {
				return true
			}
		}
	}
	return false
}
