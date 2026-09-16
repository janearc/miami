# a woodcut: solid interior, edges matched to the characters that draw them.
# a slender animal at manpage width is under four rows thick, so the edges are
# the whole read. usage: woodcut.py GREY.png COLS ROWS OUT
import sys
from PIL import Image, ImageDraw, ImageFont

FONT = '/System/Library/Fonts/Menlo.ttc'
WIDTH = 78
SUBX, SUBY = 6, 12
EDGES = " .:-=+*'\"^_/\\|(),"

grey, cols, rows, out = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), sys.argv[4]
im = Image.open(grey).convert('L')
w, h = im.size
px = im.load()

field = [[0.0] * (cols * SUBX) for _ in range(rows * SUBY)]
for r in range(rows * SUBY):
    sy0 = h * r // (rows * SUBY); sy1 = max(sy0 + 1, h * (r + 1) // (rows * SUBY))
    for c in range(cols * SUBX):
        sx0 = w * c // (cols * SUBX); sx1 = max(sx0 + 1, w * (c + 1) // (cols * SUBX))
        n = (sx1 - sx0) * (sy1 - sy0)
        on = sum(1 for y in range(sy0, sy1) for x in range(sx0, sx1) if px[x, y] < 205)
        field[r][c] = on / n

font = ImageFont.truetype(FONT, 16)
atlas = {}
for g in EDGES:
    tile = Image.new('L', (10, 20), 0)
    ImageDraw.Draw(tile).text((0, -2), g, font=font, fill=255)
    small = tile.resize((SUBX, SUBY), Image.BOX)
    atlas[g] = [small.getpixel((x, y)) / 255 for y in range(SUBY) for x in range(SUBX)]

lines = []
for r in range(rows):
    line = []
    for c in range(cols):
        vals = [field[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
        cover = sum(vals) / len(vals)
        if cover > 0.82:
            line.append('#'); continue
        if cover < 0.10:
            line.append(' '); continue
        best, err = ' ', None
        for g, t in atlas.items():
            e = sum((vals[i] - t[i]) ** 2 for i in range(len(vals)))
            if err is None or e < err:
                best, err = g, e
        line.append(best)
    lines.append(''.join(line))

art = [(' ' * (WIDTH - cols) + ''.join(l)).rstrip() for l in lines]
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
