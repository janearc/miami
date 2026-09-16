# ascii by template matching: each cell takes the character whose drawn shape
# best matches what is in that cell, so an edge picks an edge character and a
# solid picks a dense one. usage: ascii.py COLS ROWS OUT [--spray]
import colorsys, sys
from collections import deque
from PIL import Image, ImageDraw, ImageFont

SRC = '/Users/jane/.claude/image-cache/42bdcb2f-9043-4d2f-9889-5284f91e8cd3/4.png'
FONT = '/System/Library/Fonts/Menlo.ttc'
WIDTH = 78
SUBX, SUBY = 6, 12          # samples per cell used for matching
GLYPHS = " .,'`:;-_~^\"*+=<>/\\|()il!coshv%#@&"

cols = int(sys.argv[1]); rows = int(sys.argv[2]); out = sys.argv[3]
style = sys.argv[4] if len(sys.argv) > 4 else 'filled'   # filled | outline

# ---- the animals, as in trace.py: dark and not very blue, two largest pieces
src = Image.open(SRC).convert('RGB')
w, h = src.size
px = src.load()
mask = [[False] * w for _ in range(h)]
for y in range(h):
    for x in range(w):
        r, g, b = px[x, y]
        v = max(r, g, b) / 255
        mx, mn = max(r, g, b), min(r, g, b)
        s = 0 if mx == 0 else (mx - mn) / mx
        mask[y][x] = v < 0.52 and s < 0.58
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
# fill highlight holes
outside = [[False] * w for _ in range(h)]
q = deque()
for x in range(w):
    for y in (0, h - 1):
        if not keep[y][x]:
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

# foam: bright, low-saturation pixels below the animals say "leaping"
foam = [[False] * w for _ in range(h)]
for y in range(h):
    for x in range(w):
        r, g, b = px[x, y]
        v = max(r, g, b) / 255
        mx, mn = max(r, g, b), min(r, g, b)
        sat = 0 if mx == 0 else (mx - mn) / mx
        foam[y][x] = v > 0.88 and sat < 0.20 and y > 250

xs = [x for y in range(h) for x in range(w) if keep[y][x]]
ys = [y for y in range(h) for x in range(w) if keep[y][x]]
x0, x1, y0, y1 = min(xs), max(xs) + 1, min(ys), max(ys) + 1

# ---- the cell field: coverage at SUBX by SUBY per cell
field = [[0.0] * (cols * SUBX) for _ in range(rows * SUBY)]
foamf = [[0.0] * (cols * SUBX) for _ in range(rows * SUBY)]
for r in range(rows * SUBY):
    sy0 = y0 + (y1 - y0) * r // (rows * SUBY)
    sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // (rows * SUBY))
    for c in range(cols * SUBX):
        sx0 = x0 + (x1 - x0) * c // (cols * SUBX)
        sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // (cols * SUBX))
        area = (sx1 - sx0) * (sy1 - sy0)
        n = sum(1 for yy in range(sy0, sy1) for xx in range(sx0, sx1) if keep[yy][xx])
        field[r][c] = n / area
        if n == 0:
            f = sum(1 for yy in range(sy0, sy1) for xx in range(sx0, sx1) if foam[yy][xx])
            foamf[r][c] = f / area

# ---- glyph atlas, drawn in the face a terminal will use
font = ImageFont.truetype(FONT, 16)
cellw, cellh = 10, 20
atlas = {}
for g in GLYPHS:
    im = Image.new('L', (cellw, cellh), 0)
    ImageDraw.Draw(im).text((0, -2), g, font=font, fill=255)
    small = im.resize((SUBX, SUBY), Image.BOX)
    atlas[g] = [small.getpixel((x, y)) / 255 for y in range(SUBY) for x in range(SUBX)]

def pick(vals):
    if max(vals) < 0.10:
        return ' '
    best, bestErr = ' ', None
    for g, t in atlas.items():
        e = sum((vals[i] - t[i]) ** 2 for i in range(len(vals)))
        if bestErr is None or e < bestErr:
            best, bestErr = g, e
    return best

def cover(r, c):
    return sum(field[r * SUBY + y][c * SUBX + x]
               for y in range(SUBY) for x in range(SUBX)) / (SUBX * SUBY)

lines = []
for r in range(rows):
    line = []
    for c in range(cols):
        vals = [field[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
        ch = pick(vals)
        if style == 'outline' and ch != ' ':
            # inside, away from any edge, the body empties out: the outline is
            # what carries the animal, and the hollow is the negative space
            inside = all(cover(min(max(r + dr, 0), rows - 1),
                               min(max(c + dc, 0), cols - 1)) > 0.92
                         for dr in (-1, 0, 1) for dc in (-1, 0, 1))
            if inside:
                ch = ' '
        if ch == ' ':
            f = sum(foamf[r * SUBY + y][c * SUBX + x]
                    for y in range(SUBY) for x in range(SUBX)) / (SUBX * SUBY)
            if f > 0.34:
                ch = "'"
            elif f > 0.16:
                ch = '.'
        line.append(ch)
    lines.append(''.join(line).rstrip())

art = [(' ' * (WIDTH - cols) + l).rstrip() for l in lines]
open(out + '.txt', 'w').write('\n'.join(art) + '\n')

pw, ph = 9, 18
img = Image.new('RGB', (WIDTH * pw, len(art) * ph), (250, 249, 246))
d = ImageDraw.Draw(img)
pf = ImageFont.truetype(FONT, 15)
for i, l in enumerate(art):
    for j, ch in enumerate(l):
        if ch != ' ':
            d.text((j * pw, i * ph - 2), ch, font=pf, fill=(24, 22, 28))
img.save(out + '.png')
print(out, cols, 'x', rows)
