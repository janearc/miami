package main

import "os"

// miami draws for terminals. This command is one caller of the library, and
// every verb it has is one entry in the table in help.go: what it is called,
// what it takes, what it does, and the function that does it.
//
// The table was a switch a thousand lines long, and a verb described in one
// place and implemented in another. They are the same place now, so a verb
// that is not in the table does not exist and one that is cannot go
// undocumented.
//
// the wingbeat several verbs fly. it is a stream rather than a stencil because
// the beat is the point, and it is built by "miami anim".
const starlingStream = "sprites/made/starling-flight-full.stream"

// main is the verb table, dispatched. A name that is not a verb is a scene:
// "miami macaw term 40" draws one of the built scenes whichever way is asked.
func main() {
	if len(os.Args) < 2 {
		usage()
	}
	name, args := os.Args[1], os.Args[2:]
	for _, v := range verbs {
		if v.name == name && v.do != nil {
			v.do(args)
			return
		}
	}
	if len(args) < 1 {
		usage()
	}
	renderSingle(name, args)
}
