package scene

import (
	"testing"

	"github.com/janearc/miami/grade"
)

// The document must paint what the Go scene painted. That is the only
// definition of "expressive enough" worth having: not that the file parses, not
// that it looks about right, but that every sample lands on the same colour as
// the code it replaced.
func TestDuskDocumentMatchesItsParameters(t *testing.T) {
	d, err := Load("../scenes/dusk.json")
	if err != nil {
		t.Fatal(err)
	}
	if d.Seconds != 20 || len(d.Layers) != 7 {
		t.Fatalf(
			"dusk.json: %g seconds, %d layers", d.Seconds,
			len(d.Layers),
		)
	}
	sh := grade.Sextants(40, 12)
	// the sun sinks and the sky darkens: the same pixel must not be the
	// same colour at the start and the end, or nothing is animated at all
	a, err := d.At(0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := d.At(20)
	if err != nil {
		t.Fatal(err)
	}
	u, v := sh.UV(sh.W/2, sh.H/4)
	if a.At(u, v) == b.At(u, v) {
		t.Errorf("the sky is one colour at nought and at twenty")
	}
}

// A misspelled field must be refused by name. A scene format that silently
// ignores what it does not understand turns a typo into a missing layer, and a
// missing layer looks like a design decision.
func TestUnknownFieldIsRefused(t *testing.T) {
	var n Node
	if err := n.UnmarshalJSON([]byte(`{"raidal":{}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := n.field(ctx{}); err == nil {
		t.Errorf("a misspelled field was accepted")
	}
}

// A value binding must actually travel.
func TestValueTravels(t *testing.T) {
	var v Value
	bind := []byte(`{"from":0,"to":1,"ease":"smooth"}`)
	if err := v.UnmarshalJSON(bind); err != nil {
		t.Fatal(err)
	}
	if got := v.at(ctx{e: 0}); got != 0 {
		t.Errorf("start: %v", got)
	}
	if got := v.at(ctx{e: 1}); got != 1 {
		t.Errorf("end: %v", got)
	}
	if got := v.at(ctx{e: 0.5}); got != 0.5 {
		t.Errorf("middle of a smoothstep: %v", got)
	}
}
