# forge

![coverage](https://img.shields.io/badge/coverage-~75%25-brightgreen)
![go](https://img.shields.io/badge/go-1.22-00ADD8)

A test framework for Go, built from scratch — an assertion engine, suite
lifecycle, tag-based selection, flaky-test retry, pluggable reporters, and a
standalone runner — all layered on top of the standard `testing` package.

> The coverage badge is illustrative; run `make cover` for the live number. To
> wire a real badge, publish coverage from CI to a gist or Codecov.

## Why build this when `testify`, `ginkgo`, and `gotestsum` exist?

Short answer: **to learn how a test framework works on the inside, not to
replace mature tools.** For production code, reach for the established
libraries. `forge` exists to make the hidden machinery visible:

- What an assertion engine actually does when `Expect(x).ToEqual(y)` fails, and
  why a good failure message is most of the value.
- How setup/teardown lifecycle and fixtures are modeled with interfaces and
  `t.Cleanup`.
- How flaky-test retry forces you to understand `runtime.Goexit` and why each
  attempt needs its own goroutine.
- How a reporter is decoupled from the runner so the same run emits console
  output and JUnit XML.
- Why a standalone runner should orchestrate `go test -json` rather than
  reimplement test discovery.

Each design decision is documented in the source. If you're reading this from a
CV link: the interesting parts are the *decisions*, not the line count — start
with `assert/matchers.go`, `suite/retry.go`, and `run/run.go`. A companion
write-up lives in [`docs/`](docs/building-a-test-framework-in-go.md).

## Design principle: build *on top of* `testing`, not instead of it

`forge` is a library you import into `_test.go` files. `go test` stays the
runner, so you keep subtests, `-run` filtering, `-v`, parallelism, and CI
integration for free. The standalone `forge` command (Phase 6) is itself a thin
orchestrator over `go test -json`.

## Quick start

```go
import (
    "testing"
    "github.com/annguyen34/forge/assert"
    "github.com/annguyen34/forge/suite"
)

func TestAdd(t *testing.T) {
    assert.Expect(t, 2+2).ToEqual(4)
}

func TestWithLifecycle(t *testing.T) {
    s := suite.New(t, "math", nil) // nil reporter → console
    s.BeforeEach(func() { /* setup */ })

    s.Run("adds", func(t *testing.T) {
        assert.Expect(t, 1+1).ToEqual(2)
    })
}
```

### Soft assertions — collect every failure, report once

```go
soft := assert.NewSoft(t)
defer soft.Flush()
soft.Expect(got.Name).ToEqual("ada")
soft.Expect(got.Age).ToEqual(36) // runs even if the line above failed
```

### Tags & selective execution

```go
s.Run("login", fn, suite.Tags("smoke", "auth"))
s.Run("wip",   fn, suite.Skip("feature behind flag"))

s := suite.New(t, "name", nil).OnlyTags("smoke") // or set FORGE_TAGS=smoke
```

When a filter is active, only tests carrying a matching tag run; untagged tests
are skipped. With no filter, everything runs.

### Flaky-test retry

```go
s.RunFlaky("network ping", 3, func(t assert.TB) {
    assert.Expect(t, ping()).ToBeNil() // passes if any of 3 attempts pass
})
```

The body takes `assert.TB` (not `*testing.T`) because retry needs a swappable
target — you still assert with the same `Expect` API.

### JUnit XML for CI

```go
j := &report.JUnit{}
s := suite.New(t, "math", j)
// ... run tests ...
j.WriteFile("report.xml")
```

## Standalone runner

Run every package, get a unified summary and optional JUnit — your test files
don't even need to import forge:

```bash
go build -o forge ./cmd/forge

./forge                      # run ./... with a console summary
./forge -junit report.xml    # also write JUnit XML
./forge -race ./pkg/...      # pass -race, limit to a pattern
./forge -tags smoke          # set FORGE_TAGS for forge-level tag filtering
```

## Run the examples

```bash
go test ./examples/ -v
FORGE_TAGS=smoke go test ./examples/ -run TestSelective -v
go test ./...            # everything, including forge's own tests
```

## Available matchers

`ToEqual` · `ToNotEqual` · `ToBeNil` · `ToNotBeNil` · `ToBeTrue` · `ToBeFalse` ·
`ToHaveLen` · `ToContain` · `ToMatch` · `ToError` · `ToErrorContaining`

## Roadmap

- [x] **Phase 1** — assertion engine (matchers, diff, soft assertions)
- [x] **Phase 2** — suite + lifecycle hooks (`BeforeAll/Each`, `AfterAll/Each`)
- [x] **Phase 3** — table-driven tests (see `examples/`)
- [x] **Phase 4** — reporters (console + JUnit XML)
- [x] **Phase 5** — tagging, filtering, flaky-test retry
- [x] **Phase 6** — standalone runner (`cmd/forge`) over `go test -json`
- [x] **Phase 7** — polish: godoc, coverage, blog write-up
