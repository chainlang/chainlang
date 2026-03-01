# Contributing to Hanticoin Core

Thank you for your interest in contributing. This document explains how to set up your environment, submit changes, and report issues.

---

## Before you start

- **Scope:** Hanticoin Core is a payment-focused Layer-1 blockchain (no smart contracts). Contributions should align with that scope.
- **Docs:** Read [README.md](README.md) for build and run instructions, and [docs/SPEC.md](docs/SPEC.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for protocol and design.

---

## How to contribute

1. **Open an issue** — For bugs, propose features, or ask questions. Check existing issues first.
2. **Discuss** — For larger changes, discuss in the issue before sending a big PR.
3. **Fork and branch** — Fork the repo, create a branch from `main` (or the current default branch).
4. **Implement** — Make your changes; keep commits focused and messages clear.
5. **Test and lint** — Run tests and the linter (see below).
6. **Open a pull request** — Target `main`; describe the change and link any related issue.

---

## Development setup

**Requirements**

- Go 1.21 or later  
- Git

**Clone and build**

```bash
git clone https://github.com/your-org/hanticoin.git
cd hanticoin
go build -o hanticoin ./cmd/node
go build -o loadgen ./cmd/loadgen   # optional
```

**Run tests**

```bash
go test ./...
```

**Format code**

```bash
go fmt ./...
```

**Lint**

```bash
golangci-lint run
```

Install the linter if needed:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

---

## Code style and conventions

- Follow standard Go style ([Effective Go](https://go.dev/doc/effective_go), `gofmt`).
- Use meaningful names; keep functions and files focused.
- Add or update tests for behavior changes (e.g. in `*_test.go`).
- Document exported types and functions where the behavior is not obvious from the name.

---

## Pull request process

1. **Branch** — Create a feature or fix branch (e.g. `feature/description` or `fix/issue-123`). Do not push directly to `main`.
2. **Base** — Branch from the latest `main` and keep it updated (rebase or merge as the project prefers).
3. **Scope** — One logical change per PR; split large changes into smaller PRs when possible.
4. **Description** — In the PR:
   - Summarize the change.
   - Reference any related issue (e.g. "Fixes #123").
   - Note any breaking or config change.
5. **Checks** — Ensure CI (if any) and local `go test ./...` and `golangci-lint run` pass.
6. **Review** — Address review comments; maintainers will merge when the PR is approved and green.

---

## Branch strategy

- **main** — Stable, release-ready. All merges go through PRs.
- **Feature/fix branches** — Created from `main`, merged back after review. Delete the branch after merge.

---

## Versioning

Releases follow semantic versioning (e.g. `v1.0.0`). Protocol or config changes that affect compatibility should be reflected in the version and documented in release notes or [docs/](docs/).

---

## Security and vulnerabilities

**Do not** report security vulnerabilities in public issues. Report them privately (e.g. to the maintainers or a listed security contact). See [docs/SECURITY.md](docs/SECURITY.md) for more detail. We will acknowledge and work on a fix before any public disclosure.

---

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](LICENSE)).

---

## Questions

If something is unclear, open an issue with the "question" label or reach out to the maintainers as indicated in the repository.
