// Package motion animates sprites.
//
// The sprite is daffy's: libdaffy-sprite owns the format, the frame type and
// the reader, and nothing here redefines any of them. A Frame is rows of role
// characters and how long they hold, and that is the currency throughout.
//
// What this package adds is the half daffy does not do, because daffy converges
// on stillness: turning one frame into a run of them.
//
// Every operator here is a function on a frame.
//
// There is no scene graph, no timeline object and no animation type, because a
// sprite traced from a photograph has no model underneath to re-pose -- there
// is a grid of role characters and nothing else, so the only honest vocabulary
// is operations on that grid. Two kinds earn their place:
//
//	Warp   an inverse-mapped resample of a region. rotate a wing about a
//	       shoulder, pass a wave down a body, shift a foot. supersampled
//	       and majority-voted per pixel, the same way a cell is decided,
//	       because a nearest-neighbour rotation of a one-pixel feather
//	       tears it into dashes.
//	Role   an edit addressed by what a pixel is, never by where it is.
//	       Blink closes whatever the pupil and eye-ring roles cover; a
//	       stencil carrying no eye gets no blink and no error.
//
// One warning worth putting at the top, because it cost a rebuild to find: a
// warp that is not an exact whole-pixel translation runs the stencil through
// the resampler. Smooth masses survive it.
//
// A dither does not -- a kingfisher's crown of gold speckle on slate turns to
// mud under a five-degree rotation, and next to an unrotated frame the texture
// visibly pops. Integer translations resolve to exactly one source pixel per
// subsample and are lossless; prefer them wherever the stencil is textured.
package motion

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

// None is the alphabet's transparent role.
const None = '.'

// Map is an inverse map: where an output pixel reads from.
type Map func(x, y float64) (float64, float64)

// Mask says which output pixels an operation owns. Pixels it does not own keep
// their source value, which is how one warp moves a wing and leaves the body
// alone.
type Mask func(x, y int) bool

// All owns every pixel.
func All(int, int) bool { return true }

// Box owns a rectangle.
func Box(x0, y0, x1, y1 int) Mask {
	return func(
		x, y int,
	) bool {
		return x >= x0 && x < x1 && y >= y0 && y < y1
	}
}

// Left owns the columns before x.
func Left(x int) Mask { return func(px, _ int) bool { return px < x } }

// Right owns x and the columns after it.
func Right(x int) Mask { return func(px, _ int) bool { return px >= x } }

// And is the intersection of masks.
func And(ms ...Mask) Mask {
	return func(x, y int) bool {
		for _, m := range ms {
			if !m(x, y) {
				return false
			}
		}
		return true
	}
}

// Above owns the rows up to and including y.
func Above(y int) Mask { return func(_, py int) bool { return py <= y } }

// Warp resamples a region through an inverse map.
//
// Cover is the share of subsamples that must land on ink before the output
// pixel is ink at all, and it is the one knob that matters: too high and a thin
// feather dissolves into speckle, too low and the silhouette grows a fringe.
// Sub is the supersampling; three is enough at these sizes.
type WarpOpts struct {
	Cover float64 // default 0.34
	Sub   int     // default 3
}

// Warp is the resample. The vote is by role, so a rotated pixel takes the role
// most of its subsamples saw rather than an average of two roles, which in an
// alphabet has no meaning.
func Warp(f sprite.Frame, inv Map, owns Mask, o ...WarpOpts) sprite.Frame {
	opt := WarpOpts{Cover: 0.34, Sub: 3}
	if len(o) > 0 {
		if o[0].Cover > 0 {
			opt.Cover = o[0].Cover
		}
		if o[0].Sub > 0 {
			opt.Sub = o[0].Sub
		}
	}
	h, w := f.Height(), f.Width()
	out := make([]string, h)
	order := make([]byte, 0, opt.Sub*opt.Sub)
	step := 1 / float64(opt.Sub)
	need := opt.Cover * float64(opt.Sub*opt.Sub)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		for x := 0; x < w; x++ {
			if !owns(x, y) {
				row[x] = f.Art[y][x]
				continue
			}
			var votes [256]int
			seen := order[:0]
			hits := 0
			fx, fy := float64(x), float64(y)
			for sy := 0; sy < opt.Sub; sy++ {
				for sx := 0; sx < opt.Sub; sx++ {
					px := fx + (float64(sx)+0.5)*step - 0.5
					py := fy + (float64(sy)+0.5)*step - 0.5
					ux, uy := inv(px, py)
					c := At(
						f, int(math.Round(ux)),
						int(math.Round(uy)),
					)
					if c == None {
						continue
					}
					hits++
					if votes[c] == 0 {
						seen = append(seen, c)
					}
					votes[c]++
				}
			}
			// ties go to the role seen first, scanning the
			// subsamples in order. two roles with equal support is
			// a genuine coin toss, so what matters is not which
			// rule but that there is one:
			//
			// an unstated tie-break makes a resample depend on
			// iteration order and the same input stops giving the
			// same bytes.
			best, bestN := byte(None), 0
			for _, c := range seen {
				if votes[c] > bestN {
					best, bestN = c, votes[c]
				}
				votes[c] = 0
			}
			if float64(hits) >= need {
				row[x] = best
			} else {
				row[x] = None
			}
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out, Hold: f.Hold}
}

// At is the role at a pixel, or None outside the frame.
func At(f sprite.Frame, x, y int) byte {
	if y < 0 || y >= len(f.Art) || x < 0 || x >= len(f.Art[y]) {
		return None
	}
	return f.Art[y][x]
}

// Translate moves by a fixed offset. With whole-number arguments it is
// lossless: every subsample of an output pixel rounds to the same source pixel,
// so a dithered stencil comes through untouched.
func Translate(dx, dy float64) Map {
	return func(x, y float64) (float64, float64) { return x - dx, y - dy }
}

// Rotate turns about a pivot, optionally scaling first. Scaling x below one is
// the foreshortening a limb shows as it turns out of the plane of the picture;
// without it a rotation reads as a propeller rather than a wing.
func Rotate(cx, cy, deg, sx, sy float64) Map {
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	a := deg * math.Pi / -180
	ca, sa := math.Cos(a), math.Sin(a)
	return func(x, y float64) (float64, float64) {
		dx, dy := (x-cx)/sx, (y-cy)/sy
		return cx + dx*ca - dy*sa, cy + dx*sa + dy*ca
	}
}

// Bend is a rotation whose angle grows with distance from the pivot, so the tip
// turns further than the shoulder. A wing is not a plank: the hand lags the arm
// through the stroke, and that lag is most of what separates a wingbeat from a
// windmill.
//
// It inverts exactly, which is the only reason it is affordable. A rotation
// preserves radius, so the angle to undo at an output point is the same angle
// that was applied at its source.
func Bend(cx, cy, deg, span, k, sx float64) Map {
	if sx == 0 {
		sx = 1
	}
	if k == 0 {
		k = 1.6
	}
	return func(x, y float64) (float64, float64) {
		dx, dy := (x-cx)/sx, y-cy
		r := math.Hypot(dx, dy)
		a := -deg * math.Pow(math.Min(1, r/span), k) * math.Pi / 180
		ca, sa := math.Cos(a), math.Sin(a)
		return cx + dx*ca - dy*sa, cy + dx*sa + dy*ca
	}
}

// Undulate is a vertical travelling wave. A swimming animal does not flap its
// tail, it passes a wave down its body, and the amplitude grows from a head
// that barely moves to a fluke that moves a lot. Amp is that growth.
//
// X is clamped at nought because the supersampler probes half a pixel outside
// the canvas, and a negative x under a fractional power is not a real number.
func Undulate(amp func(x float64) float64, wavelength, phase float64) Map {
	return func(x, y float64) (float64, float64) {
		grow := amp(math.Max(0, x))
		dy := grow * math.Sin(2*math.Pi*x/wavelength-phase)
		return x, y - dy
	}
}

// Layer is a warped frame and the pixels it is allowed to write.
type Layer struct {
	Frame sprite.Frame
	Owns  Mask
}

// Compose paints layers over a base in order.
func Compose(base sprite.Frame, layers ...Layer) sprite.Frame {
	h, w := base.Height(), base.Width()
	out := make([][]byte, h)
	for y := 0; y < h; y++ {
		out[y] = []byte(base.Art[y])
	}
	for _, l := range layers {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if l.Owns(x, y) {
					out[y][x] = At(l.Frame, x, y)
				}
			}
		}
	}
	rows := make([]string, h)
	for y := range out {
		rows[y] = string(out[y])
	}
	return sprite.Frame{Art: rows, Hold: base.Hold}
}

// Pad grows the canvas. Which side to grow is not a detail: a stencil cropped
// by the edge of its photograph must stay cropped, or a raised limb detaches
// and floats in space where the photograph had none.
func Pad(f sprite.Frame, top, bottom, left, right int) sprite.Frame {
	w := f.Width() + left + right
	blank := strings.Repeat(string(None), w)
	out := make([]string, 0, f.Height()+top+bottom)
	for i := 0; i < top; i++ {
		out = append(out, blank)
	}
	for _, r := range f.Art {
		out = append(out, strings.Repeat(string(None), left)+r+
			strings.Repeat(string(None), right))
	}
	for i := 0; i < bottom; i++ {
		out = append(out, blank)
	}
	return sprite.Frame{Art: out, Hold: f.Hold}
}

// Pt is a pixel position.
type Pt struct{ X, Y int }

// Components is the four-connected runs of ink, largest first. A cricket ball
// is the only thing in its frame not touching the fielder, which is how a catch
// gets animated without anybody naming a pixel.
func Components(f sprite.Frame) [][]Pt {
	h, w := f.Height(), f.Width()
	seen := make([][]bool, h)
	for i := range seen {
		seen[i] = make([]bool, w)
	}
	var out [][]Pt
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if seen[y][x] || f.Art[y][x] == None {
				continue
			}
			stack, cells := []Pt{{x, y}}, []Pt(nil)
			seen[y][x] = true
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				cells = append(cells, p)
				nbrs := []Pt{
					{p.X + 1, p.Y}, {p.X - 1, p.Y},
					{p.X, p.Y + 1}, {p.X, p.Y - 1},
				}
				for _, n := range nbrs {
					inside := n.X >= 0 && n.X < w &&
						n.Y >= 0 && n.Y < h
					if !inside || seen[n.Y][n.X] {
						continue
					}
					if f.Art[n.Y][n.X] == None {
						continue
					}
					seen[n.Y][n.X] = true
					stack = append(stack, n)
				}
			}
			out = append(out, cells)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && len(out[j]) > len(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Piece is a component taken out of a frame, with its roles.
type Piece []struct {
	Pt
	Role byte
}

// Lift removes a component and hands it back as its own little sprite, so it
// can be moved independently of what it was lying on.
func Lift(f sprite.Frame, cells []Pt) (sprite.Frame, Piece) {
	rows := make([][]byte, f.Height())
	for y := range rows {
		rows[y] = []byte(f.Art[y])
	}
	var p Piece
	for _, c := range cells {
		p = append(p, struct {
			Pt
			Role byte
		}{c, rows[c.Y][c.X]})
		rows[c.Y][c.X] = None
	}
	art := make([]string, len(rows))
	for y := range rows {
		art[y] = string(rows[y])
	}
	return sprite.Frame{Art: art, Hold: f.Hold}, p
}

// Stamp draws a piece at an offset.
func Stamp(f sprite.Frame, p Piece, dx, dy int) sprite.Frame {
	h, w := f.Height(), f.Width()
	rows := make([][]byte, h)
	for y := range rows {
		rows[y] = []byte(f.Art[y])
	}
	for _, c := range p {
		x, y := c.X+dx, c.Y+dy
		if x >= 0 && x < w && y >= 0 && y < h {
			rows[y][x] = c.Role
		}
	}
	art := make([]string, h)
	for y := range rows {
		art[y] = string(rows[y])
	}
	return sprite.Frame{Art: art, Hold: f.Hold}
}

// Blink closes the lid over whatever the pupil and eye-ring roles cover, from
// the top down, filling with whatever role surrounds the eye.
//
// No subject is named. The eye is wherever those roles are, which is the whole
// point of having an alphabet:
//
// one cadence drives any stencil that carries an eye, and a stencil that
// carries none is returned unchanged rather than erroring, because a library of
// effects that refuses half its objects is unusable.
func Blink(f sprite.Frame, shut float64, eye, crease string) sprite.Frame {
	if eye == "" {
		eye = "EX"
	}
	if crease == "" {
		crease = "D"
	}
	if shut <= 0 {
		return f
	}
	x0, y0, x1, y1 := f.Width(), f.Height(), -1, -1
	for y, r := range f.Art {
		for x := 0; x < len(r); x++ {
			if strings.IndexByte(eye, r[x]) >= 0 {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	if x1 < 0 {
		return f
	}

	// the fill is the commonest role in the ring just outside the eye, so a
	// closed lid is made of face rather than of a guess
	var count [256]int
	fill, fillN := byte(None), 0
	for y := y0 - 1; y <= y1+1; y++ {
		for x := x0 - 1; x <= x1+1; x++ {
			if x >= x0 && x <= x1 && y >= y0 && y <= y1 {
				continue
			}
			c := At(f, x, y)
			if c == None || strings.IndexByte(eye, c) >= 0 {
				continue
			}
			count[c]++
			if count[c] > fillN {
				fill, fillN = c, count[c]
			}
		}
	}
	if fillN == 0 {
		return f
	}
	// do not introduce a role the stencil does not already use: a sprite's
	// set of roles is what a colourway has to bind, and a lid is not worth
	// adding a twelfth colour for.
	//
	// if the crease role is absent, the rarest role around the eye stands
	// in, which on a puffin's white face is the dark of the eye patch.
	if !strings.ContainsRune(strings.Join(f.Art, ""), rune(crease[0])) {
		rarest, rarestN := byte(0), 1<<30
		for c, n := range count {
			if n > 0 && n < rarestN {
				rarest, rarestN = byte(c), n
			}
		}
		if rarestN < 1<<30 {
			crease = string(rarest)
		}
	}

	rows := make([][]byte, f.Height())
	for y := range rows {
		rows[y] = []byte(f.Art[y])
	}
	h := y1 - y0 + 1
	lid := min(h, int(math.Round(shut*float64(h))))
	for y := y0; y < y0+lid; y++ {
		for x := x0; x <= x1; x++ {
			if strings.IndexByte(eye, rows[y][x]) >= 0 {
				rows[y][x] = fill
			}
		}
	}
	if lid >= h {
		// shut: a lid is a crease, not a blank patch of cheek
		for x := x0; x <= x1; x++ {
			rows[y1][x] = crease[0]
		}
	}
	art := make([]string, len(rows))
	for y := range rows {
		art[y] = string(rows[y])
	}
	return sprite.Frame{Art: art, Hold: f.Hold}
}

// ReadSprite reads a one-frame .sprite or the first frame of a stream, using
// daffy's own reader.
//
// A .sprite is not quite a stream: the stream grammar wants a "frame" line
// before its art and a .sprite has none, so daffy's reader refuses one with
// "art before any frame line". That is the same refusal that stops the player
// opening a .sprite directly.
//
// Rather than write a second reader for a format daffy owns, the missing line
// is supplied on the way in.
func ReadSprite(path string) (sprite.Frame, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return sprite.Frame{}, err
	}
	text := string(src)
	if !hasFrameLine(text) {
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			if strings.TrimSpace(l) == "art" {
				rest := append([]string{"frame"}, lines[i:]...)
				lines = append(lines[:i], rest...)
				break
			}
		}
		text = strings.Join(lines, "\n")
	}
	f, err := sprite.New(strings.NewReader(text)).Next()
	if err != nil {
		return sprite.Frame{}, fmt.Errorf("%s: %w", path, err)
	}
	// a .sprite's rows may be ragged where trailing transparency was
	// trimmed; every operator here indexes by column, so square it up once,
	// on the way in
	w := 0
	for _, r := range f.Art {
		w = max(w, len(r))
	}
	for i, r := range f.Art {
		f.Art[i] = r + strings.Repeat(string(None), w-len(r))
	}
	return f, nil
}

// WriteStream writes frames in the .stream grammar.
//
// Consecutive identical frames are merged into one longer hold. The player
// sends per-frame deltas, so a repeated frame is already nothing on the wire,
// but it is also a lie in the file: it reads as two beats where the animation
// has one.
func WriteStream(w io.Writer, name, note string, frames []sprite.Frame) error {
	var merged []sprite.Frame
	for _, f := range frames {
		if n := len(merged); n > 0 && sameArt(merged[n-1], f) {
			merged[n-1].Hold += f.Hold
			continue
		}
		merged = append(merged, f)
	}
	var b strings.Builder
	if note != "" {
		b.WriteString(strings.TrimRight(note, "\n") + "\n")
	}
	fmt.Fprintf(&b, "sprite %s\n", name)
	for _, f := range merged {
		fmt.Fprintf(&b, "\nframe %s\nart\n", hold(f.Hold))
		for _, r := range f.Art {
			b.WriteString(r + "\n")
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteStreamFile is WriteStream to a path.
func WriteStreamFile(path, name, note string, frames []sprite.Frame) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := WriteStream(fh, name, note, frames); err != nil {
		fh.Close()
		return err
	}
	return fh.Close()
}

// hold is a duration as a stream writes it, seconds when it divides evenly.
func hold(d time.Duration) string {
	if d == 0 {
		d = 100 * time.Millisecond
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", int(d/time.Second))
	}
	return fmt.Sprintf("%dms", int(d/time.Millisecond))
}

// sameArt reports whether two frames hold the same picture, ignoring how long
// each one is shown.
func sameArt(a, b sprite.Frame) bool {
	if len(a.Art) != len(b.Art) {
		return false
	}
	for i := range a.Art {
		if a.Art[i] != b.Art[i] {
			return false
		}
	}
	return true
}

// Hold is a frame with a duration attached, for building a run.
func Hold(f sprite.Frame, ms int) sprite.Frame {
	f.Hold = time.Duration(ms) * time.Millisecond
	return f
}

// hasFrameLine reports whether a file already declares a frame, which is what
// separates a stream from a bare sprite.
func hasFrameLine(s string) bool {
	for _, l := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(l); t == "frame" || strings.HasPrefix(
			t, "frame ",
		) {
			return true
		}
	}
	return false
}

// ReadStream reads every frame of a stream, with its hold.
func ReadStream(path string) ([]sprite.Frame, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	s := sprite.New(fh)
	var out []sprite.Frame
	for {
		f, err := s.Next()
		if err != nil {
			if len(out) == 0 {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return out, nil
		}
		if len(f.Art) > 0 {
			out = append(out, f)
		}
	}
}
