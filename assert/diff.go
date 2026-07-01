package assert

import (
	"fmt"
	"reflect"
	"strings"
)

// sprintf is a tiny indirection so fail() and soft.record() share formatting.
func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// typedRepr renders a value with its type so that 1 (int) and 1.0 (float64)
// don't look identical in a failure message. This is the single most useful
// thing a diff can do — most "why did this fail, they look equal!" moments are
// type mismatches hiding behind identical-looking %v output.
func typedRepr(v any) string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v (%T)", v, v)
}

// diff produces a human-readable expected-vs-actual block. For composite types
// (structs, maps, slices) it falls back to %#v which is verbose but unambiguous.
//
// This is deliberately simple. A real production framework would do a
// line-by-line structural diff (see go-cmp). Writing that yourself is a great
// follow-up exercise — and a good blog section on why it's harder than it looks.
func diff(expected, actual any) string {
	ek := kindOf(expected)
	ak := kindOf(actual)
	if isComposite(ek) || isComposite(ak) {
		return fmt.Sprintf("\n  expected: %#v\n  actual:   %#v", expected, actual)
	}
	return fmt.Sprintf("\n  expected: %s\n  actual:   %s", typedRepr(expected), typedRepr(actual))
}

func kindOf(v any) reflect.Kind {
	if v == nil {
		return reflect.Invalid
	}
	return reflect.TypeOf(v).Kind()
}

func isComposite(k reflect.Kind) bool {
	switch k {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array, reflect.Ptr:
		return true
	default:
		return false
	}
}

// indent left-pads every line — used when nesting messages under a header.
func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}
