package report

import (
	"encoding/xml"
	"io"
	"os"
)

// JUnit collects results and writes them as JUnit XML — the lingua franca that
// Jenkins, GitLab CI, TeamCity, and GitHub Actions all parse to render test
// reports. Being able to emit this is the single most "CV-visible" feature of
// the framework: it's what makes your runner a first-class CI citizen.
//
// Usage: create one, pass it as the suite's reporter, then call WriteFile after
// the run (or wire it into a top-level TestMain).
type JUnit struct {
	suites []SuiteResult
}

func (j *JUnit) SuiteStart(name string) {}
func (j *JUnit) TestStart(name string)  {}
func (j *JUnit) TestEnd(r TestResult)   {}
func (j *JUnit) SuiteEnd(s SuiteResult) { j.suites = append(j.suites, s) }

// XML schema structs. Field tags map to the standard JUnit element/attribute
// names that CI servers expect.
type xmlTestsuites struct {
	XMLName xml.Name       `xml:"testsuites"`
	Suites  []xmlTestsuite `xml:"testsuite"`
}

type xmlTestsuite struct {
	Name     string        `xml:"name,attr"`
	Tests    int           `xml:"tests,attr"`
	Failures int           `xml:"failures,attr"`
	Skipped  int           `xml:"skipped,attr"`
	Time     float64       `xml:"time,attr"`
	Cases    []xmlTestcase `xml:"testcase"`
}

type xmlTestcase struct {
	Name    string      `xml:"name,attr"`
	Time    float64     `xml:"time,attr"`
	Failure *xmlFailure `xml:"failure,omitempty"`
	Skipped *xmlSkipped `xml:"skipped,omitempty"`
}

type xmlFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type xmlSkipped struct{}

// Marshal renders all collected suites to JUnit XML bytes.
func (j *JUnit) Marshal() ([]byte, error) {
	root := xmlTestsuites{}
	for _, s := range j.suites {
		pass, fail, skip := s.Counts()
		_ = pass
		xs := xmlTestsuite{
			Name:     s.Name,
			Tests:    len(s.Tests),
			Failures: fail,
			Skipped:  skip,
			Time:     s.Duration.Seconds(),
		}
		for _, tc := range s.Tests {
			xc := xmlTestcase{Name: tc.Name, Time: tc.Duration.Seconds()}
			switch tc.Status {
			case Failed:
				xc.Failure = &xmlFailure{Message: firstLine(tc.Message), Body: tc.Message}
			case Skipped:
				xc.Skipped = &xmlSkipped{}
			}
			xs.Cases = append(xs.Cases, xc)
		}
		root.Suites = append(root.Suites, xs)
	}
	out, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

// Write emits the XML to w.
func (j *JUnit) Write(w io.Writer) error {
	b, err := j.Marshal()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// WriteFile emits the XML to a path (e.g. "report.xml").
func (j *JUnit) WriteFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return j.Write(f)
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
