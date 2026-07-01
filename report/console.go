package report

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ANSI colors. Kept tiny on purpose; disable with NO_COLOR (a de facto standard).
const (
	cReset = "\033[0m"
	cRed   = "\033[31m"
	cGreen = "\033[32m"
	cGray  = "\033[90m"
	cBold  = "\033[1m"
)

// Console is a human-readable Reporter. It writes a line per test and a summary
// per suite. Zero value is usable (writes to stdout, color auto-detected).
type Console struct {
	W     io.Writer // defaults to os.Stdout
	color bool
	init  bool
}

func (c *Console) ensure() {
	if c.init {
		return
	}
	if c.W == nil {
		c.W = os.Stdout
	}
	// Disable color if NO_COLOR is set or output isn't a terminal-ish writer.
	c.color = os.Getenv("NO_COLOR") == ""
	c.init = true
}

func (c *Console) paint(color, s string) string {
	if !c.color {
		return s
	}
	return color + s + cReset
}

func (c *Console) SuiteStart(name string) {
	c.ensure()
	fmt.Fprintf(c.W, "%s\n", c.paint(cBold, "▶ "+name))
}

func (c *Console) TestStart(name string) {}

func (c *Console) TestEnd(r TestResult) {
	c.ensure()
	var mark string
	switch r.Status {
	case Passed:
		mark = c.paint(cGreen, "✓")
	case Failed:
		mark = c.paint(cRed, "✗")
	case Skipped:
		mark = c.paint(cGray, "○")
	}
	fmt.Fprintf(c.W, "  %s %s %s\n", mark, r.Name, c.paint(cGray, dur(r.Duration)))
	if r.Status == Failed && r.Message != "" {
		fmt.Fprintf(c.W, "      %s\n", c.paint(cRed, r.Message))
	}
}

func (c *Console) SuiteEnd(s SuiteResult) {
	c.ensure()
	pass, fail, skip := s.Counts()
	summary := fmt.Sprintf("%d passed, %d failed, %d skipped (%s)", pass, fail, skip, dur(s.Duration))
	color := cGreen
	if fail > 0 {
		color = cRed
	}
	fmt.Fprintf(c.W, "%s\n\n", c.paint(color, summary))
}

func dur(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}
