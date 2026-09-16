package scenes

import (
	"bytes"
	"os"
	"testing"

	"miami/motion"
)

// the library takes the sprite directory as an argument, so the test can point
// at the repository's own without a chdir
const made = "../sprites/made/"

// The streams in sprites/made are golden. They were first produced by a
// throwaway python pipeline and then this library was written to replace it;
// the only test that a port is a port is that it emits the same bytes, so
// that is the test. It also holds the library honest afterwards: the vote
// tie-break, the alpha quantisation and the blink's fill role are all choices
// with no single right answer, and this is what stops one of them drifting
// silently the next time somebody tidies a loop.
func TestStreamsMatchGolden(t *testing.T) {
	runs, err := Animations(made)
	if err != nil {
		t.Fatal(err)
	}
	lit, err := Lit(made, made+"starling.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range append(runs, lit...) {
		t.Run(r.name, func(t *testing.T) {
			var got bytes.Buffer
			if err := motion.WriteStream(&got, r.name, r.note, r.frames); err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(made + r.name + ".stream")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Bytes(), want) {
				t.Errorf("%s differs from the golden stream (%d bytes vs %d)",
					r.name, got.Len(), len(want))
			}
		})
	}
}

// A stencil that carries no eye is returned unchanged rather than erroring: a
// library of effects that refuses half its objects is unusable.
func TestBlinkWithoutAnEye(t *testing.T) {
	f, err := motion.ReadSprite(made + "daffy.sprite")
	if err != nil {
		t.Fatal(err)
	}
	if got := motion.Blink(f, 1, "", ""); got.Art[0] != f.Art[0] || len(got.Art) != len(f.Art) {
		t.Errorf("blink changed a stencil with no eye roles")
	}
}

// An integer translation must be lossless, because that is the property the
// kingfisher's dithered crown depends on.
func TestIntegerTranslateIsLossless(t *testing.T) {
	f, err := motion.ReadSprite(made + "kingfisher.sprite")
	if err != nil {
		t.Fatal(err)
	}
	down := motion.Warp(f, motion.Translate(0, 3), motion.All)
	back := motion.Warp(down, motion.Translate(0, -3), motion.All)
	for y := 0; y < f.Height()-3; y++ {
		if back.Art[y] != f.Art[y] {
			t.Fatalf("row %d changed under a whole-pixel round trip", y)
		}
	}
}
