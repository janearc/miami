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
