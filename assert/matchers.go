package assert

import (
	"reflect"
	"regexp"
	"strings"
)

// ----------------------------------------------------------------------------
// Equality
// ----------------------------------------------------------------------------

// ToEqual asserts deep equality between actual and expected.
//
// We use reflect.DeepEqual here. Know its sharp edges (and document them — good
// blog material):
//   - It is strict about types: int(1) != int64(1).
//   - Two distinct pointers to equal values ARE DeepEqual (it follows pointers).
//   - func values are only equal if both nil.
//   - NaN != NaN (DeepEqual returns false), which matches IEEE-754 but surprises people.
//
// A production framework often swaps this for google/go-cmp to get better diffs
// and Equal() method support. Doing that swap yourself is Phase 1's stretch goal.
func (a *Assertion) ToEqual(expected any) *Assertion {
	a.t.Helper()
	if !reflect.DeepEqual(a.actual, expected) {
		a.fail("expected values to be equal:%s", diff(expected, a.actual))
	}
	return a
}

// ToNotEqual asserts actual and expected are not deeply equal.
func (a *Assertion) ToNotEqual(expected any) *Assertion {
	a.t.Helper()
	if reflect.DeepEqual(a.actual, expected) {
		a.fail("expected values to differ, but both were: %s", typedRepr(a.actual))
	}
	return a
}

// ----------------------------------------------------------------------------
// Nil / boolean
// ----------------------------------------------------------------------------

// ToBeNil asserts actual is nil. Handles the classic Go gotcha where an
// interface holding a nil pointer is itself non-nil — we unwrap via reflect.
func (a *Assertion) ToBeNil() *Assertion {
	a.t.Helper()
	if !isNil(a.actual) {
		a.fail("expected nil, got: %s", typedRepr(a.actual))
	}
	return a
}

// ToNotBeNil asserts actual is not nil.
func (a *Assertion) ToNotBeNil() *Assertion {
	a.t.Helper()
	if isNil(a.actual) {
		a.fail("expected non-nil value, got nil")
	}
	return a
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

// ToBeTrue asserts actual is the boolean true.
func (a *Assertion) ToBeTrue() *Assertion {
	a.t.Helper()
	if b, ok := a.actual.(bool); !ok || !b {
		a.fail("expected true, got: %s", typedRepr(a.actual))
	}
	return a
}

// ToBeFalse asserts actual is the boolean false.
func (a *Assertion) ToBeFalse() *Assertion {
	a.t.Helper()
	if b, ok := a.actual.(bool); !ok || b {
		a.fail("expected false, got: %s", typedRepr(a.actual))
	}
	return a
}

// ----------------------------------------------------------------------------
// Collections / strings
// ----------------------------------------------------------------------------

// ToHaveLen asserts the length of actual (string, slice, array, map, chan).
func (a *Assertion) ToHaveLen(n int) *Assertion {
	a.t.Helper()
	rv := reflect.ValueOf(a.actual)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		if rv.Len() != n {
			a.fail("expected length %d, got %d (value: %s)", n, rv.Len(), typedRepr(a.actual))
		}
	default:
		a.fail("ToHaveLen: value of type %T has no length", a.actual)
	}
	return a
}

// ToContain asserts actual contains expected. For strings it's substring; for
// slices/arrays it's element membership (via DeepEqual); for maps it's key
// membership.
func (a *Assertion) ToContain(expected any) *Assertion {
	a.t.Helper()
	rv := reflect.ValueOf(a.actual)
	switch rv.Kind() {
	case reflect.String:
		sub, ok := expected.(string)
		if !ok {
			a.fail("ToContain: actual is string but expected is %T", expected)
			return a
		}
		if !strings.Contains(rv.String(), sub) {
			a.fail("expected string %q to contain %q", rv.String(), sub)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if reflect.DeepEqual(rv.Index(i).Interface(), expected) {
				return a
			}
		}
		a.fail("expected collection to contain %s", typedRepr(expected))
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			if reflect.DeepEqual(k.Interface(), expected) {
				return a
			}
		}
		a.fail("expected map to contain key %s", typedRepr(expected))
	default:
		a.fail("ToContain: unsupported type %T", a.actual)
	}
	return a
}

// ToMatch asserts actual (a string) matches the given regular expression.
func (a *Assertion) ToMatch(pattern string) *Assertion {
	a.t.Helper()
	s, ok := a.actual.(string)
	if !ok {
		a.fail("ToMatch: actual is %T, want string", a.actual)
		return a
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		a.fail("ToMatch: invalid pattern %q: %v", pattern, err)
		return a
	}
	if !re.MatchString(s) {
		a.fail("expected %q to match pattern %q", s, pattern)
	}
	return a
}

// ----------------------------------------------------------------------------
// Errors
// ----------------------------------------------------------------------------

// ToError asserts actual is a non-nil error.
func (a *Assertion) ToError() *Assertion {
	a.t.Helper()
	if a.actual == nil {
		a.fail("expected an error, got nil")
		return a
	}
	if _, ok := a.actual.(error); !ok {
		a.fail("expected an error, got %s", typedRepr(a.actual))
	}
	return a
}

// ToErrorContaining asserts actual is an error whose message contains substr.
func (a *Assertion) ToErrorContaining(substr string) *Assertion {
	a.t.Helper()
	err, ok := a.actual.(error)
	if !ok || err == nil {
		a.fail("expected an error containing %q, got %s", substr, typedRepr(a.actual))
		return a
	}
	if !strings.Contains(err.Error(), substr) {
		a.fail("expected error message %q to contain %q", err.Error(), substr)
	}
	return a
}
