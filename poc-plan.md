# PoC Plan: deepseek-code-whale

## Project Classification
- **Type:** llm-app
- **Key Technologies:** Go 1.26, DeepSeek API, Cobra CLI framework, Bubble Tea TUI, MCP (Model Context Protocol)
- **ODH Relevance:** Whale is a terminal-based coding agent powered by the DeepSeek LLM API. It demonstrates how LLM-backed developer tools can be containerized and run in an OpenShift/ODH environment, useful for automated code review, code generation, and AI-assisted development workflows in CI/CD pipelines.

## PoC Objectives
What we want to prove:
1. The Whale Go binary compiles successfully in a container build and produces a working CLI executable
2. Core CLI commands (`--help`, `--version`, `doctor`) execute correctly inside the container
3. The tool handles missing/invalid API credentials gracefully without crashing
4. The container image is suitable for use as a Job-based workload on OpenShift (e.g., for batch code analysis tasks)

## Infrastructure Requirements
- **Inference Server:** none (Whale calls the external DeepSeek API directly)
- **Vector Database:** none
- **Embedding Model:** none
- **GPU Required:** no
- **Persistent Storage:** none
- **Resource Profile:** small (256Mi RAM, 250m CPU — Go binary, no heavy compute)
- **Sidecar Containers:** none

## Environment Variables
- `DEEPSEEK_API_KEY` — Required. API key for the DeepSeek platform. The tool calls the DeepSeek API for LLM completions. For basic PoC validation (help, version, doctor), the key is not strictly needed, but `exec` commands require it.

## Test Scenarios

### Scenario 1: help-output
- **Description:** Verify the whale CLI shows help with available commands and options
- **Type:** cli
- **Input:** `whale --help`
- **Expected:** Job exits 0, outputs usage info listing available subcommands (setup, doctor, exec, etc.)
- **Timeout:** 15 seconds

### Scenario 2: version-check
- **Description:** Verify the whale CLI reports its version
- **Type:** cli
- **Input:** `whale --version`
- **Expected:** Job exits 0, outputs a version string
- **Timeout:** 10 seconds

### Scenario 3: doctor-check
- **Description:** Run `whale doctor` to verify the tool's self-diagnostic works
- **Type:** cli
- **Input:** `whale doctor`
- **Expected:** Job exits 0 or with a known diagnostic code, outputs diagnostic information about the runtime environment (Go version, API connectivity status, etc.)
- **Timeout:** 30 seconds

### Scenario 4: exec-no-key
- **Description:** Run `whale exec` without a valid API key to verify graceful error handling
- **Type:** cli
- **Input:** `whale exec "hello"`
- **Expected:** Job exits with a non-zero code and a clear error message about missing or invalid DEEPSEEK_API_KEY. No panics or unhandled crashes.
- **Timeout:** 30 seconds

## Dockerfile Considerations

This is a **Go CLI tool**. The Dockerfile should use a multi-stage build:

1. **Builder stage:** Use `golang:1.26` (or latest compatible) as the build image. Copy `go.mod`, `go.sum`, and run `go mod download`. Then copy all source and run `go build -o /whale ./cmd/whale/`.
2. **Runtime stage:** Use a minimal base image (e.g., `gcr.io/distroless/static-debian12` or `registry.access.redhat.com/ubi9-minimal`). Copy the compiled binary from the builder stage.

Key points:
- **ENTRYPOINT** should be `["whale"]` (the compiled binary).
- **CMD** should default to `["--help"]` so the container does something useful if run without arguments.
- **Do NOT add EXPOSE** — there is no port to expose. Whale is a CLI tool, not a server.
- The binary is statically compiled Go, so the runtime image can be very minimal.
- Include `git` in the runtime image if possible, since Whale uses git operations. If using UBI minimal, `microdnf install -y git` is appropriate.

## Deployment Considerations

- **Do NOT deploy as a Deployment** — Whale is a CLI tool that runs a command and exits. Deploying it as a Kubernetes Deployment would cause CrashLoopBackOff.
- **Do NOT create a Service** — there is no port to expose.
- **Deploy as a Kubernetes Job** for each test scenario. Each Job runs a specific `whale` command, and success is verified by checking the Job's exit code and output via `kubectl logs`.
- For the `doctor` and `exec` scenarios, the `DEEPSEEK_API_KEY` environment variable should be injected via a Kubernetes Secret if available. If no key is provided, the tool should still handle it gracefully (the PoC validates this graceful failure).
- Test via `kubectl logs` on completed Jobs to verify expected output patterns.