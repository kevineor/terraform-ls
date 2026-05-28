// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"context"
	"encoding/json"

	lsp "github.com/hashicorp/terraform-ls/internal/protocol"
)

// quickFixBuilder decodes a diagnostic's raw Data and turns it into actions.
type quickFixBuilder func(in Input, diag lsp.Diagnostic, raw json.RawMessage) ([]lsp.CodeAction, error)

// quickFixProvider dispatches diagnostic-driven quickfixes by the "kind"
// discriminant carried in each diagnostic's Data. It is the single place that
// understands the Data wire format, so new quickfixes register a builder here
// rather than extending a switch elsewhere.
type quickFixProvider struct {
	builders map[string]quickFixBuilder
}

func newQuickFixProvider() *quickFixProvider {
	return &quickFixProvider{builders: make(map[string]quickFixBuilder)}
}

func (p *quickFixProvider) Kinds() []lsp.CodeActionKind {
	return []lsp.CodeActionKind{lsp.QuickFix}
}

func (p *quickFixProvider) CodeActions(ctx context.Context, in Input) ([]lsp.CodeAction, error) {
	var actions []lsp.CodeAction
	for _, diag := range in.Diagnostics {
		raw, ok := rawData(diag.Data)
		if !ok {
			continue
		}
		kind, ok := extraKind(raw)
		if !ok {
			continue
		}
		build, ok := p.builders[kind]
		if !ok {
			continue
		}
		got, err := build(in, diag, raw)
		if err != nil {
			// A malformed payload for one diagnostic shouldn't sink the
			// rest of the request.
			continue
		}
		actions = append(actions, got...)
	}
	return actions, nil
}

// RegisterQuickFix wires a typed builder for diagnostics whose Data carries
// the given kind. The Data is unmarshaled into T before build is called;
// build returns false to decline (emitting no action).
//
// This is a free function rather than a method because Go methods cannot
// introduce their own type parameters.
func RegisterQuickFix[T any](p *quickFixProvider, kind string, build func(extra T, in Input, diag lsp.Diagnostic) (lsp.CodeAction, bool)) {
	p.builders[kind] = func(in Input, diag lsp.Diagnostic, raw json.RawMessage) ([]lsp.CodeAction, error) {
		var extra T
		if err := json.Unmarshal(raw, &extra); err != nil {
			return nil, err
		}
		action, ok := build(extra, in, diag)
		if !ok {
			return nil, nil
		}
		return []lsp.CodeAction{action}, nil
	}
}

// rawData normalises a diagnostic's Data into raw JSON. Depending on how the
// request was decoded, Data arrives either as json.RawMessage or as a generic
// map[string]interface{}; both are handled.
func rawData(data interface{}) (json.RawMessage, bool) {
	switch v := data.(type) {
	case nil:
		return nil, false
	case json.RawMessage:
		return v, len(v) > 0
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		return raw, true
	}
}

// extraKind reads the "kind" discriminant from a raw Data payload.
func extraKind(raw json.RawMessage) (string, bool) {
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil || probe.Kind == "" {
		return "", false
	}
	return probe.Kind, true
}
