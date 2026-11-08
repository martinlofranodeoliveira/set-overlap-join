package setjoin

import (
	"errors"
	"fmt"
	"sort"
)

const (
	// MaxSets is the maximum admitted number of input sets.
	MaxSets = 1024

	// MaxTokensPerSet is the maximum number of distinct tokens in one set.
	MaxTokensPerSet = 1024

	// MaxTotalMemberships is the maximum cumulative token count across all sets.
	MaxTotalMemberships = 65536

	// MaxDistinctPairs is the theoretical maximum pair combinations for 1024 sets (1024*1023/2).
	MaxDistinctPairs = 523776
)

var (
	ErrTooManySets        = errors.New("input exceeds maximum allowed sets (1024)")
	ErrTooManyTokensInSet = errors.New("set exceeds maximum allowed tokens (1024)")
	ErrTooManyMemberships = errors.New("total token memberships exceed limit (65536)")
	ErrDuplicateSetID     = errors.New("duplicate set identifier found")
	ErrDuplicateToken     = errors.New("duplicate token found in set")
	ErrInvalidThreshold   = errors.New("invalid threshold: require 1 <= p <= q <= 65535")
	ErrNegativeBudget     = errors.New("budget limits must be non-negative")
)

// Set represents an input set with a unique identifier and distinct tokens.
type Set struct {
	ID     string   `json:"id"`
	Tokens []uint32 `json:"tokens"`
}

// Threshold represents a rational similarity bound p/q (1 <= p <= q <= 65535).
type Threshold struct {
	P int `json:"p"`
	Q int `json:"q"`
}

// Budget caps logical operations to avoid unbounded computation on adversarial inputs.
type Budget struct {
	MaxPrefixVisits int `json:"max_prefix_visits"` // 0 uses default limit
	MaxMergeSteps   int `json:"max_merge_steps"`   // 0 uses default limit
	MaxOutputPairs  int `json:"max_output_pairs"`  // 0 uses maximum possible (523776)
}

// Pair represents a verified pair of sets meeting or exceeding the similarity threshold.
type Pair struct {
	SetA         string `json:"set_a"`
	SetB         string `json:"set_b"`
	Intersection int    `json:"intersection"`
	Union        int    `json:"union"`
	Numerator    int    `json:"numerator"`
	Denominator  int    `json:"denominator"`
}

// Stats captures counters of logical work performed during the join.
type Stats struct {
	TotalSets      int `json:"total_sets"`
	EmptySets      int `json:"empty_sets"`
	NonEmptySets   int `json:"non_empty_sets"`
	PrefixVisits   int `json:"prefix_visits"`
	MergeSteps     int `json:"merge_steps"`
	CandidatePairs int `json:"candidate_pairs"`
	VerifiedPairs  int `json:"verified_pairs"`
}

// Status indicates the final evaluation state of the join.
type Status string

const (
	StatusCompleteMatches   Status = "COMPLETE_MATCHES"
	StatusCompleteNoMatches Status = "COMPLETE_NO_MATCHES"
	StatusIncomplete        Status = "INCOMPLETE"
	StatusInvalidInput      Status = "INVALID_INPUT"
)

// Result is the deterministic payload returned by JoinSets.
type Result struct {
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
	Pairs  []Pair `json:"pairs"`
	Stats  Stats  `json:"stats"`
}

// gcd computes greatest common divisor using Euclidean algorithm.
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// internalSet wraps a set with ordered tokens and prefix length.
type internalSet struct {
	id        string
	tokens    []uint32 // sorted by global token order
	rawTokens []uint32 // sorted numerically for fast two-pointer intersection
	prefixLen int
}

// JoinSets computes an exact batch Jaccard similarity join under rational threshold p/q.
func JoinSets(sets []Set, th Threshold, budget Budget) (Result, error) {
	stats := Stats{
		TotalSets: len(sets),
	}

	// 1. Validate inputs
	if th.P < 1 || th.Q < 1 || th.P > th.Q || th.Q > 65535 {
		return Result{Status: StatusInvalidInput, Reason: ErrInvalidThreshold.Error(), Stats: stats}, ErrInvalidThreshold
	}
	if len(sets) > MaxSets {
		return Result{Status: StatusInvalidInput, Reason: ErrTooManySets.Error(), Stats: stats}, ErrTooManySets
	}
	if budget.MaxPrefixVisits < 0 || budget.MaxMergeSteps < 0 || budget.MaxOutputPairs < 0 {
		return Result{Status: StatusInvalidInput, Reason: ErrNegativeBudget.Error(), Stats: stats}, ErrNegativeBudget
	}

	// Set budget defaults if unspecified
	maxPrefixVisits := budget.MaxPrefixVisits
	if maxPrefixVisits == 0 {
		maxPrefixVisits = MaxDistinctPairs * 64
	}
	maxMergeSteps := budget.MaxMergeSteps
	if maxMergeSteps == 0 {
		maxMergeSteps = MaxDistinctPairs * 64
	}
	maxOutputPairs := budget.MaxOutputPairs
	if maxOutputPairs == 0 {
		maxOutputPairs = MaxDistinctPairs
	}

	seenIDs := make(map[string]struct{}, len(sets))
	totalMemberships := 0

	var emptySetIDs []string
	type rawSet struct {
		id     string
		tokens []uint32
	}
	var nonEmpties []rawSet

	for _, s := range sets {
		if _, exists := seenIDs[s.ID]; exists {
			return Result{Status: StatusInvalidInput, Reason: fmt.Sprintf("%s: %s", ErrDuplicateSetID.Error(), s.ID), Stats: stats}, ErrDuplicateSetID
		}
		seenIDs[s.ID] = struct{}{}

		if len(s.Tokens) > MaxTokensPerSet {
			return Result{Status: StatusInvalidInput, Reason: ErrTooManyTokensInSet.Error(), Stats: stats}, ErrTooManyTokensInSet
		}
		totalMemberships += len(s.Tokens)
		if totalMemberships > MaxTotalMemberships {
			return Result{Status: StatusInvalidInput, Reason: ErrTooManyMemberships.Error(), Stats: stats}, ErrTooManyMemberships
		}

		if len(s.Tokens) == 0 {
			emptySetIDs = append(emptySetIDs, s.ID)
			continue
		}

		// Check for duplicate tokens within set
		tokSet := make(map[uint32]struct{}, len(s.Tokens))
		for _, tok := range s.Tokens {
			if _, dup := tokSet[tok]; dup {
				return Result{Status: StatusInvalidInput, Reason: fmt.Sprintf("%s in set %s: %d", ErrDuplicateToken.Error(), s.ID, tok), Stats: stats}, ErrDuplicateToken
			}
			tokSet[tok] = struct{}{}
		}

		// Store copy of tokens
		copied := make([]uint32, len(s.Tokens))
		copy(copied, s.Tokens)
		nonEmpties = append(nonEmpties, rawSet{id: s.ID, tokens: copied})
	}

	stats.EmptySets = len(emptySetIDs)
	stats.NonEmptySets = len(nonEmpties)

	pairs := make([]Pair, 0)
	incomplete := false
	incompleteReason := ""

	// 2. Handle empty-empty pairs: J(empty, empty) = 1/1 >= p/q
	sort.Strings(emptySetIDs)
	for i := 0; i < len(emptySetIDs); i++ {
		for j := i + 1; j < len(emptySetIDs); j++ {
			if len(pairs) >= maxOutputPairs {
				incomplete = true
				incompleteReason = "output capacity reached during empty-set matching"
				break
			}
			pairs = append(pairs, Pair{
				SetA:         emptySetIDs[i],
				SetB:         emptySetIDs[j],
				Intersection: 0,
				Union:        0,
				Numerator:    1,
				Denominator:  1,
			})
			stats.VerifiedPairs++
		}
		if incomplete {
			break
		}
	}

	if incomplete {
		stats.VerifiedPairs = len(pairs)
		return Result{
			Status: StatusIncomplete,
			Reason: incompleteReason,
			Pairs:  pairs,
			Stats:  stats,
		}, nil
	}

	if len(nonEmpties) == 0 {
		return finalizeResult(pairs, stats), nil
	}

	// 3. Compute global token frequencies for prefix ordering
	tokenFreq := make(map[uint32]int)
	for _, s := range nonEmpties {
		for _, tok := range s.tokens {
			tokenFreq[tok]++
		}
	}

	// Helper to sort tokens by global order: frequency ascending, then value ascending
	tokenLess := func(a, b uint32) bool {
		fa, fb := tokenFreq[a], tokenFreq[b]
		if fa != fb {
			return fa < fb
		}
		return a < b
	}

	// 4. Construct internal representations and calculate prefix lengths
	internalSets := make([]internalSet, len(nonEmpties))
	for i, s := range nonEmpties {
		n := len(s.tokens)
		// Prefix length: |S| - ceil(p*|S|/q) + 1
		// ceil(p*n/q) = (p*n + q - 1) / q
		ceilVal := (th.P*n + th.Q - 1) / th.Q
		pLen := n - ceilVal + 1
		if pLen < 1 {
			pLen = 1
		}
		if pLen > n {
			pLen = n
		}

		tokGlobal := make([]uint32, n)
		copy(tokGlobal, s.tokens)
		sort.Slice(tokGlobal, func(i, j int) bool {
			return tokenLess(tokGlobal[i], tokGlobal[j])
		})

		tokNumeric := make([]uint32, n)
		copy(tokNumeric, s.tokens)
		sort.Slice(tokNumeric, func(i, j int) bool {
			return tokNumeric[i] < tokNumeric[j]
		})

		internalSets[i] = internalSet{
			id:        s.id,
			tokens:    tokGlobal,
			rawTokens: tokNumeric,
			prefixLen: pLen,
		}
	}

	// Sort sets by ascending length, break ties by ID for stability
	sort.Slice(internalSets, func(i, j int) bool {
		if len(internalSets[i].tokens) != len(internalSets[j].tokens) {
			return len(internalSets[i].tokens) < len(internalSets[j].tokens)
		}
		return internalSets[i].id < internalSets[j].id
	})

	// 5. Index and Candidate Generation
	index := make(map[uint32][]int) // token -> list of set indices in internalSets
	candidateCandidates := make([]int, 0, len(internalSets))
	candidateSeen := make(map[int]int) // candidateSetIdx -> match count in prefix

	for curIdx := 0; curIdx < len(internalSets); curIdx++ {
		curSet := internalSets[curIdx]
		curLen := len(curSet.tokens)

		// Reset candidate accumulator
		candidateCandidates = candidateCandidates[:0]
		for k := range candidateSeen {
			delete(candidateSeen, k)
		}

		// Inspect prefix tokens of curSet
		for p := 0; p < curSet.prefixLen; p++ {
			tok := curSet.tokens[p]
			stats.PrefixVisits++
			if stats.PrefixVisits > maxPrefixVisits {
				incomplete = true
				incompleteReason = "prefix visits budget exceeded"
				break
			}

			postings := index[tok]
			for _, candIdx := range postings {
				candSet := internalSets[candIdx]
				candLen := len(candSet.tokens)

				// Cardinality / Length Filter:
				// Since internalSets is sorted by length, candLen <= curLen.
				// For J(cand, cur) >= P/Q, we must have Q * candLen >= P * curLen.
				if th.Q*candLen < th.P*curLen {
					continue
				}

				if count, exists := candidateSeen[candIdx]; exists {
					candidateSeen[candIdx] = count + 1
				} else {
					candidateSeen[candIdx] = 1
					candidateCandidates = append(candidateCandidates, candIdx)
				}
			}
		}

		if incomplete {
			break
		}

		// Sort candidate indices for deterministic traversal
		sort.Ints(candidateCandidates)

		// 6. Verify each candidate via exact two-pointer intersection merge
		for _, candIdx := range candidateCandidates {
			stats.CandidatePairs++
			candSet := internalSets[candIdx]

			// Exact two-pointer intersection merge on numerically sorted tokens
			inter := 0
			i, j := 0, 0
			candTokens := candSet.rawTokens
			curTokens := curSet.rawTokens
			candN := len(candTokens)
			curN := len(curTokens)

			for i < candN && j < curN {
				stats.MergeSteps++
				if stats.MergeSteps > maxMergeSteps {
					incomplete = true
					incompleteReason = "merge steps budget exceeded"
					break
				}

				if candTokens[i] == curTokens[j] {
					inter++
					i++
					j++
				} else if candTokens[i] < curTokens[j] {
					i++
				} else {
					j++
				}
			}

			if incomplete {
				break
			}

			// Exact rational comparison: J = inter / (candN + curN - inter) >= P / Q
			// (Q + P) * inter >= P * (candN + curN)
			union := candN + curN - inter
			if (th.Q+th.P)*inter >= th.P*(candN+curN) {
				if len(pairs) >= maxOutputPairs {
					incomplete = true
					incompleteReason = "output pairs capacity exceeded"
					break
				}

				// Compute reduced fraction
				d := gcd(inter, union)
				num := inter / d
				den := union / d

				setA, setB := candSet.id, curSet.id
				if setA > setB {
					setA, setB = setB, setA
				}

				pairs = append(pairs, Pair{
					SetA:         setA,
					SetB:         setB,
					Intersection: inter,
					Union:        union,
					Numerator:    num,
					Denominator:  den,
				})
				stats.VerifiedPairs++
			}
		}

		if incomplete {
			break
		}

		// Index prefix tokens of curSet
		for p := 0; p < curSet.prefixLen; p++ {
			tok := curSet.tokens[p]
			index[tok] = append(index[tok], curIdx)
		}
	}

	// 7. Sort pairs deterministically by SetA, then SetB
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].SetA != pairs[j].SetA {
			return pairs[i].SetA < pairs[j].SetA
		}
		return pairs[i].SetB < pairs[j].SetB
	})

	stats.VerifiedPairs = len(pairs)

	if incomplete {
		return Result{
			Status: StatusIncomplete,
			Reason: incompleteReason,
			Pairs:  pairs,
			Stats:  stats,
		}, nil
	}

	return finalizeResult(pairs, stats), nil
}

func finalizeResult(pairs []Pair, stats Stats) Result {
	status := StatusCompleteNoMatches
	if len(pairs) > 0 {
		status = StatusCompleteMatches
	}
	return Result{
		Status: status,
		Pairs:  pairs,
		Stats:  stats,
	}
}
