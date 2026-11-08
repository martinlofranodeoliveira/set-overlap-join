# Set Overlap Join

**Development version 0.1.0** — an original, offline Go library and CLI from [Lofrano Tecnologia](https://lofranotecnologia.com.br).  
**Repository:** [martinlofranodeoliveira/set-overlap-join](https://github.com/martinlofranodeoliveira/set-overlap-join)

---

## Overview

`set-overlap-join` performs an exact batch **Jaccard similarity join** across collections of pretokenized sets. It applies mathematically proven prefix filtering and cardinality bounds to prune non-qualifying pairs, followed by exact integer-only verification.

Unlike probabilistic sketches (MinHash, HyperLogLog, SimHash), this library guarantees **100% precision and 100% recall** under a rational threshold $p/q$, with zero floating-point rounding errors and zero external runtime dependencies.

### Key Invariants

- **Exact Rational Arithmetic**: Evaluates $(q + p) \cdot |A \cap B| \ge p \cdot (|A| + |B|)$ using checked integer arithmetic.
- **Prefix-Completeness**: Prunes candidate pairs using the proven prefix bound $|S| - \lceil p|S|/q \rceil + 1$ without missing any true matches.
- **Strict Boundary Semantics**: Deterministically handles empty sets: $J(\emptyset, \emptyset) = 1/1$ and $J(\emptyset, B) = 0$.
- **Fail-Closed Budgets**: Explicit counters for prefix visits and merge steps allow callers to cap execution. Interrupted runs return `INCOMPLETE` with individually verified pairs, never misrepresenting an incomplete search as a complete negative.
- **Deterministic Output**: Output pairs are canonically ordered such that `SetA < SetB` and sorted lexicographically. Shuffling input order produces byte-for-byte identical results.

---

## Requirements and Installation

- **Go**: Version 1.22 or newer (standard library only; no external dependencies).
- **Operating System**: Linux, macOS, or Windows.

```bash
# Clone the repository
git clone https://github.com/martinlofranodeoliveira/set-overlap-join.git
cd set-overlap-join

# Run unit and race tests
go test -v -race ./...

# Build the CLI
go build -o bin/set-overlap-join ./cmd/set-overlap-join
```

---

## CLI Usage

### 1. Deterministic Offline Demo

Run the built-in self-contained demonstration asserting threshold matching on synthetic component label sets:

```bash
# Human-readable output with assertion checks
./bin/set-overlap-join demo --assert

# Machine-readable JSON output
./bin/set-overlap-join demo --assert --json
```

Example output:

```json
{
  "status": "COMPLETE_MATCHES",
  "pairs": [
    {
      "set_a": "empty:placeholder-a",
      "set_b": "empty:placeholder-b",
      "intersection": 0,
      "union": 0,
      "numerator": 1,
      "denominator": 1
    },
    {
      "set_a": "svc:auth-core",
      "set_b": "svc:auth-proxy",
      "intersection": 4,
      "union": 6,
      "numerator": 2,
      "denominator": 3
    }
  ],
  "stats": {
    "total_sets": 7,
    "empty_sets": 2,
    "non_empty_sets": 5,
    "prefix_visits": 9,
    "merge_steps": 9,
    "candidate_pairs": 2,
    "verified_pairs": 2
  }
}
```

### 2. Joining Custom Input Files

```bash
./bin/set-overlap-join run --threshold 3/4 --input data/sets.json --json
```

Input JSON format:

```json
{
  "sets": [
    {"id": "doc1", "tokens": [10, 20, 30, 40]},
    {"id": "doc2", "tokens": [20, 30, 40, 50]}
  ]
}
```

---

## SDK Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/martinlofranodeoliveira/set-overlap-join"
)

func main() {
	sets := []setjoin.Set{
		{ID: "set_a", Tokens: []uint32{1, 2, 3, 4}},
		{ID: "set_b", Tokens: []uint32{2, 3, 4, 5}},
	}

	threshold := setjoin.Threshold{P: 3, Q: 5} // Jaccard >= 3/5 (60%)
	budget := setjoin.Budget{}                 // Use default limits

	result, err := setjoin.JoinSets(sets, threshold, budget)
	if err != nil {
		log.Fatalf("Join error: %v", err)
	}

	fmt.Printf("Status: %s\n", result.Status)
	for _, pair := range result.Pairs {
		fmt.Printf("Match: %s <-> %s (Jaccard = %d/%d)\n",
			pair.SetA, pair.SetB, pair.Numerator, pair.Denominator)
	}
}
```

---

## Specifications and Contracts

- [Contract Specification](docs/contract.md): Mathematical definitions, integer proof, and input bounds.
- [Architecture & Design](docs/design.md): Inverted prefix index, length filtering, and memory discipline.

---

## Scope and Limits

- Maximum admitted sets: 1,024
- Maximum tokens per set: 1,024
- Maximum total cumulative tokens: 65,536
- This library assumes tokens are pre-extracted `uint32` values. Natural language text tokenization, embeddings, MinHash approximations, and external database adapters are outside its scope.

---

## License and Security

- **License**: MIT License. Copyright © 2026 Lofrano Tecnologia.
- **Security**: Please report vulnerabilities via [GitHub Private Vulnerability Reporting](https://github.com/martinlofranodeoliveira/set-overlap-join/security/advisories/new).
- **Official Homepage**: [lofranotecnologia.com.br](https://lofranotecnologia.com.br)
