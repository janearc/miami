package motion

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

// A sprite as a daffy document, with frames either written out or differenced.
//
// The shape follows libdaffy's own file: a "daffy" version, a name, and a
// "sprite" object. Daffy's carries one still drawing as `raw`; this adds
// `frames`, and a frame says either what it is or how it differs from the one
// before.
//
//	{"daffy":1,"name":"ball","sprite":{"name":"ball","w":24,"h":18,
//	  "frames":[{"hold":"66ms","art":["....","...."]},
//	            {"hold":"66ms","delta":[[3,5,"KKWW"]]}]}}
//
// A delta is a list of [row, column, run] -- a row, where in it to start, and
// the characters to put there. Runs rather than single cells because a change
// in a picture is almost always a span:
//
// a ball moving two columns changes two long runs per row, not forty scattered
// pixels, and a triple per pixel would cost more than the frame it replaced.
//
// The trade is not the one you would expect, and the ball proved it. A
// starling's wingbeat writes at 59 per cent of full size, because most of the
// frame is unchanged sky.
//
// A spinning checkered ball writes at 131 per cent -- *larger* -- because a
// rotating pattern changes almost every pixel, and it changes them in short
// alternating runs, so every one of them costs a [row, col, "K"] wrapper
// heavier than the character it saves.
//
// So the choice is made per frame, not per file: a frame is written whichever
// way is shorter, and a document may hold both kinds. That is what "trade
// compute for filesize" should mean -- it should never cost more.
//
// The compute is still real and not free either way: a reader cannot start at
// frame two hundred without replaying two hundred frames.

// FormatVersion matches libdaffy's, because this is meant to be the same file
// with one more key in it rather than a second format standing beside it.
const FormatVersion = 1

type doc struct {
	Daffy  int       `json:"daffy"`
	Name   string    `json:"name"`
	Sprite docSprite `json:"sprite"`
}

type docSprite struct {
	Name   string     `json:"name"`
	W      int        `json:"w"`
	H      int        `json:"h"`
	Frames []docFrame `json:"frames"`
}

type docFrame struct {
	Hold  string   `json:"hold,omitempty"`
	Art   []string `json:"art,omitempty"`
	Delta []docRun `json:"delta,omitempty"`
}

// docRun is [row, column, characters].
type docRun struct {
	Row, Col int
	Run      string
}

// MarshalJSON writes a run as [row, column, characters].
func (r docRun) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{r.Row, r.Col, r.Run})
}

// UnmarshalJSON reads a run, and refuses anything that is not those three.
func (r *docRun) UnmarshalJSON(b []byte) error {
	var v []any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if len(v) != 3 {
		return fmt.Errorf(
			"a delta run is [row, col, chars], got %d values",
			len(v),
		)
	}
	row, ok1 := v[0].(float64)
	col, ok2 := v[1].(float64)
	run, ok3 := v[2].(string)
	if !ok1 || !ok2 || !ok3 {
		return fmt.Errorf("a delta run is [number, number, string]")
	}
	r.Row, r.Col, r.Run = int(row), int(col), run
	return nil
}

// WriteDoc writes frames as a daffy document. If delta is set every frame after
// the first is written as its difference from the one before.
func WriteDoc(path, name string, frames []sprite.Frame, delta bool) error {
	if len(frames) == 0 {
		return fmt.Errorf("%s: no frames", path)
	}
	d := doc{Daffy: FormatVersion, Name: name, Sprite: docSprite{
		Name: name, W: frames[0].Width(), H: frames[0].Height(),
	}}
	for i, f := range frames {
		df := docFrame{Hold: holdText(f.Hold)}
		if i == 0 || !delta {
			df.Art = append([]string(nil), f.Art...)
		} else {
			// whichever is shorter. a delta that is longer than the
			// frame it replaces is not a saving, it is a second
			// copy with punctuation.
			runs := runsBetween(frames[i-1], f)
			if size(runs) < artSize(f.Art) {
				df.Delta = runs
			} else {
				df.Art = append([]string(nil), f.Art...)
			}
		}
		d.Sprite.Frames = append(d.Sprite.Frames, df)
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// ReadDoc reads a daffy document back into frames, replaying any deltas.
func ReadDoc(path string) ([]sprite.Frame, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d doc
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if d.Daffy != FormatVersion {
		return nil, fmt.Errorf("%s: daffy version %d, expected %d",
			path, d.Daffy, FormatVersion)
	}
	var out []sprite.Frame
	var prev []string
	for i, df := range d.Sprite.Frames {
		art := df.Art
		if art == nil {
			if prev == nil {
				return nil, fmt.Errorf(
					"%s: frame %d is a delta with "+
						"nothing before it",
					path, i)
			}
			art = apply(prev, df.Delta)
		}
		out = append(
			out, sprite.Frame{Art: art, Hold: parseHold(df.Hold)},
		)
		prev = art
	}
	return out, nil
}

// runsBetween is the spans where two frames differ.
func runsBetween(a, b sprite.Frame) []docRun {
	var out []docRun
	for y, row := range b.Art {
		var old string
		if y < len(a.Art) {
			old = a.Art[y]
		}
		x := 0
		for x < len(row) {
			if x < len(old) && old[x] == row[x] {
				x++
				continue
			}
			start := x
			unchanged := func(i int) bool {
				return i < len(old) && old[i] == row[i]
			}
			for x < len(row) && !unchanged(x) {
				x++
			}
			out = append(
				out,
				docRun{Row: y, Col: start, Run: row[start:x]},
			)
		}
	}
	return out
}

// apply replays a delta onto the frame before it.
func apply(prev []string, runs []docRun) []string {
	out := make([][]byte, len(prev))
	for i, r := range prev {
		out[i] = []byte(r)
	}
	for _, r := range runs {
		if r.Row < 0 || r.Row >= len(out) {
			continue
		}
		for i := 0; i < len(r.Run); i++ {
			if c := r.Col + i; c >= 0 && c < len(out[r.Row]) {
				out[r.Row][c] = r.Run[i]
			}
		}
	}
	res := make([]string, len(out))
	for i := range out {
		res[i] = string(out[i])
	}
	return res
}

// holdText is a duration as a document writes it.
func holdText(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return d.String()
}

// parseHold reads a duration back, and treats anything unreadable as unset.
func parseHold(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return d
}

// size is roughly what a run list costs once written: the characters, plus the
// brackets, commas and two numbers around each run.
func size(runs []docRun) int {
	n := 0
	for _, r := range runs {
		n += len(r.Run) + 12
	}
	return n
}

// artSize is roughly what a frame costs once written, for choosing between a
// full frame and a delta.
func artSize(art []string) int {
	n := 0
	for _, r := range art {
		n += len(r) + 3
	}
	return n
}
