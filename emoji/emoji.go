// Package emoji pulls an emoji's own bitmap out of the system font.
//
// Apple Color Emoji is not an outline font. Each glyph is a PNG, stored at a
// handful of fixed sizes in an `sbix` table, and the largest is 160 pixels
// square. So there is nothing to rasterise: the picture wanted is already a PNG
// sitting in a file on the disk, and the whole job is finding it.
//
// That matters more than it sounds. The alternative is a Python step with a
// font renderer in it, which is a second pipeline producing an asset that
// nothing else in the tree can regenerate or check.
//
// Reading the table is about a hundred lines and keeps the emoji work inside
// the same build and the same gate as everything else.
//
// The format, in the order this file walks it:
//
//	ttcf      a collection header, when several fonts share one file
//	sfnt      a table directory: tag, checksum, offset, length --
//	          sixteen bytes each
//	maxp      how many glyphs there are, which sizes the sbix offset table
//	sbix      version, flags, a count of strikes, then an offset per strike
//	strike    pixels-per-em, resolution, then an offset per glyph, plus one
//	          past the end so a glyph's length is the difference
//	glyph     two origin offsets, a four-byte kind, then the bytes
//
// A strike entry whose length is zero is a glyph with no picture at that size,
// which is most of them in most strikes; that is not an error.
package emoji

import (
	"encoding/binary"
	"fmt"
	"os"

	"golang.org/x/image/font/sfnt"
)

// Path is the system's colour emoji font.
const Path = "/System/Library/Fonts/Apple Color Emoji.ttc"

// Bitmap is one emoji's picture as it is stored.
type Bitmap struct {
	PNG   []byte
	PPEM  int // the strike it came from, in pixels per em
	Glyph int
}

// PNG is the largest stored picture for a rune, or for a sequence such as a
// flag or a skin-toned person, whose first rune carries the glyph.
func PNG(r rune) (Bitmap, error) { return FromFile(Path, r) }

// FromFile is PNG against a named font file.
func FromFile(path string, r rune) (Bitmap, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Bitmap{}, err
	}
	gid, err := glyphOf(src, r)
	if err != nil {
		return Bitmap{}, err
	}
	tables, err := tablesOf(src)
	if err != nil {
		return Bitmap{}, err
	}
	sbix, ok := tables["sbix"]
	if !ok {
		return Bitmap{}, fmt.Errorf("%s: no sbix table; not a bitmap emoji font", path)
	}
	numGlyphs, err := glyphCount(src, tables)
	if err != nil {
		return Bitmap{}, err
	}

	best := Bitmap{Glyph: gid}
	numStrikes := int(be32(src, sbix+4))
	for i := 0; i < numStrikes; i++ {
		strike := sbix + int(be32(src, sbix+8+4*i))
		if strike+4+4*(numGlyphs+1) > len(src) {
			continue
		}
		ppem := int(be16(src, strike))
		if ppem <= best.PPEM {
			continue // a smaller strike than one already found
		}
		start := int(be32(src, strike+4+4*gid))
		end := int(be32(src, strike+4+4*(gid+1)))
		if end-start <= 8 {
			continue // no picture for this glyph at this size
		}
		g := strike + start
		if kind := string(src[g+4 : g+8]); kind != "png " {
			continue
		}
		data := src[g+8 : strike+end]
		best = Bitmap{PNG: append([]byte(nil), data...), PPEM: ppem, Glyph: gid}
	}
	if best.PNG == nil {
		return Bitmap{}, fmt.Errorf("%U: no bitmap in any strike (glyph %d)", r, gid)
	}
	return best, nil
}

// glyphOf is the glyph index for a rune, using the font's cmap. The collection
// is searched rather than assuming the emoji font is first in the file.
func glyphOf(src []byte, r rune) (int, error) {
	var b sfnt.Buffer
	col, err := sfnt.ParseCollection(src)
	if err != nil {
		return 0, err
	}
	for i := 0; i < col.NumFonts(); i++ {
		f, err := col.Font(i)
		if err != nil {
			continue
		}
		g, err := f.GlyphIndex(&b, r)
		if err == nil && g != 0 {
			return int(g), nil
		}
	}
	return 0, fmt.Errorf("%U: no glyph in this font", r)
}

// tablesOf is the table directory of the first font in the file. Collections
// share their tables, so the sbix of the emoji font is reachable from any of
// them; this reads whichever directory the file starts with.
func tablesOf(src []byte) (map[string]int, error) {
	off := 0
	if len(src) >= 16 && string(src[:4]) == "ttcf" {
		off = int(be32(src, 12)) // the first font's directory
	}
	if off+12 > len(src) {
		return nil, fmt.Errorf("truncated font")
	}
	n := int(be16(src, off+4))
	out := make(map[string]int, n)
	for i := 0; i < n; i++ {
		rec := off + 12 + 16*i
		if rec+16 > len(src) {
			break
		}
		out[string(src[rec:rec+4])] = int(be32(src, rec+8))
	}
	return out, nil
}

// glyphCount is how many glyphs the font has, which sizes the sbix table.
func glyphCount(src []byte, tables map[string]int) (int, error) {
	maxp, ok := tables["maxp"]
	if !ok || maxp+6 > len(src) {
		return 0, fmt.Errorf("no maxp table")
	}
	return int(be16(src, maxp+4)), nil
}

// be16 is a big-endian sixteen-bit number, which is how a font stores one.
func be16(b []byte, at int) uint16 { return binary.BigEndian.Uint16(b[at : at+2]) }

// be32 is a big-endian thirty-two-bit number.
func be32(b []byte, at int) uint32 { return binary.BigEndian.Uint32(b[at : at+4]) }
