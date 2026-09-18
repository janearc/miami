package view

// Camera is an eye, and what a photograph adds to one.
//
// An Eye is only geometry: where the picture is seen from, how long the
// lens is, which way it is turned. That is all a shape needs in order to
// be drawn, so every Draw and Render here takes an Eye and none changes.
//
// A Camera is the rest: a name a viewport can ask for it by, and how much
// of the light it lets through. Two cameras on one scene are two pictures
// of it, and nothing in the scene has to know there are two.
type Camera struct {
	Eye
	Name string
	// Exposure multiplies the light before it is brought to the screen.
	// Zero means the camera has no opinion and the scene's own is used,
	// which is what a document written before cameras gets.
	Exposure float64
}

// MainCamera is the name of the camera a document gets when it names none.
const MainCamera = "main"

// DefaultCamera is DefaultEye with a name, so a stage with no document and a
// document with no cameras are the same stage.
func DefaultCamera() Camera {
	return Camera{Eye: DefaultEye(), Name: MainCamera}
}

// Exposed is the exposure to use, given what the scene would have used.
//
// Zero cannot be asked for, because zero is how a camera says nothing. A
// camera that wants a black frame wants a small number instead; that is a
// loss, and it buys every older document working without being touched.
func (c Camera) Exposed(scene float64) float64 {
	if c.Exposure > 0 {
		return c.Exposure
	}
	if scene > 0 {
		return scene
	}
	return 1
}
