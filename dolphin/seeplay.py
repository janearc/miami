# renders what bandersnatch actually drew: parse its escape stream, keep a
# cell grid, and snapshot at the end of every synchronised frame. looking at
# the player's own output rather than at the file it was given.
import re, subprocess, sys
from PIL import Image

cmd = sys.argv[1:-1]
out = sys.argv[-1]
raw = subprocess.run(cmd, capture_output=True, text=True).stdout

CELL = 24          # preview pixels per cell, wide
grid = {}          # (row, col) -> (glyph, fg, bg)
fg, bg = (220, 220, 220), None
r = c = 1
frames = []
i = 0
tok = re.compile(r'\x1b\[([0-9;?]*)([A-Za-z])')
while i < len(raw):
    m = tok.match(raw, i)
    if m:
        args, verb = m.group(1), m.group(2)
        if verb == 'm':
            parts = [p for p in args.split(';') if p != '']
            k = 0
            while k < len(parts):
                if parts[k] == '0':
                    fg, bg = (220, 220, 220), None; k += 1
                elif parts[k] == '38' and parts[k+1] == '2':
                    fg = tuple(int(v) for v in parts[k+2:k+5]); k += 5
                elif parts[k] == '48' and parts[k+1] == '2':
                    bg = tuple(int(v) for v in parts[k+2:k+5]); k += 5
                elif parts[k] == '49':
                    bg = None; k += 1
                else:
                    k += 1
        elif verb == 'H':
            a = args.split(';')
            r = int(a[0] or 1); c = int(a[1]) if len(a) > 1 and a[1] else 1
        elif verb == 'J':
            grid.clear()
        elif verb == 'l' and args == '?2026':
            frames.append(dict(grid))
        i = m.end(); continue
    ch = raw[i]
    if ch == '\n':
        r += 1; c = 1
    elif ch == '\r':
        c = 1
    else:
        grid[(r, c)] = (ch, fg, bg)
        c += 1
    i += 1

if not frames:
    frames = [grid]
maxr = max(k[0] for f in frames for k in f) if frames[0] else 1
maxc = max(k[1] for f in frames for k in f) if frames[0] else 1
W, H = maxc * CELL, maxr * CELL * 2
sheet = Image.new('RGB', (W * len(frames) + 12 * (len(frames) - 1), H), (60, 60, 66))
for n, f in enumerate(frames):
    img = Image.new('RGB', (W, H), (16, 14, 22))
    px = img.load()
    for (rr, cc), (ch, f_, b_) in f.items():
        x0, y0 = (cc - 1) * CELL, (rr - 1) * CELL * 2
        top = f_ if ch == '▀' else (b_ or (16, 14, 22))
        bot = f_ if ch == '▄' else (b_ or (16, 14, 22))
        if ch == '█':
            top = bot = f_
        for y in range(CELL):
            for x in range(CELL):
                px[x0 + x, y0 + y] = top
                px[x0 + x, y0 + CELL + y] = bot
    sheet.paste(img, (n * (W + 12), 0))
sheet.save(out)
print(f'{len(frames)} frames, {maxc} x {maxr} cells -> {out}')
