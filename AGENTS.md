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

### 2. Build All Commands

Verify that all tools compile successfully:

```sh
make all
```

This builds all executables under `cmd/`, including `keeper` which has special build requirements.

### 3. Tests

While iterating during development, use `-short` for faster feedback:

```sh
go run ./build/ci.go test -short
```

Before committing, run the full test suite **without** `-short` to ensure all tests pass, including the Ethereum execution-spec tests and all state/block test permutations:

```sh
go run ./build/ci.go test
```

### 4. Linting

```sh
go run ./build/ci.go lint
```

This runs additional style checks. Fix any issues before committing.

### 5. Generated Code

```sh
go run ./build/ci.go check_generate
```

Ensures that all generated files (e.g., `gen_*.go`) are up to date. If this fails, first install the required code generators by running `make devtools`, then run the appropriate `go generate` commands and include the updated files in your commit.

### 6. Dependency Hygiene

```sh
go run ./build/ci.go check_baddeps
```

Verifies that no forbidden dependencies have been introduced.

## Commit Message Format

Commit messages must be prefixed with the package(s) they modify, followed by a short lowercase description:

```
<package(s)>: description
```

Examples:
- `core/vm: fix stack overflow in PUSH instruction`
- `eth, rpc: make trace configs optional`
- `cmd/geth: add new flag for sync mode`

Use comma-separated package names when multiple areas are affected. Keep the description concise.

## Pull Request Title Format

PR titles follow the same convention as commit messages:

```
<list of modified paths>: description
```

Examples:
- `core/vm: fix stack overflow in PUSH instruction`
- `core, eth: add arena allocator support`
- `cmd/geth, internal/ethapi: refactor transaction args`
- `trie/archiver: streaming subtree archival to fix OOM`

Use the top-level package paths, comma-separated if multiple areas are affected. Only mention the directories with functional changes, interface changes that trickle all over the codebase should not generate an exhaustive list. The description should be a short, lowercase summary of the change.
