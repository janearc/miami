# miami

miami draws for terminals: stills, motion, effects and lighting. It paints
with fields and ramps, moves stencils in daffy's eleven-role alphabet, and
composites the two by alpha.

    sh bootstrap.sh     clone the libraries it builds against
    game check          gofmt, vet, the tests
    game build          bin/miami
    bin/miami           what it is, and every verb

It is also a library: a program with its own event loop calls package `app`
and never touches the drawing (`example/builder`).

The documentation is a program, as libtheme's is: painted, one idea per page,
and `--go` prints the source that painted each, so the two cannot drift.

    bin/miami docs            roles, fields, alpha, viewport, library, traps
    bin/miami check FILE      what a document holds, and what is wrong with it

If you are an agent: render it and look. Reading the bytes of an escape
sequence will make you confident and wrong. `bin/miami docs PAGE --png`
renders any page and `bin/miami png FILE.ansi` any file, to `out/`.

`check.sh` regenerates 29 artefacts and compares them where they live.
