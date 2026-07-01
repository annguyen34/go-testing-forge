// Package examples demonstrates using forge to test real code. Run with:
//
//	go test ./examples/ -v
//
// This is "dogfooding": the framework tests something with itself, which is the
// best smoke test that the API actually feels good to use.
package examples

import (
	"errors"
	"testing"

	"github.com/annguyen34/forge/assert"
	"github.com/annguyen34/forge/suite"
)

// --- code under test (normally this lives in your real package) -------------

func add(a, b int) int { return a + b }

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// --- plain assertion style (no suite) ---------------------------------------

func TestCalc_Plain(t *testing.T) {
	assert.Expect(t, add(2, 3)).ToEqual(5)

	q, err := divide(10, 2)
	assert.Expect(t, err).ToBeNil()
	assert.Expect(t, q).ToEqual(5)

	_, err = divide(1, 0)
	assert.Expect(t, err).ToErrorContaining("division by zero")
}

// --- suite style with lifecycle ---------------------------------------------

func TestCalc_Suite(t *testing.T) {
	var setupCount int
	s := suite.New(t, "calculator", nil) // nil reporter → console
	s.BeforeEach(func() { setupCount++ })

	s.Run("adds positives", func(t *testing.T) {
		assert.Expect(t, add(2, 2)).ToEqual(4)
	})

	s.Run("divides evenly", func(t *testing.T) {
		q, err := divide(9, 3)
		assert.Expect(t, err).ToBeNil()
		assert.Expect(t, q).ToEqual(3)
	})

	s.Run("rejects divide by zero", func(t *testing.T) {
		_, err := divide(5, 0)
		assert.Expect(t, err).ToError()
	})
}

// --- table-driven style (Phase 3 preview) -----------------------------------

func TestCalc_Table(t *testing.T) {
	cases := []struct {
		name string
		a, b int
		want int
	}{
		{"two plus two", 2, 2, 4},
		{"neg plus pos", -1, 1, 0},
		{"zeros", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Expect(t, add(tc.a, tc.b)).ToEqual(tc.want)
		})
	}
}

// --- soft assertions: see all failures at once ------------------------------

func TestCalc_Soft(t *testing.T) {
	soft := assert.NewSoft(t)
	defer soft.Flush()

	soft.Expect(add(1, 1)).ToEqual(2)
	soft.Expect(add(2, 2)).ToEqual(4)
	// Flip a value to 99 here to see soft mode report every failure together.
}
