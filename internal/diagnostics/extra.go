// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"github.com/hashicorp/hcl/v2"
)

// Extra "kind" discriminants. The kind is stored alongside the diagnostic
// payload (as lsp.Diagnostic.Data on the wire) so the code-action layer can
// dispatch to the right handler without knowing the concrete Go type up front.
const (
	MissingRequiredAttributesKind = "missingRequiredAttributes"
)

// MissingRequiredAttributesData is the machine-readable payload attached to a
// "required attribute(s) not specified" diagnostic. It is the terraform-ls
// owned contract: validators populate it on hcl.Diagnostic.Extra, and the
// code-action layer reads it back to build a quickfix.
//
// The JSON shape is the contract; producers in other modules (e.g. the
// hcl-lang validator used for .tftest.hcl files) only need to emit a
// JSON-compatible value carrying the same "kind".
type MissingRequiredAttributesData struct {
	Kind              string    `json:"kind"`
	MissingAttributes []string  `json:"missingAttributes"`
	InsertAfterRange  hcl.Range `json:"insertAfterRange"`
}

// DiagnosticExtraUnwrap makes the value a well-behaved hcl diagnostic Extra.
func (e MissingRequiredAttributesData) DiagnosticExtraUnwrap() interface{} {
	return nil
}
