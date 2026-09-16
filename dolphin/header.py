# the flipr manpage header: two dolphins in colour character art, with a
# figlet wordmark. figlet rather than a rasterised typeface: its letterforms
# are drawn for this size, and outlines sampled into seven rows were mush.
import subprocess, sys
from PIL import Image, ImageDraw, ImageFont
sys.path.insert(0, 'dolphin')
from ansi import WIDTH, SUBX, SUBY, RAMP, EDGY, photo_mask, foam_mask, sample, chars

out = sys.argv[1] if len(sys.argv) > 1 else 'dolphin/flipr'
FIGFONT = sys.argv[2] if len(sys.argv) > 2 else 'slant'
STYLE = sys.argv[3] if len(sys.argv) > 3 else 'photo'   # photo | outline
DCOLS, DROWS, DX = 78, 24, 0

keep, tone, w, h, px, near, far = photo_mask()
foam = foam_mask(w, h, px)
cov, ton = sample(keep, tone, w, h, DCOLS, DROWS)
fcov, _ = sample(foam, foam, w, h, DCOLS, DROWS)
ncov, _ = sample(near, near, w, h, DCOLS, DROWS)
farcov, _ = sample(far, far, w, h, DCOLS, DROWS)
if STYLE == 'outline':
    # the boundary only: the line-art reading, in colour
    edge = [[0.0] * len(keep[0]) for _ in range(len(keep))]
    for y in range(1, h - 1):
        for x in range(1, w - 1):
            if keep[y][x] and not (keep[y-1][x] and keep[y+1][x] and keep[y][x-1] and keep[y][x+1]):
                for dy in (-1, 0, 1):
                    for dx in (-1, 0, 1):
                        edge[y + dy][x + dx] = 1.0
    cov, _ = sample(edge, edge, w, h, DCOLS, DROWS)
glyphs = chars(cov, DCOLS, DROWS, RAMP + EDGY[1:])

# stretch the tone across the ramp: inside these animals the photograph's
# range is narrow, and unstretched every cell came out the same blue
vals = sorted(t for row in ton for t in row if t > 0)
lo = vals[int(len(vals) * 0.06)] if vals else 0.0
hi = vals[int(len(vals) * 0.94)] if vals else 1.0

word = subprocess.run(['figlet', '-f', FIGFONT, '-w', '60', 'flipr'],
                      capture_output=True, text=True).stdout.rstrip('\n').split('\n')
ROWS = DROWS

# each animal gets its own family, which is the only thing that says there
# are two of them rather than one long shape
def sea(t, front):
    # deep ends lifted clear of the record's band, so the animals never fall
    # into the same lightness as the text behind them
    a, b = ((92, 134, 206), (210, 234, 250)) if front else ((64, 126, 150), (146, 198, 208))
    t = max(0.0, min(1.0, (t - lo) / max(1e-6, hi - lo)))
    return tuple(int(b[i] + (a[i] - b[i]) * t) for i in range(3))

FOAM = (150, 198, 224)
GOLD = (226, 170, 84)

# a flag record, printed behind the animals: the dolphins are drawn after it
# and keep every cell they hold, so the art sits on top of the data and it is
# plain that this is art.
BLOB = '''{
  "flag": {
    "key": "nightly.enabled",
    "value": {"boolValue": false},
    "description": "on: the nightly runs. off: it does not.",
    "expensive": true,
    "updatedAt": "2026-09-16T01:23:08.706697Z"
  },
  "revision": "3"
}'''.split('\n')
BLOBX, BLOBY = 3, 2
def oklab_l(rgb):
    """oklab lightness, ottosson's matrices, so the two bands can be set by
    measurement rather than by eye."""
    def lin(v):
        v /= 255
        return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    r, g, b = (lin(v) for v in rgb)
    l = (0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b) ** (1 / 3)
    m = (0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b) ** (1 / 3)
    s_ = (0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b) ** (1 / 3)
    return 0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s_

def at_l(rgb, want):
    """the same colour taken to a given lightness, by scaling toward black or
    white; close enough for a background and it keeps the hue."""
    cur = oklab_l(rgb)
    if cur <= 0:
        return tuple(int(255 * want) for _ in range(3))
    lo, hi = 0.0, 3.0
    for _ in range(24):
        k = (lo + hi) / 2
        test = tuple(min(255, int(v * k)) for v in rgb)
        if oklab_l(test) < want:
            lo = k
        else:
            hi = k
    return tuple(min(255, int(v * (lo + hi) / 2)) for v in rgb)

# the record lives a clear step below the animals: it is the page they swim
# over, and at equal lightness it was competing with them
PUNCT = at_l((104, 112, 146), 0.26)
KEY = at_l((120, 142, 180), 0.33)
STR = at_l((104, 146, 144), 0.33)
LIT = at_l((160, 136, 96), 0.32)

def blob_colour(line, i):
    """muted syntax colour: keys, strings, literals and punctuation, all dim
    enough that the animals stay the subject."""
    before = line[:i]
    quotes = before.count('"')
    if quotes % 2 == 1:
        colon = line.find(':')
        return KEY if colon == -1 or i < colon else STR
    ch = line[i]
    if ch.isalnum() or ch in '.-':
        return LIT
    return PUNCT

cells = [[(' ', None)] * WIDTH for _ in range(ROWS)]
for i, line in enumerate(BLOB):
    r = BLOBY + i
    if r >= ROWS:
        break
    for j, ch in enumerate(line):
        if ch != ' ' and BLOBX + j < WIDTH:
            cells[r][BLOBX + j] = (ch, blob_colour(line, j))
for r in range(DROWS):
    for c in range(DCOLS):
        g = glyphs[r][c]
        if g != ' ':
            n = sum(ncov[r * SUBY + y][c * SUBX + x]
                    for y in range(SUBY) for x in range(SUBX))
            f2 = sum(farcov[r * SUBY + y][c * SUBX + x]
                     for y in range(SUBY) for x in range(SUBX))
            cells[r][DX + c] = (g, sea(ton[r][c], n >= f2))
            continue
        f = sum(fcov[r * SUBY + y][c * SUBX + x]
                for y in range(SUBY) for x in range(SUBX)) / (SUBX * SUBY)
        if f > 0.30:
            cells[r][DX + c] = ("'", FOAM)
        elif f > 0.13:
            cells[r][DX + c] = ('.', FOAM)
# the wordmark sits inside the picture's own empty corner, as cassowary's
# label does, rather than below it as a separate block
WTOP = ROWS - len(word) - 1
for i, l in enumerate(word):
    for j, g in enumerate(l):
        if g != ' ' and 1 + j < WIDTH:
            cells[WTOP + i][1 + j] = (g, GOLD)

lines = []
for row in cells:
    s, last, trail = '', None, ''
    for g, col in row:
        if g == ' ':
            trail += ' '; continue
        s += trail; trail = ''
        if col != last:
            s += '\033[38;2;%d;%d;%dm' % col; last = col
        s += g
    lines.append(s + ('\033[0m' if last else ''))
open(out + '.ansi', 'w').write('\n'.join(lines) + '\n')

pw, ph = 9, 18
img = Image.new('RGB', (WIDTH * pw, ROWS * ph), (23, 15, 45))
d = ImageDraw.Draw(img)
pf = ImageFont.truetype('/System/Library/Fonts/Menlo.ttc', 15)
for i, row in enumerate(cells):
    for j, (g, col) in enumerate(row):
        if g != ' ':
            d.text((j * pw, i * ph - 2), g, font=pf, fill=col)
img.save(out + '.png')
print(out, WIDTH, 'x', ROWS)
