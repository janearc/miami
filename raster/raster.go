package raster

import (
	"image/color"
	"miami/canvas"
	"miami/geom"
)

// a Part is a region of the picture that means something anatomically. role
// codes are parts, never colours. colour is a later lookup.
type Part struct {
	Role  byte
	Shape geom.Shape
}

type Scene struct {
	W, H  float64 // canvas extent in square units
	Parts []Part  // painted back to front
	Pal   map[byte]color.RGBA
	Bg    func(x, y float64) color.RGBA // resolves the empty role

	// MinCover lets a role win a sample at partial coverage instead of by
	// majority. a hairline stroke rarely holds most of a subcell, so
	// without this the delicate parts of a letter drop out. nil keeps pure
	// majority.
	MinCover map[byte]float64

	// Shade runs after the palette and background have given a sample its
	// colour. it sees the role and the PHYSICAL position, so it can lay a
	// gradient on a part or light spilling across everything. nil does
	// nothing.
	Shade func(role byte, x, y float64, c color.RGBA) color.RGBA

	// Tone is the palette for renders with no colour: a lightness from 0 to
	// 1 per role, and ToneBg for the empty role in physical units. a one
	// colour medium needs its own choices -- mapping colour lightness
	// straight across turned a faint glow into a solid grey disc.
	//
	// nil falls back to lightness.
	Tone   map[byte]float64
	ToneBg func(x, y float64) float64
}

// ToneField resolves every sample to a lightness for plain text renders. ok is
// false when the scene has no tone palette of its own.
func ToneField(g *Grid, sc *Scene) (f [][]float64, ok bool) {
	if sc.Tone == nil {
		return nil, false
	}
	f = make([][]float64, g.H)
	for y := 0; y < g.H; y++ {
		row := make([]float64, g.W)
		for x := 0; x < g.W; x++ {
			if v, has := sc.Tone[g.Rows[y][x]]; has {
				row[x] = v
				continue
			}
			if sc.ToneBg != nil {
				row[x] = sc.ToneBg((float64(x)+0.5)/float64(g.W)*sc.W, (float64(y)+0.5)/float64(g.H)*sc.H)
			}
		}
		f[y] = row
	}
	return f, true
}

// Empty is a space, so a role dump reads as the picture's own negative space
// rather than as a field of dots.
const Empty = ' '

// Grid is a rectangular field of role codes plus the canvas mapping that
// produced it, so downstream stages can still ask where a sample came from.
type Grid struct {
	W, H int
	Rows [][]byte
	// canvas coordinate of sample (x,y)
	SX, SY float64 // scale: canvas units per sample
}

// At is the role at a pixel, or Empty outside the grid.
func (g *Grid) At(x, y int) byte {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return Empty
	}
	return g.Rows[y][x]
}

// Set writes a role at a pixel, and ignores anything outside the grid.
func (g *Grid) Set(x, y int, r byte) {
	if x < 0 || y < 0 || x >= g.W || y >= g.H {
		return
	}
	g.Rows[y][x] = r
}

// Rasterize samples the scene into a gw x gh field of role codes. ss is the
// supersample factor per axis; the winning role is the one holding the most
// subsamples, which keeps boundaries crisp but subpixel-accurate.
func Rasterize(sc *Scene, gw, gh, ss int) *Grid {
	g := &Grid{W: gw, H: gh, SX: sc.W / float64(gw), SY: sc.H / float64(gh)}
	g.Rows = make([][]byte, gh)
	var tally [256]int
	for y := 0; y < gh; y++ {
		row := make([]byte, gw)
		for x := 0; x < gw; x++ {
			for i := range tally {
				tally[i] = 0
			}
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := (float64(x) + (float64(sx)+0.5)/float64(ss)) * g.SX
					py := (float64(y) + (float64(sy)+0.5)/float64(ss)) * g.SY
					r := byte(Empty)
					p := geom.Vec{X: px, Y: py}
					for i := len(sc.Parts) - 1; i >= 0; i-- {
						if sc.Parts[i].Shape.In(p) {
							r = sc.Parts[i].Role
							break
						}
					}
					tally[r]++
				}
			}
			best, bn := byte(Empty), -1
			for r, n := range tally {
				if n > bn {
					best, bn = byte(r), n
				}
			}
			if sc.MinCover != nil {
				total := float64(ss * ss)
				pn := 0
				for r := 0; r < 256; r++ {
					f, ok := sc.MinCover[byte(r)]
					if ok && tally[r] > pn && float64(tally[r]) >= f*total {
						best, pn = byte(r), tally[r]
					}
				}
			}
			row[x] = best
		}
		g.Rows[y] = row
	}
	return g
}

// Contour marks empty samples that touch the silhouette, giving the figure a
// dark edge so it survives being laid over a bright background.
func Contour(g *Grid, role byte, thick int) {
	for t := 0; t < thick; t++ {
		add := [][2]int{}
		for y := 0; y < g.H; y++ {
			for x := 0; x < g.W; x++ {
				if g.At(x, y) != Empty {
					continue
				}
				hit := false
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					n := g.At(x+d[0], y+d[1])
					if n != Empty && n != role {
						hit = true
					}
				}
				if hit {
					add = append(add, [2]int{x, y})
				}
			}
		}
		for _, p := range add {
			g.Set(p[0], p[1], role)
		}
	}
}

// Resolve turns role codes into RGB, consulting the scene background for empty
// samples. ox,oy is the grid's offset within a larger composite so the
// background gradient stays continuous.
func Resolve(g *Grid, sc *Scene, ox, oy int, gw, gh int) [][]color.RGBA {
	out := make([][]color.RGBA, g.H)
	for y := 0; y < g.H; y++ {
		row := make([]color.RGBA, g.W)
		for x := 0; x < g.W; x++ {
			r := g.Rows[y][x]
			fx := float64(x+ox) / float64(gw)
			fy := float64(y+oy) / float64(gh)
			c, ok := sc.Pal[r]
			if !ok || r == Empty {
				c = sc.Bg(fx, fy)
			}
			if sc.Shade != nil {
				// sample centre, in physical units
				px := (float64(x+ox) + 0.5) / float64(gw) * sc.W
				py := (float64(y+oy) + 0.5) / float64(gh) * sc.H
				c = sc.Shade(r, px, py, c)
			}
			row[x] = c
		}
		out[y] = row
	}
	return out
}

// Drop removes parts by role, so a sub-picture can shed scenery the composite
// supplies for itself.
func (s *Scene) Drop(roles ...byte) {
	kill := map[byte]bool{}
	for _, r := range roles {
		kill[r] = true
	}
	var keep []Part
	for _, p := range s.Parts {
		if !kill[p.Role] {
			keep = append(keep, p)
		}
	}
	s.Parts = keep
}

// Blit copies non-empty samples of src into dst at a subsample offset.
func Blit(dst, src *Grid, ox, oy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			if r := src.Rows[y][x]; r != Empty {
				dst.Set(x+ox, y+oy, r)
			}
		}
	}
}

// NewGrid returns an empty field.
func NewGrid(w, h int) *Grid {
	g := &Grid{W: w, H: h, Rows: make([][]byte, h)}
	for y := range g.Rows {
		row := make([]byte, w)
		for x := range row {
			row[x] = Empty
		}
		g.Rows[y] = row
	}
	return g
}

// RoleText is the diffable review artefact: one character per sample.
func RoleText(g *Grid) string {
	s := make([]byte, 0, (g.W+1)*g.H)
	for y := 0; y < g.H; y++ {
		s = append(s, g.Rows[y]...)
		s = append(s, '\n')
	}
	return string(s)
}

// TermRows gives the row count that keeps the scene undistorted.
func TermRows(sc *Scene, cols int) int { return canvas.Rows(sc.W, sc.H, cols) }
