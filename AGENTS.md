# AGENTS

## Guidelines

- **Keep changes minimal and focused.** Only modify code directly related to the task at hand. Do not refactor unrelated code, rename existing variables or functions for style, or bundle unrelated fixes into the same commit or PR.
- **Do not add, remove, or update dependencies** unless the task explicitly requires it.

## Pre-Commit Checklist

Before every commit, run **all** of the following checks and ensure they pass:

### 1. Formatting

Before committing, always run `gofmt` and `goimports` on all modified files:

```sh
gofmt -w <modified files>
goimports -w <modified files>
```

If command `goimports` is not found, install it by running `go install golang.org/x/tools/cmd/goimports@latest`.

### 2. Build All Commands

Verify that all tools compile successfully:

```sh
make all
```

This builds all executables under `cmd/`.

### 3. Tests

During development, run the quick test suite for faster feedback:

```sh
make quick-test
```

Before committing, run the full test suite to ensure all tests pass:

```sh
make test
```

### 4. Tidy

```sh
make tidy
```

Tidy makes sure go.mod matches the source code in the module. It adds any missing modules necessary to build the current module's packages and dependencies, and it removes unused modules that don't provide any relevant packages. It also adds any missing entries to go.sum and removes any unnecessary ones. If this command fails, report the error and exit.

### 5. Generated Code

```sh
make generate
```

Ensures that all generated files (e.g., `gen_*.go`) are up to date. If this fails, first install the required code generators by running `make devtools`, then run the appropriate `go generate` commands and include the updated files in your commit.

## What to include in commits

Do not commit binaries, whether they are produced by the main build or byproducts of investigations.

## Commit Message Format

Commit messages should follow the Conventional Commits format:

```text
<type>(<scope>): description
```

Examples:

- `fix(core/vm): fix stack overflow in PUSH instruction`
- `feat(eth,rpc): make trace configs optional`
- `chore(cmd/XDC): add new flag for sync mode`

Use a concise, lowercase description. Use a comma-separated scope list when multiple areas are affected, or omit the scope if it doesn't add clarity.

## Pull Request Title Format

PR titles should follow the same Conventional Commits format as commit messages:

```text
<type>(<scope>): description
```

Examples:

- `fix(core/vm): fix stack overflow in PUSH instruction`
- `feat(core,eth): add arena allocator support`
- `refactor(cmd/XDC,internal/ethapi): refactor transaction args`
- `feat(trie/archiver): streaming subtree archival to fix OOM`

Use the top-level package paths as the scope, comma-separated if multiple areas are affected. Only mention the directories with functional changes; interface changes that trickle all over the codebase should not generate an exhaustive list. The description should be a short, lowercase summary of the change.
