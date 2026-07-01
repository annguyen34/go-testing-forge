// Command forge is a standalone test runner: it orchestrates `go test -json`,
// aggregates results across all packages, prints a unified console summary, and
// can emit JUnit XML for CI — without your test files importing forge.
//
//	forge                      # run ./... with a console summary
//	forge -junit report.xml    # also write JUnit XML
//	forge -race ./pkg/...      # pass -race, limit to a pattern
//	forge -tags smoke          # set FORGE_TAGS for forge-level tag filtering
//
// This is the Phase 6 deliverable. Note it builds ON `go test` rather than
// reimplementing test discovery — the honest, robust choice (see package run).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/annguyen34/forge/report"
	"github.com/annguyen34/forge/run"
)

func main() {
	junitPath := flag.String("junit", "", "write JUnit XML to this path")
	race := flag.Bool("race", false, "run tests with the race detector")
	tags := flag.String("tags", "", "forge-level tag filter (sets FORGE_TAGS)")
	flag.Parse()

	cfg := run.Config{
		Patterns:  flag.Args(),
		Race:      *race,
		ForgeTags: *tags,
	}

	suites, passed, err := run.Go(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Replay parsed results through the console reporter for a clean summary.
	console := &report.Console{}
	junit := &report.JUnit{}
	var totalPass, totalFail, totalSkip int
	for _, s := range suites {
		console.SuiteStart(s.Name)
		for _, tr := range s.Tests {
			console.TestEnd(tr)
		}
		console.SuiteEnd(s)
		junit.SuiteEnd(s)
		p, f, k := s.Counts()
		totalPass, totalFail, totalSkip = totalPass+p, totalFail+f, totalSkip+k
	}

	fmt.Printf("TOTAL: %d passed, %d failed, %d skipped across %d package(s)\n",
		totalPass, totalFail, totalSkip, len(suites))

	if *junitPath != "" {
		if err := junit.WriteFile(*junitPath); err != nil {
			fmt.Fprintln(os.Stderr, "writing junit:", err)
			os.Exit(2)
		}
		fmt.Printf("JUnit XML written to %s\n", *junitPath)
	}

	if !passed {
		os.Exit(1)
	}
}
