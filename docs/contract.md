# Set Overlap Join — Contract Specification

**Version:** 0.1.0  
**Status:** Frozen Contract  
**Author:** Lofrano Tecnologia (https://lofranotecnologia.com.br)  
**License:** MIT  

---

## 1. Domain and Problem Statement

The `setjoin` package computes the **exact batch Jaccard similarity join** across a bounded collection of pretokenized sets. Given a set of records $\mathcal{S} = \{S_1, S_2, \dots, S_N\}$ and a rational threshold $T = p/q \in (0, 1]$, the engine outputs all unordered pairs $(S_i, S_j)$ with $i < j$ such that:

$$J(S_i, S_j) \ge \frac{p}{q}$$

where:

$$J(A, B) = \frac{|A \cap B|}{|A \cup B|}$$

### 1.1 Exact Rational Arithmetic
Floating-point arithmetic is strictly forbidden in similarity testing. The condition:

$$\frac{|A \cap B|}{|A| + |B| - |A \cap B|} \ge \frac{p}{q}$$

is algebraically transformed to the exact integer inequality:

$$(q + p) \cdot |A \cap B| \ge p \cdot (|A| + |B|)$$

Under the maximum admitted cardinalities ($|A|, |B| \le 1024$) and maximum threshold components ($p, q \le 65535$), the maximum cross-product $(q + p) \cdot 1024$ cannot exceed $134,215,680$, which is strictly less than $2^{28}$ and fits safely within standard integer representations without overflow.

### 1.2 Boundary Semantics for Empty Sets
- $J(\emptyset, \emptyset) = 1$: Two empty sets have identical empty contents. For any valid threshold $p/q \le 1$, every pair of empty sets qualifies with intersection $0$ and union $0$.
- $J(\emptyset, B) = 0$ for $B \neq \emptyset$: An empty set and a nonempty set have zero similarity and never match.

---

## 2. Input Limits and Constraints

| Dimension | Bound | Error on Violation |
| :--- | :--- | :--- |
| Maximum Sets | $1,024$ | `ErrTooManySets` |
| Maximum Tokens per Set | $1,024$ | `ErrTooManyTokensInSet` |
| Maximum Total Memberships | $65,536$ | `ErrTooManyMemberships` |
| Maximum Output Pairs | $523,776$ | Output truncated to `StatusIncomplete` |
| Threshold Bounds | $1 \le p \le q \le 65,535$ | `ErrInvalidThreshold` |
| Duplicate Set Identifiers | Prohibited | `ErrDuplicateSetID` |
| Duplicate Tokens within a Set | Prohibited | `ErrDuplicateToken` |

---

## 3. Algorithm and Filtering Invariants

### 3.1 Global Token Order
All nonempty set tokens are ordered globally by:
1. **Ascending Corpus Frequency**: tokens appearing in fewer sets appear earlier.
2. **Ascending Numeric Token ID**: breaks frequency ties deterministically.

### 3.2 Proven Prefix Bound
For a set $S$ sorted under the global token order, the prefix length $L(S)$ required to guarantee completeness is:

$$L(S) = |S| - \left\lceil \frac{p \cdot |S|}{q} \right\rceil + 1$$

Implemented using integer arithmetic:

$$\left\lceil \frac{p \cdot |S|}{q} \right\rceil = \frac{p \cdot |S| + q - 1}{q}$$

**Theorem (Completeness):** If $J(A, B) \ge p/q$, then $A$ and $B$ must share at least one token in their prefixes of length $L(A)$ and $L(B)$. If they share no tokens in their prefixes, the maximum possible overlap is bounded below the required threshold.

### 3.3 Cardinality (Length) Filter
When evaluating candidate $A$ against query set $B$ (with $|A| \le |B|$):
The maximum possible intersection is $|A|$. Therefore, the maximum possible Jaccard similarity is $|A| / |B|$.
If:

$$q \cdot |A| < p \cdot |B|$$

candidate $A$ cannot possibly qualify and is skipped before computing set intersections.

---

## 4. Evaluation Statuses

The result status is an explicit enum:
- `COMPLETE_MATCHES`: Entire search completed within budget; at least one matching pair verified.
- `COMPLETE_NO_MATCHES`: Entire search completed within budget; zero matching pairs found.
- `INCOMPLETE`: Operation budget exhausted (e.g., prefix visits, merge steps, or output pair capacity). Returned pairs are individually verified, but the caller is informed that the search is incomplete.
- `INVALID_INPUT`: Validation failure (bad threshold, duplicate tokens, out-of-bound counts).
