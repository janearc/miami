package scene

import "fmt"

// MainCamera is the camera a document has whether or not it says so: the
// one its "eye" describes, or the default when it describes none.
const MainCamera = "main"

// Camera is a named eye and an exposure, as a document says them.
//
// The eye's fields sit beside the name rather than under a key of their
// own, so a camera reads the way "eye" always has, with two words added.
//
// Exposure is a binding, like everything else that may move with the music.
// It is a pointer so that saying nothing and saying nought are different
// documents, and the second one means a black frame.
//
// Layer.Gain met the same trap first: a number that reached nought
// legitimately and was read as absent.
type Camera struct {
	Name string `json:"name"`
	Eye
	Exposure *Value `json:"exposure"`

	// Default says the document described no camera at all, so the
	// stage should use its own rather than an eye made of zeroes.
	Default bool `json:"-"`
}

// Viewport is a camera and where on the sheet its picture goes.
//
// In fractions of the sheet, because a terminal is resized and a document
// is not rewritten when it is. The stage turns them into whole cells.
type Viewport struct {
	Camera string  `json:"camera"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	W      float64 `json:"w"`
	H      float64 `json:"h"`
}

// slack is how far outside the sheet a viewport may reach before it is
// refused. Thirds do not add up to one in binary, and a document that says
// 0.333, 0.333, 0.334 should not be told it is wrong.
const slack = 1e-9

// Shots is the document's cameras and viewports with what it left out
// filled in. It is the one place a document from before cameras becomes one
// that has them, so nothing downstream asks whether "eye" was used.
//
// "eye" is the camera called main, for ever. A document may add cameras
// beside it but may not name another one main, because then which of the
// two a viewport meant would depend on the order they were read in.
//
// With no viewports the first camera fills the sheet. Viewports may overlap
// and are drawn in the order written, later over earlier: a picture in a
// corner of another is two viewports, not a third kind of thing.
func (d *Doc) Shots() ([]Camera, []Viewport, error) {
	var cams []Camera
	seen := map[string]bool{}
	if d.Eye != nil {
		cams = append(cams, Camera{Name: MainCamera, Eye: *d.Eye})
		seen[MainCamera] = true
	}
	for i, c := range d.Cameras {
		switch {
		case c.Name == "":
			return nil, nil, fmt.Errorf("camera %d has no name", i)
		case seen[c.Name] && c.Name == MainCamera && d.Eye != nil:
			return nil, nil, fmt.Errorf(
				"camera %d is called %q, which "+
					"\"eye\" already is", i, c.Name)
		case seen[c.Name]:
			return nil, nil, fmt.Errorf(
				"two cameras are called %q", c.Name)
		}
		seen[c.Name] = true
		cams = append(cams, c)
	}
	if len(cams) == 0 {
		cams = []Camera{{Name: MainCamera, Default: true}}
		seen[MainCamera] = true
	}

	if len(d.Viewports) == 0 {
		full := Viewport{Camera: cams[0].Name, W: 1, H: 1}
		return cams, []Viewport{full}, nil
	}
	for i, v := range d.Viewports {
		if !seen[v.Camera] {
			return nil, nil, fmt.Errorf(
				"viewport %d wants camera %q, which is not one",
				i, v.Camera)
		}
		if v.W <= 0 || v.H <= 0 {
			return nil, nil, fmt.Errorf(
				"viewport %d is %g by %g; both must be "+
					"more than zero", i, v.W, v.H)
		}
		if v.X < 0 || v.Y < 0 ||
			v.X+v.W > 1+slack || v.Y+v.H > 1+slack {
			return nil, nil, fmt.Errorf(
				"viewport %d reaches outside the sheet: "+
					"x %g y %g w %g h %g",
				i, v.X, v.Y, v.W, v.H)
		}
	}
	return cams, d.Viewports, nil
}
