// Package hex reads a colour written as text, and is the only thing here
// that does.
//
// There were three of these, hand-written, in grade, in scene and in paint,
// and they did not agree. All three read a digit the same way and differed in
// everything around it: what a hash in front meant, what a space meant, what a
// letter that is not a digit meant, and how many digits were enough.
//
// So a document's colour could be refused by one, accepted by another, and
// drawn as a third thing by the one that actually put it on the screen. That
// is not a bug in any of them; it is three answers to one question, which is
// the shape of every fault found in this house tonight.
//
// One answer now. The others call it.
package hex

import "strings"

// Bytes is a colour's three bytes, and whether the string was a colour.
//
// A colour is six hex digits. It may be written in either case, with a hash
// in front of it, and with space around it, because those are the ways people
// write colours and a design tool emits the middle one.
func Bytes(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	var out [3]uint8
	for i := 0; i < 6; i++ {
		v, good := digit(s[i])
		if !good {
			return 0, 0, 0, false
		}
		out[i/2] = out[i/2]<<4 | v
	}
	return out[0], out[1], out[2], true
}

// Is is whether a string is a colour, for whoever is checking rather than
// drawing.
func Is(s string) bool {
	_, _, _, ok := Bytes(s)
	return ok
}

// digit is one hex digit's value.
func digit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
