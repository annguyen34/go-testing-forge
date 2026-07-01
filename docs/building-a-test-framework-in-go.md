# Building a test framework in Go: what `pytest` hides from you

I spend most of my day writing tests, not test frameworks. Like a lot of SDETs,
I'd internalized `pytest` fixtures and Playwright's `expect()` as primitives —
things that simply *exist*, the way `+` exists. So I built a small test framework
in Go from scratch, on purpose, to find out what those primitives actually are
underneath. This is what fell out.

The framework is called `forge`. It is emphatically **not** a replacement for
`testify` or `ginkgo`; it's a study. But building it changed how I read failure
output, how I think about flaky tests, and how I'll design the next bit of test
infrastructure at work. Here are the parts that surprised me.

## 1. The assertion is the cheap part. The *message* is the product.

My first matcher was four lines:

```go
func (a *Assertion) ToEqual(expected any) *Assertion {
    if !reflect.DeepEqual(a.actual, expected) {
        a.fail("not equal")
    }
    return a
}
```

It worked, and it was useless. The first time a test failed with `not equal`, I
understood viscerally why every real assertion library pours its effort into the
failure path. The comparison is trivial; *explaining the comparison* is the job.

The single highest-leverage thing a message can do is print types, not just
values. These two values render identically with `%v`:

```
expected: 1
actual:   1
```

…and yet the test fails, because one is `int` and the other is `int64`.
`reflect.DeepEqual` is strict about types and will never call those equal. So the
fix is to render `%v (%T)`:

```
expected: 1 (int)
actual:   1 (int64)
```

That one change resolved more "but they're obviously equal!" confusion than any
other line in the project. `pytest`'s rich assert rewriting and Playwright's
diff output are doing a fancier version of exactly this — they just hide the
machinery so well you forget it's a feature someone had to build.

## 2. `reflect.DeepEqual` is a convenient footgun

Reaching for `reflect.DeepEqual` is the obvious move, and it's fine — as long as
you know its edges, because they will bite a test author eventually:

- It is **type-strict**: `int(1) != int64(1)`, as above.
- It **follows pointers**: two distinct `*T` pointing at equal values *are*
  `DeepEqual`. Sometimes that's what you want; sometimes it hides an aliasing
  bug you were trying to catch.
- `NaN != NaN`. Correct per IEEE-754, surprising in a test.
- Funcs are only equal if both are nil.

A production framework eventually swaps this for `google/go-cmp`, which gives
structural diffs and respects `Equal()` methods. I left `DeepEqual` in and
documented the trade-off in a comment, because the *point* was to understand the
trade-off — not to paper over it. That's the difference between using a tool and
knowing one.

## 3. Build *on top of* `testing`, don't replace it

The biggest architectural decision came early: do I write my own runner, or
build on `go test`? It's tempting to write your own — it feels more like "a real
framework." It's also a trap. `go test` gives you, for free: subtest trees,
`-run` filtering, `-v`, parallelism, race detection, caching, and CI
integration. Reimplementing even half of that well is months of work that
teaches you about process orchestration, not about test frameworks.

So `forge` is a *library* you import into `_test.go` files. The suite wraps
`t.Run`; assertions report through `t`. `go test` stays the engine. This one
constraint kept the whole project honest and small, and it's the same instinct
behind a clean layered API client: depend on the stable thing, don't rebuild it.

## 4. The interface you can't implement

Here's a genuinely Go-specific lesson. My assertion engine naturally wanted to
take a `testing.TB`:

```go
func Expect(t testing.TB, actual any) *Assertion
```

Then I went to test the *failure* paths of my own matchers — does `ToEqual`
actually fail when given unequal values, and is the message right? To test that,
I need a fake `TB` that records failures instead of aborting. And you cannot
build one:

```go
// testing.TB has an unexported method specifically to prevent this.
type TB interface {
    ...
    private()
}
```

The standard library deliberately seals `testing.TB` so nothing outside the
`testing` package can implement it. The fix is the same one `testify` uses:
define your own *narrow* interface with only the methods you need.

```go
type TB interface {
    Helper()
    Fatalf(format string, args ...any)
    Error(args ...any)
}
```

`*testing.T` satisfies it for free, so callers pass `t` exactly as before — but
now I can write a `fakeT` and unit-test the engine that tests everything else.
This is the kind of thing you only learn by hitting the wall.

## 5. Flaky-test retry taught me `runtime.Goexit`

Retry sounds trivial: run the test, if it fails, run it again. It is not
trivial, and the reason is subtle.

Once a `*testing.T` is marked failed, you cannot un-fail it. So "retry" cannot
mean "re-run against the same `t`." It has to mean "run the body against
something I control, decide the verdict myself, and only *then* report once to
`go test`."

That "something I control" is a capturing `TB`. But there's a second wrinkle: a
real assertion failure calls `t.FailNow`, which the `testing` package implements
via `runtime.Goexit` — it unwinds the current goroutine and runs deferred
functions, but doesn't return normally. If my capturing `TB.Fatalf` does the
same (and it should, to behave like the real thing), then it terminates whatever
goroutine it's running in.

So each attempt has to run in **its own goroutine**:

```go
func runAttempt(fn func(t assert.TB)) (passed bool, msg string) {
    c := &captureTB{}
    done := make(chan struct{})
    go func() {
        defer func() {
            if r := recover(); r != nil { // a real panic, not a test failure
                c.failed, c.msg = true, fmt.Sprintf("panic: %v", r)
            }
            close(done)
        }()
        fn(c) // may call Goexit; that just ends THIS goroutine
    }()
    <-done
    return !c.failed, c.msg
}
```

I had used `t.Fatal` thousands of times without once thinking about what
"stops the test" mechanically means. Building retry forced the question. That's
the whole reason for an exercise like this.

## 6. Decouple the reporter, or regret it

The output format is the part most likely to change — console today, JUnit for
CI tomorrow, maybe Allure or a TeamCity service message after that. So the runner
must not know about formats. It knows about an interface:

```go
type Reporter interface {
    SuiteStart(name string)
    TestStart(name string)
    TestEnd(r TestResult)
    SuiteEnd(s SuiteResult)
}
```

The suite *drives* the reporter; the reporter never reaches back. Adding JUnit
XML meant writing one new type, not touching the suite at all. This is the same
dependency-direction discipline that keeps an API client maintainable: high-level
orchestration depends on an abstraction, concrete outputs plug in. Get the arrow
pointing the right way and extension becomes additive instead of surgical.

## 7. A standalone runner should *orchestrate*, not reinvent

The last phase was a standalone `forge` binary that runs across all packages and
emits one unified report. The naive version walks the filesystem, parses ASTs,
compiles, and runs — and it's a fragile reimplementation of `go test` that will
never keep up with the real thing.

The honest version is what `gotestsum` does: shell out to `go test -json`, which
emits a structured event stream (the test2json format), and parse *that*:

```go
type event struct {
    Action  string  // run, pass, fail, skip, output
    Package string
    Test    string
    Elapsed float64
    Output  string
}
```

Aggregate the stream into per-package results, feed it through the same reporters,
and you can produce JUnit XML for *any* `go test` run — even one whose tests never
import forge. Less code, more robust, and it taught me the test2json protocol,
which is genuinely useful knowledge for CI work.

## What I'd tell my past self

If you only ever consume test frameworks, you carry a fuzzy mental model of what
they do. Building a small one — even a deliberately incomplete one — converts
that fuzz into mechanism. I now know why a good diff matters more than the
comparison, why `testify` defines its own `TestingT`, what `t.Fatal` actually
does to a goroutine, and why the smart move is almost always to build *on* the
standard library rather than around it.

None of this makes me want to replace `testify` at work. It makes me much better
at *using* it — and at building the test infrastructure around it, which is most
of the job anyway.

---

*Source: the `forge` repository, with each decision documented inline. Start
with `assert/matchers.go`, `suite/retry.go`, and `run/run.go`.*
