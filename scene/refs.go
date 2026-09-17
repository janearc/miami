package scene

import (
	"encoding/json"
	"io"
	"sort"
	"strings"
)

// What a document reads, so a stage can refuse one that reads nothing.
//
// A binding that names a constant nobody supplies does not fail. It reads
// nought, because a missing name in a map is nought, and the picture simply
// does not do the thing.
//
// A ball bound to @beat with no music attached sits on the floor for ever and
// nothing says why. The only cure is to know every name before the first
// frame and refuse the ones with no source.
//
// The names are read from the document's JSON rather than from its decoded
// values, and that is not a shortcut. A field keeps its parameters as raw JSON
// and decodes them when drawn, so "@horizon" inside a vband inside an alpha is
// not a Value anybody could walk to at load.
//
// Reading the text finds it wherever it is, and a field added next month is
// covered without anyone remembering to add it here.

// Builtins are the names every document may read without declaring them.
var Builtins = []string{"dur", "e", "t"}

// literal are the keys whose strings are words somebody wrote rather than
// values: a line of type that says "@home" is not a constant.
var literal = map[string]bool{
	"name": true, "kind": true, "text": true, "label": true,
	"font": true, "format": true, "rune": true,
}

// Refs is every name a document reads with "@", once each, in order.
//
// It covers a document that came through Read. One built in Go has no text to
// read, so it has no names here -- and a stage built from one gets no ref
// check at all.
func (d *Doc) Refs() []string {
	var top any
	if json.Unmarshal(d.src, &top) != nil {
		return nil
	}
	found := map[string]bool{}
	walkRefs(top, "", found)
	out := make([]string, 0, len(found))
	for name := range found {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// walkRefs collects "@name" strings. An array inherits the key it sits under,
// so the operands of a mul count and the words of a text do not.
func walkRefs(v any, key string, found map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			walkRefs(e, k, found)
		}
	case []any:
		for _, e := range x {
			walkRefs(e, key, found)
		}
	case string:
		if len(x) > 1 && strings.HasPrefix(x, "@") && !literal[key] {
			found[x[1:]] = true
		}
	}
}

// IsScene says whether a document is a scene, from its top-level keys.
//
// Not from its text. The first version looked for the word "layers" anywhere
// in the file, comments included.
//
// So a sprite document mentioning layers in a comment went to the scene
// reader, and a scene made only of shapes went to the sprite reader, which
// failed on the first comment and blamed the comment.
func IsScene(b []byte) bool {
	trimmed, err := io.ReadAll(newTrimmer(b))
	if err != nil {
		return false
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(trimmed, &top) != nil {
		return false
	}
	_, layers := top["layers"]
	_, shapes := top["shapes"]
	return layers || shapes
}
