package scene

import (
	"strings"
	"testing"
)

// A number a document writes that cannot draw anything is refused when the
// document is read, and a number it leaves out still means the look's own.
//
// Both halves were found by opezart reviewing the scenery branch. A spiral
// with no radius divided by it, so every particle's place was a NaN and the
// emitter drew nothing while saying nothing; and an explicit zero or minus
// sign was thrown away by a guard asking whether the number was above nought,
// which a document that said nothing answers the same way.
func TestNumbersThatCannotDrawAreRefused(t *testing.T) {
	for _, c := range []struct {
		what, doc, says string
	}{
		{"a spiral with no radius",
			`{"kind":"emitter","emit":{"kind":"spiral","rate":40,
			  "life":7}}`, "needs a radius"},
		{"a drift with no radius",
			`{"kind":"emitter","emit":{"kind":"drift","rate":40,
			  "life":7}}`, "needs a radius"},
		{"a negative radius",
			`{"kind":"emitter","emit":{"kind":"spiral","rate":40,
			  "life":7,"radius":-6}}`, "radius is -6"},
		{"a rate of nought",
			`{"kind":"emitter","emit":{"kind":"fountain","rate":0,
			  "life":7}}`, "rate is 0"},
		{"a planet of negative radius",
			`{"kind":"planet","radius":-5}`, "radius is -5"},
		{"a hole sampling minus seven",
			`{"kind":"blackhole","hole":{"samples":-7}}`, "-7"},
		{"a world of minus three tiles",
			`{"kind":"planet","world":{"tiles":-3}}`, "tiles is -3"},
	} {
		_, err := Read("test", []byte(`{"shapes":[`+c.doc+`]}`))
		if err == nil {
			t.Errorf("%s: read without a word", c.what)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("%s: said %q, wanted it to mention %q", c.what,
				err, c.says)
		}
	}
}

// What a document leaves out is still left out, and a binding is still read
// when it is drawn rather than judged now: it may be nought at this moment
// and right at the next, or read a constant nobody has supplied yet.
func TestAbsentAndBoundNumbersAreLeftAlone(t *testing.T) {
	for _, doc := range []string{
		`{"kind":"planet","world":{"look":"terran"}}`,
		`{"kind":"blackhole","hole":{"look":"ember"}}`,
		`{"kind":"emitter","emit":{"kind":"fountain","rate":40,
		  "life":2}}`,
		`{"kind":"emitter","emit":{"kind":"spiral","rate":40,"life":2,
		  "radius":{"mid":6,"amp":2,"period":8}}}`,
		`{"kind":"planet","world":{"tiles":"@tiles"}}`,
	} {
		if _, err := Read("test", []byte(
			`{"const":{"tiles":12},"shapes":[`+doc+`]}`)); err != nil {
			t.Errorf("%s: refused: %v", doc, err)
		}
	}
}
