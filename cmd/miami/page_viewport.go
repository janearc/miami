package main

import (
	"time"

	"miami/cells"
	"miami/compose"
	"miami/motion"
	"miami/view"
)

// viewport: one sprite, three distances. Nothing here resizes anything; the
// objects are asked what they look like and they answer smaller.
func viewportPage() ([][]cells.Cell, string) {
	frames, err := motion.ReadStream("sprites/made/starling-flight-full.stream")
	if err != nil {
		return nil, "no starling: " + err.Error()
	}
	col, err := compose.Colours("sprites/made/starling.css")
	if err != nil {
		return nil, err.Error()
	}
	const cols, rows = 66, 11
	px := blank(cols*2, rows*3)
	objs := []view.Object{
		view.Billboard{Frames: frames, At: view.Vec{X: -0.46, Y: -0.02, Z: 1.5}, Height: 1.05},
		view.Billboard{Frames: frames, At: view.Vec{X: 0.06, Y: 0.02, Z: 2.5}, Height: 1.05,
			Phase: 190 * time.Millisecond},
		view.Billboard{Frames: frames, At: view.Vec{X: 0.52, Y: 0.0, Z: 3.7}, Height: 1.05,
			Phase: 370 * time.Millisecond},
	}
	for _, p := range view.Resolve(objs, view.DefaultEye(), 0, cols*2, rows*3) {
		compose.Blit(px, p.Frame, col, p.X, p.Y)
	}
	return cells.ToCells(px, 2, 3), `
One sprite, three distances, and nothing in the code resized anything.

An object has one job: say what it looks like from here, at the size asked
for. A billboard answers by scaling the frame it was drawn at. A sprite drawn
from eight sides answers by picking the one that faces you. A ball answers by
solving its own surface, which is exactly the ray-sphere intersection and is
why tracing is cheaper here than in a scene made of triangles.

Then: sort far to near, blit, and alpha does the occlusion. That is the whole
of it. Apparent size is size over distance, and apparent speed falls out of
the same division -- nothing tells a distant bird to move slower.

Each one keeps its own clock. The phases above are offset by a fraction of a
second, because a flock beating its wings in unison is the single thing that
gives away that they are all the same animation.`
}
