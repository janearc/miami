package light

import (
	"strings"
	"testing"
)

// A ramp's ends are the two places anybody writes by hand, and nought and one
// are untyped ints in a call. Asserting float64 on them panicked with
// "interface {} is int, not float64", which names neither the ramp nor the
// stop. Found writing a test for this library, in this library.
func TestARampTakesTheNumbersPeopleWrite(t *testing.T) {
	want := HexRamp(0.0, "ff2200", 1.0, "00ff44")
	for _, got := range []Ramp{
		HexRamp(0, "ff2200", 1, "00ff44"),
		HexRamp(0, "ff2200", 1.0, "00ff44"),
		HexRamp(float32(0), "ff2200", float32(1), "00ff44"),
	} {
		for _, at := range []float64{0, 0.25, 0.5, 1} {
			if got.At(at) != want.At(at) {
				t.Errorf("at %g: %v, want %v", at, got.At(at),
					want.At(at))
			}
		}
	}
}

// And a stop that is not a number at all says so, naming what it got.
func TestARampSaysWhatIsWrongWithAStop(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a ramp took a stop that is not a number")
		}
		said := strings.ToLower(strings.TrimSpace(
			strings.Join(strings.Fields(toString(r)), " ")))
		for _, want := range []string{"ramp", "string"} {
			if !strings.Contains(said, want) {
				t.Errorf("the panic does not mention %q: %s",
					want, said)
			}
		}
	}()
	HexRamp("nought", "ff2200", 1.0, "00ff44")
}

// toString is whatever recover gave, as text.
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return ""
}

// A place with no colour after it is a call that is wrong, and it is refused
// by name rather than quietly becoming a shorter ramp.
//
// Taking the panic away is not the same as fixing it. porter caught this: the
// bound that stopped the index going off the end also made an odd count
// silent, which is the outcome the function's own doc calls the one not worth
// having.
func TestARampRefusesAPlaceWithNoColour(t *testing.T) {
	for _, c := range []struct {
		what  string
		stops []any
		says  string
	}{
		{"a place with nothing after it",
			[]any{0.0, "ff2200", 1.0}, "left over"},
		{"a colour that is not a string",
			[]any{0.0, 0xff2200, 1.0, "00ff44"}, "hex digits"},
	} {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s: taken without a word",
						c.what)
					return
				}
				if !strings.Contains(toString(r), c.says) {
					t.Errorf("%s: said %q, wanted %q",
						c.what, toString(r), c.says)
				}
			}()
			HexRamp(c.stops...)
		}()
	}
}
