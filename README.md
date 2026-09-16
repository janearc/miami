# miami

miami draws for terminals: stills, motion, effects and lighting. It paints
with fields and ramps, moves stencils in daffy's eleven-role alphabet, and
composites the two by alpha.

    sh bootstrap.sh     clone the libraries it builds against
    game check          gofmt, vet, the tests
    game build          bin/miami
    ./miami             what it is, and every verb

The documentation is a program, as libtheme's is: painted, one idea per page,
and `--go` prints the source that painted each, so the two cannot drift.

    ./miami docs             roles, fields, alpha, viewport, traps
    ./miami docs traps       the refusals, and what each one cost to find
    ./miami check FILE       what a document holds, and what is wrong with it

If you are an agent: render it and look. Reading the bytes of an escape
sequence will make you confident and wrong.

    ./miami png FILE.ansi    any ansi file, to out/
    ./miami docs PAGE --png  any page

`check.sh` regenerates 29 artefacts and compares them where they live.
