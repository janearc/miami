// Package compose puts a sprite over a painted background.
//
// This is the seam between the two halves of miami. One half paints with fields
// and ramps and knows only colour; the other moves stencils and knows only
// roles. Neither should learn the other's vocabulary, so the join is here:
//
// a colourway turns roles into colours, and the sprite's pixels are written
// into the background's subpixels before either becomes a cell.
//
// Compositing below the cell is the whole point. Fit the background to cells
// first and the bird would have to fight for two colours a cell it does not
// own; write it into the subpixels and the cell fitter sees one picture and
// gives the bird's edge the same sextant treatment it gives everything else.
//
// The two grids are not the same shape and that is not a mistake. A daffy pixel
// is one column wide and half a row tall; a sextant subcell is half a column
// wide and a third of a row tall.
//
// So one sprite pixel is two subcells across and one and a half down, and the
// half is why the mapping is done by division rather than by a scale factor.
package compose

import (
	"fmt"
	"image/color"
	"os"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// roles maps a colourway's property names onto the alphabet.
var roles = map[string]byte{
	"none": '.', "dark": 'K', "light": 'W', "wing": 'V', "stripe": 'D',
	"beak-base": 'B', "beak-band": 'Y', "beak-tip": 'R', "feet": 'O',
	"pupil": 'E', "eye-ring": 'X',
}

// Colours reads a colourway into the alphabet, using libtheme's own reader.
func Colours(path string) (map[byte]color.RGBA, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, err := ColoursFrom(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// ColoursFrom is a colourway already in hand, for a caller holding the
// stylesheet rather than a path to it.
func ColoursFrom(src []byte) (map[byte]color.RGBA, error) {
	cw := css.Read(string(src))
	out := map[byte]color.RGBA{}
	for name, r := range roles {
		sw, ok := cw.Roles.Get(name)
		if !ok {
			continue
		}
		c, _ := srgb.FromSwatch(sw)
		q := func(v float64) uint8 {
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			return uint8(v*255 + 0.5)
		}
		out[r] = color.RGBA{q(c.R), q(c.G), q(c.B), 255}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no roles of the alphabet found")
	}
	return out, nil
}

// Over writes a sprite frame into a subpixel field at a subpixel position. The
// transparent role is left alone, which is what makes a stencil a stencil.
func Over(px [][]color.RGBA, f sprite.Frame, col map[byte]color.RGBA, atX, atY int) {
	if len(px) == 0 {
		return
	}
	h, w := len(px), len(px[0])
	sh := f.Height()
	for y := 0; y < h; y++ {
		sy := (y - atY) * 2 / 3 // three subcell rows per two sprite rows
		if y-atY < 0 || sy >= sh {
			continue
		}
		row := f.Art[sy]
		for x := 0; x < w; x++ {
			sx := (x - atX) / 2 // two subcells per sprite column
			if x-atX < 0 || sx >= len(row) {
				continue
			}
			c, ok := col[row[sx]]
			if !ok || row[sx] == '.' {
				continue
			}
			px[y][x] = c
		}
	}
}

// Size is a sprite frame's size in subcells.
func Size(f sprite.Frame) (w, h int) { return f.Width() * 2, f.Height() * 3 / 2 }

// At is the frame a looping stream is showing at time t, and nothing about it
// is tied to the background's frame rate. A wingbeat has its own clock: the
// bird beats at its own pace whatever the sky is doing, which is the thing a
// compositor is for.
func At(frames []sprite.Frame, t time.Duration) sprite.Frame {
	var total time.Duration
	for _, f := range frames {
		total += hold(f)
	}
	if total <= 0 {
		return frames[0]
	}
	t %= total
	if t < 0 {
		t += total
	}
	for _, f := range frames {
		if t < hold(f) {
			return f
		}
		t -= hold(f)
	}
	return frames[len(frames)-1]
}

// hold is a frame's duration, or the default when the stream did not say.
func hold(f sprite.Frame) time.Duration {
	if f.Hold <= 0 {
		return 100 * time.Millisecond
	}
	return f.Hold
}

// Blit draws a frame one sprite pixel to one subpixel.
//
// Over is for a sprite at its natural size, where one sprite pixel covers two
// subcells across and one and a half down. A sprite that has already been
// scaled for a viewport has had that correction folded into its size, so
// applying it twice would stretch it.
//
// Two functions rather than a flag, because which one you want is a fact about
// where the frame came from.
func Blit(px [][]color.RGBA, f sprite.Frame, col map[byte]color.RGBA, atX, atY int) {
	if len(px) == 0 {
		return
	}
	h, w := len(px), len(px[0])
	for y, row := range f.Art {
		py := atY + y
		if py < 0 || py >= h {
			continue
		}
		for x := 0; x < len(row); x++ {
			pxx := atX + x
			if pxx < 0 || pxx >= w || row[x] == '.' {
				continue
			}
			if c, ok := col[row[x]]; ok {
				px[py][pxx] = c
			}
		}
	}
}
