package canvas

// cells are not square. everything that maps physical space onto terminal cells
// reads this constant and nothing else.

// CellAspect is the height of a terminal cell divided by its width. THIS IS THE
// ONLY PLACE THE TERMINAL'S NON-SQUARE PIXEL IS HANDLED.
const CellAspect = 2.0

// Rows gives the row count that keeps a picture of physical size w by h
// undistorted at the requested column count.
func Rows(w, h float64, cols int) int {
	r := float64(cols) * h / (w * CellAspect)
	return int(r + 0.5)
}

// Canvas is a page with a locked width and, if Rows is set, a locked height.
// with Rows at zero the scene decides how tall it is and the page descends as
// far as that takes it.
type Canvas struct {
	Name string
	Cols int
	Rows int
}

// Manpage is the libreadme page: the header of its LICENSE.txt is exactly 77
// columns and the body indents five.
var Manpage = Canvas{Name: "manpage", Cols: 77}

// W is the physical width: one unit per cell.
func (c Canvas) W() float64 { return float64(c.Cols) }

// RowsFor is how many rows a physical height needs.
func (c Canvas) RowsFor(h float64) int {
	r := int(h / CellAspect)
	if float64(r)*CellAspect < h {
		r++
	}
	return r
}

// RowsToPhys is the physical height of n rows.
func (c Canvas) RowsToPhys(n int) float64 { return float64(n) * CellAspect }
