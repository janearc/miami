package app

import (
	"fmt"
	"strings"
	"testing"
)

// A session as the terminal sends it: the answers to the two questions, a
// press, a drag of three moves, a release, two wheel notches, an arrow key, a
// letter, and a bare escape.
//
// Written as bytes rather than built, because the point of the test is that
// the bytes are what arrive and nothing knows where one read ended.
const session = "\x1b[6;38;19t" + // a cell is 19 by 38 pixels
	"\x1b[?1016;1$y" + // 1016 is on
	"\x1b[<0;400;760M" + // left button down
	"\x1b[<32;408;764M" + // dragged
	"\x1b[<32;420;770M" +
	"\x1b[<32;437;781M" +
	"\x1b[<0;437;781m" + // let go
	"\x1b[<64;437;781M" + // wheel away
	"\x1b[<65;437;781M" + // wheel toward
	"\x1b[<35;500;800M" + // moved with nothing held
	"\x1b[A" + // up arrow
	"q" + // a letter
	"\x1b" // and an escape with nothing after it

// events is everything the machine made of a stream, as text: one line each,
// so a difference between two runs is a line and not a struct.
func events(chunks []string) []string {
	var out []string
	t := &term{}
	key := func(k Key) {
		if k.Name != "" {
			out = append(out, "key "+k.Name)
			return
		}
		out = append(out, fmt.Sprintf("key %q", k.Rune))
	}
	point := func(m Mouse) {
		// the machine is told reports are in pixels once it has been
		// told a cell's size, the same as the loop does
		out = append(out, m.String())
	}
	for _, c := range chunks {
		for i := 0; i < len(c); i++ {
			t.feed(c[i], key, point)
			t.pixels = t.caps.Pixels && t.caps.CellW > 0
		}
	}
	t.flush(key)
	return out
}

// The stream cut at every single byte position gives the same events as the
// stream whole.
//
// This is the property the whole parser exists for. A drag reports every few
// milliseconds and a read returns whatever has arrived, so a report split
// across two reads is not an edge case, it is most of a drag. The old reader
// looked for an escape, a bracket and a letter inside one bufferful, and a cut
// anywhere in a report lost it and left its tail to arrive as typed letters.
//
// Said plainly: no change to the machine below can make this fail, because it
// is fed a byte at a time and cannot see a boundary to get wrong. It is here
// to fail on the day somebody parses a bufferful again, which is the shape the
// fault had the first time. TestWhatTheReportsSay is the one that catches a
// report taken apart wrongly.
func TestAStreamSplitAnywhereGivesTheSameEvents(t *testing.T) {
	whole := events([]string{session})
	if len(whole) == 0 {
		t.Fatal("the recorded session produced no events at all")
	}
	for at := 1; at < len(session); at++ {
		split := events([]string{session[:at], session[at:]})
		if len(split) != len(whole) {
			t.Fatalf("cut at byte %d: %d events, whole stream gave"+
				" %d\n  split: %s\n  whole: %s", at,
				len(split), len(whole),
				strings.Join(split, " | "),
				strings.Join(whole, " | "))
		}
		for i := range whole {
			if split[i] != whole[i] {
				t.Fatalf("cut at byte %d, event %d: %q,"+
					" whole stream gave %q", at, i,
					split[i], whole[i])
			}
		}
	}
}

// And cut at every byte, one byte a chunk, which is the worst a terminal can
// do and what a slow pipe actually does.
func TestAStreamOneByteAtATimeGivesTheSameEvents(t *testing.T) {
	whole := events([]string{session})
	var each []string
	for i := 0; i < len(session); i++ {
		each = append(each, session[i:i+1])
	}
	got := events(each)
	if len(got) != len(whole) {
		t.Fatalf("one byte at a time gave %d events, whole stream %d",
			len(got), len(whole))
	}
	for i := range whole {
		if got[i] != whole[i] {
			t.Errorf("event %d: %q, whole stream gave %q", i,
				got[i], whole[i])
		}
	}
}

// What the session means, read once. This is the test that says the numbers
// are being taken apart correctly rather than merely consistently: a parser
// that lost every report would pass the two above.
func TestWhatTheReportsSay(t *testing.T) {
	got := events([]string{session})
	want := []string{
		// the reports are one-based and a cell is 19 across by 38
		// down, so 400,760 is pixel 399,759: column 399/19 = 21 with
		// nothing left over, so the left subpixel of the two; row
		// 759/38 = 19 with 37 left over, so the last of its three.
		"press button 0 px 399,759 cell 21,19 sub 42,59",
		"move button 0 px 407,763 cell 21,20 sub 42,60",
		"move button 0 px 419,769 cell 22,20 sub 44,60",
		"move button 0 px 436,780 cell 22,20 sub 45,61",
		"release button 0 px 436,780 cell 22,20 sub 45,61",
		"wheel wheel -1 px 436,780 cell 22,20 sub 45,61",
		"wheel wheel +1 px 436,780 cell 22,20 sub 45,61",
		// 35 is a motion report with no button held: 35 and 3 is 3,
		// which is the no-button code and not the fourth button
		"move button 3 px 499,799 cell 26,21 sub 52,63",
		"key up",
		"key 'q'",
		"key '\\x1b'",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d:\n  %s", len(got), len(want),
			strings.Join(got, "\n  "))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d:\n  got  %s\n  want %s", i, got[i],
				want[i])
		}
	}
}

// A terminal that says nothing about pixels reports cells, and the pointer is
// put in the middle of the cell it is in. This is the degrade, and it is the
// only thing a multiplexer leaves working.
func TestWithoutPixelsThePointerIsTheMiddleOfItsCell(t *testing.T) {
	got := events([]string{"\x1b[<0;12;7M"})
	want := []string{"press button 0 cell 11,6 sub 22,19"}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// An escape with a key after it is the escape key and then that key. A
// terminal sends both for alt-x as well, which is why the escape is not simply
// dropped when something follows it.
func TestAnEscapeBeforeAKeyIsBothKeys(t *testing.T) {
	got := events([]string{"\x1bx"})
	want := []string{`key '\x1b'`, `key 'x'`}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Of a run of moves only the last is kept, and everything a person did is
// kept. A frame acting on every sample of a drag acts on where the hand was.
func TestOnlyTheLastMoveOfARunSurvives(t *testing.T) {
	at := func(k MouseKind, x int) Mouse {
		return Mouse{Kind: k, X: x}
	}
	in := []Mouse{
		at(Move, 1), at(Move, 2), at(Move, 3),
		at(Press, 3),
		at(Move, 4), at(Move, 5),
		at(Release, 5),
		at(Wheel, 5), at(Wheel, 5),
		at(Move, 6),
	}
	var got []string
	for _, m := range coalesce(in) {
		got = append(got, fmt.Sprintf("%s%d", m.Kind, m.X))
	}
	want := []string{"move3", "press3", "move5", "release5", "wheel5",
		"wheel5", "move6"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("at %d: got %v, want %v", i, got, want)
		}
	}
}
