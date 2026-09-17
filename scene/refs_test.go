package scene

import (
	"reflect"
	"testing"
)

// TestRefsFindsNamesWhereverTheyAre puts a name in every kind of place a
// binding can be, including the one a walk over decoded values cannot reach:
// the raw parameters of a field, which are not decoded until it is drawn.
func TestRefsFindsNamesWhereverTheyAre(t *testing.T) {
	d, err := Read("refs", []byte(`{
	  "name": "refs",
	  "const": { "a": 1 },
	  // a comment saying "@commented" is not a document reading anything
	  "eye": { "at": { "x": "@eye", "y": 0, "z": 0 } },
	  "layers": [
	    { "name": "l",
	      "field": { "linear": { "a": [0, 0], "b": [0, "@inside"] } },
	      "ramp": [ { "at": 0, "c": "000000" } ],
	      "gain": "@a" }
	  ],
	  "shapes": [
	    { "kind": "sphere", "name": "s", "radius": 1,
	      "at": { "x": "@place", "y": "@twice", "z": "@twice" } },
	    { "kind": "cylinder", "name": "c", "radius": 1, "length": 1,
	      "texture": { "kind": "split", "frac": "@fill",
	        "done": { "kind": "solid", "a": "W" },
	        "rest": { "kind": "solid", "a": "K" } } },
	    { "kind": "bar", "name": "b", "value": "@level" },
	    { "kind": "text", "name": "t", "text": "@home" }
	  ]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"a", "eye", "fill", "inside", "level", "place", "twice",
	}
	if got := d.Refs(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// TestIsSceneReadsKeysNotText pins the routing that check depends on.
func TestIsSceneReadsKeysNotText(t *testing.T) {
	for _, c := range []struct {
		src  string
		want bool
	}{
		{`{"layers": []}`, true},
		{`{"shapes": []}`, true},
		// a sprite document that talks about layers and shapes in a
		// comment is still a sprite document
		{"{\"frames\": [] // layers, shapes\n}", false},
		{`{"voices": []}`, false},
		{`not json`, false},
	} {
		if got := IsScene([]byte(c.src)); got != c.want {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}
