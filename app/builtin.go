package app

import (
	"embed"
	"fmt"
	"image/color"
	"io/fs"
	"strings"

	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/scene"
)

// Scenes and colourways that travel with the library.
//
// A program that depends on miami should not also depend on miami's working
// directory. NewStage takes two paths, and the two paths anyone wants first are
// in this repository.
//
// So either they copy a document and a stylesheet into their own tree and the
// copies drift, or they cannot draw at all without a checkout of this one.
//
// So the useful ones are compiled in. A name beginning with "@" is a built-in:
//
//	app.NewStage("@clock", "@starling")   what miami clock draws
//	app.NewStage("my.json", "@starling")  your scene, the house colours
//	app.NewStage("my.json", "my.css")     neither of them ours
//
// Built in rather than fetched, because a build tool that has to reach the
// network to draw a progress bar is a build tool that hangs on an aeroplane.

//go:embed builtin/*
var builtin embed.FS

// Builtin lists what is compiled in, for a caller that wants to know what it
// can ask for without reading this file.
func Builtin() []string {
	var out []string
	entries, err := fs.ReadDir(builtin, "builtin")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		out = append(out, "@"+strings.TrimSuffix(e.Name(),
			ext(e.Name())))
	}
	return out
}

// ext is a filename's extension, dot included.
func ext(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i:]
	}
	return ""
}

// read is a document or a colourway, from the embedded set if it is named with
// an "@" and from the filesystem otherwise.
func read(name string) ([]byte, error) {
	if !strings.HasPrefix(name, "@") {
		return nil, nil // not ours: the caller reads it from disk
	}
	stem := name[1:]
	entries, err := fs.ReadDir(builtin, "builtin")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if strings.TrimSuffix(e.Name(), ext(e.Name())) == stem {
			return builtin.ReadFile("builtin/" + e.Name())
		}
	}
	return nil, fmt.Errorf("no built-in %q; there is %s",
		name, strings.Join(Builtin(), " "))
}

// loadScene is a scene document, built in or from disk.
func loadScene(name string) (*scene.Doc, error) {
	b, err := read(name)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return scene.Load(name)
	}
	return scene.Read(name, b)
}

// loadColours is a colourway, built in or from disk.
func loadColours(name string) (map[byte]color.RGBA, error) {
	b, err := read(name)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return compose.Colours(name)
	}
	return compose.ColoursFrom(b)
}
