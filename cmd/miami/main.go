package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"time"

	"miami/canvas"
	"miami/cells"
	"miami/compose"
	"miami/emoji"
	"miami/grade"
	"miami/motion"
	"miami/paint"
	"miami/proof"
	"miami/raster"
	"miami/scene"
	"miami/scenes"
	"miami/typeset"
	"miami/view"

	sprite "github.com/janearc/libdaffy-sprite"
)

var single = map[string]func() *raster.Scene{
	"macaw":     scenes.Macaw,
	"palm":      scenes.Palm,
	"car":       scenes.Car,
	"miami":     scenes.Miami,
	"cassowary": scenes.Cassowary,
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "bb":
		bigBirds(os.Args[2:])
		return
	case "bulb":
		var c int
		var h float64
		fmt.Sscan(os.Args[2], &c)
		fmt.Sscan(os.Args[3], &h)
		sc := scenes.BulbTest(c, h)
		proof.RenderScene(sc, c, 3, fmt.Sprintf("bulb-%d", c))
		g := raster.Rasterize(sc, c*4, int(h*4), 3) // square samples, or the silhouette lies
		proof.Must(proof.WritePNG(fmt.Sprintf("out/bulb-%d-sil.png", c), proof.Silhouette(g, 'r'), 2))
		return
	case "sprite":
		cols := 78
		if len(os.Args) > 2 {
			fmt.Sscan(os.Args[2], &cols)
		}
		proof.Must(scenes.WriteSprite(scenes.DolphinSprite(), "dolphins",
			"sprites/made/dolphins.sprite", cols))
		fmt.Println("wrote sprites/made/dolphins.sprite")
		return
	case "bulbsprite":
		cols := 44
		if len(os.Args) > 2 {
			fmt.Sscan(os.Args[2], &cols)
		}
		proof.Must(scenes.WriteSprite(scenes.BulbSprite(), "bulb",
			"sprites/made/bulb.sprite", cols))
		fmt.Println("wrote sprites/made/bulb.sprite")
		return
	case "sunset":
		cols, rows := 100, 30
		if len(os.Args) > 3 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
		}
		sh := grade.Sextants(cols, rows)
		px := sh.Paint(scenes.Sunset(sh, 0.52, 0))
		cs := cells.ToCells(px, 2, 3)
		proof.Must(write("out/sunset.ansi", []byte(cells.ANSI(cs))))
		proof.Must(proof.WritePNG("out/sunset-px.png", px, 4))
		proof.Must(proof.WritePNG("out/sunset.png", cells.CellsToPixels(cs, 2, 3), 4))
		cols2, als, names := sh.Contact(scenes.Sunset(sh, 0.52, 0))
		for i, n := range names {
			proof.Must(proof.WritePNG("out/sunset-l-"+n+".png", cols2[i], 2))
			proof.Must(proof.WritePNG("out/sunset-a-"+n+".png", als[i], 2))
		}
		fmt.Println("wrote out/sunset.ansi and out/sunset*.png; layers:", names)
		return
	case "sunsetanim":
		cols, rows, n := 100, 30, 16
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
			fmt.Sscan(os.Args[4], &n)
		}
		sh := grade.Sextants(cols, rows)
		var frames []string
		for i := 0; i < n; i++ {
			ph := float64(i) / float64(n)
			st := append(scenes.Sunset(sh, 0.52, ph), scenes.Effects(sh, 0.52, ph)...)
			px := sh.Paint(st)
			cs := cells.ToCells(px, 2, 3)
			frames = append(frames, cells.ANSI(cs))
			proof.Must(proof.WritePNG(fmt.Sprintf("out/anim-%02d.png", i),
				cells.CellsToPixels(cs, 2, 3), 2))
		}
		proof.Must(write("out/sunset-frames.ansi",
			[]byte(strings.Join(frames, "\x1b[H"))))
		fmt.Println("wrote", n, "frames")
		return
	case "anim":
		dir := "out/anim/"
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		proof.Must(os.MkdirAll(dir, 0o755))
		rs, err := scenes.Animations("sprites/made/")
		proof.Must(err)
		lit, err := scenes.Lit("sprites/made/", "sprites/made/starling.css")
		proof.Must(err)
		proof.Must(scenes.WriteAll(dir, append(rs, lit...)))
		return
	case "play":
		// plays a frames file to stdout. synchronised output so a frame is
		// presented whole rather than torn, the cursor hidden, and the screen
		// put back the way it was found on the way out.
		raw, err := os.ReadFile(os.Args[2])
		proof.Must(err)
		loop := len(os.Args) > 3 && os.Args[3] == "loop"
		type fr struct {
			hold time.Duration
			body string
		}
		var frames []fr
		for _, block := range strings.Split(string(raw), "\nframe ")[1:] {
			nl := strings.Index(block, "\n")
			ms := 100
			fmt.Sscanf(block[:nl], "%dms", &ms)
			frames = append(frames, fr{time.Duration(ms) * time.Millisecond, block[nl+1:]})
		}
		fmt.Print("\x1b[?25l\x1b[2J")
		defer fmt.Print("\x1b[?25h\x1b[0m\n")
		for {
			for i, f := range frames {
				fmt.Print("\x1b[?2026h")
				if i == 0 {
					fmt.Print("\x1b[H")
				}
				fmt.Print(f.body)
				fmt.Print("\x1b[?2026l")
				time.Sleep(f.hold)
			}
			if !loop {
				return
			}
		}
	case "png":
		if len(os.Args) > 2 && strings.HasSuffix(os.Args[2], ".ansi") {
			out := "out/" + strings.TrimSuffix(filepath.Base(os.Args[2]), ".ansi") + ".png"
			if len(os.Args) > 3 {
				out = os.Args[3]
			}
			ansiToPNG(os.Args[2], out)
			return
		}
	case "docs":
		docs(os.Args[2:])
		return
	case "check":
		checkFile(os.Args[2])
		return
	case "card":
		// one piece of art per repository, seventy-five by seventeen, with a
		// figlet wordmark set as characters beside it. colour escapes only:
		// these are cat'd inside a readme and in a release tree, so nothing
		// here may move the cursor or clear the screen.
		name := os.Args[2]
		const cols, rows = 75, 17
		var cs [][]cells.Cell
		var pal []color.RGBA
		var cardPx [][]color.RGBA
		switch name {
		case "libtheme-css":
			sh := grade.Sextants(cols, rows)
			px := sh.Paint(scenes.Libtheme(sh))
			cardPx = px
			cs = cells.ToCells(px, 2, 3)
			pal = cells.PaletteOf(px, 10, 0.05)
		default:
			var sc *raster.Scene
			if name == "cassowary" {
				sc = scenes.Cassowary()
			} else {
				sc = scenes.Bender()
			}
			g := raster.Rasterize(sc, cols*2, rows*3, 4)
			px := raster.Resolve(g, sc, 0, 0, cols*2, rows*3)
			cardPx = px
			cs = cells.ToCells(px, 2, 3)
			// the scene's own declared palette, not the palette of the
			// rendered pixels: most of this card is background, so sampling
			// the render gives a ramp made of ground and the word comes out
			// the colour of the dark it is sitting on
			for r, c := range sc.Pal {
				if r != 'i' && r != 'e' {
					pal = append(pal, c)
				}
			}
		}

		// libtheme-css carries no wordmark at all. the card's whole argument
		// is that the library owns no form, and it sits under a heading that
		// already says its name; a word punched into the ramps was the one
		// thing on it pretending to be an object.
		word := map[string]string{
			"cassowary": "cassowary", "bender": "bender",
			"libtheme-css": "libtheme-css",
		}[name]
		// the art moves over to make room for type that can actually be read.
		// contessa fitted the gap at three rows and held none of its
		// letterforms: cassowary came out as punctuation. fitting the space
		// was the wrong thing to optimise.
		shift := map[string]int{"cassowary": 11, "bender": 12}[name]
		if shift > 0 {
			cells.ShiftLeft(cs, shift, color.RGBA{16, 14, 24, 255})
		}
		wx := map[string]int{"cassowary": 26, "bender": 40, "libtheme-css": 2}[name]
		wy := map[string]int{"cassowary": 6, "bender": 6, "libtheme-css": 0}[name]
		var art []string
		if word != "" {
			ff, err := typeset.LoadFig("small")
			proof.Must(err)
			art = typeset.Trim(ff.Render(word))
		}
		// only the bright half of the palette: Hot runs from the lightest
		// colour to the middle one, and on a card that is mostly dark ground
		// the middle one is nearly the ground itself
		ramp := scenes.Ramp(pal)
		if len(ramp) > 3 {
			ramp = ramp[len(ramp)/2:]
		}
		ink := func(x, y int) color.RGBA {
			return scenes.Hot(ramp, float64(y)/math.Max(1, float64(len(art)-1)))
		}
		if name == "libtheme-css" {
			// coloured from the ramps it stands above, column by column, so
			// the mark is the palette. a colour library's own wordmark ought
			// to be made of the thing the library does.
			marks := scenes.LibthemeMarks
			wide := 0
			for _, l := range art {
				if len(l) > wide {
					wide = len(l)
				}
			}
			// blended rather than stepped: six hard switches across a word
			// land in the middle of letters, so a single glyph comes out half
			// gold and half blue and reads as a fault rather than a spectrum
			ink = func(x, _ int) color.RGBA {
				t := float64(x) / math.Max(1, float64(wide-1)) * float64(len(marks)-1)
				i := int(t)
				if i >= len(marks)-1 {
					return scenes.Hex(marks[len(marks)-1])
				}
				return scenes.Blend(marks[i], marks[i+1], t-float64(i))
			}
		}
		if art != nil {
			cells.Letters(cs, art, wx, wy, ink)
		}

		proof.Must(write("dist/"+name+".ansi", []byte(cells.ANSI(cs))))

		// and a plain companion, because markdown needs four spaces of indent
		// to hold art preformatted and escapes do not survive that. colour is
		// gone, so the picture has to be carried by lightness alone: the same
		// subpixels, fitted to sextant masks against how bright each one is.
		// the wordmark is stamped after, as the characters it already was.
		lum := make([][]float64, len(cardPx))
		lo, hi := math.Inf(1), math.Inf(-1)
		for y, r2 := range cardPx {
			lum[y] = make([]float64, len(r2))
			for x, c := range r2 {
				v := paint.Luminance(c)
				lum[y][x] = v
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
		// stretched across the range the picture actually uses. a cassowary
		// painted in mid blues and tans occupies a third of the scale, and
		// left unstretched every one of them fits the same shade: the bird
		// came out as one flat patch of light grey with a word next to it.
		if hi > lo {
			for y := range lum {
				for x := range lum[y] {
					lum[y][x] = math.Pow((lum[y][x]-lo)/(hi-lo), 0.75)
				}
			}
		}
		tone := cells.Tone(lum, 3)
		for y, line := range art {
			for x, ch := range line {
				if ch == ' ' {
					continue
				}
				if r2, c2 := wy+y, wx+x; r2 < len(tone) && c2 < len(tone[r2]) {
					tone[r2][c2] = ch
				}
			}
		}
		var plain strings.Builder
		for _, r2 := range tone {
			plain.WriteString(strings.TrimRight(string(r2), " ") + "\n")
		}
		proof.Must(write("dist/"+name+"-plain.txt", []byte(plain.String())))
		proof.Must(proof.SaveImage("out/card-"+name+".png",
			proof.NewCellImageOn(proof.Mono(), 9, 18, color.RGBA{16, 14, 24, 255}).Draw(cs)))
		fmt.Printf("dist/%s.ansi  %d cols x %d rows\n", name, cols, rows)
		return
	case "shadow":
		// a ball over a floor, with a light somewhere in the scene and a
		// shadow that is cast rather than drawn: for every point of floor,
		// one ray toward the lamp, and whether it arrives.
		cols, rows, secs, fps := 90, 26, 8.0, 15
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
			fmt.Sscan(os.Args[4], &secs)
		}
		col, err := compose.Colours("sprites/made/starling.css")
		proof.Must(err)
		var out strings.Builder
		var prev [][]cells.Cell
		hold := time.Duration(float64(time.Second) / float64(fps))
		n := int(secs*float64(fps) + 0.5)
		fmt.Fprintf(&out, "# a ball over a floor, and a shadow cast by a located light.\n")
		for i := 0; i < n; i++ {
			t := float64(i) / float64(fps)
			e := t / secs
			sc := view.Lit{
				// the floor sits below the eye and the eye looks along it
				Eye: view.Eye{At: view.Vec{Y: 0.30}, Lens: 1.0, Near: 0.05},
				Floor: view.Floor{Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
					// a shadow is a remap, so every colour in it was already
					// chosen by the colourway
					Shade: map[byte]byte{'V': 'K', 'K': 'D'}},
				// the lamp: high, to the left, and a little in front, so the
				// shadow falls away to the right and lengthens as the ball
				// travels from under it
				// behind the ball as well as above it, so the shadow is thrown
				// toward the viewer. with the lamp in front, every shadow fell
				// away from the camera and hid behind the thing casting it.
				Light: view.Vec{X: -1.2, Y: 3.0, Z: 1.6},
				Ball: view.Ball{
					At: view.Vec{
						X: 1.30 * math.Sin(2*math.Pi*e),
						// a bounce: the height is the absolute of a sine, so
						// it arrives at the floor rather than easing into it
						Y: -0.21 + 0.66*math.Abs(math.Sin(3*math.Pi*e)),
						Z: 3.6 - 0.7*math.Cos(2*math.Pi*e),
					},
					Radius: 0.34,
					Spin:   view.Vec{X: 0.11, Y: 0.43, Z: 0.07},
				},
			}
			f := sc.Trace(time.Duration(t*float64(time.Second)), cols, rows)
			px := make([][]color.RGBA, rows*3)
			for y := range px {
				px[y] = make([]color.RGBA, cols*2)
				for x := range px[y] {
					px[y][x] = color.RGBA{10, 9, 16, 255}
				}
			}
			compose.Blit(px, f, col, 0, 0)
			cs := cells.ToCells(px, 2, 3)
			fmt.Fprintf(&out, "\nframe %dms\n", hold.Milliseconds())
			if prev == nil {
				out.WriteString(cells.ANSI(cs))
			} else {
				out.WriteString(cells.Delta(prev, cs))
			}
			prev = cs
			if i%(n/6) == 0 && i/(n/6) < 6 {
				proof.Must(proof.WritePNGXY(fmt.Sprintf("out/shadow-%d.png", i/(n/6)),
					cells.CellsToPixels(cs, 2, 3), 3, 4))
			}
		}
		proof.Must(write("out/shadow.frames", []byte(out.String())))
		fmt.Printf("%d frames -> out/shadow.frames\n", n)
		return
	case "ball":
		// a checkered ball on nothing at all: spinning, moving, and coming
		// toward the viewer. no background, so what is being looked at is the
		// object and not the picture around it.
		cols, rows, secs, fps := 90, 26, 12.0, 15
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
			fmt.Sscan(os.Args[4], &secs)
		}
		sh := grade.Sextants(cols, rows)
		col, err := compose.Colours("sprites/made/starling.css")
		proof.Must(err)
		eye := view.DefaultEye()
		ground := color.RGBA{14, 13, 22, 255}

		var out strings.Builder
		var prev [][]cells.Cell
		hold := time.Duration(float64(time.Second) / float64(fps))
		n := int(secs*float64(fps) + 0.5)
		fmt.Fprintf(&out, "# a checkered ball: spinning, scaling and moving.\n")
		for i := 0; i < n; i++ {
			t := float64(i) / float64(fps)
			e := t / secs
			px := make([][]color.RGBA, sh.H)
			for y := range px {
				px[y] = make([]color.RGBA, sh.W)
				for x := range px[y] {
					px[y][x] = ground
				}
			}
			ball := view.Ball{
				// across, and toward the viewer and away again, so the same
				// object is seen at several sizes without anything resizing it
				At: view.Vec{
					X: 0.62 * math.Sin(2*math.Pi*e),
					Y: 0.22 * math.Sin(4*math.Pi*e),
					Z: 2.9 - 1.25*math.Sin(math.Pi*e),
				},
				Radius: 0.42,
				Spin:   view.Vec{X: 0.13, Y: 0.37, Z: 0.05},
			}
			for _, p := range view.Resolve([]view.Object{ball}, eye,
				time.Duration(t*float64(time.Second)), sh.W, sh.H) {
				compose.Blit(px, p.Frame, col, p.X, p.Y)
			}
			cs := cells.ToCells(px, 2, 3)
			fmt.Fprintf(&out, "\nframe %dms\n", hold.Milliseconds())
			if prev == nil {
				out.WriteString(cells.ANSI(cs))
			} else {
				out.WriteString(cells.Delta(prev, cs))
			}
			prev = cs
			if i%(n/8) == 0 && i/(n/8) < 8 {
				proof.Must(proof.WritePNGXY(fmt.Sprintf("out/ball-%d.png", i/(n/8)),
					cells.CellsToPixels(cs, 2, 3), 3, 4))
			}
		}
		proof.Must(write("out/ball.frames", []byte(out.String())))

		// and the same spin baked out as a daffy document, both ways, so the
		// trade between size and compute is a measurement rather than a claim
		var baked []sprite.Frame
		spin := view.Ball{At: view.Vec{Z: 2.4}, Radius: 0.42,
			Spin: view.Vec{X: 0.13, Y: 0.37, Z: 0.05}}
		for i := 0; i < n; i++ {
			f := spin.Render(eye, time.Duration(float64(i)/float64(fps)*float64(time.Second)), 24)
			f.Hold = hold
			baked = append(baked, f)
		}
		proof.Must(motion.WriteDoc("out/ball-full.json", "ball", baked, false))
		proof.Must(motion.WriteDoc("out/ball.json", "ball", baked, true))
		back, err := motion.ReadDoc("out/ball.json")
		proof.Must(err)
		for i := range baked {
			for y := range baked[i].Art {
				if back[i].Art[y] != baked[i].Art[y] {
					proof.Must(fmt.Errorf("frame %d row %d did not replay", i, y))
				}
			}
		}
		a, _ := os.Stat("out/ball-full.json")
		b2, _ := os.Stat("out/ball.json")
		fmt.Printf("%d frames -> out/ball.frames\n", n)
		fmt.Printf("baked %d frames: full %d bytes, deltas %d bytes (%.0f%%), replayed exactly\n",
			len(baked), a.Size(), b2.Size(), 100*float64(b2.Size())/float64(a.Size()))
		return
	case "flock":
		// a flock at different depths over the dusk. every bird is the same
		// sprite; what differs is where it is, and the viewport does the rest.
		cols, rows, secs, fps := 110, 30, 20.0, 15
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
			fmt.Sscan(os.Args[4], &secs)
		}
		sh := grade.Sextants(cols, rows)
		doc, err := scene.Load("scenes/dusk.json")
		proof.Must(err)
		frames, err := motion.ReadStream("sprites/made/starling-flight-full.stream")
		proof.Must(err)
		col, err := compose.Colours("sprites/made/starling.css")
		proof.Must(err)

		eye := view.DefaultEye()
		// nine birds, spread through the depth of the picture. the near ones
		// are big and cross fast; the far ones are small and barely move.
		// that is parallax and it comes out of the projection for nothing --
		// nothing here is told to make the distant ones slower.
		type bird struct {
			z, y, x0, speed float64
			phase           time.Duration
		}
		var birds []bird
		for i := 0; i < 9; i++ {
			f := float64(i)
			birds = append(birds, bird{
				z:     1.2 + 1.35*f,
				y:     -0.12 + 0.075*math.Sin(f*2.1),
				x0:    -1.8 - 0.55*f,
				speed: 0.55 + 0.05*math.Sin(f*1.7),
				phase: time.Duration(f*137) * time.Millisecond,
			})
		}

		var out strings.Builder
		var prev [][]cells.Cell
		hold := time.Duration(float64(time.Second) / float64(fps))
		n := int(secs*float64(fps) + 0.5)
		fmt.Fprintf(&out, "# a flock at nine depths over a dusk. one sprite, one viewport.\n")
		for i := 0; i < n; i++ {
			t := float64(i) / float64(fps)
			st, err := doc.At(t)
			proof.Must(err)
			px := sh.Paint(st)

			objs := make([]view.Object, 0, len(birds))
			for _, b := range birds {
				objs = append(objs, view.Billboard{
					Frames: frames, Height: 0.5, Phase: b.phase,
					// wrapped, so the sky does not empty out: a bird that
					// leaves on the right comes back on the left, and at its
					// own depth nobody can tell it is the same one
					At: view.Vec{X: math.Mod(b.x0+b.speed*t+4.5, 9) - 4.5,
						Y: b.y + 0.05*math.Sin(t*0.9+b.z), Z: b.z},
				})
			}
			for _, p := range view.Resolve(objs, eye, time.Duration(t*float64(time.Second)), sh.W, sh.H) {
				compose.Blit(px, p.Frame, col, p.X, p.Y)
			}

			cs := cells.ToCells(px, 2, 3)
			fmt.Fprintf(&out, "\nframe %dms\n", hold.Milliseconds())
			if prev == nil {
				out.WriteString(cells.ANSI(cs))
			} else {
				out.WriteString(cells.Delta(prev, cs))
			}
			prev = cs
			if i%(n/6) == 0 && i/(n/6) < 6 {
				proof.Must(proof.WritePNGXY(fmt.Sprintf("out/flock-%d.png", i/(n/6)),
					cells.CellsToPixels(cs, 2, 3), 3, 4))
			}
		}
		proof.Must(write("out/flock.frames", []byte(out.String())))
		fmt.Printf("%d frames -> out/flock.frames\n", n)
		return
	case "fly":
		cols, rows, secs, fps := 100, 28, 20.0, 15
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
			fmt.Sscan(os.Args[4], &secs)
		}
		if len(os.Args) > 5 {
			fmt.Sscan(os.Args[5], &fps)
		}
		sh := grade.Sextants(cols, rows)
		doc, err := scene.Load("scenes/dusk.json")
		proof.Must(err)
		secs = doc.Seconds
		bird, err := motion.ReadStream("sprites/made/starling-flight-full.stream")
		proof.Must(err)
		col, err := compose.Colours("sprites/made/starling.css")
		proof.Must(err)
		bw, bh := compose.Size(bird[0])

		n := int(secs*float64(fps) + 0.5)
		hold := time.Duration(float64(time.Second) / float64(fps))
		var out strings.Builder
		var prev [][]cells.Cell
		fmt.Fprintf(&out, "# a dusk that does not repeat, and a starling crossing it.\n")
		fmt.Fprintf(&out, "# %d by %d cells, %d frames at %d per second.\n", cols, rows, n, fps)
		for i := 0; i < n; i++ {
			t := float64(i) / float64(fps)
			st, err := doc.At(t)
			proof.Must(err)
			px := sh.Paint(st)

			// the bird crosses once, which is its own kind of not repeating,
			// and rises as the light goes. its wingbeat keeps its own clock.
			e := t / secs
			x := int(float64(-bw) + float64(cols*2+2*bw)*e)
			y := int(float64(rows*3)*(0.62-0.34*e) - float64(bh)/2 +
				float64(rows)*0.22*math.Sin(2*math.Pi*t/6.4))
			compose.Over(px, compose.At(bird, time.Duration(t*float64(time.Second))), col, x, y)

			cs := cells.ToCells(px, 2, 3)
			fmt.Fprintf(&out, "\nframe %dms\n", hold.Milliseconds())
			if prev == nil {
				out.WriteString(cells.ANSI(cs))
			} else {
				out.WriteString(cells.Delta(prev, cs))
			}
			// the encoder is checked as it runs: replay the delta onto the
			// previous frame and it must give this one back exactly
			if prev != nil {
				echo := make([][]cells.Cell, len(prev))
				for i := range prev {
					echo[i] = append([]cells.Cell(nil), prev[i]...)
				}
				cells.Apply(echo, cells.Delta(prev, cs))
				for r := range echo {
					for c := range echo[r] {
						if echo[r][c] != cs[r][c] {
							proof.Must(fmt.Errorf("frame %d: delta does not replay at %d,%d", i, r, c))
						}
					}
				}
			}
			prev = cs
			if i%(n/6) == 0 && i/(n/6) < 6 {
				proof.Must(proof.WritePNGXY(fmt.Sprintf("out/fly-%d.png", i/(n/6)),
					cells.CellsToPixels(cs, 2, 3), 3, 4))
			}
		}
		proof.Must(write("out/fly.frames", []byte(out.String())))
		fmt.Printf("%d frames, %.0fs -> out/fly.frames\n", n, secs)
		return
	case "scene":
		// renders a json scene, and if a go scene of the same name exists,
		// compares them: the only proof a document expresses a scene is that
		// it paints the same pixels as the code it replaced.
		doc, err := scene.Load(os.Args[2])
		proof.Must(err)
		cols, rows := 100, 30
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[3], &cols)
			fmt.Sscan(os.Args[4], &rows)
		}
		sh := grade.Sextants(cols, rows)
		same := true
		for i, t := range []float64{0, 4, 8, 12, 16, 20} {
			st, err := doc.At(t)
			proof.Must(err)
			got := sh.Paint(st)
			cs := cells.ToCells(got, 2, 3)
			proof.Must(proof.WritePNGXY(fmt.Sprintf("out/scene-%d.png", i),
				cells.CellsToPixels(cs, 2, 3), 3, 4))
			want := sh.Paint(scenes.Dusk(sh, t, doc.Seconds))
		compare:
			for y := range want {
				for x := range want[y] {
					if want[y][x] != got[y][x] {
						fmt.Printf("t=%.0f differs first at %d,%d: go %v json %v\n",
							t, x, y, want[y][x], got[y][x])
						same = false
						break compare
					}
				}
			}
			_ = i
		}
		if same {
			fmt.Println("json scene paints the same pixels as the go scene, at every sample")
		}
		return
	case "dusk":
		cols, rows := 100, 30
		if len(os.Args) > 3 {
			fmt.Sscan(os.Args[2], &cols)
			fmt.Sscan(os.Args[3], &rows)
		}
		sh := grade.Sextants(cols, rows)
		for i, t := range []float64{0, 4, 8, 12, 16, 20} {
			cs := cells.ToCells(sh.Paint(scenes.Dusk(sh, t, 20)), 2, 3)
			proof.Must(proof.WritePNGXY(fmt.Sprintf("out/dusk-%d.png", i),
				cells.CellsToPixels(cs, 2, 3), 3, 4))
		}
		fmt.Println("wrote six dusk samples")
		return
	case "emoji":
		// the rune is given as a hex codepoint, because argv and a terminal
		// between them will not reliably deliver an astral character
		var cp int64
		fmt.Sscanf(os.Args[2], "%x", &cp)
		cols, n, word, fig := 44, 8, "", "smslant"
		if len(os.Args) > 3 {
			fmt.Sscan(os.Args[3], &cols)
		}
		if len(os.Args) > 4 {
			fmt.Sscan(os.Args[4], &n)
		}
		if len(os.Args) > 5 {
			word = os.Args[5]
		}
		if len(os.Args) > 6 {
			fig = os.Args[6]
		}
		bm, err := emoji.PNG(rune(cp))
		proof.Must(err)
		img, err := png.Decode(bytes.NewReader(bm.PNG))
		proof.Must(err)
		// keep the picture square: a cell is twice as tall as it is wide, so
		// the row count is the column count scaled by the image's own aspect
		// and then halved. that is the only place the aspect is applied.
		ib := img.Bounds()
		rows := int(float64(cols)*float64(ib.Dy())/float64(ib.Dx())/2 + 0.5)
		// palette first, then snap, then vote down: reducing after sampling
		// lets the source's anti-aliased edges into the palette
		px := cells.SampleFlat(img, cols*2, rows*3, n, 0.055)

		// the wordmark is figlet letterforms placed as characters, not type
		// rasterised into the block-glyph fitter: at this size the fitter
		// cannot hold a letterform and the N comes out as an H
		var art []string
		if word != "" {
			ff, err := typeset.LoadFig(fig)
			proof.Must(err)
			art = typeset.Trim(ff.Render(word))
			wide := 0
			for _, l := range art {
				wide = max(wide, len(l))
			}
			px = cells.Widen(px, (cols+2+wide+1)*2)
		}
		cs := cells.ToCellsAlpha(px, 2, 3, 128)
		if art != nil {
			ramp := scenes.Ramp(cells.PaletteOf(px, n, 0.055))
			top := (len(cs) - len(art)) / 2
			cells.Letters(cs, art, cols+2, top, func(x, y int) color.RGBA {
				return scenes.Hot(ramp, float64(y)/math.Max(1, float64(len(art)-1)))
			})
		}
		name := fmt.Sprintf("out/emoji-%X", cp)
		proof.Must(write(name+".ansi", []byte(cells.ANSIAlpha(cs))))
		// the same cells on two grounds, because "the background is
		// transparent" is a claim about what the terminal supplies and the
		// only way to see it is to supply two different things.
		//
		// drawn through the cell-image renderer rather than by expanding the
		// block masks: a literal letter is not a sextant, so the mask path
		// draws nothing where the wordmark is and the preview quietly lies
		// about what the file contains.
		for _, g := range []struct {
			suffix string
			ground color.RGBA
		}{{"", color.RGBA{18, 16, 28, 255}}, {"-light", color.RGBA{238, 236, 230, 255}}} {
			proof.Must(proof.SaveImage(name+g.suffix+".png",
				proof.NewCellImageOn(proof.Mono(), 14, 28, g.ground).Draw(cs)))
		}
		fmt.Printf("%U at %dx%d cells (ppem %d) -> %s.ansi\n",
			cp, len(cs[0]), len(cs), bm.PPEM, name)
		return
	case "word":
		text := "LIGHTS"
		if len(os.Args) > 2 {
			text = os.Args[2]
		}
		cols := 62
		if len(os.Args) > 3 {
			fmt.Sscan(os.Args[3], &cols)
		}
		sc := scenes.WordSprite(text, 100, 5)
		proof.Must(scenes.WriteSprite(sc, "word", "sprites/made/word.sprite", cols))
		fmt.Println("wrote sprites/made/word.sprite")
		return
	case "dolphins":
		sc := scenes.Dolphins()
		g := raster.Rasterize(sc, int(sc.W*5), int(sc.H*5), 3)
		proof.Must(proof.WritePNG("out/dolphins-sil.png", proof.Silhouette(g), 1))
		proof.Must(proof.WritePNG("out/dolphins-grey.png", raster.Resolve(g, sc, 0, 0, int(sc.W*5), int(sc.H*5)), 1))
		cols := 76
		if len(os.Args) > 2 {
			fmt.Sscan(os.Args[2], &cols)
		}
		proof.Page(sc, cols, fmt.Sprintf("dolphins-%d", cols), nil, nil)
		return
	case "lights":
		sc, marks := scenes.Lights(canvas.Manpage)
		proof.Page(sc, canvas.Manpage.Cols, "lights", scenes.LightsText, marks)
		return
	case "specimen":
		proof.RenderScene(scenes.Specimen(canvas.Manpage), canvas.Manpage.Cols, 3, "specimen")
		return
	case "faces":
		for _, f := range os.Args[2:] {
			names, err := typeset.Faces(f)
			proof.Must(err)
			fmt.Printf("%s\n  %s\n", f, strings.Join(names, "\n  "))
		}
		return
	case "miami-quad":
		fmt.Print(cells.ANSI(scenes.BuildMiami("miami-quad", 2)))
		return
	case "miami":
		if len(os.Args) == 2 || os.Args[2] == "term" {
			fmt.Print(cells.ANSI(scenes.BuildMiami("miami", 3)))
			return
		}
	}
	if len(os.Args) < 3 {
		usage()
	}
	mk, ok := single[os.Args[1]]
	if !ok {
		fmt.Println("unknown scene", os.Args[1])
		os.Exit(2)
	}
	sc := mk()
	mode := os.Args[2]
	cols := 34
	if len(os.Args) > 3 {
		fmt.Sscan(os.Args[3], &cols)
	}

	switch mode {
	case "png", "sil", "roles":
		gw := int(sc.W * 2)
		gh := int(sc.H * 2)
		g := raster.Rasterize(sc, gw, gh, 3)
		switch mode {
		case "sil":
			proof.Must(proof.WritePNG("out/"+os.Args[1]+"-sil.png", proof.Silhouette(g, 'P', 'x'), 2))
		case "roles":
			fmt.Print(raster.RoleText(g))
		default:
			raster.Contour(g, 'O', 1)
			proof.Must(proof.WritePNG("out/"+os.Args[1]+".png", raster.Resolve(g, sc, 0, 0, gw, gh), 2))
		}
	case "term":
		rows := raster.TermRows(sc, cols)
		gw, gh := cols*2, rows*3
		g := raster.Rasterize(sc, gw, gh, 4)
		raster.Contour(g, 'O', 1)
		rgb := raster.Resolve(g, sc, 0, 0, gw, gh)
		cs := cells.ToCells(rgb, 2, 3)
		fmt.Print(cells.ANSI(cs))
		proof.Must(proof.WritePNGXY("out/"+os.Args[1]+"-term.png", cells.CellsToPixels(cs, 2, 3), 7, 9))
		fmt.Fprintf(os.Stderr, "%d cols x %d rows\n", cols, rows)
	}
}

func bigBirds(args []string) {
	if len(args) < 2 {
		usage()
	}
	var c, r int
	fmt.Sscan(args[0], &c)
	fmt.Sscan(args[1], &r)
	name := fmt.Sprintf("bb-%dx%d", c, r)
	if len(args) > 2 {
		fmt.Sscan(args[2], &scenes.BBWordFrac)
	}
	if len(args) > 4 {
		fmt.Sscan(args[3], &scenes.BBErodeUp)
		fmt.Sscan(args[4], &scenes.BBErodeIter)
	}
	if len(args) > 5 {
		name = args[5]
	}
	proof.RenderScene(scenes.BigBirds(c, r), c, 3, name)
}
