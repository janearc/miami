package hex_test

import (
	"encoding/json"
	"image/color"
	"math"
	"testing"

	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/scene"
)

// One table, asked of every way into a colour.
//
// porter's shape, after this library was found with three hand-written
// parsers that did not agree: a document's colour could be refused by one,
// accepted by another, and drawn as a third thing by the one that put it on
// the screen. Fixing the first left the other two, which is how the fault
// found the same house twice in one night.
//
// So: whatever any of them does with a string, they all do.
var table = []struct {
	written string
	r, g, b uint8
	colour  bool
}{
	{"ff6ec7", 0xff, 0x6e, 0xc7, true},
	{"FF6EC7", 0xff, 0x6e, 0xc7, true},
	{"#ff6ec7", 0xff, 0x6e, 0xc7, true},
	{"#FF6EC7", 0xff, 0x6e, 0xc7, true},
	{" ff6ec7 ", 0xff, 0x6e, 0xc7, true},
	{"000000", 0, 0, 0, true},
	{"red", 0, 0, 0, false},
	{"ff6ec", 0, 0, 0, false},
	{"ff6ec7ff", 0, 0, 0, false},
	{"", 0, 0, 0, false},
	{"gg6ec7", 0, 0, 0, false},
}

// paint.Hex is the one that reaches a terminal.
func TestPaintDrawsWhatIsWritten(t *testing.T) {
	for _, c := range table {
		want := color.RGBA{c.r, c.g, c.b, 255}
		if got := paint.Hex(c.written); got != want {
			t.Errorf("paint.Hex(%q) = %v, want %v", c.written, got,
				want)
		}
	}
}

// grade.Swatch is the one a ramp is made of, and IsColour is what a document
// is checked against: they must agree with each other.
func TestGradeAgreesWithItself(t *testing.T) {
	for _, c := range table {
		if grade.IsColour(c.written) != c.colour {
			t.Errorf("grade.IsColour(%q) = %v, want %v", c.written,
				!c.colour, c.colour)
		}
	}
}

// And a document: what Read accepts, Read draws. This is the one that was
// broken -- IsColour took "#ff6ec7" and scene's own parser then drew a pale
// blue from it, with nothing said.
func TestADocumentDrawsWhatItAccepted(t *testing.T) {
	for _, c := range table {
		raw, err := json.Marshal(c.written)
		if err != nil {
			t.Fatal(err)
		}
		doc := `{"shapes":[{"kind":"blackhole","hole":{"disk":[` +
			`{"at":0,"c":` + string(raw) + `}]}}]}`
		d, err := scene.Read("test", []byte(doc))
		if (err == nil) != c.colour {
			t.Errorf("%q: read said %v, the table says %v",
				c.written, err, c.colour)
			continue
		}
		if err != nil {
			continue
		}
		got := light.FromSwatch(
			scene.RampAt(d.Shapes[0].Hole.Disk, scene.Ctx{}).At(0))
		want := light.FromRGBA(paint.Hex(c.written))
		// to a part in a billion, not to the bit: the two reach a
		// colour by different conversions and disagree in the last
		// place, which is far below anything a terminal can show
		if !near(got, want) {
			t.Errorf("%q: the document drew %v, and the same "+
				"colour through paint is %v", c.written, got,
				want)
		}
	}
}

// near is whether two colours are the same to a part in a billion.
func near(a, b light.RGB) bool {
	return math.Abs(a.R-b.R) < 1e-9 && math.Abs(a.G-b.G) < 1e-9 &&
		math.Abs(a.B-b.B) < 1e-9
}
