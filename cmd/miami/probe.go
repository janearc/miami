package main

import (
	"fmt"
	"os"

	"github.com/janearc/miami/app"
)

// probe says what this terminal can do about the pointer.
//
// It exists because the answer is different in a multiplexer, over ssh, and
// in a terminal that is not Ghostty, and because "the drag feels coarse" is a
// thing somebody will say before they think to ask whether their terminal
// reports pixels at all.
//
// It asks the terminal directly rather than guessing from TERM: a name says
// what a terminal calls itself, not what it does.
func probeVerb([]string) {
	caps, err := app.Probe()
	if err != nil {
		fmt.Println("cannot ask this terminal:", err)
		fmt.Println("the pointer needs a terminal on standard input;" +
			" run it in one rather than through a pipe")
		os.Exit(2)
	}
	fmt.Println(caps)
	if why := caps.Missing(); why != "" {
		fmt.Println(why)
		return
	}
	fmt.Println("a drag moves the camera by the half of a cell, which is" +
		" the resolution the picture is drawn at")
}
