# draws a .ansi file the way a terminal draws it: one monospace glyph per
# cell, solid, on whatever background the terminal supplies.
#
# this exists because the go side previews cells by filling block masks and
# supersampling outlines, which is right for judging a block-glyph picture and
# wrong for judging a wordmark. a letter drawn as a supersampled outline at a
# few pixels is a hairline; the same letter drawn by a terminal is solid. two
# rounds of font choice nearly got made against that difference.
import re, sys
from PIL import Image, ImageDraw, ImageFont

FONT = '/System/Library/Fonts/Menlo.ttc'
SGR = re.compile(r'\x1b\[([0-9;]*)m')


def cells(path):
    out = []
    for line in open(path).read().rstrip('\n').split('\n'):
        row, fg, bg, i = [], (220, 220, 220), None, 0
        for m in SGR.finditer(line):
            for ch in line[i:m.start()]:
                row.append((ch, fg, bg))
            i = m.end()
            parts = [p for p in m.group(1).split(';') if p]
            k = 0
            while k < len(parts):
                if parts[k] == '0':
                    fg, bg, k = (220, 220, 220), None, k + 1
                elif parts[k] == '49':
                    bg, k = None, k + 1
                elif parts[k] == '38' and parts[k + 1] == '2':
                    fg, k = tuple(int(v) for v in parts[k + 2:k + 5]), k + 5
                elif parts[k] == '48' and parts[k + 1] == '2':
                    bg, k = tuple(int(v) for v in parts[k + 2:k + 5]), k + 5
                else:
                    k += 1
        for ch in line[i:]:
            row.append((ch, fg, bg))
        out.append(row)
    return out


def draw(rows, ground, cw=9, ch=18):
    w = max(len(r) for r in rows)
    im = Image.new('RGB', (w * cw, len(rows) * ch), ground)
    d = ImageDraw.Draw(im)
    f = ImageFont.truetype(FONT, ch)
    for y, row in enumerate(rows):
        for x, (c, fg, bg) in enumerate(row):
            if bg is not None:
                d.rectangle([x * cw, y * ch, x * cw + cw - 1, y * ch + ch - 1], fill=bg)
            if c != ' ':
                d.text((x * cw, y * ch - ch * 0.12), c, font=f, fill=fg)
    return im


if __name__ == '__main__':
    src, dst = sys.argv[1], sys.argv[2]
    ground = (18, 16, 28)
    if len(sys.argv) > 3 and sys.argv[3] == 'light':
        ground = (238, 236, 230)
    draw(cells(src), ground).save(dst)
    print(f'{src} -> {dst}')
