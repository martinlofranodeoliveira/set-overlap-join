package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/martinlofranodeoliveira/set-overlap-join"
)

func parseThreshold(s string) (setjoin.Threshold, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return setjoin.Threshold{}, fmt.Errorf("threshold must be in 'P/Q' rational format, e.g. '1/2' or '3/4'")
	}
	p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return setjoin.Threshold{}, fmt.Errorf("invalid numerator P: %w", err)
	}
	q, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return setjoin.Threshold{}, fmt.Errorf("invalid denominator Q: %w", err)
	}
	return setjoin.Threshold{P: p, Q: q}, nil
}

func runDemo(assertMode bool, jsonMode bool) int {
	// Synthetic near-duplicate service component label sets
	demoSets := []setjoin.Set{
		{ID: "svc:auth-core", Tokens: []uint32{101, 102, 103, 104, 105}},
		{ID: "svc:auth-proxy", Tokens: []uint32{101, 102, 103, 104, 106}},
		{ID: "svc:billing-api", Tokens: []uint32{201, 202, 203, 204}},
		{ID: "svc:billing-worker", Tokens: []uint32{201, 202, 203, 205}},
		{ID: "svc:metrics-collector", Tokens: []uint32{301, 302}},
		{ID: "empty:placeholder-a", Tokens: []uint32{}},
		{ID: "empty:placeholder-b", Tokens: []uint32{}},
	}

	th := setjoin.Threshold{P: 2, Q: 3} // Jaccard >= 2/3 (0.666...)
	res, err := setjoin.JoinSets(demoSets, th, setjoin.Budget{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running demo join: %v\n", err)
		return 1
	}

	if jsonMode {
		payload, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
			return 1
		}
		fmt.Println(string(payload))
	} else {
		fmt.Println("Set Overlap Join — Development 0.1.0")
		fmt.Printf("Evaluated %d sets under rational threshold %d/%d (%.2f)\n\n",
			res.Stats.TotalSets, th.P, th.Q, float64(th.P)/float64(th.Q))

		fmt.Printf("Status: %s\n", res.Status)
		fmt.Printf("Verified Matching Pairs: %d\n", len(res.Pairs))
		fmt.Printf("Counters: %d prefix visits, %d merge steps, %d candidate pairs\n\n",
			res.Stats.PrefixVisits, res.Stats.MergeSteps, res.Stats.CandidatePairs)

		for i, p := range res.Pairs {
			fmt.Printf("  [%d] %s <-> %s: Jaccard = %d/%d (intersection: %d, union: %d)\n",
				i+1, p.SetA, p.SetB, p.Numerator, p.Denominator, p.Intersection, p.Union)
		}
	}

	if assertMode {
		// Invariant checks
		if res.Status != setjoin.StatusCompleteMatches {
			fmt.Fprintf(os.Stderr, "Assertion failed: expected COMPLETE_MATCHES, got %s\n", res.Status)
			return 1
		}
		// Expect auth-core vs auth-proxy (4/6 = 2/3 >= 2/3)
		// Expect billing-api vs billing-worker (3/5 = 0.60 < 2/3 -> should not match)
		// Expect placeholder-a vs placeholder-b (empty/empty -> 1/1 >= 2/3 -> matches)
		if len(res.Pairs) != 2 {
			fmt.Fprintf(os.Stderr, "Assertion failed: expected exactly 2 pairs, got %d\n", len(res.Pairs))
			return 1
		}
	}

	return 0
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "demo":
		demoFlags := flag.NewFlagSet("demo", flag.ExitOnError)
		assertFlag := demoFlags.Bool("assert", false, "Assert demo invariants")
		jsonFlag := demoFlags.Bool("json", false, "Output machine-readable JSON")
		_ = demoFlags.Parse(os.Args[2:])
		os.Exit(runDemo(*assertFlag, *jsonFlag))

	case "run", "join":
		runFlags := flag.NewFlagSet("run", flag.ExitOnError)
		thFlag := runFlags.String("threshold", "1/2", "Rational threshold in P/Q format")
		inputFlag := runFlags.String("input", "", "Path to input JSON file containing sets (or '-' for stdin)")
		jsonFlag := runFlags.Bool("json", false, "Output machine-readable JSON")
		_ = runFlags.Parse(os.Args[2:])

		th, err := parseThreshold(*thFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if *inputFlag == "" {
			fmt.Fprintf(os.Stderr, "Error: --input required (specify JSON file path or '-')\n")
			os.Exit(1)
		}

		var rawData []byte
		if *inputFlag == "-" {
			rawData, err = io.ReadAll(os.Stdin)
		} else {
			rawData, err = os.ReadFile(*inputFlag)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}

		var payload struct {
			Sets []setjoin.Set `json:"sets"`
		}
		if err := json.Unmarshal(rawData, &payload); err != nil {
			// Try decoding as bare list
			var listPayload []setjoin.Set
			if err2 := json.Unmarshal(rawData, &listPayload); err2 != nil {
				fmt.Fprintf(os.Stderr, "Error parsing JSON input: %v\n", err)
				os.Exit(1)
			}
			payload.Sets = listPayload
		}

		res, err := setjoin.JoinSets(payload.Sets, th, setjoin.Budget{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Evaluation error: %v\n", err)
			os.Exit(1)
		}

		if *jsonFlag {
			enc, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Printf("Status: %s\nVerified Pairs: %d\n", res.Status, len(res.Pairs))
			for i, p := range res.Pairs {
				fmt.Printf("[%d] %s <-> %s (J = %d/%d)\n", i+1, p.SetA, p.SetB, p.Numerator, p.Denominator)
			}
		}

	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Set Overlap Join — Development 0.1.0
Usage:
  set-overlap-join demo [--assert] [--json]
  set-overlap-join run --threshold P/Q --input <file.json> [--json]

Examples:
  set-overlap-join demo --assert
  set-overlap-join run --threshold 2/3 --input sample.json --json
`)
}
