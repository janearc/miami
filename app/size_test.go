package app

import (
	"os"
	"testing"
)

// TestSttySizeRefusesWhatIsNotATerminal pins the path that hid the bug.
//
// Size asks stty about a file. Asked about something that is not a terminal it
// must say so, because the caller's next move is to try another file -- and
// the first version of this asked about nothing at all, got "stdin isn't a
// terminal", and quietly handed back ninety by twenty-four. Every window,
// however large, drew a small picture in the corner of itself.
//
// What that version actually needed was to ask about stdout, which is the
// terminal being drawn to. There is no pty here to prove that half; it was
// proved outside, with a 137 by 40 terminal reporting 137 by 39.
func TestSttySizeRefusesWhatIsNotATerminal(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("no", os.DevNull)
	}
	defer f.Close()
	if _, _, ok := sttySize(f); ok {
		t.Error("stty reported a size for /dev/null")
	}
}

// TestSizeAlwaysGivesSomethingDrawable checks the fallback is usable.
func TestSizeAlwaysGivesSomethingDrawable(t *testing.T) {
	c, r := Size()
	if c < 20 || r < 4 {
		t.Fatalf("got %dx%d, which nothing can be drawn in", c, r)
	}
}
