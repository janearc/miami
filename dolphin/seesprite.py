# renders a .sprite or .stream straight to png, without going through the
# player. the player is still the thing that says whether it plays; this is
# for the fifty looks in between, where waiting on a subprocess per frame is
# the slow part.
#
# a daffy pixel is one column by half a row, which is close to square, so a
# pixel here is a square block and no aspect correction is applied.
import re, sys
from PIL import Image, ImageDraw

# role character -> colour, read out of made/starling.css so the two agree
CSSROLE = {
    'none': '.', 'dark': 'K', 'wing': 'V', 'light': 'W', 'stripe': 'D',
    'beak-base': 'B', 'beak-band': 'Y', 'beak-tip': 'R', 'feet': 'O',
    'pupil': 'E', 'eye-ring': 'X',
}


def colourway(path):
    src = open(path).read()
    pal = {}
    for name, hexv in re.findall(r'--([a-z-]+):\s*#([0-9a-fA-F]{6})', src):
        if name in CSSROLE:
            pal[CSSROLE[name]] = tuple(int(hexv[i:i + 2], 16) for i in (0, 2, 4))
    return pal


def frames(path):
    # a .sprite is one art block; a .stream is a run of them. both are read
    # the same way: everything after an `art` line until a blank or keyword.
    out, cur, inart = [], None, False
    for line in open(path).read().splitlines():
        if line.strip() == 'art':
            cur, inart = [], True
            continue
        if inart:
            if line.strip() == '' or re.match(r'^(frame|sprite|pose)\b', line):
                out.append(cur)
                inart = False
                if re.match(r'^frame\b', line):
                    continue
            else:
                cur.append(line)
    if inart and cur:
        out.append(cur)
    return [f for f in out if f]


def render(grid, pal, px=6):
    w = max(len(r) for r in grid)
    im = Image.new('RGB', (w * px, len(grid) * px), pal.get('.', (0, 0, 0)))
    d = ImageDraw.Draw(im)
    for y, row in enumerate(grid):
        for x, ch in enumerate(row):
            if ch == '.':
                continue
            d.rectangle([x * px, y * px, x * px + px - 1, y * px + px - 1],
                        fill=pal.get(ch, (255, 0, 255)))
    return im


def sheet(fs, pal, px=6, cols=4, gap=8):
    ims = [render(f, pal, px) for f in fs]
    w = max(i.width for i in ims)
    h = max(i.height for i in ims)
    rows = (len(ims) + cols - 1) // cols
    n = min(cols, len(ims))
    out = Image.new('RGB', (n * (w + gap) + gap, rows * (h + gap) + gap), (20, 20, 24))
    for k, i in enumerate(ims):
        out.paste(i, (gap + (k % cols) * (w + gap), gap + (k // cols) * (h + gap)))
    return out


if __name__ == '__main__':
    src, dst = sys.argv[1], sys.argv[2]
    px = int(sys.argv[3]) if len(sys.argv) > 3 else 6
    pal = colourway('sprites/made/starling.css')
    fs = frames(src)
    (fs[0] and sheet(fs, pal, px) if len(fs) > 1 else render(fs[0], pal, px)).save(dst)
    print(f'{src}: {len(fs)} frame(s) -> {dst}')
