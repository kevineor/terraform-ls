// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validations

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-ls/internal/diagnostics"
	"github.com/zclconf/go-cty/cty"
)

func TestMissingRequiredAttribute(t *testing.T) {
	bodySchema := &schema.BodySchema{
		Attributes: map[string]*schema.AttributeSchema{
			"ami": {
				IsRequired: true,
				Constraint: schema.LiteralType{Type: cty.String},
			},
			"instance_type": {
				IsRequired: true,
				Constraint: schema.LiteralType{Type: cty.String},
			},
			"tags": {
				Constraint: schema.LiteralType{Type: cty.String},
			},
		},
	}

	tests := []struct {
		name string
		cfg  string
		want hcl.Diagnostics
	}{
		{
			name: "single missing required attribute",
			cfg:  "instance_type = \"t2.micro\"\n",
			want: hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Required attribute "ami" not specified`,
					Detail:   `An attribute named "ami" is required here`,
					Subject: &hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 2, Column: 1, Byte: 27},
					},
					Extra: diagnostics.MissingRequiredAttributesData{
						Kind:              diagnostics.MissingRequiredAttributesKind,
						MissingAttributes: []string{"ami"},
						InsertAfterRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 27},
							End:      hcl.Pos{Line: 2, Column: 1, Byte: 27},
						},
					},
				},
			},
		},
		{
			name: "multiple missing required attributes are consolidated",
			cfg:  "tags = \"x\"\n",
			want: hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Required attributes not specified: "ami", "instance_type"`,
					Detail:   `The following attributes are required here: "ami", "instance_type"`,
					Subject: &hcl.Range{
						Filename: "test.tf",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 2, Column: 1, Byte: 11},
					},
					Extra: diagnostics.MissingRequiredAttributesData{
						Kind:              diagnostics.MissingRequiredAttributesKind,
						MissingAttributes: []string{"ami", "instance_type"},
						InsertAfterRange: hcl.Range{
							Filename: "test.tf",
							Start:    hcl.Pos{Line: 2, Column: 1, Byte: 11},
							End:      hcl.Pos{Line: 2, Column: 1, Byte: 11},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, diags := hclsyntax.ParseConfig([]byte(tt.cfg), "test.tf", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatalf("unexpected parse errors: %s", diags)
			}
			body := f.Body.(*hclsyntax.Body)

			_, got := MissingRequiredAttribute{}.Visit(context.Background(), body, bodySchema)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("diagnostics mismatch: %s", diff)
			}
		})
	}
}
