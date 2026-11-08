# Set Overlap Join — Architecture and Design

**Version:** 0.1.0  
**Author:** Lofrano Tecnologia (https://lofranotecnologia.com.br)  
**License:** MIT  

---

## 1. Architectural Overview

`set-overlap-join` is designed as a standalone, zero-dependency Go SDK and CLI for discovering identical and near-duplicate entity sets. Unlike approximate nearest-neighbor approaches (such as MinHash or SimHash) that introduce false positives and false negatives, this library implements an **exact filtering-verification pipeline**.

```
Input Sets
    │
    ▼
[Input Validation & Deduplication Check]
    │
    ├── Empty Sets ──────────► [Combinatorial Empty-Empty Matcher]
    │                                  │
    └── Non-Empty Sets                 │
            │                          │
            ▼                          │
    [Corpus Frequency Analysis]        │
            │                          │
            ▼                          │
    [Global Token Ordering]            │
            │                          │
            ▼                          │
    [Prefix Index Construction &       │
     Length Filtering]                 │
            │                          │
            ▼                          │
    [Two-Pointer Exact Merge]          │
            │                          │
            ▼                          │
    [Integer Threshold Test]           │
            │                          │
            ▼                          │
    [Deterministic Pair Ordering] ◄────┘
            │
            ▼
     Final Result
```

---

## 2. Memory and Allocation Discipline

1. **Zero Runtime Reflection or Dynamic Allocation**:
   Internal candidate tracking reuses pre-allocated slices and maps cleared between query sets, avoiding thousands of transient heap allocations during batch joins.
2. **Deterministic Output Guarantee**:
   Pairs are canonically ordered such that `SetA < SetB`, and the final list is sorted lexicographically by `(SetA, SetB)`. Input permutations or thread scheduling variations produce byte-for-byte identical results.
3. **Budgeted Execution**:
   Callers operating in resource-constrained environments can provide finite bounds for prefix visits, merge steps, and output capacities. When a budget is reached, the engine fails closed with status `INCOMPLETE`, retaining only validly verified pairs without ever reporting false negatives as clean passes.

---

## 3. Comparison with Related Approaches

| Technique | Guarantee | Memory Footprint | Edge Case Behavior |
| :--- | :--- | :--- | :--- |
| **All-Pairs $O(N^2)$** | Exact | Minimal | Quadratic execution time |
| **MinHash / LSH** | Approximate (probabilistic) | Constant | False negatives on threshold boundaries |
| **Prefix-Filtered Join (Ours)** | **Exact (100% precision & recall)** | Bounded by corpus tokens | Verified exact boundaries & fail-closed budgets |
