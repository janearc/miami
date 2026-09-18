#!/bin/sh
# builds, vets and proves the renderer against the committed baseline. every
# step is checked on its own, so a failure early cannot pass itself off as a
# difference later.
set -eu
cd "$(dirname "$0")"

# every tracked go file, rather than a list of packages. the list was one and
# went stale the moment a package was added to the tree and not to it -- three
# of them, unchecked, before anyone noticed. git knows what is in the
# repository and libs/ is not, so this cannot drift.
git ls-files '*.go' | xargs gofmt -l | sed 's/^/gofmt: /' | grep . && exit 1 || true
go vet ./...
go build -o bin/miami ./cmd/miami
go test ./...

# the library embeds a copy of the colourway, because a program importing
# miami cannot read files out of miami's working directory. two copies is
# two things that drift, so this is the gate that says they have not.
if ! cmp -s app/builtin/starling.css sprites/made/starling.css; then
	echo "DIFFERS  app/builtin/starling.css is not sprites/made/starling.css"
	exit 1
fi

# the 404 license lives at the top, where a reader and github look for it,
# and in licenses/ beside the MIT license some of the code is under. the
# same gate, for the same reason.
if ! cmp -s LICENSE.txt licenses/LICENSE.404.txt; then
	echo "DIFFERS  licenses/LICENSE.404.txt is not LICENSE.txt"
	exit 1
fi

rm -rf regress && mkdir -p regress out
bin/miami macaw term 34 > regress/macaw.ansi 2>/dev/null
bin/miami palm term 22 > regress/palm.ansi 2>/dev/null
bin/miami car term 30 > regress/car.ansi 2>/dev/null
bin/miami miami term > regress/miami.ansi 2>/dev/null
cp out/miami-roles.txt regress/
bin/miami miami-quad > regress/miami-quad.ansi 2>/dev/null
cp out/miami-quad-roles.txt regress/
for s in 144x44 80x40 80x25 72x22 36x11; do
	bin/miami bb "${s%x*}" "${s#*x}" 2>/dev/null
	cp "out/bb-$s.ansi" "out/bb-$s-roles.txt" regress/
done

# the pointer's probe can only fail on a real terminal, which is why the first
# one shipped broken: os.Stdin takes no read deadline on a tty, so it answered
# "this terminal has no pixels" everywhere and nothing said otherwise. every
# unit test passed, because a test has no terminal.
#
# script gives a pty. the two answers are fed in as a terminal sends them, and
# what comes back has to be the cell size that was in them.
if command -v script >/dev/null 2>&1; then
	said=$(printf '\033[6;38;19t\033[?1016;1$y' |
		script -q /dev/null bin/miami probe 2>/dev/null | tr -d '\r')
	case "$said" in
	*"cell 19x38 px"*) ;;
	*) echo "DIFFERS  miami probe on a pty did not read the terminal's answer"
	   echo "         it said: $said"
	   exit 1 ;;
	esac
fi

# the icons are rendered rather than drawn, so the picture in dist/ and the
# code that makes it can part company without a word. porter asked for this
# after the first pair went out: a drifted icon is the quiet kind of wrong.
for i in dark light; do
	cp "dist/miami-icon-$i-1024.png" "regress/miami-icon-$i.was"
done
bin/miami icon 1024 > /dev/null
for i in dark light; do
	if ! cmp -s "regress/miami-icon-$i.was" "dist/miami-icon-$i-1024.png"; then
		echo "DIFFERS  dist/miami-icon-$i-1024.png"
		exit 1
	fi
done

# the cards in dist/ are what other repositories put in their readmes, and
# nothing was comparing them. cassowary had been stale since the first commit
# in this repository: the colour code moved underneath it several times --
# oklab mixing among them -- and no gate said so, because the gate only ever
# looked at what lives under regress/.
#
# they are written where they are read from, like the streams below, so they
# are compared in place rather than copied.
for c in cassowary bender libtheme-css; do
	cp "dist/$c.ansi" "regress/$c.ansi.was"
	bin/miami card "$c" > /dev/null 2>&1
	if ! cmp -s "regress/$c.ansi.was" "dist/$c.ansi"; then
		echo "DIFFERS  dist/$c.ansi"
	fi
done

# the animations are golden too: the library that makes them replaced a
# throwaway pipeline, and the only proof a port is a port is the same bytes.
# they are regenerated into a scratch directory and compared where they live,
# rather than copied into baseline, because sprites/made is where the player
# reads them from and a second copy would be the thing that drifts.
bin/miami anim regress/anim/ > /dev/null
for f in regress/anim/*.stream; do
	b=$(basename "$f")
	if ! cmp -s "$f" "sprites/made/$b"; then echo "DIFFERS  sprites/made/$b"; fi
done

total=0; bad=0
for f in regress/anim/*.stream; do
	total=$((total + 1))
	b=$(basename "$f")
	cmp -s "$f" "sprites/made/$b" || bad=$((bad + 1))
done
for f in baseline/*; do
	total=$((total + 1))
	b=$(basename "$f")
	if [ ! -f "regress/$b" ]; then echo "MISSING  $b"; bad=$((bad + 1)); continue; fi
	if ! cmp -s "$f" "regress/$b"; then echo "DIFFERS  $b"; bad=$((bad + 1)); fi
done
echo "baseline artefacts: $total   mismatched: $bad"
[ "$bad" -eq 0 ]
