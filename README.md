# miami

miami draws for terminals: stills, motion, effects and lighting. It paints
with fields and ramps, moves stencils in daffy's eleven-role alphabet, and
composites the two by alpha. Output is cells — sextant glyphs, two colours
each — and every verb writes files rather than needing a window.

    sh bootstrap.sh     clone the libraries it builds against
    game check          gofmt, vet, the tests
    game build          bin/miami
    ./miami             what it is, and every verb

## if you are an agent, read this first

You cannot see an escape sequence. Handed one you will read the bytes, form a
confident opinion, and be wrong. That happened repeatedly while this was being
built, and once shipped a card whose wordmark was punctuation because the
render was judged from a description instead of a picture.

So every verb writes a png beside its output, and anything else can be
rendered:

    ./miami png dist/cassowary.ansi     any ansi file, to out/
    ./miami docs alpha --png            any documentation page

Open the png. Look at it. Change one thing. Look again. That loop is how this
library was built and nothing substitutes for it — not reasoning about the
bytes, not trusting the parameters, not asking whether it ought to work.

## where to start

    ./miami                  the verb table; nothing is documented elsewhere
    ./miami docs             pages: roles, fields, alpha, viewport, traps
    ./miami docs traps       the refusals, and what each one cost to find
    ./miami check FILE       what a document holds, and what is wrong with it

`docs` follows libtheme's visualdocs pattern: each page is painted rather than
described, and `--go` prints the source file that painted it, so the picture
and the code cannot drift. `--png` renders the page for the reason above.

Read `docs traps` before writing anything. Every entry in it is a mistake this
library already made, and each one cost a rebuild.

## the layout

    geom raster cells paint   stills: shapes, role grids, cell fitting, colour
    typeset                   type, and figlet fonts read from .flf
    grade  scene              gradients stacked; the same, as a json document
    motion effect             sprite animation, and effects as an alpha layer
    view   compose            objects in space, and putting them together
    emoji                     an emoji's own png, out of the system font
    scenes                    the drawings; cmd/miami the verbs
    dolphin                   the photograph tracer, still python — see below

The libraries under `libs/` are cloned, not vendored. libdaffy owns the sprite
format and the frame reader, libtheme-css owns colour, and miami redefines
neither: `go.work` expects all three beside it.

## what is proven, and how

`check.sh` regenerates 29 artefacts and compares them to `baseline/`. It also
regenerates every sprite animation and diffs it where it lives. Those files are
golden: the motion library replaced a throwaway python pipeline, and the only
proof a port is a port is that it emits the same bytes. It caught a rewrite of
300 composite literals across 30 files without a pixel moving.

The tests that matter are in `scene`, `motion` and `view`. Coverage across the
tree is low and the golden files are change-detectors rather than correctness
tests: they prove the bytes did not move, not that they were right. The only
thing checking correctness is somebody looking at a png.

## known debt, stated plainly

`dolphin/art.py` traces a photograph into character art, and it is the last
thing Go cannot regenerate. The six headers in `dist/` came from it and are not
in the gate. It has to move; until it does, the cassowary card is drawn from
geometry rather than traced from its photograph, and looks it beside the other
six birds.

The source photographs are not in this repository. `dist/` holds what was made
from them.
