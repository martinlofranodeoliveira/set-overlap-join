package setjoin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// naiveExhaustiveJoin implements simple O(N^2) Jaccard join for testing ground truth.
func naiveExhaustiveJoin(sets []Set, th Threshold) []Pair {
	var pairs []Pair
	for i := 0; i < len(sets); i++ {
		setA := sets[i]
		tokA := make(map[uint32]struct{}, len(setA.Tokens))
		for _, t := range setA.Tokens {
			tokA[t] = struct{}{}
		}

		for j := i + 1; j < len(sets); j++ {
			setB := sets[j]
			tokB := make(map[uint32]struct{}, len(setB.Tokens))
			for _, t := range setB.Tokens {
				tokB[t] = struct{}{}
			}

			// J(empty, empty) = 1
			if len(tokA) == 0 && len(tokB) == 0 {
				idA, idB := setA.ID, setB.ID
				if idA > idB {
					idA, idB = idB, idA
				}
				pairs = append(pairs, Pair{
					SetA:         idA,
					SetB:         idB,
					Intersection: 0,
					Union:        0,
					Numerator:    1,
					Denominator:  1,
				})
				continue
			}

			if len(tokA) == 0 || len(tokB) == 0 {
				continue // J = 0
			}

			inter := 0
			for t := range tokA {
				if _, exists := tokB[t]; exists {
					inter++
				}
			}
			union := len(tokA) + len(tokB) - inter

			// (Q + P) * inter >= P * (lenA + lenB)
			if (th.Q+th.P)*inter >= th.P*(len(tokA)+len(tokB)) {
				d := gcd(inter, union)
				idA, idB := setA.ID, setB.ID
				if idA > idB {
					idA, idB = idB, idA
				}
				pairs = append(pairs, Pair{
					SetA:         idA,
					SetB:         idB,
					Intersection: inter,
					Union:        union,
					Numerator:    inter / d,
					Denominator:  union / d,
				})
			}
		}
	}

	return pairs
}

func TestBasicMatches(t *testing.T) {
	sets := []Set{
		{ID: "s1", Tokens: []uint32{1, 2, 3}},
		{ID: "s2", Tokens: []uint32{2, 3, 4}},
		{ID: "s3", Tokens: []uint32{10, 20}},
	}

	res, err := JoinSets(sets, Threshold{P: 1, Q: 2}, Budget{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != StatusCompleteMatches {
		t.Fatalf("expected COMPLETE_MATCHES, got %s", res.Status)
	}
	if len(res.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(res.Pairs))
	}
	p := res.Pairs[0]
	if p.SetA != "s1" || p.SetB != "s2" || p.Intersection != 2 || p.Union != 4 || p.Numerator != 1 || p.Denominator != 2 {
		t.Fatalf("unexpected pair details: %+v", p)
	}
}

func TestEmptySets(t *testing.T) {
	sets := []Set{
		{ID: "empty1", Tokens: []uint32{}},
		{ID: "empty2", Tokens: []uint32{}},
		{ID: "s1", Tokens: []uint32{1, 2}},
	}

	res, err := JoinSets(sets, Threshold{P: 1, Q: 1}, Budget{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Pairs) != 1 {
		t.Fatalf("expected 1 empty-empty match, got %d", len(res.Pairs))
	}
	if res.Pairs[0].SetA != "empty1" || res.Pairs[0].SetB != "empty2" {
		t.Fatalf("unexpected empty match: %+v", res.Pairs[0])
	}
}

func TestThresholdBoundaries(t *testing.T) {
	// Intersection 2, union 3 -> J = 2/3
	sets := []Set{
		{ID: "a", Tokens: []uint32{1, 2}},
		{ID: "b", Tokens: []uint32{1, 2, 3}},
	}

	// At threshold 2/3: matches
	res, err := JoinSets(sets, Threshold{P: 2, Q: 3}, Budget{})
	if err != nil || len(res.Pairs) != 1 {
		t.Fatalf("expected match at exact 2/3 threshold, got %v (pairs: %d)", err, len(res.Pairs))
	}

	// At threshold 3/4 (0.75 > 0.666): does not match
	res2, err := JoinSets(sets, Threshold{P: 3, Q: 4}, Budget{})
	if err != nil || len(res2.Pairs) != 0 || res2.Status != StatusCompleteNoMatches {
		t.Fatalf("expected no match at 3/4 threshold, got %v (pairs: %d)", err, len(res2.Pairs))
	}
}

func TestInputValidation(t *testing.T) {
	// Duplicate set IDs
	_, err := JoinSets([]Set{{ID: "dup"}, {ID: "dup"}}, Threshold{P: 1, Q: 2}, Budget{})
	if err != ErrDuplicateSetID {
		t.Fatalf("expected ErrDuplicateSetID, got %v", err)
	}

	// Duplicate token within set
	_, err = JoinSets([]Set{{ID: "s1", Tokens: []uint32{1, 2, 1}}}, Threshold{P: 1, Q: 2}, Budget{})
	if err != ErrDuplicateToken {
		t.Fatalf("expected ErrDuplicateToken, got %v", err)
	}

	// Invalid thresholds
	_, err = JoinSets([]Set{{ID: "s1"}}, Threshold{P: 0, Q: 1}, Budget{})
	if err != ErrInvalidThreshold {
		t.Fatalf("expected ErrInvalidThreshold for P=0, got %v", err)
	}
	_, err = JoinSets([]Set{{ID: "s1"}}, Threshold{P: 2, Q: 1}, Budget{})
	if err != ErrInvalidThreshold {
		t.Fatalf("expected ErrInvalidThreshold for P > Q, got %v", err)
	}
}

func TestBudgetLimits(t *testing.T) {
	sets := []Set{
		{ID: "s1", Tokens: []uint32{1, 2, 3, 4, 5}},
		{ID: "s2", Tokens: []uint32{1, 2, 3, 4, 6}},
		{ID: "s3", Tokens: []uint32{1, 2, 3, 4, 7}},
	}

	// Restrict merge steps to 1 -> should fail closed with StatusIncomplete
	res, err := JoinSets(sets, Threshold{P: 1, Q: 2}, Budget{MaxMergeSteps: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != StatusIncomplete {
		t.Fatalf("expected StatusIncomplete due to merge steps budget, got %s", res.Status)
	}

	// Restrict output pairs to 1
	res2, err := JoinSets(sets, Threshold{P: 1, Q: 2}, Budget{MaxOutputPairs: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Status != StatusIncomplete {
		t.Fatalf("expected StatusIncomplete due to output pairs capacity, got %s", res2.Status)
	}
	if len(res2.Pairs) != 1 {
		t.Fatalf("expected exactly 1 partial pair, got %d", len(res2.Pairs))
	}
}

func TestPermutationInvariance(t *testing.T) {
	baseSets := []Set{
		{ID: "doc_a", Tokens: []uint32{10, 20, 30, 40, 50}},
		{ID: "doc_b", Tokens: []uint32{20, 30, 40, 50, 60}},
		{ID: "doc_c", Tokens: []uint32{10, 20, 70, 80}},
		{ID: "doc_d", Tokens: []uint32{99, 100}},
		{ID: "empty_x", Tokens: []uint32{}},
	}

	th := Threshold{P: 1, Q: 2}
	expected, err := JoinSets(baseSets, th, Budget{})
	if err != nil {
		t.Fatalf("base run failed: %v", err)
	}

	rng := rand.New(rand.NewSource(42))
	for iter := 0; iter < 10; iter++ {
		shuffled := make([]Set, len(baseSets))
		copy(shuffled, baseSets)
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		// Also shuffle tokens within sets
		for s := range shuffled {
			toks := make([]uint32, len(shuffled[s].Tokens))
			copy(toks, shuffled[s].Tokens)
			rng.Shuffle(len(toks), func(i, j int) {
				toks[i], toks[j] = toks[j], toks[i]
			})
			shuffled[s].Tokens = toks
		}

		res, err := JoinSets(shuffled, th, Budget{})
		if err != nil {
			t.Fatalf("shuffled run %d failed: %v", iter, err)
		}
		if res.Status != expected.Status {
			t.Fatalf("mismatched status on iter %d: expected %s, got %s", iter, expected.Status, res.Status)
		}
		if !reflect.DeepEqual(res.Pairs, expected.Pairs) {
			t.Fatalf("mismatched pairs on iter %d:\nexpected %+v\ngot      %+v", iter, expected.Pairs, res.Pairs)
		}
	}
}

func TestExhaustiveCompletenessAgainstNaiveOracle(t *testing.T) {
	// Generate random corpora across diverse token universes and test against naive ground truth
	rng := rand.New(rand.NewSource(1337))

	thresholds := []Threshold{
		{P: 1, Q: 4},
		{P: 1, Q: 2},
		{P: 2, Q: 3},
		{P: 3, Q: 4},
		{P: 1, Q: 1},
	}

	for trial := 0; trial < 20; trial++ {
		numSets := rng.Intn(25) + 5
		sets := make([]Set, numSets)

		for i := 0; i < numSets; i++ {
			setLen := rng.Intn(15)
			tokMap := make(map[uint32]struct{})
			for len(tokMap) < setLen {
				tokMap[uint32(rng.Intn(50)+1)] = struct{}{}
			}
			toks := make([]uint32, 0, len(tokMap))
			for tok := range tokMap {
				toks = append(toks, tok)
			}
			sets[i] = Set{
				ID:     fmt.Sprintf("set_%03d", i),
				Tokens: toks,
			}
		}

		th := thresholds[trial%len(thresholds)]
		expectedPairs := naiveExhaustiveJoin(sets, th)

		res, err := JoinSets(sets, th, Budget{})
		if err != nil {
			t.Fatalf("trial %d failed with error: %v", trial, err)
		}

		if len(res.Pairs) != len(expectedPairs) {
			t.Fatalf("trial %d threshold %d/%d: expected %d matches, got %d",
				trial, th.P, th.Q, len(expectedPairs), len(res.Pairs))
		}

		// Verify every returned pair matches ground truth
		for k, p := range res.Pairs {
			exp := expectedPairs[k]
			if p.SetA != exp.SetA || p.SetB != exp.SetB || p.Intersection != exp.Intersection || p.Union != exp.Union {
				t.Fatalf("trial %d pair %d mismatch:\nexpected %+v\ngot      %+v", trial, k, exp, p)
			}
		}
	}
}

func TestAgainstPythonOracle(t *testing.T) {
	// Verify that the independent Python oracle agrees on exact matches
	sets := []Set{
		{ID: "c1", Tokens: []uint32{1, 2, 3, 5, 8}},
		{ID: "c2", Tokens: []uint32{1, 2, 4, 8, 16}},
		{ID: "c3", Tokens: []uint32{100, 200}},
		{ID: "c4", Tokens: []uint32{}},
	}

	th := Threshold{P: 2, Q: 5}
	res, err := JoinSets(sets, th, Budget{})
	if err != nil {
		t.Fatalf("JoinSets failed: %v", err)
	}

	// Prepare payload for Python oracle
	type oracleInput struct {
		Sets []Set `json:"sets"`
	}
	inputBytes, err := json.Marshal(oracleInput{Sets: sets})
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	oracleScript, err := filepath.Abs("internal/oracle/oracle.py")
	if err != nil {
		t.Fatalf("failed to get oracle path: %v", err)
	}

	cmd := exec.Command("python3", "-c", fmt.Sprintf(`
import json, sys
sys.path.insert(0, '.')
from internal.oracle.oracle import exhaustive_join
data = json.loads('''%s''')
res = exhaustive_join(data['sets'], %d, %d)
print(json.dumps(res))
`, string(inputBytes), th.P, th.Q))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python oracle failed (%v): %s (script path: %s)", err, string(output), oracleScript)
	}

	type oracleMatch struct {
		SetA         string `json:"set_a"`
		SetB         string `json:"set_b"`
		Intersection int    `json:"intersection"`
		Union        int    `json:"union"`
		Numerator    int    `json:"numerator"`
		Denominator  int    `json:"denominator"`
	}
	var oracleMatches []oracleMatch
	if err := json.Unmarshal(output, &oracleMatches); err != nil {
		t.Fatalf("failed to decode python oracle output: %v, raw: %s", err, string(output))
	}

	if len(res.Pairs) != len(oracleMatches) {
		t.Fatalf("oracle count mismatch: expected %d, got %d", len(oracleMatches), len(res.Pairs))
	}
	for i, p := range res.Pairs {
		o := oracleMatches[i]
		if p.SetA != o.SetA || p.SetB != o.SetB || p.Intersection != o.Intersection || p.Union != o.Union {
			t.Fatalf("pair %d mismatch between Go and Python:\nGo:     %+v\nPython: %+v", i, p, o)
		}
	}
}

func TestConcurrentCallsRace(t *testing.T) {
	sets := []Set{
		{ID: "alpha", Tokens: []uint32{1, 2, 3}},
		{ID: "beta", Tokens: []uint32{2, 3, 4}},
		{ID: "gamma", Tokens: []uint32{3, 4, 5}},
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			p := (iter % 3) + 1
			res, err := JoinSets(sets, Threshold{P: p, Q: 4}, Budget{})
			if err != nil {
				t.Errorf("concurrent run failed: %v", err)
			}
			if res.Status != StatusCompleteMatches && res.Status != StatusCompleteNoMatches {
				t.Errorf("unexpected status: %s", res.Status)
			}
		}(i)
	}
	wg.Wait()
}
