package typeset

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FIGlet fonts, read here rather than shelled out to.
//
// There are two ways to use FIGlet letterforms and they are not the same thing.
// One is as a stencil -- the letters become a shape, and the shape is filled
// with picture, which is what the big.birds work does. The other is as
// characters:
//
// the letters stay characters, they go into cells as themselves, and a colour
// is applied per cell. That second one is the only thing that works at a small
// size, because outline type rasterised to six subpixels of cap height loses
// its own letterforms -- the N closes up into an H and the R loses its leg.
//
// A FIGlet font has already solved that problem; its letters were designed at
// exactly this size out of exactly these characters.
//
// The format is plain and worth reading rather than shelling out for: miami
// gains a runtime dependency on a binary otherwise, for something that is under
// a hundred lines.
//
//	flf2a$ 6 5 16 15 13 0 24463 229
//	         |  | |  |  |  |    |
//	         |  | |  |  |  |    codetag count
//	         |  | |  |  |  full layout
//	         |  | |  |  print direction
//	         |  | |  comment lines to skip
//	         |  | old layout
//	         |  max line length
//	         baseline
//	height
//
// After the comments, the characters from space to tilde in order, each of them
// `height` lines, each line ending in an endmark character that is stripped.
// The hardblank stands for a space that must not be kerned away.

// FigFont is a loaded FIGlet font.
type FigFont struct {
	Name      string
	Height    int
	Baseline  int
	hardblank byte
	glyphs    map[rune][]string
}

// FigFontDirs are where the fonts live. The homebrew path is listed because
// that is where figlet puts them on this machine and a font that cannot be
// found is a worse failure than a dependency.
var FigFontDirs = []string{
	"/opt/homebrew/share/figlet/fonts",
	"/opt/homebrew/Cellar/figlet/2.2.5/share/figlet/fonts",
	"/usr/local/share/figlet/fonts",
	"/usr/share/figlet/fonts",
}

// LoadFig loads a font by name, searching the usual directories.
func LoadFig(name string) (*FigFont, error) {
	if strings.HasSuffix(name, ".flf") {
		return readFig(name)
	}
	for _, d := range FigFontDirs {
		p := filepath.Join(d, name+".flf")
		if _, err := os.Stat(p); err == nil {
			return readFig(p)
		}
	}
	return nil, fmt.Errorf("figlet font %q not found in %v", name, FigFontDirs)
}

// readFig reads a font file: a header, comments to skip, then the characters
// from space to tilde in order.
func readFig(path string) (*FigFont, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "flf2a") {
		return nil, fmt.Errorf("%s: not a flf2a font", path)
	}
	head := strings.Fields(lines[0])
	f := &FigFont{
		Name:      strings.TrimSuffix(filepath.Base(path), ".flf"),
		hardblank: lines[0][5],
		glyphs:    map[rune][]string{},
	}
	num := func(i int) int {
		if i >= len(head) {
			return 0
		}
		n, _ := strconv.Atoi(head[i])
		return n
	}
	f.Height, f.Baseline = num(1), num(2)
	comments := num(5)
	if f.Height <= 0 {
		return nil, fmt.Errorf("%s: height %d", path, f.Height)
	}

	at := 1 + comments
	take := func() ([]string, bool) {
		if at+f.Height > len(lines) {
			return nil, false
		}
		g := make([]string, f.Height)
		for i := 0; i < f.Height; i++ {
			l := lines[at+i]
			// the endmark is whatever the line ends with, one or
			// two of them
			if n := len(l); n > 0 {
				end := l[n-1]
				for len(l) > 0 && l[len(l)-1] == end {
					l = l[:len(l)-1]
				}
			}
			g[i] = strings.ReplaceAll(l, string(f.hardblank), " ")
		}
		at += f.Height
		return g, true
	}
	for r := rune(' '); r <= '~'; r++ {
		g, ok := take()
		if !ok {
			break
		}
		f.glyphs[r] = g
	}
	return f, nil
}

// Render sets text as lines of characters.
//
// Kerned, not smushed.
//
// FIGlet's smushing rules merge touching letters into shared characters, which
// is how the fonts were meant to be set; kerning alone -- slide each letter
// left until something would collide -- is most of the benefit for a few lines
// of code, and at a wordmark's size the difference is a column here and there.
//
// Fonts whose letters are designed to interlock will look looser than figlet
// would set them.
func (f *FigFont) Render(text string) []string {
	out := make([]string, f.Height)
	for _, r := range text {
		g, ok := f.glyphs[r]
		if !ok {
			if g, ok = f.glyphs['?']; !ok {
				continue
			}
		}
		slide := overlap(out, g)
		for i := range out {
			keep := len(out[i]) - slide
			if keep < 0 {
				keep = 0
			}
			left, lap := out[i], ""
			if keep < len(left) {
				lap, left = left[keep:], left[:keep]
			}
			out[i] = left + merge(lap, g[i])
		}
	}
	return out
}

// overlap is how far a glyph can slide left before it touches what is there.
func overlap(left []string, g []string) int {
	best := -1
	for i := range left {
		room := trailing(left[i]) + leading(g[i])
		if len(g[i]) == 0 || strings.TrimSpace(g[i]) == "" {
			room = trailing(left[i]) + len(g[i])
		}
		if best < 0 || room < best {
			best = room
		}
	}
	if best < 0 {
		return 0
	}
	return best
}

// trailing is how many spaces a line ends with.
func trailing(s string) int {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == ' '; i-- {
		n++
	}
	return n
}

// leading is how many spaces a line begins with.
func leading(s string) int {
	n := 0
	for i := 0; i < len(s) && s[i] == ' '; i++ {
		n++
	}
	return n
}

// merge overlays a glyph row onto the tail of what is already set, letting ink
// win over space in either direction.
func merge(lap, g string) string {
	n := len(lap)
	if len(g) > n {
		n = len(g)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		a, b := byte(' '), byte(' ')
		if i < len(lap) {
			a = lap[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if b != ' ' {
			out[i] = b
		} else {
			out[i] = a
		}
	}
	return string(out)
}

// Trim drops blank rows and the common left margin, so a wordmark can be placed
// by its ink rather than by its font's padding.
func Trim(art []string) []string {
	first, last := -1, -1
	for i, l := range art {
		if strings.TrimSpace(l) != "" {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil
	}
	art = art[first : last+1]
	pad := -1
	for _, l := range art {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if n := leading(l); pad < 0 || n < pad {
			pad = n
		}
	}
	out := make([]string, len(art))
	for i, l := range art {
		if len(l) >= pad {
			l = l[pad:]
		}
		out[i] = strings.TrimRight(l, " ")
	}
	return out
}
