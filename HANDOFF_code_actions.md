# Handoff: Code Action quickfix for missing required attributes

Status as of commit `7fb2b26` on branch `claude/code-actions-missing-attrs-qJmXe`.

This branch is **not yet mergeable or even buildable on a fresh checkout**. Two
blocking issues remain, described below. Read this whole document before
touching anything.

-----

## What was done (committed and pushed)

On `kevineor/terraform-ls`, branch `claude/code-actions-missing-attrs-qJmXe`:

- `internal/diagnostics/extra.go` (+ test): `extraRegistry` and
  `DeserializeExtra`, the kind-based registry that turns `lsp.Diagnostic.Data`
  back into a typed Go struct.
- `internal/codeaction/missing_required_attrs.go` (+ test):
  `BuildMissingAttrsAction`, builds a `QuickFix` `lsp.CodeAction` that inserts
  `  <attr> = null` lines before the closing `}`.
- `internal/lsp/diagnostics.go`: `HCLDiagsToLSP` now JSON-marshals
  `hcl.Diagnostic.Extra` into `lsp.Diagnostic.Data`.
- `internal/lsp/code_actions.go`: registered `lsp.QuickFix` in
  `SupportedCodeActions`.
- `internal/langserver/handlers/code_action.go`: added a `case lsp.QuickFix`
  branch that deserializes `Data` and dispatches to `BuildMissingAttrsAction`.
- `internal/langserver/handlers/handlers_test.go`: updated the server
  capabilities assertion to include `quickfix`.
- `go.mod`: added a `replace` directive (see Blocking Issue 1).

The `terraform-ls` side compiles and its unit tests pass. The three failing
tests in the suite (`TestExec_cancel`, `TestCompletion_module*`,
`TestLangServer_DidChangeWatchedFiles_moduleInstalled`) are **pre-existing**
failures on the base branch (expired PGP key in test fixtures), unrelated to
this work.

-----

## Blocking issue 1: the hcl-lang fork is ephemeral, build is broken

The design requires changes to `hashicorp/hcl-lang`. Those changes were made in
a **local, throwaway** directory `/home/user/hcl-lang` and `go.mod` was pointed
at it:

```
replace github.com/hashicorp/hcl-lang => /home/user/hcl-lang
```

`/home/user/hcl-lang` does **not** exist in a fresh container and is **not** in
any repo. Anyone checking out this branch gets a build failure. This must be
fixed.

Tooling constraint: GitHub access in these sessions is restricted to the
`kevineor/terraform-ls` repo only, and the task rules say do not publish to
official repositories. So pushing the fork to a separate `kevineor/hcl-lang`
repo may not be possible with the available tools. Confirm what you can reach.

Recommended approach (portable, no external repo needed): **vendor the fork
inside this repo and use a relative replace directive.**

1. Create `third_party/hcl-lang/` in the terraform-ls repo and copy the full
   hcl-lang source (the exact version `v0.0.0-20250117153936-66cdc97e9d3b`)
   into it, including the two modified files below.
2. Change `go.mod` to:
   ```
   replace github.com/hashicorp/hcl-lang => ./third_party/hcl-lang
   ```
   A relative path resolves against the repo, so it works on any checkout.
3. Commit `third_party/hcl-lang/` so it persists.
4. Run `go build ./...` and `go mod tidy` to confirm.

If you instead get the changes merged into a real hcl-lang fork/version, drop
the replace and bump the `require` to that pseudo-version. That is cleaner but
depends on tooling access you may not have.

### The two hcl-lang changes to reproduce

**New file `lang/diagnostic_extra.go`:**

```go
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lang

import (
	"github.com/hashicorp/hcl/v2"
)

type MissingRequiredAttributesDiagnosticExtra struct {
	Kind              string    `json:"kind"`
	MissingAttributes []string  `json:"missingAttributes"`
	InsertAfterRange  hcl.Range `json:"insertAfterRange"`
}

func (e MissingRequiredAttributesDiagnosticExtra) DiagnosticExtraUnwrap() interface{} {
	return nil
}
```

**Updated `validator/attribute_missing_required.go`:** consolidates to a single
diagnostic per body and populates `Extra`. Full content:

```go
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl-lang/lang"
	"github.com/hashicorp/hcl-lang/schema"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

type MissingRequiredAttribute struct{}

func (v MissingRequiredAttribute) Visit(ctx context.Context, node hclsyntax.Node, nodeSchema schema.Schema) (context.Context, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	body, ok := node.(*hclsyntax.Body)
	if !ok {
		return ctx, diags
	}
	if nodeSchema == nil {
		return ctx, diags
	}

	bodySchema := nodeSchema.(*schema.BodySchema)
	if bodySchema.Attributes == nil {
		return ctx, diags
	}

	var missing []string
	for name, attr := range bodySchema.Attributes {
		if attr.IsRequired {
			if _, ok := body.Attributes[name]; !ok {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) == 0 {
		return ctx, diags
	}
	sort.Strings(missing)

	quoted := make([]string, len(missing))
	for i, name := range missing {
		quoted[i] = fmt.Sprintf("%q", name)
	}

	var summary, detail string
	if len(missing) == 1 {
		summary = fmt.Sprintf("Missing required attribute %s", quoted[0])
		detail = fmt.Sprintf("An attribute named %s is required here", quoted[0])
	} else {
		summary = fmt.Sprintf("Missing required attributes: %s", strings.Join(quoted, ", "))
		detail = fmt.Sprintf("The following attributes are required: %s", strings.Join(quoted, ", "))
	}

	diags = append(diags, &hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  summary,
		Detail:   detail,
		Subject:  body.SrcRange.Ptr(),
		Extra: lang.MissingRequiredAttributesDiagnosticExtra{
			Kind:              "missingRequiredAttributes",
			MissingAttributes: missing,
			InsertAfterRange:  body.EndRange,
		},
	})

	return ctx, diags
}
```

Also update the hcl-lang decoder test
`decoder/validate_test.go` "missing required attribute" case: the expected
`Summary` becomes `Missing required attribute "wakka"` and the expected
diagnostic gains the `Extra` field (the `InsertAfterRange` is the zero-length
range at `body.EndRange`, which for the one-line fixture `bar = "baz"` is
`{Line:1, Column:12, Byte:11}` start and end).

-----

## Blocking issue 2 (most important): module/stack .tf files do NOT use the hcl-lang validator

This is the gap that makes the feature currently do nothing for the primary use
case. The design doc assumed the diagnostic comes from
`hcl-lang/validator.MissingRequiredAttribute`. That is only true for the
**tests** feature.

For regular `.tf` files, terraform-ls registers its own **local fork** of the
validator:

- `internal/features/modules/decoder/validators.go` uses
  `validations.MissingRequiredAttribute{}` (local), not the hcl-lang one.
- `internal/features/stacks/decoder/validators.go` does the same.

The local implementations live in:

- `internal/features/modules/decoder/validations/missing_required_attribute.go`
- `internal/features/stacks/decoder/validations/missing_required_attribute.go`

These local validators:
1. Emit one diagnostic per missing attribute (not consolidated).
2. Do **not** set `Extra`.
3. Have extra logic the hcl-lang one lacks: they skip top-level `provider`
   blocks (dynamic-default attributes, see vscode-terraform#1616) via the
   `HasUnknownRequiredAttributes` / `WithUnknownRequiredAttributes` context
   helpers.

**Consequence:** with only the hcl-lang change, a missing required attribute in
a normal `main.tf` produces a diagnostic with no `Data`, so no code action
appears. The quickfix would only show up for `.tftest.hcl` files.

### What to do

Port the `Extra`-population (and ideally the consolidation) into the **local**
`validations.MissingRequiredAttribute` for both modules and stacks. Keep the
existing provider-exclusion logic intact. Concretely, in each local
`missing_required_attribute.go`, in the `*hclsyntax.Body` case:

- collect missing names into a slice instead of emitting per-attribute,
- after the loop, if non-empty, emit a single `hcl.Diagnostic` with
  `Extra: lang.MissingRequiredAttributesDiagnosticExtra{ Kind: "missingRequiredAttributes", MissingAttributes: missing, InsertAfterRange: nodeType.EndRange }`
  (import `github.com/hashicorp/hcl-lang/lang`).

Decide whether to keep the diagnostic message consolidated or per-attribute. If
you keep it per-attribute for messaging reasons, you still need a single Extra
carrying the full list (otherwise the code action title/edit count is wrong).
Consolidated is simpler and matches the hcl-lang change. Whichever you pick,
make modules, stacks, and the hcl-lang validator consistent.

Then update any tests that assert the old `Required attribute %q not specified`
message. Search: `grep -rn "Required attribute.*not specified" internal/`.

Note the `code_action.go` handler already tolerates `Data` arriving as either
`json.RawMessage` or a decoded map, so the LSP round-trip is handled. No change
needed there.

-----

## Remaining smaller items

- **Integration test** (design step 8): add a langserver handler test that
  opens a `.tf` with a missing required attribute, publishes diagnostics,
  sends `textDocument/codeAction` with the diagnostic (including its `Data`) in
  `params.Context.Diagnostics`, and asserts a `quickfix` action with the
  expected `WorkspaceEdit`. Mirror the style of the existing cases in
  `internal/langserver/handlers/code_action_test.go`. Make sure the test
  exercises the module path (Blocking Issue 2), not just a unit-level Extra.
- **CHANGELOG.md**: add a feature entry.
- **Verify end to end in an editor** if possible: the code action edits insert
  `  <attr> = null`. Confirm indentation/formatting is acceptable, or run
  `terraform fmt` mentally against the result. The insert position uses
  `body.EndRange.Start` (just before `}`); confirm it lands correctly for both
  single-line and multi-line blocks.

## Open questions from the design (still open)

- Use schema default instead of `null` when available (would extend the Extra
  struct with an optional `AttributeDefaults map[string]string`).
- `codeAction/resolve` lazy edit support (hcl-lang#363). Current design is
  compatible; not implemented.
- `hcl.Range` JSON round-trip: verified working out of the box (the unit tests
  in `internal/diagnostics/extra_test.go` and `internal/lsp/diagnostics_test.go`
  cover marshal/unmarshal), so no wrapper type is needed.

## Constraints to respect

- Do all work on branch `claude/code-actions-missing-attrs-qJmXe`.
- Do NOT publish to official HashiCorp repositories. Stay inside
  `kevineor/terraform-ls`. Do not open public PRs or tag anyone/any issue.
