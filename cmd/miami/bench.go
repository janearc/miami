package main

import (
	"fmt"
	"time"

	"miami/app"
	"miami/cells"
)

// What a size costs, and which ceiling you hit first.
//
// There are two, and they are not the same. One is what it costs to build a
// frame, which is ours and scales with the number of subpixels. The other is
// how many bytes the frame is once it is escape sequences, which is the
// terminal's, and it is usually the one that gives way first:
//
// a cell costs about twenty bytes when its colour changes, so a full repaint of
// a hundred by thirty is sixty kilobytes and a hundred of those a second is six
// megabytes a second down a pipe.
//
// Deltas are what make the second number bearable, so both are measured: the
// first frame in full, and the steady state as a difference.
func bench(args []string) {
	sizes := [][2]int{{80, 24}, {96, 28}, {120, 34}, {160, 44}, {200, 56}, {240, 68}}
	const runs = 12

	c := newClock(time.Time{})

	fmt.Printf("%-10s %8s %9s %9s %9s %9s\n",
		"size", "cells", "build", "full", "delta", "ceiling")
	for _, s := range sizes {
		cols, rows := s[0], s[1]
		base := time.Now()

		// warm, so the first allocation is not counted as the cost of
		// the size
		prev := c.Draw(app.Frame{At: base}, cols, rows)

		start := time.Now()
		var full, delta int
		for i := 0; i < runs; i++ {
			at := base.Add(time.Duration(i) * 83 * time.Millisecond)
			cs := c.Draw(app.Frame{At: at}, cols, rows)
			full = len(cells.ANSI(cs))
			delta = len(cells.Delta(prev, cs))
			prev = cs
		}
		build := time.Since(start) / runs

		// the terminal's ceiling, assuming it can take about eight
		// megabytes a second down a pipe. that is a guess in the right
		// order rather than a measurement of any particular terminal,
		// and it is stated as one so nobody reads it as gospel.
		const bytesPerSec = 8 << 20
		byFrame := int(time.Second / build)
		byWire := bytesPerSec / max(1, delta)
		limit := "build"
		if byWire < byFrame {
			limit = "wire"
		}
		fmt.Printf("%-10s %8d %9s %9s %9s %6d/s %s\n",
			fmt.Sprintf("%dx%d", cols, rows), cols*rows*6,
			build.Round(10*time.Microsecond),
			kb(full), kb(delta), min(byFrame, byWire), limit)
	}
	fmt.Println("\ncells is subpixels: a cell is six of them, and every one is a ray.")
	fmt.Println("build is ours and scales with that. full and delta are what the")
	fmt.Println("terminal has to swallow per frame; ceiling is whichever gives first.")
}

// kb is a byte count, readably.
func kb(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	return fmt.Sprintf("%.1fK", float64(n)/1024)
}
