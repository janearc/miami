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

for repo in libdaffy libdaffy-sprite libtheme-css bandersnatch; do
	if [ -d "libs/$repo" ] && [ "${1:-}" != "--force" ]; then
		echo "have  libs/$repo"
		continue
	fi
	rm -rf "libs/$repo"
	git clone -q "$ROOTS/$repo.git" "libs/$repo"
	echo "clone libs/$repo"
done

# the player, if its tree came down. sprites/bin is where SPRITES.md looks.
if [ -d libs/bandersnatch ]; then
	mkdir -p sprites/bin
	(cd libs/bandersnatch && go build -o ../../sprites/bin/bandersnatch ./cmd/bandersnatch) \
		2>/dev/null && echo "built sprites/bin/bandersnatch" ||
		echo "note: bandersnatch did not build; streams still render, they just do not play"
fi

echo
echo "now: game check   (gofmt, vet, tests)"
echo "     game build   (bin/miami)"
echo "     ./miami      (what it is, and every verb)"
