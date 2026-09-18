package app

import (
	"strings"
	"testing"
)

// What a terminal can do is said in words, because the person reading it is
// deciding whether their terminal is the problem.
func TestWhatIsMissingIsSaidInWords(t *testing.T) {
	for _, c := range []struct {
		what  string
		caps  Caps
		says  string
		quiet bool // nothing is missing
	}{
		{"a terminal that does both",
			Caps{Pixels: true, CellW: 9, CellH: 18}, "", true},
		{"pixels, but no cell size",
			Caps{Pixels: true}, "middle", false},
		{"a cell size, but no pixels",
			Caps{CellW: 9, CellH: 18}, "whole cell", false},
		{"a terminal that answered nothing",
			Caps{}, "whole cell", false},
	} {
		got := c.caps.Missing()
		if c.quiet {
			if got != "" {
				t.Errorf("%s: said %q, and nothing is missing",
					c.what, got)
			}
			continue
		}
		if !strings.Contains(got, c.says) {
			t.Errorf("%s: said %q, wanted it to mention %q",
				c.what, got, c.says)
		}
	}
}

// And the one line a probe prints says which of the two it got, separately:
// a terminal can give pixels and not a cell size, and that is a different
// thing to fix from giving neither.
func TestTheProbeLineSaysWhichOfTheTwo(t *testing.T) {
	both := Caps{Pixels: true, CellW: 9, CellH: 18}.String()
	if !strings.Contains(both, "pixels") || !strings.Contains(both, "9x18") {
		t.Errorf("a terminal that does both prints %q", both)
	}
	none := Caps{}.String()
	if !strings.Contains(none, "cells") ||
		!strings.Contains(none, "unknown") {
		t.Errorf("a terminal that does neither prints %q", none)
	}
}
