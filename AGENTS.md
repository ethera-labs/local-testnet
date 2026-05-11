# Repository Guidelines

This Go 1.26 project orchestrates a local L1/L2 stack for Ethera Labs contributors. Follow the practices below to keep the control plane predictable and easy to debug.

## Project Structure & Module Organization
- `cmd/localnet` contains the Cobra CLI entry point that wires configuration and subcommands.
- `internal/l1`, `internal/l2`, and `internal/observability` host service-specific orchestration logic; shared logging lives in `internal/logger`.
- `configs/` stores the source-of-truth `config.yaml` plus per-service templates consumed at runtime.
- `internal/l2/infra/docker/` holds the docker-compose overlays and Dockerfiles that drive the L2 stack; `.localnet/` is generated at runtime and can be wiped via `make clean-l2`.
- Tests live beside their packages as Go `_test.go` files; architecture notes sit in `docs/`.

## Build, Test, and Development Commands
- `make build` compiles the CLI to `cmd/localnet/bin/localnet`.
- `make run` starts the CLI with the default stack; use `make run-l1`, `make run-l2`, or `make run-observability` for targeted bring-up.
- `make test` executes `go test ./...`.
- `make lint` runs `golangci-lint` with `.golangci.yml` defaults.
- `make clean`, `make clean-l1`, and `make clean-l2` tear down Docker/Kurtosis resources and clear `.localnet/`.

## Coding Style & Naming Conventions
- Always format Go code with `gofmt` (tabs for indentation, single blank line between logical blocks) before sending a review.
- Adopt Go naming: exported types/functions use PascalCase, private helpers use lowerCamelCase, and constants remain in ALL_CAPS when shared.
- Prefer structured logging via `slog` from `internal/logger` and load configs through `configs.Values` rather than manual file reads.

## Testing Guidelines
- Place unit and integration tests in the same package using `_test.go` files with functions like `TestCommand_Run`.
- Favour table-driven tests for multi-scenario coverage and assert external effects via temporary directories instead of mutating `state/`.
- Run `make test` before pushing; include Kurtosis/Docker smoke checks when touching deployment logic.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects (`Refine L2 slot sync`) following the existing history style, and group related changes logically.
- For pull requests, provide: purpose, touched services (`l1`, `l2`, `observability`), manual test results or logs, and references to issues/spec tickets.
- Screenshot CLI output or `docker ps` summaries when UI-visible behavior changes, and call out required config tweaks in `configs/config.yaml`.
