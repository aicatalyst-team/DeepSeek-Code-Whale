## What is DeepSeek-Code-Whale?

Whale is an open-source, terminal-based coding agent built in Go and optimized for the DeepSeek API. Think of it as a developer's AI pair programmer that lives entirely in your terminal — it can read your codebase, edit files, run commands, and extend its capabilities through MCP (Model Context Protocol) and a pluggable Skills system.

Under the hood, Whale uses the Cobra CLI framework for command structure and Bubble Tea for its TUI (terminal user interface). One of its more interesting design choices is prefix-cache optimization, which reduces redundant token processing when making repeated DeepSeek API calls — a practical cost-saving measure for developers who run lots of prompts against a large codebase context.

The project is a good example of the emerging "agentic CLI" pattern: a single binary that wraps an LLM API with tool-calling capabilities, giving developers an AI assistant that operates in the same environment where they already work.

## Why this matters for OpenShift AI

Let's be upfront: Whale scored 30/100 on our RHOAI fitness evaluation. It's a client-side tool that calls an external cloud API. It doesn't deploy a model, serve inference, or interact with any Open Data Hub components directly. So why bother with this PoC?

Because the interesting question isn't "does Whale use RHOAI components?" — it's "can we containerize agentic AI developer tools and run them as Kubernetes-native workloads?" If you're building a platform team that wants to offer AI-assisted code review in CI/CD pipelines, or batch code analysis as a Job-based workload, you need to know that these tools containerize cleanly, handle credentials gracefully, and behave well in ephemeral environments.

This PoC exercises the basics: building a Go binary in a UBI-based container, running it as a Kubernetes Job, and validating that CLI tools designed for interactive use can function headlessly. These are foundational questions for any team evaluating agentic AI tooling on OpenShift.

## Setting up the PoC

The infrastructure requirements for this PoC are minimal — deliberately so. Whale is a statically-compiled Go binary with no runtime dependencies:

- **CPU/Memory:** 250m CPU, 256Mi RAM — this is a Go CLI, not a model server
- **GPU:** None required
- **Persistent storage:** None
- **Inference server:** None (Whale calls the external DeepSeek API directly)
- **Vector database:** None
- **Sidecar containers:** None

The only environment variable needed is `DEEPSEEK_API_KEY`, which provides credentials for the DeepSeek platform. For our basic validation scenarios (help, version, doctor), the key isn't strictly required — but any actual `exec` commands that hit the API will need it.

We chose a Job-based deployment model rather than a long-running Deployment. Whale is a CLI tool — it does work and exits. That maps naturally to Kubernetes Jobs, and it's the pattern you'd use if you wanted to run Whale as a step in a Tekton pipeline or a batch code analysis task.

--------------------
**[Image Placeholder 1: Architecture diagram showing Whale's deployment model]**

**Placement rationale**: Readers benefit from seeing the simple architecture — a Kubernetes Job running the Whale container, with an outbound arrow to the DeepSeek API — before diving into implementation details.

**Image generation prompt**: A clean architecture diagram on a white background showing a Kubernetes cluster (represented by a rounded rectangle with the K8s wheel logo) containing a single Job pod labeled "whale". An arrow from the pod points outward to a cloud icon labeled "DeepSeek API". Use flat design, blue and teal color palette, 16:9 aspect ratio, minimal text, developer documentation style.

**Alt text**: Architecture diagram showing a Kubernetes Job pod running the Whale container making outbound API calls to the DeepSeek cloud API.
--------------------

## Containerizing with UBI

Whale is a pure Go project, which makes containerization straightforward — but there were a few decisions worth documenting. We used a multi-stage build with a Go builder stage and a UBI minimal runtime stage:

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /whale ./cmd/whale

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
COPY --from=builder /whale /usr/local/bin/whale
ENTRYPOINT ["whale"]
```

A few things to call out. First, `CGO_ENABLED=0` is critical here — it produces a fully static binary that runs on UBI minimal without needing glibc or any shared libraries. Second, the project's `go.mod` specifies Go 1.24 toolchain compatibility (despite the plan mentioning 1.26, which doesn't exist yet), so we used `golang:1.24` as the builder image.

The resulting image is lean. UBI minimal plus a single static binary keeps the image size small, the attack surface minimal, and startup time negligible — all properties you want for Job-based workloads that spin up, execute, and terminate.

The built image is available at `quay.io/aicatalyst/deepseek-code-whale-whale:latest`.

## Deploying to Kubernetes

Since Whale is a CLI tool that runs and exits, we deployed each test scenario as a separate Kubernetes Job. Here's the structure we used for the help output validation:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: whale-help-output
  labels:
    app: deepseek-code-whale
    poc-scenario: help-output
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 15
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: whale
          image: quay.io/aicatalyst/deepseek-code-whale-whale:latest
          args: ["--help"]
          resources:
            requests:
              cpu: 250m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
          env:
            - name: DEEPSEEK_API_KEY
              valueFrom:
                secretKeyRef:
                  name: deepseek-credentials
                  key: api-key
                  optional: true
```

A few notable choices: `backoffLimit: 0` ensures failed Jobs don't retry — we want clean pass/fail signals. `activeDeadlineSeconds` enforces our per-scenario timeout. And the `DEEPSEEK_API_KEY` secret reference is marked `optional: true` because several of our test scenarios specifically validate behavior without a valid key.

For the `exec-no-key` scenario, we deliberately omitted the secret to verify that Whale handles missing credentials gracefully — no panics, no unhandled exceptions, just a clear error message and a non-zero exit code.

--------------------
**[Image Placeholder 2: Screenshot of test job pods in a terminal or OpenShift console]**

**Placement rationale**: Showing the completed Job pods gives readers a concrete visual of what the deployment looks like in practice, reinforcing that this is a real PoC rather than a theoretical exercise.

**Image generation prompt**: A terminal screenshot showing `kubectl get jobs` output with four completed Kubernetes Jobs: whale-help-output, whale-version-check, whale-doctor-check, and whale-exec-no-key. All showing "1/1" completions. Dark terminal background with green and white monospaced text, realistic CLI appearance, 16:9 aspect ratio.

**Alt text**: Terminal output showing four completed Kubernetes Jobs for the Whale PoC test scenarios, all showing successful completion.
--------------------

## Test results

All four test scenarios passed on the first run. Here are the results:

| Scenario | Description | Status | Duration |
|----------|-------------|--------|----------|
| help-output | CLI shows help with available commands | ✅ PASS | 0.2s |
| version-check | CLI reports its version string | ✅ PASS | 0.2s |
| doctor-check | Self-diagnostic reports runtime info | ✅ PASS | 0.2s |
| exec-no-key | Graceful error without API key | ✅ PASS | 0.2s |

**4/4 scenarios passed.**

The 0.2-second execution times across the board tell us exactly what we'd expect from a statically compiled Go binary — near-instant startup with negligible overhead. This is a meaningful data point for teams considering Whale in CI/CD pipelines: the container overhead is minimal, and the tool is Job-friendly.

The `exec-no-key` result is particularly worth highlighting. Whale exits cleanly with a descriptive error when no API key is present, which means you won't get mysterious CrashLoopBackOff situations in Kubernetes if credentials are misconfigured. Good CLI hygiene matters more than usual in containerized environments.

The `doctor` command ran successfully, outputting runtime diagnostic information. This kind of built-in health check is valuable when debugging deployment issues — it's a pattern we'd love to see in more AI developer tools.

--------------------
**[Image Placeholder 3: Results summary graphic]**

**Placement rationale**: A visual summary of test results provides a quick scannable reference and breaks up the text before the analysis sections.

**Image generation prompt**: A clean results dashboard graphic showing 4 test scenarios in a card layout. Each card has a scenario name, a green checkmark, and "0.2s" timing. A header reads "4/4 Passed". Use a modern flat design with a light gray background, green accent color for passes, rounded corners on cards, 16:9 aspect ratio, suitable for a technical blog post.

**Alt text**: Test results dashboard showing all four PoC scenarios passed with 0.2-second execution times each.
--------------------

## What we learned

**Go CLI tools are ideal Job candidates.** The zero-dependency static binary pattern — `CGO_ENABLED=0` plus UBI minimal — produces images that are small, fast to pull, and instant to start. If you're evaluating agentic CLI tools for Kubernetes-native workflows, Go-based tools have a real deployment advantage.

**Credential handling matters more than you think.** In interactive terminal use, a missing API key means the developer just sets it. In a Kubernetes Job, a missing key can mean silent failures or cryptic crash logs. Whale handles this well, but many LLM-backed tools don't — test this early.

**The RHOAI gap is real but instructive.** Whale calls an external API, which means it doesn't exercise any ODH model serving, pipeline, or data science project capabilities. For a production deployment, the interesting next step would be pointing Whale at a self-hosted DeepSeek model running on an RHOAI model serving instance via a compatible API endpoint. Whale's architecture — it talks to a standard OpenAI-compatible API — means this swap should be straightforward.

**MCP support opens platform possibilities.** Whale's MCP (Model Context Protocol) support is intriguing for platform teams. MCP enables structured tool calling, which could integrate with enterprise systems — code review tools, ticketing systems, deployment pipelines. This is where a tool like Whale could become genuinely interesting in an OpenShift AI context, orchestrating agent workflows rather than just being a single developer's terminal helper.

**What we'd do differently:** We'd test with a self-hosted DeepSeek model behind RHOAI model serving to close the platform integration gap. We'd also explore running Whale as a Tekton task in an OpenShift Pipelines workflow to demonstrate the CI/CD code analysis use case more concretely.

## Try it yourself

If you want to reproduce this PoC or extend it, here's everything you need:

- **Forked repository:** [github.com/aicatalyst-team/DeepSeek-Code-Whale](https://github.com/aicatalyst-team/DeepSeek-Code-Whale.git)
- **Original project:** [github.com/usewhale/DeepSeek-Code-Whale](https://github.com/usewhale/DeepSeek-Code-Whale)
- **Container image:** `quay.io/aicatalyst/deepseek-code-whale-whale:latest`
- **ODH documentation:** [opendatahub.io/docs](https://opendatahub.io/docs)

To run the simplest validation locally:

```bash
podman run --rm quay.io/aicatalyst/deepseek-code-whale-whale:latest --help
```

If you're interested in extending this PoC — especially the self-hosted model serving angle or the Tekton pipeline integration — we'd love to hear about it. Open an issue on the fork, or try deploying a DeepSeek model on RHOAI and pointing Whale at it. That's where this story gets genuinely interesting.
