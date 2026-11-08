# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - Unreleased

### Added
- Core `JoinSets` engine implementing exact rational Jaccard similarity joins.
- Prefix-filtering index with proven prefix bound `|S| - ceil(p*|S|/q) + 1`.
- Cardinality and length filter pruning unqualified set pairs prior to full intersection verification.
- Exact two-pointer intersection merge over sorted token slices using integer arithmetic.
- Dedicated empty-set handling establishing `J(empty, empty) = 1` and `J(empty, non-empty) = 0`.
- Logical operation budgets for prefix visits, merge steps, and maximum output pairs.
- Full status reporting distinguishing `COMPLETE_MATCHES`, `COMPLETE_NO_MATCHES`, and `INCOMPLETE`.
- CLI utility with `demo` and `run` subcommands.
- Independent Python Fraction ground-truth oracle and exhaustive property test suites.
- GitHub Actions CI workflow with formatting, vet, race detection, and offline demo assertion.
