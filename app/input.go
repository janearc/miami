package app

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Bytes from the terminal, turned into keys and pointer reports.
//
// The old reader took a bufferful and looked for an escape, a bracket and a
// letter next to each other in it. That works for arrows, which arrive as
// three bytes when a person presses a key.
//
// It does not work for a pointer: a drag reports every few milliseconds and a
// read returns whatever has arrived, so a report is cut in half routinely.
//
// So this is a state machine fed one byte at a time. It cannot see a boundary,
// which means there is no boundary it can get wrong, and the test says exactly
// that: a recorded session split at every one of its byte positions gives the
// same events as the whole of it at once.

// the states: plain bytes, just after an escape, and inside a CSI sequence.
const (
	ground = iota
	afterEsc
	inCSI
)

// term turns bytes into what the terminal meant by them.
type term struct {
	state int
	priv  byte  // the private marker after CSI: '<' for a pointer, '?'
	num   []int // the numbers of the sequence so far
	got   bool  // whether the number being gathered has a digit in it
	caps  Caps
	// pixels is whether reports are currently arriving in pixels, which is
	// not the same as whether the terminal can: it is only set once mode
	// 1016 has actually been turned on.
	pixels bool
}

// feed gives the machine one byte, and calls back with whatever that byte
// completed. Most bytes complete nothing.
func (t *term) feed(b byte, key func(Key), point func(Mouse)) {
	switch t.state {
	case ground:
		if b == 0x1b {
			t.state = afterEsc
			return
		}
		key(Key{Rune: rune(b)})
	case afterEsc:
		if b == '[' {
			t.state, t.priv = inCSI, 0
			t.num, t.got = t.num[:0], false
			return
		}
		// an escape followed by anything else: the escape was the key
		// and this byte is the next one, whatever it is
		t.state = ground
		key(Key{Rune: 0x1b})
		t.feed(b, key, point)
	case inCSI:
		t.csi(b, key, point)
	}
}

// csi gathers one byte of a CSI sequence and acts on the last of them.
//
// An escape inside a sequence abandons it and starts the next one, which is
// what a terminal does and what recovers from a sequence cut short: without
// it the half-finished one goes on eating bytes and swallows the sequence
// after it whole.
//
// The numbers are bounded. A sequence carrying nothing but semicolons is a
// list that grows for as long as somebody sends them, and a pointer's
// numbers are three.
func (t *term) csi(b byte, key func(Key), point func(Mouse)) {
	switch {
	case b == 0x1b:
		t.state, t.priv = afterEsc, 0
		t.num, t.got = t.num[:0], false
	case marker(b) && len(t.num) == 0 && !t.got:
		t.priv = b
	case b >= '0' && b <= '9':
		if !t.got {
			if len(t.num) >= maxNums {
				return // said enough; the rest is noise
			}
			t.num = append(t.num, 0)
			t.got = true
		}
		t.num[len(t.num)-1] = t.num[len(t.num)-1]*10 + int(b-'0')
	case b == ';':
		if !t.got && len(t.num) < maxNums {
			t.num = append(t.num, 0)
		}
		t.got = false
	case b == '$' || b == ' ' || b == '\'':
		// an intermediate byte: part of the sequence, says nothing here
	case b >= 0x40 && b <= 0x7e:
		t.done(b, key, point)
		t.state = ground
	}
}

// maxNums is how many numbers a sequence may carry here. The longest this
// reads is three, and a terminal's own limit is sixteen.
const maxNums = 16

// marker is whether a byte is the private marker a sequence may start with,
// which says what kind of sequence it is: '<' is a pointer report.
func marker(b byte) bool {
	return b == '<' || b == '?' || b == '>' || b == '='
}

// done acts on a finished sequence.
func (t *term) done(final byte, key func(Key), point func(Mouse)) {
	switch {
	case t.priv == '<' && (final == 'M' || final == 'm'):
		if m, ok := t.mouse(final == 'm'); ok {
			point(m)
		}
	case t.priv == 0 && final >= 'A' && final <= 'D':
		key(Key{Name: map[byte]string{'A': "up", 'B': "down",
			'C': "right", 'D': "left"}[final]})
	case t.priv == 0 && final == 't' && len(t.num) == 3 && t.num[0] == 6:
		// CSI 6 ; height ; width t: how big a cell is, in pixels
		t.caps.CellH, t.caps.CellW = t.num[1], t.num[2]
	case t.priv == '?' && final == 'y' && len(t.num) == 2 &&
		t.num[0] == 1016:
		// the answer to asking: one is set, three is set and
		// cannot be unset
		t.caps.Pixels = t.num[1] == 1 || t.num[1] == 3
	}
}

// flush is what to do when the terminal has gone quiet: an escape with nothing
// after it was the escape key.
//
// This is time's job and not the parser's, which is why it is a call and not
// something feed decides. A sequence is never flushed half-finished, because
// the rest of it may be one byte behind; only a bare escape is.
func (t *term) flush(key func(Key)) {
	if t.state == afterEsc {
		t.state = ground
		key(Key{Rune: 0x1b})
	}
}

// pending is whether a bare escape is waiting to be either a key or the start
// of a sequence.
func (t *term) pending() bool { return t.state == afterEsc }

// mouse turns a gathered SGR report into a Mouse, and says false for one that
// does not carry three numbers.
func (t *term) mouse(up bool) (Mouse, bool) {
	if len(t.num) < 3 {
		return Mouse{}, false
	}
	cb, x, y := t.num[0], t.num[1], t.num[2]
	m := Mouse{
		Shift: cb&4 != 0, Alt: cb&8 != 0, Ctrl: cb&16 != 0,
		X: -1, Y: -1,
	}
	switch base := Button(cb & 3); {
	case cb&64 != 0:
		// a wheel notch: nought is away from the reader, one toward
		m.Kind, m.Button, m.Wheel = Wheel, None, -1
		if cb&1 != 0 {
			m.Wheel = 1
		}
	case up:
		m.Kind, m.Button = Release, base
	case cb&32 != 0:
		m.Kind, m.Button = Move, base
		if base == None {
			m.Button = None // moving with nothing held
		}
	default:
		m.Kind, m.Button = Press, base
	}
	// the numbers are one-based, and are pixels only while 1016 is on
	if t.pixels && t.caps.CellW > 0 && t.caps.CellH > 0 {
		m.Pixels = true
		m.X, m.Y = x-1, y-1
		m.Col, m.Row = m.X/t.caps.CellW, m.Y/t.caps.CellH
		m.SubX = m.Col*2 + (m.X%t.caps.CellW)*2/t.caps.CellW
		m.SubY = m.Row*3 + (m.Y%t.caps.CellH)*3/t.caps.CellH
		return m, true
	}
	// cells: the pointer is put in the middle of the cell it is in, which
	// is the most that is known about it
	m.Col, m.Row = x-1, y-1
	m.SubX, m.SubY = m.Col*2, m.Row*3+1
	return m, true
}

// escGap is how long a bare escape waits for the rest of a sequence before it
// is taken as the escape key. A sequence arrives in one piece or within a
// millisecond or two of itself; a person pressing escape and then something
// else is slower than this by three orders of magnitude.
const escGap = 40 * time.Millisecond

// Probe asks the terminal what it can do about the pointer, and puts it back
// the way it was found.
//
// For a verb that reports rather than draws. Run does this for itself, at the
// moment it already holds the terminal.
func Probe() (Caps, error) {
	undo, err := raw()
	if err != nil {
		return Caps{}, err
	}
	defer undo()
	return probe(os.Stdout), nil
}

// probe asks the terminal what it can do, and waits a moment for the answer.
//
// Asked before any reporting is turned on, so nothing can arrive out of order:
// the only bytes that can come back are the two answers, and after this the
// decision about pixels is already made when the first report lands.
//
// The terminal times the read, not Go. os.Stdin.SetReadDeadline fails on a
// tty -- "file type does not support deadline" -- so the first version of
// this returned empty capabilities on every real terminal, and pixels were
// never turned on anywhere.
//
// porter found it under a pty, which is the only place it could be found:
// with no terminal the deadline is accepted and everything looks right.
//
// "stty min 0 time 2" is how a terminal is asked to do it: a read comes back
// after two tenths of a second with whatever has arrived, or with nothing.
// Raw mode is min 1 time 0, which is a read that waits for a key, and that is
// what it is put back to.
//
// A terminal that answers neither leaves Caps empty, which is not an error.
// miami is a library and this is a pointer: it degrades to cells, and says so
// in a sentence if the caller asks.
func probe(out *os.File) Caps {
	if err := stty("min", "0", "time", "2"); err != nil {
		return Caps{}
	}
	defer stty("min", "1", "time", "0")
	fmt.Fprint(out, askCell+askPixels)
	var t term
	buf := make([]byte, 64)
	for deadline := time.Now().Add(600 * time.Millisecond); ; {
		n, err := os.Stdin.Read(buf)
		for i := 0; i < n; i++ {
			t.feed(buf[i], func(Key) {}, func(Mouse) {})
		}
		if err != nil {
			break
		}
		if t.caps.Pixels && t.caps.CellW > 0 {
			break
		}
		if time.Now().After(deadline) {
			break
		}
	}
	return t.caps
}

// stty tells the terminal on standard input something about itself.
func stty(args ...string) error {
	c := exec.Command("stty", args...)
	c.Stdin = os.Stdin
	return c.Run()
}

// modes are what to write to start reporting, given what the terminal can do.
//
// Pixels are only asked for when a cell's size is known as well, because a
// report in pixels cannot be turned into a cell without it: the two answers
// are one capability here, whatever the terminal thinks.
func (c Caps) modes() string {
	on := "\x1b[?1003h\x1b[?1006h"
	if c.Pixels && c.CellW > 0 {
		return on + "\x1b[?1016h"
	}
	return on
}

// readInput reads the terminal until it ends, turning bytes into keys and
// pointer reports. It is the only place a read boundary exists, and the
// machine it feeds cannot see one.
func readInput(t *term, keys chan<- Key, mice chan<- Mouse) {
	chunks := make(chan []byte, 8)
	go func() {
		defer close(chunks)
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()
	key := func(k Key) { keys <- k }
	point := func(m Mouse) { mice <- m }
	var wait <-chan time.Time
	for {
		select {
		case b, ok := <-chunks:
			if !ok {
				return
			}
			for _, c := range b {
				t.feed(c, key, point)
			}
		case <-wait:
			t.flush(key)
		}
		wait = nil
		if t.pending() {
			wait = time.After(escGap)
		}
	}
}
