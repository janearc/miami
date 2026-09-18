package scene

import (
	"strings"
	"testing"
)

// read is a document from a string, so a test can say what it means in the
// words a person writing a scene would use.
func read(t *testing.T, doc string) (*Doc, error) {
	t.Helper()
	return Read("test", []byte(doc))
}

// A document from before cameras has to come out as the one camera it always
// was, filling the sheet. This is the promise the rest of the change rests on.
func TestAnEyeIsTheMainCamera(t *testing.T) {
	d, err := read(t, `{"name":"old","eye":{"lens":2,"near":0.1}}`)
	if err != nil {
		t.Fatal(err)
	}
	cams, views, err := d.Shots()
	if err != nil {
		t.Fatal(err)
	}
	if len(cams) != 1 || cams[0].Name != MainCamera || cams[0].Lens != 2 {
		t.Fatalf("cameras = %+v, want one called main with lens 2", cams)
	}
	if cams[0].Default {
		t.Fatal("a document with an eye was given the default camera")
	}
	want := Viewport{Camera: MainCamera, W: 1, H: 1}
	if len(views) != 1 || views[0] != want {
		t.Fatalf("viewports = %+v, want %+v", views, want)
	}
}

// With neither an eye nor cameras the stage is told to use its own, rather
// than being handed an eye made of zeroes that happens to work.
func TestNoCameraAtAllIsTheDefault(t *testing.T) {
	d, err := read(t, `{"name":"bare"}`)
	if err != nil {
		t.Fatal(err)
	}
	cams, _, err := d.Shots()
	if err != nil {
		t.Fatal(err)
	}
	if len(cams) != 1 || !cams[0].Default || cams[0].Name != MainCamera {
		t.Fatalf("cameras = %+v, want the default called main", cams)
	}
}

// Two cameras and two viewports: the split she asked for, as a document.
func TestTwoCamerasSideBySide(t *testing.T) {
	d, err := read(t, `{"name":"two",
	  "eye": {"lens":1,"near":0.05},
	  "cameras": [{"name":"axis","lens":1,"near":0.05,"pitch":1.2,
	               "exposure":2}],
	  "viewports": [{"camera":"main","x":0,"y":0,"w":0.5,"h":1},
	                {"camera":"axis","x":0.5,"y":0,"w":0.5,"h":1}]}`)
	if err != nil {
		t.Fatal(err)
	}
	cams, views, err := d.Shots()
	if err != nil {
		t.Fatal(err)
	}
	if len(cams) != 2 || cams[1].Name != "axis" || cams[1].Exposure != 2 ||
		cams[1].Pitch != 1.2 {
		t.Fatalf("cameras = %+v", cams)
	}
	if len(views) != 2 || views[1].Camera != "axis" || views[1].X != 0.5 {
		t.Fatalf("viewports = %+v", views)
	}
}

// With cameras and no viewports, the first one written fills the sheet. A
// document that declares one camera and never sees it would be a trap.
func TestTheFirstCameraFillsTheSheet(t *testing.T) {
	d, err := read(t, `{"name":"one","cameras":[{"name":"axis","lens":1}]}`)
	if err != nil {
		t.Fatal(err)
	}
	_, views, _ := d.Shots()
	if len(views) != 1 || views[0].Camera != "axis" {
		t.Fatalf("viewports = %+v, want the sheet given to axis", views)
	}
}

// Thirds do not add up to one in binary. A document that says so is right.
func TestThirdsFitTheSheet(t *testing.T) {
	_, err := read(t, `{"name":"thirds","viewports":[
	  {"camera":"main","x":0,"y":0,"w":0.3333333333333333,"h":1},
	  {"camera":"main","x":0.3333333333333333,"y":0,
	   "w":0.3333333333333333,"h":1},
	  {"camera":"main","x":0.6666666666666666,"y":0,
	   "w":0.3333333333333334,"h":1}]}`)
	if err != nil {
		t.Fatalf("three thirds were refused: %v", err)
	}
}

// Every refusal names what is wrong, and happens when the document is read,
// while there is still a file name to say it about.
func TestBadCamerasAreRefusedAtRead(t *testing.T) {
	for _, tc := range []struct{ name, doc, want string }{
		{"a camera with no name",
			`{"name":"x","cameras":[{"lens":1}]}`, "has no name"},
		{"two cameras with one name",
			`{"name":"x","cameras":[{"name":"a"},{"name":"a"}]}`,
			"two cameras are called"},
		{"a second main beside an eye",
			`{"name":"x","eye":{"lens":1},"cameras":[{"name":"main"}]}`,
			`which "eye" already is`},
		{"a viewport of a camera that is not there",
			`{"name":"x","viewports":[{"camera":"b","w":1,"h":1}]}`,
			"which is not one"},
		{"a viewport with no width",
			`{"name":"x","viewports":[{"camera":"main","w":0,"h":1}]}`,
			"more than zero"},
		{"a viewport off the edge",
			`{"name":"x","viewports":[{"camera":"main","x":0.6,"w":0.5,"h":1}]}`,
			"outside the sheet"},
	} {
		_, err := read(t, tc.doc)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want one saying %q", tc.name, err, tc.want)
		}
		if err != nil && !strings.Contains(err.Error(), "test:") {
			t.Errorf("%s: the refusal does not name the document: %v",
				tc.name, err)
		}
	}
}
