package view

// What a shape has to be able to say for a ray to find it.
//
// The tracer used to know two shapes by name -- a ball and a floor -- and asked
// each the questions it happened to need. That works for exactly those two, and
// nothing else can ever be added without editing the loop that draws them.
//
// A shape is not much: it can say where a ray meets it and what its surface is
// there, and whether it is in the way of a light. Those are one question asked
// twice, and they are the whole interface.
//
// Everything drawn in space answers them -- a sphere, a cylinder, a floor, and
// whatever is wanted next -- and the tracer no longer knows what it has hit.

// Solid is a shape a ray can meet.
type Solid interface {
	// Hit is the nearest distance along the ray at which this shape is met
	// between near and far, and the role of its surface at that point.
	//
	// The role rather than a colour, because the picture is a stencil all
	// the way down: a colourway dresses it afterwards, and a shape that
	// returned a colour would be the one thing in the library a theme could
	// not touch.
	//
	// The seconds are for the surface and not for the shape. Where a thing
	// is at a moment is set by whoever is moving it, before the frame is
	// drawn; what a spinning ball is showing at a moment is the shape's own
	// business, and this is how it knows.
	Hit(r Ray, t float64, near, far float64) (
		dist float64, role byte, ok bool,
	)

	// Blocks says whether this shape stands in the segment of the ray
	// between near and far. It is the shadow question, and it is separate
	// from Hit because it can stop at the first answer: what a shadow ray
	// meets first does not matter, only that it met something.
	Blocks(r Ray, near, far float64) bool
}

// Shader is a solid whose surface goes darker where a light cannot reach it.
//
// Optional, and most shapes do not want it. A sphere lit this way looks wrong:
// the shadow terminator lands on a surface with no shading of its own, so the
// ball reads as a flat disc with a bite out of it. A floor is the surface a
// shadow belongs on, and it is the one that implements this.
type Shader interface {
	Darken(role byte, blocked int) byte
}
