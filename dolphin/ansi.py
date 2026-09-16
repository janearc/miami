# colour character art: the grain of ascii, with a colour per cell. one
# medium for the wordmark and the animals, default background throughout so
# the negative space is the reader's own terminal.
import colorsys, sys
from collections import deque
from PIL import Image, ImageDraw, ImageFont

FONT = '/System/Library/Fonts/Menlo.ttc'
PHOTO = '/Users/jane/.claude/image-cache/42bdcb2f-9043-4d2f-9889-5284f91e8cd3/4.png'
GREY = 'out/dolphins-grey.png'
WIDTH = 78
SUBX, SUBY = 6, 12
RAMP = " .:-=+*#%@"
EDGY = " .:-=+*'\"^_/\\|(),"

def atlas(glyphs):
    f = ImageFont.truetype(FONT, 16)
    out = {}
    for g in glyphs:
        tile = Image.new('L', (10, 20), 0)
        ImageDraw.Draw(tile).text((0, -2), g, font=f, fill=255)
        s = tile.resize((SUBX, SUBY), Image.BOX)
        out[g] = [s.getpixel((x, y)) / 255 for y in range(SUBY) for x in range(SUBX)]
    return out

def foam_mask(w, h, px):
    """the splash: bright and nearly colourless, below the animals."""
    import colorsys
    m = [[0.0] * w for _ in range(h)]
    for y in range(250, h):
        for x in range(w):
            r, g, b = px[x, y]
            hsv = colorsys.rgb_to_hsv(r / 255, g / 255, b / 255)
            if hsv[2] > 0.86 and hsv[1] < 0.22:
                m[y][x] = 1.0
    return m


def photo_mask():
    im = Image.open(PHOTO).convert('RGB')
    w, h = im.size
    px = im.load()
    m = [[False] * w for _ in range(h)]
    for y in range(h):
        for x in range(w):
            r, g, b = px[x, y]
            hsv = colorsys.rgb_to_hsv(r / 255, g / 255, b / 255)
            m[y][x] = hsv[2] < 0.52 and hsv[1] < 0.58
    seen = [[False] * w for _ in range(h)]
    comps = []
    for y in range(h):
        for x in range(w):
            if not m[y][x] or seen[y][x]:
                continue
            q = deque([(x, y)]); seen[y][x] = True; c = []
            while q:
                cx, cy = q.popleft(); c.append((cx, cy))
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    nx, ny = cx + dx, cy + dy
                    if 0 <= nx < w and 0 <= ny < h and m[ny][nx] and not seen[ny][nx]:
                        seen[ny][nx] = True; q.append((nx, ny))
            comps.append(c)
    comps.sort(key=len, reverse=True)
    keep = [[0.0] * w for _ in range(h)]
    near = [[0.0] * w for _ in range(h)]   # the leading animal only
    far = [[0.0] * w for _ in range(h)]
    for i, c in enumerate(comps[:2]):
        for x, y in c:
            keep[y][x] = 1.0
            (near if i == 0 else far)[y][x] = 1.0
    # tone from the photograph, so the bodies keep their grain
    tone = [[0.0] * w for _ in range(h)]
    for y in range(h):
        for x in range(w):
            if keep[y][x]:
                r, g, b = px[x, y]
                tone[y][x] = 1 - (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255
    return keep, tone, w, h, px, near, far

def grey_mask():
    im = Image.open(GREY).convert('L')
    w, h = im.size
    px = im.load()
    keep = [[1.0 if px[x, y] < 205 else 0.0 for x in range(w)] for y in range(h)]
    tone = [[(1 - px[x, y] / 255) if keep[y][x] else 0.0 for x in range(w)] for y in range(h)]
    return keep, tone, w, h

def sample(keep, tone, w, h, cols, rows):
    """coverage and mean tone per subcell, plus per-cell means."""
    cov = [[0.0] * (cols * SUBX) for _ in range(rows * SUBY)]
    ton = [[0.0] * cols for _ in range(rows)]
    for r in range(rows * SUBY):
        y0 = h * r // (rows * SUBY); y1 = max(y0 + 1, h * (r + 1) // (rows * SUBY))
        for c in range(cols * SUBX):
            x0 = w * c // (cols * SUBX); x1 = max(x0 + 1, w * (c + 1) // (cols * SUBX))
            n = (x1 - x0) * (y1 - y0)
            cov[r][c] = sum(keep[y][x] for y in range(y0, y1) for x in range(x0, x1)) / n
    for r in range(rows):
        y0 = h * r // rows; y1 = max(y0 + 1, h * (r + 1) // rows)
        for c in range(cols):
            x0 = w * c // cols; x1 = max(x0 + 1, w * (c + 1) // cols)
            vals = [tone[y][x] for y in range(y0, y1) for x in range(x0, x1) if keep[y][x]]
            ton[r][c] = sum(vals) / len(vals) if vals else 0.0
    return cov, ton

def chars(cov, cols, rows, glyphs):
    a = atlas(glyphs)
    out = []
    for r in range(rows):
        row = []
        for c in range(cols):
            v = [cov[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
            if max(v) < 0.10:
                row.append(' '); continue
            best, err = ' ', None
            for g, t in a.items():
                e = sum((v[i] - t[i]) ** 2 for i in range(len(v)))
                if err is None or e < err:
                    best, err = g, e
            row.append(best)
        out.append(row)
    return out
