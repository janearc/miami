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
# where the libraries come from.
#
# the published urls by default, so a clone of this on a machine that has
# never heard of us builds. GAME_ROOTS points at a directory of bare
# repositories instead, which is how it is built here: the local road has
# today's work on it and the published one has the last release.
#
# the name on the left is what go.work calls it. the name on the right is what
# the local repository is called, and they differ because some of these have a
# working remote and a clean one -- cloning from the name go uses got "does
# not exist" on a fresh machine and nowhere else, which is the only kind of
# bug a bootstrap has.
ROOTS="${GAME_ROOTS:-}"
mkdir -p libs

for pair in libdaffy:libdaffy-road libdaffy-sprite:libdaffy-sprite \
	libtheme-css:libtheme-css-road bandersnatch:bandersnatch; do
	repo=${pair%%:*}
	from=${pair##*:}
	if [ -d "libs/$repo" ] && [ "${1:-}" != "--force" ]; then
		echo "have  libs/$repo"
		continue
	fi
	if [ -n "$ROOTS" ]; then
		url="$ROOTS/$from.git"
		if [ ! -d "$url" ]; then
			echo "skip  libs/$repo (no $url)"
			continue
		fi
	elif [ -d "$HOME/mesh/frontier/roots/$from.git" ]; then
		# bandersnatch is on this machine and not on github, so a
		# fresh clone here would skip it and say so for ever. the
		# frontier root is tried first when it is there, and the
		# github url is still what a machine without one gets.
		url="$HOME/mesh/frontier/roots/$from.git"
	else
		url="https://github.com/janearc/$repo.git"
	fi
	rm -rf "libs/$repo"
	if ! git clone -q "$url" "libs/$repo" 2>/dev/null; then
		# bandersnatch is the player and is optional: it is how you
		# watch a stream rather than how you build one.
		echo "skip  libs/$repo (could not clone $url)"
		continue
	fi
	echo "clone libs/$repo"
done

# the workspace, written here rather than kept in the repository.
#
# it names libs/, which is not in the repository, so tracked it made every
# checkout that had not been bootstrapped fail to build -- including a plain
# git worktree, where the error is "cannot load module libs/libdaffy-sprite
# listed in go.work file" and says nothing about what to run.
#
# without it the module builds against the published libraries, which is what
# anyone cloning gets and what the release is cut from. With it the local
# trees win, which is what working on two repositories at once needs.
#
# the replace is what makes it build, not tidiness: libdaffy's own go.mod
# replaces libtheme-css with a path from the layout it was committed in, which
# exists nowhere else, and a workspace replace overrides a module's.
if [ -d libs/libdaffy ] && [ -d libs/libdaffy-sprite ] &&
	[ -d libs/libtheme-css ]; then
	cat > go.work <<'WORK'
go 1.26

use (
	.
	./libs/libdaffy
	./libs/libdaffy-sprite
)

replace github.com/janearc/libtheme-css => ./libs/libtheme-css
WORK
	echo "wrote go.work (the libraries beside this one are used)"
elif [ -f go.work ]; then
	rm -f go.work go.work.sum
	echo "removed go.work (the libraries are not here)"
else
	echo "no go.work (building against the published libraries)"
fi

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
