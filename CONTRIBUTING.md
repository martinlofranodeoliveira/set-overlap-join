# Contributing to Set Overlap Join

Thank you for your interest in contributing to Set Overlap Join.

## Development Principles

1. **Deterministic and Exact**: All algorithms avoid floating-point approximations, random sampling, or heuristic pruning that compromises exact Jaccard similarity.
2. **Offline Reproducibility**: The core SDK relies solely on the Go standard library and operates completely offline without network calls or runtime external processes.
3. **Bounded Resources**: Input sets, token counts, and logical operation budgets remain strictly bounded to prevent Denial-of-Service or runaway computations.
4. **Adversarial Testing**: Every algorithm modification must pass comprehensive regression tests and comparisons against the independent Fraction ground-truth oracle.

## Submitting Pull Requests

1. Fork and create a topic branch.
2. Ensure `go test -v -race ./...` passes without errors or data races.
3. Verify formatting with `gofmt -s -w .` and static analysis with `go vet ./...`.
4. Run the offline assertion demo: `go run ./cmd/set-overlap-join demo --assert`.
5. Open a pull request describing the changes and test evidence.
