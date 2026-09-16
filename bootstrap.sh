#!/bin/sh
# clone the libraries miami builds against.
#
# they are not vendored. libdaffy owns the sprite format, libdaffy-sprite the
# frame reader and libtheme-css the colour, and miami redefines none of them;
# go.work expects all three under libs/. bandersnatch is the player and is
# optional -- it is how you watch a stream rather than how you build one.
#
#   sh bootstrap.sh          clone what is missing
#   sh bootstrap.sh --force  clone again over what is there
set -eu
cd "$(dirname "$0")"
ROOTS="${GAME_ROOTS:-$HOME/mesh/frontier/roots}"
mkdir -p libs

# the name on the left is what go.work calls it; the name on the right is what
# the root is actually called. they differ because some of these have a working
# remote and a clean one, and it is the working one a build wants -- cloning
# from the name go uses got "does not exist" on a fresh machine and nowhere
# else, which is the only kind of bug a bootstrap has.
for pair in libdaffy:libdaffy-road libdaffy-sprite:libdaffy-sprite \
	libtheme-css:libtheme-css-road bandersnatch:bandersnatch; do
	repo=${pair%%:*}
	from=${pair##*:}
	if [ -d "libs/$repo" ] && [ "${1:-}" != "--force" ]; then
		echo "have  libs/$repo"
		continue
	fi
	if [ ! -d "$ROOTS/$from.git" ]; then
		echo "skip  libs/$repo (no $ROOTS/$from.git)"
		continue
	fi
	rm -rf "libs/$repo"
	git clone -q "$ROOTS/$from.git" "libs/$repo"
	echo "clone libs/$repo"
done

# the player, if its tree came down. sprites/bin is where SPRITES.md looks.
#
# it needs its own workspace for two reasons. built from in here it would pick
# up miami's go.work and decide miami is the main module; and its go.mod, like
# libdaffy's, replaces libtheme-css with a path from the layout it was
# committed in. a go.work beside it fixes both, and is an overlay rather than
# an edit: these checkouts are not ours to change.
if [ -d libs/bandersnatch ]; then
	cat > libs/bandersnatch/go.work <<'WORK'
go 1.26

use (
	.
	../libdaffy
	../libdaffy-sprite
)

replace github.com/janearc/libtheme-css => ../libtheme-css
WORK
	mkdir -p sprites/bin
	if (cd libs/bandersnatch && go build -o ../../sprites/bin/bandersnatch ./cmd/bandersnatch) 2>/dev/null; then
		echo "built sprites/bin/bandersnatch"
	else
		echo "note: bandersnatch did not build; streams still render, they just do not play"
	fi
fi

echo
echo "now: game check   (gofmt, vet, tests)"
echo "     game build   (bin/miami)"
echo "     bin/miami    (what it is, and every verb)"
