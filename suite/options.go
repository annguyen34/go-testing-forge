package suite

import (
	"os"
	"strings"
)

// Opt configures a single test run (tags, skip, retry). Pass any number of
// these to Suite.Run / Suite.RunFlaky.
type Opt func(*testOpts)

type testOpts struct {
	tags       []string
	skipReason string
	hasSkip    bool
}

// Tags labels a test so it can be selected by a tag filter (see Suite.OnlyTags
// and the FORGE_TAGS environment variable).
//
//	s.Run("login smoke", fn, suite.Tags("smoke", "auth"))
func Tags(tags ...string) Opt {
	return func(o *testOpts) { o.tags = append(o.tags, tags...) }
}

// Skip marks a test to be skipped with a recorded reason. The reason shows up
// in reports, unlike a silently commented-out test.
func Skip(reason string) Opt {
	return func(o *testOpts) { o.skipReason = reason; o.hasSkip = true }
}

func buildOpts(opts []Opt) testOpts {
	var o testOpts
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// OnlyTags restricts the suite to tests carrying at least one of the given tags.
// Returns the suite for chaining. An empty call (no tags) clears the filter.
//
// Precedence: an explicit OnlyTags call wins over the FORGE_TAGS env var.
func (s *Suite) OnlyTags(tags ...string) *Suite {
	s.filter = tags
	s.filterSet = true
	return s
}

// activeFilter resolves the effective tag filter: explicit OnlyTags first, then
// the FORGE_TAGS env var (comma-separated), else none.
func (s *Suite) activeFilter() []string {
	if s.filterSet {
		return s.filter
	}
	if env := os.Getenv("FORGE_TAGS"); env != "" {
		parts := strings.Split(env, ",")
		out := parts[:0]
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

// shouldSkipForTags decides whether a test with the given tags should be skipped
// under the active filter.
//
// Convention: when a filter is active, a test runs only if it carries at least
// one matching tag. Untagged tests are skipped while a filter is active — this
// is what lets `FORGE_TAGS=smoke` isolate exactly the smoke set. With no filter,
// everything runs.
func (s *Suite) shouldSkipForTags(tags []string) (skip bool, reason string) {
	filter := s.activeFilter()
	if len(filter) == 0 {
		return false, ""
	}
	for _, want := range filter {
		for _, have := range tags {
			if want == have {
				return false, ""
			}
		}
	}
	return true, "filtered out by tags " + strings.Join(filter, ",")
}
