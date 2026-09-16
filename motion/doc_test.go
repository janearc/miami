package motion

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

func frames(t *testing.T) []sprite.Frame {
	t.Helper()
	f, err := ReadStream("../sprites/made/starling-flight-full.stream")
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// Both ways of writing a document must read back as the same frames. That is
// the only thing that makes deltas safe to prefer: they are smaller, and they
// are not a different animation.
func TestDocRoundTripsBothWays(t *testing.T) {
	want := frames(t)
	dir := t.TempDir()
	for _, delta := range []bool{false, true} {
		name := filepath.Join(dir, "full.json")
		if delta {
			name = filepath.Join(dir, "delta.json")
		}
		if err := WriteDoc(name, "starling", want, delta); err != nil {
			t.Fatal(err)
		}
		got, err := ReadDoc(name)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("delta=%v: %d frames back, wanted %d", delta, len(got), len(want))
		}
		for i := range want {
			if got[i].Hold != want[i].Hold {
				t.Errorf("delta=%v frame %d: hold %v, wanted %v",
					delta, i, got[i].Hold, want[i].Hold)
			}
			for y := range want[i].Art {
				if got[i].Art[y] != want[i].Art[y] {
					t.Fatalf("delta=%v frame %d row %d differs", delta, i, y)
				}
			}
		}
	}
}

// The trade is meant to be size for compute, so the size had better go down.
func TestDeltaIsSmaller(t *testing.T) {
	f := frames(t)
	dir := t.TempDir()
	full := filepath.Join(dir, "full.json")
	delta := filepath.Join(dir, "delta.json")
	if err := WriteDoc(full, "x", f, false); err != nil {
		t.Fatal(err)
	}
	if err := WriteDoc(delta, "x", f, true); err != nil {
		t.Fatal(err)
	}
	a, _ := os.Stat(full)
	b, _ := os.Stat(delta)
	if b.Size() >= a.Size() {
		t.Errorf("deltas are not smaller: %d vs %d", b.Size(), a.Size())
	}
	t.Logf("full %d bytes, deltas %d bytes (%.0f%%)",
		a.Size(), b.Size(), 100*float64(b.Size())/float64(a.Size()))
}

// A delta with nothing before it is refused rather than guessed at.
func TestOrphanDeltaIsRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte(`{"daffy":1,"name":"x","sprite":{"name":"x","w":1,"h":1,
		"frames":[{"delta":[[0,0,"K"]]}]}}`), 0o644)
	if _, err := ReadDoc(p); err == nil {
		t.Errorf("a leading delta was accepted")
	}
	_ = time.Second
}
