#!/usr/bin/env python3
"""Independent ground-truth oracle for exact set Jaccard similarity join.

Uses Python standard library Fractions and sets. Has zero dependencies on any
Go implementation. Evaluates exhaustive all-pairs ground truth for testing.
"""
from fractions import Fraction
import json
import sys


def jaccard_similarity(tokens_a: set, tokens_b: set) -> Fraction:
    """Compute exact rational Jaccard similarity between two token sets."""
    if not tokens_a and not tokens_b:
        return Fraction(1, 1)
    if not tokens_a or not tokens_b:
        return Fraction(0, 1)
    intersection_len = len(tokens_a & tokens_b)
    union_len = len(tokens_a | tokens_b)
    return Fraction(intersection_len, union_len)


def exhaustive_join(sets_data, p: int, q: int):
    """Exhaustive O(N^2) join comparing each pair (i, j) with i < j."""
    threshold = Fraction(p, q)
    matches = []
    
    # Sort sets by ID for deterministic output
    sorted_sets = sorted(sets_data, key=lambda s: s["id"])
    n = len(sorted_sets)
    
    for i in range(n):
        set_a = sorted_sets[i]
        tokens_a = set(set_a["tokens"])
        for j in range(i + 1, n):
            set_b = sorted_sets[j]
            tokens_b = set(set_b["tokens"])
            
            sim = jaccard_similarity(tokens_a, tokens_b)
            if sim >= threshold:
                inter = len(tokens_a & tokens_b)
                uni = len(tokens_a | tokens_b)
                matches.append({
                    "set_a": min(set_a["id"], set_b["id"]),
                    "set_b": max(set_a["id"], set_b["id"]),
                    "intersection": inter,
                    "union": uni,
                    "numerator": sim.numerator,
                    "denominator": sim.denominator,
                })
                
    matches.sort(key=lambda m: (m["set_a"], m["set_b"]))
    return matches


def main():
    if len(sys.argv) < 4:
        print("Usage: oracle.py <input.json> <p> <q>", file=sys.stderr)
        sys.exit(1)
        
    input_file = sys.argv[1]
    p = int(sys.argv[2])
    q = int(sys.argv[3])
    
    with open(input_file, "r", encoding="utf-8") as f:
        data = json.load(f)
        
    results = exhaustive_join(data["sets"], p, q)
    print(json.dumps(results, indent=2))


if __name__ == "__main__":
    main()
