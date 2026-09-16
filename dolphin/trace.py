# traces the dolphins out of the photo into ascii, cassowary style: a density
# ramp, the animals on the right, everything else empty. run:
#   python3 dolphin/trace.py COLS ROWS PAD OUT
import colorsys, sys
from collections import deque
from PIL import Image, ImageDraw, ImageFont

SRC = '/Users/jane/.claude/image-cache/42bdcb2f-9043-4d2f-9889-5284f91e8cd3/4.png'
RAMP = ' .:-=+*#%@'          # cassowary's, lightest to densest
WIDTH = 78                    # the manpage column

cols = int(sys.argv[1]) if len(sys.argv) > 1 else 44
rows = int(sys.argv[2]) if len(sys.argv) > 2 else 14
pad = int(sys.argv[3]) if len(sys.argv) > 3 else 0     # blank rows above
out = sys.argv[4] if len(sys.argv) > 4 else 'dolphin/dolphins'

src = Image.open(SRC).convert('RGB')
w, h = src.size
px = src.load()

# an animal is dark and not very blue; the sea is dark but heavily saturated
mask = [[False] * w for _ in range(h)]
for y in range(h):
    for x in range(w):
        r, g, b = px[x, y]
        hsv = colorsys.rgb_to_hsv(r / 255, g / 255, b / 255)
        mask[y][x] = hsv[2] < 0.52 and hsv[1] < 0.58

# keep the two largest pieces: the specks are splash and horizon
seen = [[False] * w for _ in range(h)]
comps = []
for y in range(h):
    for x in range(w):
        if not mask[y][x] or seen[y][x]:
            continue
        q = deque([(x, y)]); seen[y][x] = True; cells = []
        while q:
            cx, cy = q.popleft(); cells.append((cx, cy))
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                nx, ny = cx + dx, cy + dy
                if 0 <= nx < w and 0 <= ny < h and mask[ny][nx] and not seen[ny][nx]:
                    seen[ny][nx] = True; q.append((nx, ny))
        comps.append(cells)
comps.sort(key=len, reverse=True)
keep = [[False] * w for _ in range(h)]
for c in comps[:2]:
    for x, y in c:
        keep[y][x] = True

# fill the holes the highlights punched, so the body is solid and only its
# tone varies
outside = [[False] * w for _ in range(h)]
q = deque()
for x in range(w):
    for y in (0, h - 1):
        if not keep[y][x] and not outside[y][x]:
            outside[y][x] = True; q.append((x, y))
for y in range(h):
    for x in (0, w - 1):
        if not keep[y][x] and not outside[y][x]:
            outside[y][x] = True; q.append((x, y))
while q:
    cx, cy = q.popleft()
    for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
        nx, ny = cx + dx, cy + dy
        if 0 <= nx < w and 0 <= ny < h and not keep[ny][nx] and not outside[ny][nx]:
            outside[ny][nx] = True; q.append((nx, ny))
for y in range(h):
    for x in range(w):
        if not outside[y][x]:
            keep[y][x] = True

xs = [x for y in range(h) for x in range(w) if keep[y][x]]
ys = [y for y in range(h) for x in range(w) if keep[y][x]]
x0, x1, y0, y1 = min(xs), max(xs) + 1, min(ys), max(ys) + 1

# one character per cell: coverage says how much animal is here, tone says how
# dark it is. an empty cell stays a space, which is the negative space.
lines = []
for r in range(rows):
    line = []
    for c in range(cols):
        sx0 = x0 + (x1 - x0) * c // cols
        sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // cols)
        sy0 = y0 + (y1 - y0) * r // rows
        sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // rows)
        n = 0; tot = 0; lum = 0.0
        for y in range(sy0, sy1):
            for x in range(sx0, sx1):
                tot += 1
                if keep[y][x]:
                    n += 1
                    r8, g8, b8 = px[x, y]
                    lum += (0.2126 * r8 + 0.7152 * g8 + 0.0722 * b8) / 255
        if n == 0:
            line.append(' '); continue
        cover = n / tot
        tone = 1 - (lum / n)                    # dark animal, dense character
        # these animals are almost uniformly dark, so tone inside the body
        # says very little and a coverage ramp only blurs the outline. the
        # boundary is thresholded instead, and tone shades within the body.
        if cover < 0.18:
            line.append(' '); continue
        if cover < 0.55:
            line.append('=' if cover > 0.36 else '-'); continue
        body = ' *#%@'
        k = 1 + min(3, max(0, int(round((tone - 0.45) / 0.18))))
        line.append(body[k])
    lines.append(''.join(line).rstrip())

art = [''] * pad + [(' ' * (WIDTH - cols) + l).rstrip() for l in lines]
text = '\n'.join(art) + '\n'
open(out + '.txt', 'w').write(text)

# a preview in a real monospace face, because reading art as characters is
# not reading it
cw, ch = 9, 18
font = ImageFont.truetype('/System/Library/Fonts/Menlo.ttc', 15)
img = Image.new('RGB', (WIDTH * cw, len(art) * ch), (250, 249, 246))
d = ImageDraw.Draw(img)
for i, l in enumerate(art):
    for j, chx in enumerate(l):
        if chx != ' ':
            d.text((j * cw, i * ch - 2), chx, font=font, fill=(24, 22, 28))
img.save(out + '.png')
print(f'{cols}x{rows} art, {len(art)} lines, widest {max(len(l) for l in art)}')
