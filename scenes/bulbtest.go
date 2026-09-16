package scenes

import (
	"image/color"
	"math"

	"github.com/janearc/miami/motifs"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/raster"
)

// BulbTest is the bulb on its own, at the size it will be used, so it can pass
// the silhouette test before it meets a sign.
func BulbTest(cols int, h float64) *raster.Scene {
	u := float64(cols)
	b := motifs.NewBulb(u/2, 0.30*h, 0.62*h)
	burst := motifs.Sunburst(b.Centre, b.R*1.42, []float64{b.R * 2.25, b.R * 1.62, b.R * 1.9, b.R * 1.62}, 28, -208, 28, b.R*0.07)

	sc := &raster.Scene{W: u, H: h}
	sc.Parts = []raster.Part{
		{Role: 'r', Shape: burst},
		{Role: 'G', Shape: b.Glass},
		{Role: 'w', Shape: b.Wires},
		{Role: 'h', Shape: b.Highlight},
		{Role: 'f', Shape: b.Filament},
		{Role: 'S', Shape: b.Screw},
		{Role: 'g', Shape: b.Grooves},
		{Role: 'I', Shape: b.Insulator},
		{Role: 'T', Shape: b.Tip}}
	sc.Pal = map[byte]color.RGBA{
		'r': {246, 206, 128, 255},
		'G': {255, 240, 190, 255},
		'w': {214, 150, 60, 255},
		'h': {255, 255, 250, 255},
		'f': {255, 255, 255, 255},
		'S': {178, 168, 150, 255},
		'g': {92, 84, 80, 255},
		'I': {40, 34, 40, 255},
		'T': {120, 110, 100, 255},
	}
	sc.MinCover = map[byte]float64{'f': 0.25, 'g': 0.30, 'w': 0.35, 'r': 0.30}
	glass := paint.Radial(b.Centre.X, b.Centre.Y-0.1*b.R, b.R*1.25, []paint.Stop{
		{At: 0.00, C: color.RGBA{255, 252, 232, 255}},
		{At: 0.55, C: color.RGBA{255, 228, 160, 255}},
		{At: 1.00, C: color.RGBA{246, 176, 84, 255}}})
	rayRamp := []paint.Stop{
		{At: 0.00, C: color.RGBA{255, 232, 170, 255}},
		{At: 1.00, C: color.RGBA{226, 144, 78, 255}}}
	sky := []paint.Stop{
		{At: 0.00, C: color.RGBA{16, 16, 44, 255}},
		{At: 1.00, C: color.RGBA{44, 30, 86, 255}}}
	sc.Bg = func(fx, fy float64) color.RGBA { return paint.Ramp(sky, fy) }
	sc.Shade = func(role byte, x, y float64, c color.RGBA) color.RGBA {
		if role == 'G' {
			c = glass(x, y)
		}
		dx, dy := x-b.Centre.X, y-b.Centre.Y
		d := math.Sqrt(dx*dx+dy*dy) / b.R
		if role == 'r' {
			// rays cool as they leave the glass, so they never
			// outshine it
			c = paint.Ramp(rayRamp, (d-1.42)/0.85)
		}
		// light spilling past the glass: strongest right at its edge so
		// the gap glows instead of reading as a shadow every non-glass
		// sample is lit, including sky samples whose centre sits just
		// inside the radius: gating on d > 1 left dark dots on the edge
		if role != 'G' && role != 'f' && role != 'h' {
			e := math.Max(d-1, 0)
			c = paint.Screen(c, color.RGBA{255, 190, 105, 255}, 0.62/(1+3.2*e*e))
		}
		return c
	}
	return sc
}
