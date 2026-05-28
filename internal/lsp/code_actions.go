// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lsp

const (
	// SourceFormatAllTerraform is a Terraform specific format code action.
	//
	// `source.*` actions apply to the entire file. They must be explicitly
	// requested and do not show in the normal lightbulb menu. They can be run
	// on save via editor.codeActionsOnSave and appear in the source context
	// menu. For action definitions, refer to:
	// https://code.visualstudio.com/api/references/vscode-api#CodeActionKind
	//
	// We deliberately register the Terraform-specific `source.formatAll.terraform`
	// rather than the generic `source.formatAll`, so a user can enable
	// `source.formatAll` while disabling it for Terraform files (or vice versa).
	SourceFormatAllTerraform = "source.formatAll.terraform"
)
