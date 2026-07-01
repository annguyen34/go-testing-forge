// Package forge is a from-scratch test framework for Go, built as a study of how
// such frameworks work on the inside.
//
// It is intentionally a learning project, not a replacement for testify, Ginkgo,
// or gotestsum. Each subpackage isolates one concern and documents the design
// decisions behind it:
//
//   - assert  — a fluent assertion engine (matchers, diffs, soft assertions)
//     layered on a narrow TB interface so it stays testable.
//   - suite   — setup/teardown lifecycle, tag-based selection, and flaky-test
//     retry, all built on top of testing.T so `go test` remains the runner.
//   - report  — a Reporter interface with console and JUnit XML implementations,
//     decoupled so new outputs plug in without touching the runner.
//   - run     — the engine behind the standalone `forge` command, which
//     orchestrates `go test -json` and aggregates the event stream rather than
//     reimplementing test discovery.
//
// The guiding principle throughout: build ON the standard library, not instead
// of it. See the package-level docs and the README for the full rationale.
package forge
