package scenery

import (
	"sort"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// Body is a thing in space that can be drawn and can say how far away it is.
type Body interface {
	Draw(b *light.Buffer, eye view.Eye, t float64)
	Away(eye view.Eye) float64
}

// Draw puts bodies into a frame furthest first, whatever order they are given
// in. Lights -- particles, a glow, anything that adds rather than replaces --
// are drawn after it returns, never before: see light.Buffer.
//
// The order matters and cannot be left to the caller. A body's silhouette
// crosses a subpixel rather than stopping at its edge, so an edge subpixel
// holds some of the body and some of whatever is behind it.
//
// Drawn nearest first, the near body blends against a background the far body
// has not painted yet, and the far body is then kept out of the subpixel it
// should have shared.
//
// Furthest first, each body blends over what is already there and the edge
// comes out the same however the caller listed them. Which is why this takes
// the list rather than trusting it.
func Draw(b *light.Buffer, eye view.Eye, t float64, bodies ...Body) {
	order := append([]Body(nil), bodies...)
	sort.SliceStable(order, func(i, j int) bool {
		return order[i].Away(eye) > order[j].Away(eye)
	})
	for _, body := range order {
		body.Draw(b, eye, t)
	}
}
