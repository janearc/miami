package paint

import (
	"image/color"
	"os"
	"strconv"
	"strings"
)

// which ground a picture is being drawn on. the terminal can be asked -- OSC 11
// -- but that needs the tty in raw mode, and nothing here has a tty when it
// renders to a file, so the answer comes from what the caller says, then from
// what the shell was told, and only then from a default.

// DarkGround and LightGround are the two defaults.
var (
	DarkGround  = Ground{C: color.RGBA{16, 14, 22, 255}}
	LightGround = Ground{C: color.RGBA{250, 249, 246, 255}}
)

// DetectGround resolves the ground. want is "dark", "light", a #rrggbb, or
// empty to fall back to COLORFGBG and then to dark.
func DetectGround(want string) Ground {
	switch strings.ToLower(strings.TrimSpace(want)) {
	case "dark":
		return DarkGround
	case "light":
		return LightGround
	}
	if strings.HasPrefix(want, "#") && len(want) == 7 {
		v, err := strconv.ParseUint(want[1:], 16, 32)
		if err == nil {
			return Ground{C: color.RGBA{
				uint8(v >> 16), uint8(v >> 8), uint8(v), 255}}
		}
	}
	// COLORFGBG is "fg;bg" in terminal colour numbers; 0 to 6 and 8 are
	// dark
	if v := os.Getenv("COLORFGBG"); v != "" {
		parts := strings.Split(v, ";")
		if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			if n == 7 || n == 15 || n >= 230 {
				return LightGround
			}
			return DarkGround
		}
	}
	return DarkGround
}
