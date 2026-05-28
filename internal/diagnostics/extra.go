// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package diagnostics

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/hcl-lang/lang"
)

type extraFactory func(json.RawMessage) (interface{}, error)

// extraRegistry maps the "kind" discriminant (stored in
// lsp.Diagnostic.Data) to a factory that deserializes the full struct.
// Register new Extra types here as new code-action kinds are added.
var extraRegistry = map[string]extraFactory{
	"missingRequiredAttributes": func(raw json.RawMessage) (interface{}, error) {
		var extra lang.MissingRequiredAttributesDiagnosticExtra
		if err := json.Unmarshal(raw, &extra); err != nil {
			return nil, err
		}
		return extra, nil
	},
}

// DeserializeExtra reads the "kind" field from data and dispatches to the
// registered factory, returning the typed Extra value or an error.
func DeserializeExtra(data json.RawMessage) (interface{}, error) {
	var probe struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	factory, ok := extraRegistry[probe.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown diagnostic extra kind: %q", probe.Kind)
	}
	return factory(data)
}
