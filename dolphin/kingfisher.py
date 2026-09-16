# the kingfisher header: colour character art from the photograph. unlike the
# dolphins this is a colour subject, so the glyph carries tone and the cell's
# own colour carries the teal crown, the orange cheek and the white throat.
import colorsys, subprocess, sys
from collections import deque
from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, 'dolphin')
from ansi import SUBX, SUBY, RAMP, EDGY, atlas

SRC = '/Users/jane/.claude/image-cache/42bdcb2f-9043-4d2f-9889-5284f91e8cd3/5.png'
WIDTH = 78
BCOLS = int(sys.argv[1]) if len(sys.argv) > 1 else 46
FIGFONT = sys.argv[2] if len(sys.argv) > 2 else 'small'
out = sys.argv[3] if len(sys.argv) > 3 else 'dolphin/kingfisher'
TRIM = float(sys.argv[4]) if len(sys.argv) > 4 else 0.0   # cut this much off the left
PLACE = sys.argv[5] if len(sys.argv) > 5 else 'right'      # which side the bird sits
WORDX = int(sys.argv[6]) if len(sys.argv) > 6 else 1        # where the wordmark starts

im = Image.open(SRC).convert('RGB')
w, h = im.size
px = im.load()

# trim the photograph's black frame
def bright_row(y):
    return sum(sum(px[x, y]) for x in range(0, w, 7)) / (3 * len(range(0, w, 7))) > 40
top = next(y for y in range(h) if bright_row(y))
bot = next(y for y in range(h - 1, -1, -1) if bright_row(y))
def bright_col(x):
    return sum(sum(px[x, y]) for y in range(top, bot, 7)) / (3 * len(range(top, bot, 7))) > 40
left = next(x for x in range(w) if bright_col(x))
right = next(x for x in range(w - 1, -1, -1) if bright_col(x))

# the sky is pale, blue-leaning and unsaturated; the white ear patch is pale
# but not blue, which is why blue-dominance and not lightness decides
bird = [[0.0] * w for _ in range(h)]
for y in range(top, bot + 1):
    for x in range(left, right + 1):
        r, g, b = px[x, y]
        mx, mn = max(r, g, b), min(r, g, b)
        s = 0 if mx == 0 else (mx - mn) / mx
        sky = (b - r) > 20 and s < 0.32 and mx / 255 > 0.62
        bird[y][x] = 0.0 if sky else 1.0
seen = [[False] * w for _ in range(h)]
best = []
for y in range(top, bot + 1):
    for x in range(left, right + 1):
        if not bird[y][x] or seen[y][x]:
            continue
        q = deque([(x, y)]); seen[y][x] = True; c = []
        while q:
            cx, cy = q.popleft(); c.append((cx, cy))
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                nx, ny = cx + dx, cy + dy
                if left <= nx <= right and top <= ny <= bot and bird[ny][nx] and not seen[ny][nx]:
                    seen[ny][nx] = True; q.append((nx, ny))
        if len(c) > len(best):
            best = c
keep = [[0.0] * w for _ in range(h)]
for x, y in best:
    keep[y][x] = 1.0
xs = [p[0] for p in best]; ys = [p[1] for p in best]
x0, x1, y0, y1 = min(xs), max(xs) + 1, min(ys), max(ys) + 1
# the head and the bill are the bird; the shoulder behind only costs width
x0 += int((x1 - x0) * TRIM)
BROWS = max(1, round(BCOLS * (y1 - y0) / (x1 - x0) / 2))

cov = [[0.0] * (BCOLS * SUBX) for _ in range(BROWS * SUBY)]
for r in range(BROWS * SUBY):
    sy0 = y0 + (y1 - y0) * r // (BROWS * SUBY); sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // (BROWS * SUBY))
    for c in range(BCOLS * SUBX):
        sx0 = x0 + (x1 - x0) * c // (BCOLS * SUBX); sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // (BCOLS * SUBX))
        n = (sx1 - sx0) * (sy1 - sy0)
        cov[r][c] = sum(keep[y][x] for y in range(sy0, sy1) for x in range(sx0, sx1)) / n

colour = [[None] * BCOLS for _ in range(BROWS)]
tone = [[0.0] * BCOLS for _ in range(BROWS)]
for r in range(BROWS):
    sy0 = y0 + (y1 - y0) * r // BROWS; sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // BROWS)
    for c in range(BCOLS):
        sx0 = x0 + (x1 - x0) * c // BCOLS; sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // BCOLS)
        acc = [0, 0, 0]; n = 0
        for y in range(sy0, sy1):
            for x in range(sx0, sx1):
                if keep[y][x]:
                    p = px[x, y]
                    acc[0] += p[0]; acc[1] += p[1]; acc[2] += p[2]; n += 1
        if n:
            rgb = tuple(v // n for v in acc)
            colour[r][c] = rgb
            tone[r][c] = 1 - (0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2]) / 255

A = atlas(RAMP + EDGY[1:])

def glyph(r, c):
    v = [cov[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
    if max(v) < 0.10:
        return ' '
    full = sum(v) / len(v)
    if full > 0.85:
        # inside the bird the shape is solid, so the character says how dark
        # the plumage is there and the colour says what it is
        # a gamma on the tone so pale plumage thins to sparse marks and the
        # dark crown and bill stay solid: a flat mapping filled every cell
        t = tone[r][c] ** 1.25
        i = min(len(RAMP) - 1, int(t * (len(RAMP) - 1) + 0.5))
        return RAMP[i]
    best, err = ' ', None
    for g, t in A.items():
        e = sum((v[i] - t[i]) ** 2 for i in range(len(v)))
        if err is None or e < err:
            best, err = g, e
    return best

# colour comes from a few families, not from per-cell photograph averages.
# averaging gave a muddy slab; the flipr header reads because each thing in it
# belongs to one of a small number of colours, shaded by tone.
FAMILIES = {
    'cream': ((242, 236, 220), (188, 176, 152)),
    'orange': ((243, 160, 54), (146, 80, 20)),
    'teal': ((96, 206, 204), (26, 92, 112)),
    # slate, not black: the bill and the eye are the darkest things in the
    # photograph and would disappear into a dark terminal
    'dark': ((168, 176, 196), (96, 104, 124)),
}

def family(rgb):
    h_, s_, v_ = colorsys.rgb_to_hsv(*[q / 255 for q in rgb])
    deg = h_ * 360
    if v_ < 0.26:
        return 'dark'
    if s_ < 0.22 and v_ > 0.62:
        return 'cream'
    if 12 <= deg <= 52 and s_ > 0.30:
        return 'orange'
    if 140 <= deg <= 215:
        return 'teal'
    if v_ < 0.40:
        return 'dark'
    return 'teal'

def lift(rgb, t):
    light, deep = FAMILIES[family(rgb)]
    t = max(0.0, min(1.0, (t - 0.22) / 0.50))
    return tuple(int(light[i] + (deep[i] - light[i]) * t) for i in range(3))

word = subprocess.run(['figlet', '-f', FIGFONT, '-w', '78', 'kingfisher'],
                      capture_output=True, text=True).stdout.rstrip('\n').split('\n')
# the wordmark tucks under the head and the bill crosses in front of it: the
# bird is drawn first and keeps every cell it holds, so the beak reads as
# passing over the word
OVERLAP = 3
ROWS = BROWS + len(word) + 1 - OVERLAP
cells = [[(' ', None)] * WIDTH for _ in range(ROWS)]
OX = WIDTH - BCOLS if PLACE == 'right' else 0
for r in range(BROWS):
    for c in range(BCOLS):
        g = glyph(r, c)
        if g != ' ' and colour[r][c]:
            cells[r][OX + c] = (g, lift(colour[r][c], tone[r][c]))
# the wordmark goes in the largest empty corner, which for a head facing
# down-right is the bottom left
# the eye, placed from the photograph rather than hoped for: at this size it
# is one cell, and without it the head is a patterned lump
EYEX, EYEY = 448, 272
ec = OX + int((EYEX - x0) / (x1 - x0) * BCOLS)
er = int((EYEY - y0) / (y1 - y0) * BROWS)
if 0 <= er < BROWS and 0 <= ec < BCOLS:
    cells[er][ec] = ('o', (214, 220, 234))

# the wordmark sits in the emptiest corner, as flipr's does: opposite the
# bird, along the bottom
GOLD = (236, 164, 62)
wx = WORDX
wtop = BROWS - OVERLAP
for i, l in enumerate(word):
    for j, g in enumerate(l):
        if g != ' ' and 0 <= wx + j < WIDTH and cells[wtop + i][wx + j][0] == ' ':
            cells[wtop + i][wx + j] = (g, GOLD)

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
print(out, WIDTH, 'x', ROWS, '| bird', BCOLS, 'x', BROWS, '| crop', x1 - x0, 'x', y1 - y0)
