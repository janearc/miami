package scene

import (
	"strings"
	"testing"
)

// A colour a document writes is the colour it draws, or it is refused by
// name. It is never quietly a different one.
//
// grade.Swatch read anything that was not "0" to "9" or lower-case "a" to "f"
// as nought and took any string at all, so "FF6EC7" drew nearly black,
// "#ff6ec7" drew a different colour because the hash shifted every digit
// along, and "red" drew nearly black. None of them said anything, and a
// document's ambient, air and biome colours went straight there.
//
// porter found it probing the ramp beside it, after the branch it was in.
func TestAColourIsTheColourOrIsRefused(t *testing.T) {
	doc := func(colour string) string {
		return `{"shapes":[{"kind":"planet","world":{"air":"` +
			colour + `"}}]}`
	}
	for _, good := range []string{"ff6ec7", "FF6EC7", "#ff6ec7", "#FF6EC7",
		"Ff6Ec7", " ff6ec7 "} {
		if _, err := Read("test", []byte(doc(good))); err != nil {
			t.Errorf("%q is a colour people write, and it was "+
				"refused: %v", good, err)
		}
	}
	for _, bad := range []string{"red", "ff6ec", "ff6ec7ff", "", "#",
		"gg6ec7", "0x ff6ec7"} {
		_, err := Read("test", []byte(doc(bad)))
		if err == nil {
			t.Errorf("%q was taken as a colour; it would have "+
				"drawn something else without a word", bad)
			continue
		}
		if !strings.Contains(err.Error(), "not a colour") {
			t.Errorf("%q: said %q, wanted it to say what a "+
				"colour is", bad, err)
		}
	}
}

// Every field a document writes a colour in, not only the one that was
// tested: a colour that travels has two ends, and both are colours.
func TestEveryColourFieldIsChecked(t *testing.T) {
	for what, doc := range map[string]string{
		"a planet's air": `{"shapes":[{"kind":"planet",
		  "world":{"air":"red"}}]}`,
		"a sky's ambient": `{"shapes":[{"kind":"space",
		  "space":{"ambient":"red"}}]}`,
		"a biome's colour": `{"shapes":[{"kind":"planet",
		  "world":{"colours":{"sea":"red"}}}]}`,
		"a stop in a ramp": `{"shapes":[{"kind":"blackhole",
		  "hole":{"disk":[{"at":0,"c":"red"}]}}]}`,
		"where a colour travels from": `{"shapes":[{"kind":"blackhole",
		  "hole":{"disk":[{"at":0,"c":{"from":"red","to":"ff6ec7"}}]}}]}`,
		"where a colour travels to": `{"shapes":[{"kind":"blackhole",
		  "hole":{"disk":[{"at":0,"c":{"from":"ff6ec7","to":"red"}}]}}]}`,
	} {
		if _, err := Read("test", []byte(doc)); err == nil {
			t.Errorf("%s: took red as a colour", what)
		}
	}
}
