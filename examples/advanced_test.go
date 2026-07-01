package examples

import (
	"errors"
	"testing"

	"github.com/annguyen34/forge/assert"
	"github.com/annguyen34/forge/suite"
)

// Run only the smoke set with:  FORGE_TAGS=smoke go test ./examples/ -run TestSelective -v
func TestSelective_Tags(t *testing.T) {
	s := suite.New(t, "selective", nil)

	s.Run("fast smoke check", func(t *testing.T) {
		assert.Expect(t, add(1, 1)).ToEqual(2)
	}, suite.Tags("smoke"))

	s.Run("slow regression check", func(t *testing.T) {
		assert.Expect(t, add(100, 100)).ToEqual(200)
	}, suite.Tags("regression"))

	s.Run("work in progress", func(t *testing.T) {
		t.Fatal("not ready")
	}, suite.Skip("feature behind flag"))
}

// A deliberately flaky body: fails on the first call, passes afterward. With 3
// allowed attempts it ends green. Note the body takes assert.TB (retry needs a
// swappable target), but you still assert with the same Expect API.
func TestFlaky_Retry(t *testing.T) {
	calls := 0
	s := suite.New(t, "flaky-demo", nil)
	s.RunFlaky("intermittent network call", 3, func(t assert.TB) {
		calls++
		var err error
		if calls < 2 {
			err = errors.New("connection reset")
		}
		assert.Expect(t, err).ToBeNil()
	})
}
