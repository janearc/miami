package view

import "testing"

// A camera is handed to everything that takes an eye by handing over its
// Eye, so the promotion of Eye's methods is the contract being tested.
func TestCameraIsAnEye(t *testing.T) {
	c := DefaultCamera()
	e := DefaultEye()
	if c.Eye != e {
		t.Fatalf("default camera eye = %+v, want %+v", c.Eye, e)
	}
	if got, want := c.Ray(3, 4, 80, 60), e.Ray(3, 4, 80, 60); got != want {
		t.Fatalf("camera ray = %+v, eye ray = %+v", got, want)
	}
	if c.Name != MainCamera {
		t.Fatalf("default camera is called %q, want %q", c.Name, MainCamera)
	}
}

// The scene's exposure is the fallback and one is the floor, so a document
// from before cameras is brought to the screen exactly as it was.
func TestExposedFallsBackToTheScene(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cam, scene float64
		want       float64
	}{
		{"the camera's own", 2.5, 0.8, 2.5},
		{"the scene's, when the camera has none", 0, 0.8, 0.8},
		{"one, when neither says", 0, 0, 1},
		{"a negative camera says nothing", -1, 0.8, 0.8},
	} {
		c := Camera{Exposure: tc.cam}
		if got := c.Exposed(tc.scene); got != tc.want {
			t.Errorf("%s: Exposed(%v) = %v, want %v",
				tc.name, tc.scene, got, tc.want)
		}
	}
}
