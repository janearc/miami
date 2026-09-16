# ascii from a greyscale render: template matching against a menlo atlas, so
# edges take edge characters and the body takes the ramp.
# usage: fromgrey.py PNG COLS ROWS OUT
import sys
from PIL import Image, ImageDraw, ImageFont

FONT = '/System/Library/Fonts/Menlo.ttc'
WIDTH = 78
SUBX, SUBY = 6, 12
# cassowary's ramp, plus the few characters that draw an edge. the wider set
# put & and c and s through the middle of the animals and read as noise.
GLYPHS = " .:-=+*#%@'\"^_/\\"

src, cols, rows, out = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4]
outline = '--outline' in sys.argv
im = Image.open(src).convert('L')
w, h = im.size
px = im.load()

if outline:
    # the boundary of the shape, at source resolution: a slender animal reads
    # as line art at this size, where a filled ramp reads as a dark smear
    ink = [[1 if px[x, y] < 200 else 0 for x in range(w)] for y in range(h)]
    edge = Image.new('L', (w, h), 0)
    ep = edge.load()
    for y in range(1, h - 1):
        for x in range(1, w - 1):
            if ink[y][x] and not (ink[y - 1][x] and ink[y + 1][x] and ink[y][x - 1] and ink[y][x + 1]):
                for dy in (-1, 0, 1):
                    for dx in (-1, 0, 1):
                        ep[x + dx, y + dy] = 255
    im = Image.eval(edge, lambda v: 255 - v)
    px = im.load()

field = [[0.0] * (cols * SUBX) for _ in range(rows * SUBY)]
for r in range(rows * SUBY):
    sy0 = h * r // (rows * SUBY); sy1 = max(sy0 + 1, h * (r + 1) // (rows * SUBY))
    for c in range(cols * SUBX):
        sx0 = w * c // (cols * SUBX); sx1 = max(sx0 + 1, w * (c + 1) // (cols * SUBX))
        n = (sx1 - sx0) * (sy1 - sy0)
        tot = sum(px[x, y] for y in range(sy0, sy1) for x in range(sx0, sx1))
        field[r][c] = 1 - (tot / n) / 255      # ink, 0 paper to 1 black

font = ImageFont.truetype(FONT, 16)
atlas = {}
for g in GLYPHS:
    tile = Image.new('L', (10, 20), 0)
    ImageDraw.Draw(tile).text((0, -2), g, font=font, fill=255)
    small = tile.resize((SUBX, SUBY), Image.BOX)
    atlas[g] = [small.getpixel((x, y)) / 255 for y in range(SUBY) for x in range(SUBX)]

lines = []
for r in range(rows):
    line = []
    for c in range(cols):
        vals = [field[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
        if max(vals) < 0.10:
            line.append(' '); continue
        best, err = ' ', None
        for g, t in atlas.items():
            e = sum((vals[i] - t[i]) ** 2 for i in range(len(vals)))
            if err is None or e < err:
                best, err = g, e
        line.append(best)
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
