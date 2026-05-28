// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package codeaction

import (
	"github.com/hashicorp/terraform-ls/internal/diagnostics"
)

// NewRegistry builds the default code-action registry wired to deps.
//
// This is the single place that knows the full set of code actions
// terraform-ls offers. New actions are added here: a contextual action as a
// Provider, a diagnostic-driven quickfix via RegisterQuickFix.
func NewRegistry(deps Deps) *Registry {
	quickFix := newQuickFixProvider()
	RegisterQuickFix(quickFix, diagnostics.MissingRequiredAttributesKind, buildMissingAttrsAction)

	return &Registry{
		providers: []Provider{
			formatProvider{format: deps.Format},
			quickFix,
		},
	}
}
