package typeset

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/janearc/miami/geom"
)

// a Face is an outline font we did not draw. letters come from a type designer;
// this package only sizes, spaces and places them.
type Face struct {
	Name string
	f    *sfnt.Font
	buf  sfnt.Buffer
	// everything is loaded at one large ppem and scaled in float, so
	// outline precision never depends on the size a letter is finally drawn
	// at.
	capAt float64
}

const loadPPEM = 1000

var ppem = fixed.I(loadPPEM)

// f26 is a font's fixed-point number as an ordinary one.
func f26(v fixed.Int26_6) float64 { return float64(v) / 64 }

// Faces lists the full names inside a font file or collection.
func Faces(path string) ([]string, error) {
	fonts, err := open(path)
	if err != nil {
		return nil, err
	}
	var b sfnt.Buffer
	var out []string
	for _, f := range fonts {
		n, _ := f.Name(&b, sfnt.NameIDFull)
		out = append(out, n)
	}
	return out, nil
}

// open reads every font in a file, a collection being several in one.
func open(path string) ([]*sfnt.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if c, err := sfnt.ParseCollection(data); err == nil {
		var out []*sfnt.Font
		for i := 0; i < c.NumFonts(); i++ {
			f, err := c.Font(i)
			if err != nil {
				return nil, err
			}
			out = append(out, f)
		}
		return out, nil
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	return []*sfnt.Font{f}, nil
}

// LoadFace finds a face by its full name, e.g. "Copperplate Light".
func LoadFace(path, name string) (*Face, error) {
	fonts, err := open(path)
	if err != nil {
		return nil, err
	}
	var b sfnt.Buffer
	var seen []string
	for _, f := range fonts {
		n, _ := f.Name(&b, sfnt.NameIDFull)
		seen = append(seen, n)
		if !strings.EqualFold(n, name) {
			continue
		}
		m, err := f.Metrics(&b, ppem, font.HintingNone)
		if err != nil {
			return nil, err
		}
		cap := f26(m.CapHeight)
		if cap <= 0 {
			// some faces leave the OS/2 cap height empty; measure
			// an H instead
			if p, _, err := (&Face{f: f, capAt: 1}).raw('H'); err == nil {
				lo, hi := p.Bounds()
				cap = hi.Y - lo.Y
			}
		}
		return &Face{Name: n, f: f, capAt: cap}, nil
	}
	return nil, fmt.Errorf("face %q not in %s; have %s", name, path, strings.Join(seen, ", "))
}

// raw returns a glyph outline at load size, origin on the baseline, y down.
func (fc *Face) raw(r rune) (geom.Path, float64, error) {
	gi, err := fc.f.GlyphIndex(&fc.buf, r)
	if err != nil || gi == 0 {
		return geom.Path{}, 0, fmt.Errorf("no glyph for %q", r)
	}
	segs, err := fc.f.LoadGlyph(&fc.buf, gi, ppem, nil)
	if err != nil {
		return geom.Path{}, 0, err
	}
	adv, err := fc.f.GlyphAdvance(&fc.buf, gi, ppem, font.HintingNone)
	if err != nil {
		return geom.Path{}, 0, err
	}
	var contours [][]geom.Vec
	var cur []geom.Vec
	pt := func(p fixed.Point26_6) geom.Vec { return geom.Vec{X: f26(p.X), Y: f26(p.Y)} }
	last := geom.Vec{}
	const steps = 10
	for _, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			if len(cur) > 2 {
				contours = append(contours, cur)
			}
			last = pt(s.Args[0])
			cur = []geom.Vec{last}
		case sfnt.SegmentOpLineTo:
			last = pt(s.Args[0])
			cur = append(cur, last)
		case sfnt.SegmentOpQuadTo:
			c, e := pt(s.Args[0]), pt(s.Args[1])
			for i := 1; i <= steps; i++ {
				t := float64(i) / steps
				u := 1 - t
				cur = append(cur, geom.Vec{X: u*u*last.X + 2*u*t*c.X + t*t*e.X, Y: u*u*last.Y + 2*u*t*c.Y + t*t*e.Y})
			}
			last = e
		case sfnt.SegmentOpCubeTo:
			c1, c2, e := pt(s.Args[0]), pt(s.Args[1]), pt(s.Args[2])
			for i := 1; i <= steps; i++ {
				t := float64(i) / steps
				u := 1 - t
				cur = append(cur, geom.Vec{X: u*u*u*last.X + 3*u*u*t*c1.X + 3*u*t*t*c2.X + t*t*t*e.X, Y: u*u*u*last.Y + 3*u*u*t*c1.Y + 3*u*t*t*c2.Y + t*t*t*e.Y})
			}
			last = e
		}
	}
	if len(cur) > 2 {
		contours = append(contours, cur)
	}
	return geom.NewPath(contours), f26(adv), nil
}

// Glyph returns a letter scaled so its capitals are capHeight tall, placed with
// its pen origin at pen (on the baseline), and its advance at that size.
func (fc *Face) Glyph(r rune, pen geom.Vec, capHeight float64) (geom.Path, float64, error) {
	p, adv, err := fc.raw(r)
	if err != nil {
		return p, 0, err
	}
	k := capHeight / fc.capAt
	m := geom.Affine{A: k, E: k}.Then(geom.Translate(pen.X, pen.Y))
	return p.Transform(m), adv * k, nil
}

// Kern is the pair adjustment between two letters at a cap height.
func (fc *Face) Kern(a, b rune, capHeight float64) float64 {
	ga, _ := fc.f.GlyphIndex(&fc.buf, a)
	gb, _ := fc.f.GlyphIndex(&fc.buf, b)
	k, err := fc.f.Kern(&fc.buf, ga, gb, ppem, font.HintingNone)
	if err != nil {
		return 0
	}
	return f26(k) * capHeight / fc.capAt
}
