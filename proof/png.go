package proof

import (
	"image"
	"image/color"
	"image/png"
	"miami/raster"
	"os"
)

// SaveImage writes an image as it is.
func SaveImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// WritePNG blows the pixel field up by an integer factor so a low resolution
// picture can actually be looked at.
func WritePNG(path string, rgb [][]color.RGBA, scale int) error {
	return WritePNGXY(path, rgb, scale, scale)
}

// WritePNGXY scales the axes independently, which is how a terminal render is
// previewed honestly: its subpixels are not square.
func WritePNGXY(path string, rgb [][]color.RGBA, sx, sy int) error {
	h := len(rgb)
	w := len(rgb[0])
	img := image.NewRGBA(image.Rect(0, 0, w*sx, h*sy))
	for y := 0; y < h*sy; y++ {
		for x := 0; x < w*sx; x++ {
			img.Set(x, y, rgb[y/sy][x/sx])
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Silhouette is step 5: everything solid black on white. if it is not
// recognisable here the proportions are wrong and colour will not save it.
func Silhouette(g *raster.Grid, ignore ...byte) [][]color.RGBA {
	skip := map[byte]bool{raster.Empty: true}
	for _, b := range ignore {
		skip[b] = true
	}
	out := make([][]color.RGBA, g.H)
	for y := 0; y < g.H; y++ {
		row := make([]color.RGBA, g.W)
		for x := 0; x < g.W; x++ {
			if skip[g.Rows[y][x]] {
				row[x] = color.RGBA{255, 255, 255, 255}
			} else {
				row[x] = color.RGBA{0, 0, 0, 255}
			}
		}
		out[y] = row
	}
	return out
}
