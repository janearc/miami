# one tool for the header set: photograph in, colour character art out, with
# the logotype inside the frame. per subject it needs three things -- how to
# tell subject from background, the few colours it is told in, and where the
# word sits. everything else is shared.
import colorsys, math, subprocess, sys
from collections import deque
from PIL import Image, ImageDraw, ImageFont

sys.path.insert(0, 'dolphin')
from ansi import SUBX, SUBY, RAMP, EDGY, atlas

# the sources live in the tree now. they were read from a session image cache
# until that cache was deleted mid-session and took all six photographs with
# it; they were recovered out of the transcript and copied here.
BASE = 'sources/'
WIDTH = 78
GOLD = (226, 170, 84)

def hsv(r, g, b):
    return colorsys.rgb_to_hsv(r / 255, g / 255, b / 255)

# the families each subject is told in: light end, deep end. tone runs between
# them, and which family a sample belongs to is decided by hue and chroma with
# lightness weighted down -- the same rule miami's palette uses, because plain
# oklab distance is mostly lightness and matched bright teal to cream.
SUBJECTS = {
    'puffin': dict(
        src='puffin.png', cols=54, place='left', wordx=18, overlap=2, fig='slant',
        # sky is anything clearly blue-dominant, which holds where it darkens
        # at the edges too, plus the green blur at the left. the bird's own
        # blue-white breast is only just blue and stays.
        bg=lambda r, g, b: (b - r > 30 and hsv(r, g, b)[1] > 0.28) or
                           (60 <= hsv(r, g, b)[0] * 360 <= 170 and hsv(r, g, b)[2] < 0.65),
        # head and bill only: with the breast in frame the black body took the
        # picture and the face and bill were a footnote
        keep=1, trim=(0.02, 0.0, 0.0, 0.38),
        families=[('ink', (96, 102, 118), (38, 42, 54)),
                  ('white', (252, 252, 250), (214, 218, 224)),
                  ('orange', (255, 158, 64), (208, 92, 20)),
                  ('grey', (186, 176, 170), (112, 104, 102))],
        roles='K W R B'.split(), eye=(437, 246),
    ),
    'starling': dict(
        src='starling.png', cols=62, place='centre', wordx=2, overlap=1, fig='slant',
        bg=lambda r, g, b: hsv(r, g, b)[2] > 0.92 and hsv(r, g, b)[1] < 0.08,
        keep=1, trim=(0.0, 0.0, 0.0, 0.0),
        families=[('teal', (112, 208, 190), (26, 86, 88)),
                  ('blue', (124, 162, 230), (38, 68, 126)),
                  ('tan', (220, 194, 154), (118, 94, 64)),
                  ('dark', (146, 152, 166), (58, 62, 74)),
                  ('yellow', (250, 214, 110), (174, 138, 40))],
        roles='K V W D R'.split(),
    ),
    'game': dict(
        src='game.png', cols=64, place='left', wordx=30, overlap=2, fig='smslant',
        # sky is blue-dominant AND bright: the navy helmet is blue too, and
        # without the brightness test it went out with the sky
        bg=lambda r, g, b: b - r > 28 and hsv(r, g, b)[2] > 0.55 and hsv(r, g, b)[1] > 0.20,
        keep=1, keepmin=0.004, trim=(0.0, 0.0, 0.0, 0.0), wordy=17,
        # the kit is bright and nearly featureless, so its range has to be
        # wide or the whole player comes out as one white slab
        families=[('kit', (252, 252, 250), (148, 150, 158)),
                  ('skin', (236, 178, 136), (112, 68, 48)),
                  ('navy', (124, 136, 184), (44, 48, 78)),
                  ('bat', (234, 202, 152), (150, 118, 74)),
                  ('ball', (240, 122, 104), (150, 52, 44))],
        # the eleven roles this subject's families become in a stencil
        roles='W D K Y R'.split(),
    ),
    'cassowary': dict(
        src='cassowary.jpeg', cols=34, place='left', wordx=38, wordy=6,
        overlap=2, fig='small',
        # the background is a dark green blur and the bird is neutral: green
        # clearly dominant and dark is background, and so is pure black. the
        # plumage samples at r=g=b, so it never trips the green test.
        bg=lambda r, g, b: (g >= r and g > b + 8 and r + g + b < 200) or (r + g + b < 30),
        keep=1, trim=(0.02, 0.0, 0.26, 0.40), framedark=0.995,
        families=[('casque', (156, 146, 134), (52, 48, 50)),
                  ('face', (204, 236, 255), (108, 158, 202)),
                  ('neck', (132, 162, 246), (34, 42, 122)),
                  ('wattle', (238, 132, 110), (138, 48, 54)),
                  ('plume', (112, 112, 114), (18, 18, 20)),
                  ('eye', (240, 202, 120), (150, 110, 50))],
        roles='V W D R K X'.split(),
    ),
    'kingfisher': dict(
        src='kingfisher.png', cols=58, place='right', wordx=22, overlap=3, fig='slant',
        bg=lambda r, g, b: (b - r) > 20 and hsv(r, g, b)[1] < 0.32 and max(r, g, b) / 255 > 0.62,
        keep=1, trim=(0.12, 0.0, 0.0, 0.0),
        families=[('teal', (96, 206, 204), (26, 92, 112)),
                  ('orange', (243, 160, 54), (146, 80, 20)),
                  ('cream', (242, 236, 220), (188, 176, 152)),
                  ('slate', (168, 176, 196), (96, 104, 124))],
        # crown and back are the dark mass, the cheek sits just off it, the
        # throat is the light mass, and the dagger is what protrudes
        roles='K V W R'.split(), eye=(448, 272),
    ),
    'daffy': dict(
        src='daffy.png', cols=52, place='right', wordx=2, overlap=2, fig='slant',
        # the feet are the only red-orange thing in the frame
        bg=lambda r, g, b: not ((hsv(r, g, b)[0] * 360 <= 28 or hsv(r, g, b)[0] * 360 >= 348)
                                and hsv(r, g, b)[1] > 0.28 and hsv(r, g, b)[2] > 0.22),
        keep=2, trim=(0.0, 0.0, 0.0, 0.0),
        families=[('orange', (255, 150, 108), (166, 70, 46)),
                  ('claw', (156, 158, 170), (74, 76, 88))],
        roles='O K'.split(),
    ),
}

def oklab(rgb):
    def lin(v):
        v /= 255
        return v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    r, g, b = (lin(v) for v in rgb)
    l = (0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b) ** (1 / 3)
    m = (0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b) ** (1 / 3)
    s = (0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b) ** (1 / 3)
    return (0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s,
            1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s,
            0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s)

def from_oklab(lab):
    L, a, b = lab
    l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
    m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
    s_ = (L - 0.0894841775 * a - 1.2914855480 * b) ** 3
    r = 4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s_
    g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s_
    bb = -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s_
    def enc(v):
        v = max(0.0, min(1.0, v))
        v = 12.92 * v if v <= 0.0031308 else 1.055 * v ** (1 / 2.4) - 0.055
        return int(round(max(0.0, min(1.0, v)) * 255))
    return (enc(r), enc(g), enc(bb))

# the ground the art is drawn on, and the floor every colour must clear to be
# seen against it. this is miami's rule: a colour within 0.18 oklab lightness
# of the background is not dark, it is absent.
GROUND_L = 0.20
FLOOR = GROUND_L + 0.18

def clear_ground(rgb):
    L, a, b = oklab(rgb)
    if L >= FLOOR:
        return rgb
    return from_oklab((FLOOR, a, b))

LIGHTWEIGHT = 0.3

def subject_mask(cfg, px, w, h):
    """the subject, and the box it lives in: frame trim, background rule,
    components, crop. one copy, used by the renderer and by the stencil."""
    # trim the frame the picture came with. by the share of near-black pixels
    # in a line, not by its average: a two pixel black edge inside a bright row
    # survives an average, and then rings the whole picture as "subject".
    def dark_share(pts):
        n = sum(1 for p in pts if sum(p) < 90)
        return n / max(1, len(pts))
    # the share at which a line counts as frame rather than picture. it is a
    # per-subject number because a photograph whose *background* is dark is
    # indistinguishable from a black border by this test: on the cassowary,
    # 0.55 ate the whole top of the image and with it the casque, which is the
    # one feature that identifies the bird.
    fd = cfg.get('framedark', 0.55)
    top = next(y for y in range(h) if dark_share([px[x, y] for x in range(0, w, 5)]) < fd)
    bot = next(y for y in range(h - 1, -1, -1) if dark_share([px[x, y] for x in range(0, w, 5)]) < fd)
    left = next(x for x in range(w) if dark_share([px[x, y] for y in range(top, bot, 5)]) < fd)
    right = next(x for x in range(w - 1, -1, -1) if dark_share([px[x, y] for y in range(top, bot, 5)]) < fd)
    # and stand two pixels clear of whatever edge is left
    top, bot, left, right = top + 2, bot - 2, left + 2, right - 2

    mask = [[0.0] * w for _ in range(h)]
    for y in range(top, bot + 1):
        for x in range(left, right + 1):
            if not cfg['bg'](*px[x, y]):
                mask[y][x] = 1.0
    seen = [[False] * w for _ in range(h)]
    comps = []
    for y in range(top, bot + 1):
        for x in range(left, right + 1):
            if not mask[y][x] or seen[y][x]:
                continue
            q = deque([(x, y)]); seen[y][x] = True; c = []
            while q:
                cx, cy = q.popleft(); c.append((cx, cy))
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    nx, ny = cx + dx, cy + dy
                    if left <= nx <= right and top <= ny <= bot and mask[ny][nx] and not seen[ny][nx]:
                        seen[ny][nx] = True; q.append((nx, ny))
            comps.append(c)
    comps.sort(key=len, reverse=True)
    # keep the named number of pieces, plus anything else big enough to be a
    # real part of the picture: the ball is its own component and was going in
    # the bin with the dust
    pieces = comps[:cfg['keep']]
    if cfg.get('keepmin'):
        floor_px = cfg['keepmin'] * (right - left) * (bot - top)
        pieces = [c for c in comps if len(c) >= floor_px] or pieces
    keep = [[0.0] * w for _ in range(h)]
    for c in pieces:
        for x, y in c:
            keep[y][x] = 1.0

    xs = [p[0] for c in pieces for p in c]
    ys = [p[1] for c in pieces for p in c]
    x0, x1, y0, y1 = min(xs), max(xs) + 1, min(ys), max(ys) + 1
    tl, tt, tr, tb = cfg['trim']
    ww, hh = x1 - x0, y1 - y0
    x0 += int(ww * tl); x1 -= int(ww * tr)
    y0 += int(hh * tt); y1 -= int(hh * tb)

    return keep, x0, x1, y0, y1


def build(name, out):
    cfg = SUBJECTS[name]
    im = Image.open(BASE + cfg['src']).convert('RGB')
    w, h = im.size
    px = im.load()

    keep, x0, x1, y0, y1 = subject_mask(cfg, px, w, h)

    COLS = cfg['cols']
    ROWS_ART = max(1, round(COLS * (y1 - y0) / (x1 - x0) / 2))

    cov = [[0.0] * (COLS * SUBX) for _ in range(ROWS_ART * SUBY)]
    for r in range(ROWS_ART * SUBY):
        sy0 = y0 + (y1 - y0) * r // (ROWS_ART * SUBY)
        sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // (ROWS_ART * SUBY))
        for c in range(COLS * SUBX):
            sx0 = x0 + (x1 - x0) * c // (COLS * SUBX)
            sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // (COLS * SUBX))
            n = (sx1 - sx0) * (sy1 - sy0)
            cov[r][c] = sum(keep[y][x] for y in range(sy0, sy1) for x in range(sx0, sx1)) / n

    mid = []
    fams = [(n, clear_ground(l), clear_ground(d)) for n, l, d in cfg['families']]
    for _, light, deep in fams:
        m = tuple((light[i] + deep[i]) // 2 for i in range(3))
        mid.append((oklab(m), oklab(light)[0], oklab(deep)[0], light, deep))

    def classify(rgb):
        want = oklab(rgb)
        best, bestd = 0, math.inf
        for i, (lab, _, _, _, _) in enumerate(mid):
            dl = (want[0] - lab[0]) * LIGHTWEIGHT
            da, db = want[1] - lab[1], want[2] - lab[2]
            d = math.sqrt(dl * dl + da * da + db * db)
            if d < bestd:
                best, bestd = i, d
        return best, want[0]

    def shade(i, l):
        _, ll, dl_, light, deep = mid[i]
        t = 0.0 if ll == dl_ else (l - ll) / (dl_ - ll)
        t = max(0.0, min(1.0, t))
        return tuple(int(light[k] + (deep[k] - light[k]) * t) for k in range(3)), t

    colour = [[None] * COLS for _ in range(ROWS_ART)]
    tone = [[0.0] * COLS for _ in range(ROWS_ART)]
    for r in range(ROWS_ART):
        sy0 = y0 + (y1 - y0) * r // ROWS_ART; sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // ROWS_ART)
        for c in range(COLS):
            sx0 = x0 + (x1 - x0) * c // COLS; sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // COLS)
            # vote first, then shade: averaging a cell that straddles a black
            # cap and a white face gives grey, and grey is not in the picture
            votes = {}
            for y in range(sy0, sy1):
                for x in range(sx0, sx1):
                    if keep[y][x]:
                        i, l = classify(px[x, y])
                        v = votes.setdefault(i, [0, 0.0])
                        v[0] += 1; v[1] += l
            if votes:
                i = max(votes, key=lambda k: votes[k][0])
                colour[r][c], tone[r][c] = shade(i, votes[i][1] / votes[i][0])

    A = atlas(RAMP + EDGY[1:])

    def glyph(r, c):
        v = [cov[r * SUBY + y][c * SUBX + x] for y in range(SUBY) for x in range(SUBX)]
        if max(v) < 0.10:
            return ' '
        if sum(v) / len(v) > 0.85:
            # on a dark ground a character's ink is LIGHT, so density follows
            # how much light the cell gives off. the other way round drew the
            # puffin's white face as holes and its black cap as a solid block.
            l = oklab(colour[r][c])[0] if colour[r][c] else 0.0
            t = max(0.0, min(1.0, (l - FLOOR) / (0.93 - FLOOR))) ** 0.8
            # the darkest adapted colour still gets a solid character: mapping
            # it to the bottom of the ramp thinned every shadow to specks
            lo = 4
            return RAMP[min(len(RAMP) - 1, lo + int(t * (len(RAMP) - 1 - lo) + 0.5))]
        best, err = ' ', None
        for g, tpl in A.items():
            e = sum((v[i] - tpl[i]) ** 2 for i in range(len(v)))
            if err is None or e < err:
                best, err = g, e
        return best

    word = subprocess.run(['figlet', '-f', cfg['fig'], '-w', '78', name],
                          capture_output=True, text=True).stdout.rstrip('\n').split('\n')
    ROWS = ROWS_ART if cfg.get('wordy') is not None else ROWS_ART + len(word) + 1 - cfg['overlap']
    cells = [[(' ', None)] * WIDTH for _ in range(ROWS)]
    OX = {'left': 0, 'right': WIDTH - COLS, 'centre': (WIDTH - COLS) // 2}[cfg['place']]
    for r in range(ROWS_ART):
        for c in range(COLS):
            g = glyph(r, c)
            if g != ' ' and colour[r][c]:
                cells[r][OX + c] = (g, colour[r][c])
    # a word can be told where to sit. 'game' sat beside the ball, out in the
    # sky it is flying into, which read well and cost twenty-two columns of
    # width the manpage does not have. under the bat is empty anyway.
    wtop = cfg.get('wordy', ROWS_ART - cfg['overlap'])
    for i, l in enumerate(word):
        for j, g in enumerate(l):
            xx = cfg['wordx'] + j
            if g == ' ' or not (0 <= xx < WIDTH) or wtop + i >= ROWS:
                continue
            # a word placed on purpose wins its cells; one tucked under the
            # subject gives way to it, so the subject reads as being in front
            placed = cfg.get('wordy') is not None
            if placed or cells[wtop + i][xx][0] == ' ':
                cells[wtop + i][xx] = (g, GOLD)

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
    plain = [''.join(g for g, _ in row).rstrip() for row in cells]
    open(out + '-plain.txt', 'w').write('\n'.join(plain) + '\n')

    pw, ph = 9, 18
    img = Image.new('RGB', (WIDTH * pw, ROWS * ph), (23, 15, 45))
    d = ImageDraw.Draw(img)
    pf = ImageFont.truetype('/System/Library/Fonts/Menlo.ttc', 15)
    for i, row in enumerate(cells):
        for j, (g, col) in enumerate(row):
            if g != ' ':
                d.text((j * pw, i * ph - 2), g, font=pf, fill=col)
    img.save(out + '.png')
    print(f'{name:9s} {WIDTH} x {ROWS} | art {COLS} x {ROWS_ART} | crop {x1-x0} x {y1-y0}')

def sprite(name, out, cols=None):
    """emit a full size stencil: roles at daffy's pixel, one column wide and
    half a row tall, so a 78 by 23 header is a 78 by 46 sprite. no colour in
    the file -- a colourway binds the roles when it is drawn."""
    cfg = SUBJECTS[name]
    cols = cols or cfg['cols']
    im = Image.open(BASE + cfg['src']).convert('RGB')
    w, h = im.size
    px = im.load()
    keep, x0, x1, y0, y1 = subject_mask(cfg, px, w, h)
    rows = max(1, round(cols * (y1 - y0) / (x1 - x0) / 2)) * 2   # half-block pixels

    mid = []
    fams = [(n, clear_ground(l), clear_ground(d)) for n, l, d in cfg['families']]
    for _, light, deep in fams:
        m = tuple((light[i] + deep[i]) // 2 for i in range(3))
        mid.append(oklab(m))

    def pick(rgb):
        want = oklab(rgb)
        best, bestd = 0, math.inf
        for i, lab in enumerate(mid):
            dl = (want[0] - lab[0]) * LIGHTWEIGHT
            da, db = want[1] - lab[1], want[2] - lab[2]
            d = math.sqrt(dl * dl + da * da + db * db)
            if d < bestd:
                best, bestd = i, d
        return best

    letters = cfg.get('roles') or ['K'] * len(cfg['families'])
    art = []
    for r in range(rows):
        sy0 = y0 + (y1 - y0) * r // rows; sy1 = max(sy0 + 1, y0 + (y1 - y0) * (r + 1) // rows)
        line = []
        for c in range(cols):
            sx0 = x0 + (x1 - x0) * c // cols; sx1 = max(sx0 + 1, x0 + (x1 - x0) * (c + 1) // cols)
            votes = {}
            for y in range(sy0, sy1):
                for x in range(sx0, sx1):
                    if keep[y][x]:
                        i = pick(px[x, y])
                        votes[i] = votes.get(i, 0) + 1
            covered = sum(votes.values())
            total = (sx1 - sx0) * (sy1 - sy0)
            if covered / total < 0.5:
                line.append('.')
            else:
                line.append(letters[max(votes, key=votes.get)])
        art.append(''.join(line))
    # the eye is placed from the photograph rather than found: at this size it
    # is two pixels, and it is what a blink needs
    if cfg.get('eye'):
        ex, ey = cfg['eye']
        ec = int((ex - x0) / (x1 - x0) * cols)
        er = int((ey - y0) / (y1 - y0) * rows)
        for dr in (-1, 0, 1):
            for dc in (-1, 0, 1):
                rr, cc = er + dr, ec + dc
                if 0 <= rr < rows and 0 <= cc < cols and art[rr][cc] != '.':
                    art[rr] = art[rr][:cc] + 'X' + art[rr][cc + 1:]
        if 0 <= er < rows and 0 <= ec < cols and art[er][ec] != '.':
            art[er] = art[er][:ec] + 'E' + art[er][ec + 1:]

    body = (f'# {name}, traced from the photograph and reduced to roles.\n'
            f'# {cols} by {rows} at daffy pixels, which is {cols} by {rows // 2} cells.\n'
            f'sprite {name}\n\nart\n' + '\n'.join(art) + '\n')
    open(out, 'w').write(body)
    print(f'{name:9s} sprite {cols} x {rows} px -> {out}')

if __name__ == '__main__':
    args = sys.argv[1:]
    if args and args[0] == '--sprite':
        for n in (args[1:] or list(SUBJECTS)):
            sprite(n, f'sprites/made/{n}.sprite')
    else:
        for n in (args or list(SUBJECTS)):
            build(n, 'dolphin/' + n)
