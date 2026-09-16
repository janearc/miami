# sprites, and how to see them move

A sprite is a stencil, not a picture. It holds roles, never colours: a
colourway binds the roles at the moment it is drawn, so one sprite takes
any palette there is, including ones written after it.

The alphabet is the contract. Eleven characters, and nothing else may
appear in an art block. A character outside it is refused by name rather
than guessed at.

    .   none        transparent
    K   dark        the dark mass
    W   light       the light mass
    V   wing        just off dark, so a large dark area is not a blob
    D   stripe      a small dark detail on light
    B   beak-base   the first of three bands of whatever protrudes
    Y   beak-band   the second
    R   beak-tip    the third, usually largest, carrying the accent
    O   feet        whatever touches the ground
    E   pupil       the eye itself
    X   eye-ring    the eye's surround

## a .sprite file

A `sprite NAME` line, an `art` line, then rows of role characters up to a
blank line. Every row is the same width. Lines beginning with a hash are
comments. `pose NAME` blocks are kept as written and not interpreted.

    sprite stranger
    art
    ..KKKK..
    .KWWWWK.
    KWWEXWWK
    KWWWWWWK
    .KBYRRK.
    ..KOOK..

`examples/stranger.sprite` uses all eleven roles and round-trips byte for
byte, which is how the format is proven rather than asserted.

## a .stream file, which is what moves

A stream is the same grammar repeated, one frame at a time, each with how
long it holds. Frames arrive and are drawn as they arrive; nothing waits
for the whole file.

    sprite blink

    frame 400ms
    art
    ..KKKK..
    ...

    frame 200ms
    art
    ..KKKK..
    ...

## seeing it

    bin/bandersnatch play examples/blink.stream
    bin/bandersnatch play --colourway vaporwave examples/blink.stream
    bin/bandersnatch play --hold 2s FILE

`--colourway` takes a name from the shelf at `~/.config/colourways/css`
or a path to a css file; with no flag it uses its own built-in sheet.
Reading from a pipe works, so a generator can feed it directly:

    ./mygenerator | bin/bandersnatch play -

## the six that move

Each full-size stencil has a stream beside it. They were traced from
photographs, so there is no model underneath to re-pose — the motion is an
operation on the grid of role characters, and each subject gets the one
motion its photograph will support.

    made/starling-flight-full.stream   a wingbeat
    made/dolphins-swim.stream          a wave down each body
    made/game-catch.stream             the ball flies into the glove
    made/daffy-walk.stream             a waddle, read from the ankles down
    made/kingfisher-hunt.stream        a bob and a blink
    made/puffin-blink.stream           a blink

Play any of them:

    bin/bandersnatch play --colourway made/starling.css \
        made/starling-flight-full.stream

`--colourway` is not optional here. The built-in sheet binds `dark` at the
ground's own lightness, and a mostly-dark bird then plays as a floating bill.

Rebuild them all with `./miami anim sprites/made/` from the tree root. The
motion is a Go library:

    miami/motion   operators on frames: warp, bend, undulate, translate,
                   blink, components, lift and stamp, and the stream writer
    miami/effect   the alpha layer: the ladder, the compositor, glow, sweep
    scenes/anim.go the rig per subject -- where the shoulder is, which
                   columns are the near leg -- and nothing else

The sprite itself is daffy's. `libdaffy-sprite` owns the format, the frame
type and the reader, and nothing in miami redefines any of them; what miami
adds is the half daffy does not do, because daffy converges on stillness.
One adaptation was needed on the way in: the stream grammar wants a `frame`
line before its art and a `.sprite` has none, so daffy's reader refuses one
with "art before any frame line" — the same refusal that stops the player
opening a `.sprite` directly. `motion.ReadSprite` supplies the missing line
rather than carrying a second reader for a format daffy owns.

The operators name roles and extents, never pixels, so the blink runs on any
stencil that carries an eye and returns the stencil unchanged when it does
not.

Two things learned by looking at the output rather than reasoning about it:

- A warp that is not an exact whole-pixel translation runs the stencil
  through the resampler. Smooth masses survive it; a dither does not. The
  kingfisher's crown is a dense speckle of gold on slate, and rotating it
  five degrees turned it to mud next to an unrotated frame — so that bird
  bobs on integer pixels instead of tilting.
- Amplitude has to be larger than looks right on paper. The first wingbeat
  was a rigid twenty degrees and read as a shrug. What made it fly was the
  tip turning further than the shoulder, and the span shortening at the top
  and bottom of the stroke.

## light, as a second sprite

The bulb is still. It is emitted cold from miami's own geometry — clear glass,
dark metal, a filament with nowhere to go — and no frame of any animation
edits one character of it. Everything that moves is a *second* sprite holding
no picture at all, only how much light is at each pixel, and a compositor
mixes the two.

    made/bulb.sprite          the still object, off
    made/bulb-lit-fx.stream   the effect on its own
    made/bulb-lit.stream      the two composited: strike, catch, settle

    made/word.sprite          set type as a stencil
    made/word-sweep-fx.stream the effect on its own
    made/word-sweep.stream    a sweep of light across the letters

The effect sprite is a real sprite. It is in the eleven characters, it plays
on its own, and daffy can read it. What makes it an alpha channel is only how
the compositor reads it: five roles standing for five levels of coverage.

    .  0.00      D  0.25      K  0.50      V  0.75      W  1.00

Five and not a continuum on purpose. Banding is the look — a roadside sign at
night has hard rings around every bulb — and quantising bounds what the
compositor can produce, so a lit frame is still a stencil in eleven characters
and still takes any colourway.

The mixing happens in OKLab lightness, never in the role characters. A pixel's
lit lightness is its own lifted toward white by the alpha, and the output role
is whichever role the colourway binds nearest to that. A role is never made
darker than it started, because light does not subtract.

### the test is the second object, not the first

An effect that only works on the thing it was written for is worth nothing.
So both effects run over both objects, and neither was edited to make it work:

    made/bulb-sweep.stream    the word's sweep, over the bulb
    made/word-glow.stream     the bulb's halo, over the word

The halo finds its source by role. On the bulb it is told the filament; on the
word it is told the letters; the field is a distance transform out of whatever
it was handed. One number has to move between the two, and it is the one the
design note predicted would have to: extent. A bulb's halo is nine pixels
because the globe is. A letter's is two and a half, because its counters are —
at nine the glow fills them and the word becomes a lozenge, which is exactly
what the first attempt produced.

Rebuild with `./miami anim sprites/made/`, the same command. `check.sh`
regenerates all twelve streams and compares them where they live: these files
are golden, because the library replaced a throwaway pipeline and the only
proof a port is a port is the same bytes. It also holds the library honest
afterwards — the vote tie-break, the alpha quantisation and the blink's fill
role are all choices with no single right answer, and the comparison is what
stops one of them drifting the next time somebody tidies a loop.

## what to make

Sprites of your own, in your alphabet, that a colourway can dress. The
birds you have drawn are halftone art in many characters; a sprite is the
other thing, a small stencil in eleven, built to move.

Start small: eight by six reads well and animates cleanly. The blink in
the examples is six rows.
