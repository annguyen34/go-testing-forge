package run

import (
	"bytes"
	"os/exec"

	"github.com/annguyen34/forge/report"
)

// Config controls a standalone run.
type Config struct {
	Patterns  []string // package patterns, e.g. ["./..."]; default ["./..."]
	Race      bool     // pass -race
	ForgeTags string   // value for FORGE_TAGS env (forge-level tag filter)
}

// Go executes `go test -json` for the given patterns and parses the result.
// It returns the aggregated suites plus whether the run passed overall.
//
// Even when tests fail, `go test` exits non-zero; we still parse its output, so
// a non-nil error here means the tooling itself failed (e.g. a build error),
// not that tests failed. Test failures are reflected in the returned suites.
func Go(cfg Config) (suites []report.SuiteResult, passed bool, err error) {
	patterns := cfg.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	args := []string{"test", "-json"}
	if cfg.Race {
		args = append(args, "-race")
	}
	args = append(args, patterns...)

	cmd := exec.Command("go", args...)
	if cfg.ForgeTags != "" {
		cmd.Env = append(cmd.Environ(), "FORGE_TAGS="+cfg.ForgeTags)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // build errors land here; Parse skips non-JSON lines

	runErr := cmd.Run() // non-nil when tests fail OR build fails

	suites, perr := Parse(bytes.NewReader(stdout.Bytes()))
	if perr != nil {
		return nil, false, perr
	}

	// Determine pass/fail from parsed results, not exit code, so we can tell a
	// real test failure from a build error (no suites parsed + runErr != nil).
	passed = true
	for _, s := range suites {
		if _, fail, _ := s.Counts(); fail > 0 {
			passed = false
		}
	}
	if len(suites) == 0 && runErr != nil {
		// Nothing parsed but the command failed → tooling/build error.
		return suites, false, &BuildError{Output: stdout.String()}
	}
	return suites, passed, nil
}

// BuildError indicates `go test` failed before producing test events.
type BuildError struct{ Output string }

func (e *BuildError) Error() string { return "go test failed to build:\n" + e.Output }
