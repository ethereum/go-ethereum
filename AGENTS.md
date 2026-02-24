# AGENTS.md — Guidelines for AI-Assisted Contributions

This document provides instructions for AI agents (and their operators) before committing code and creating pull requests on this repository.

## Pre-Commit Checklist

Before every commit, run **all** of the following checks and ensure they pass:

### 1. Formatting

Before committing, always run `gofmt` and `goimports` on all modified files:

```sh
gofmt -w <modified files>
goimports -w <modified files>
```

### 2. Linting

```sh
go run ./build/ci.go lint
```

This runs additional style checks. Fix any issues before committing.

### 3. Generated Code

```sh
go run ./build/ci.go check_generate
```

Ensures that all generated files (e.g., `gen_*.go`) are up to date. If this fails, run the appropriate `go generate` commands and include the updated files in your commit.

### 4. Dependency Hygiene

```sh
go run ./build/ci.go check_baddeps
```

Verifies that no forbidden dependencies have been introduced.

### 5. Tests

```sh
go run ./build/ci.go test
```

Run the full test suite. All tests must pass.

### 6. Build All Commands

Verify that all tools compile successfully:

```sh
make all
```

This builds all executables under `cmd/`, including `keeper` which has special build requirements.

## Pull Request Title Format

PR titles must follow this convention:

```
<list of modified paths>: description
```

Examples:
- `core/vm: fix stack overflow in PUSH instruction`
- `core, eth: add arena allocator support`
- `cmd/geth, internal/ethapi: refactor transaction args`
- `trie/archiver: streaming subtree archival to fix OOM`

Use the top-level package paths, comma-separated if multiple areas are affected. Only mention the directories with functional changes, interface changes that trickle all over the codebase should not generate an exhaustive list. The description should be a short, lowercase summary of the change.

## Summary

Before creating a PR, confirm:

- [ ] `gofmt` and `goimports` applied to all modified files
- [ ] `go run ./build/ci.go lint` passes
- [ ] `go run ./build/ci.go check_generate` passes
- [ ] `go run ./build/ci.go check_baddeps` passes
- [ ] `go run ./build/ci.go test` passes
- [ ] `make all` succeeds
