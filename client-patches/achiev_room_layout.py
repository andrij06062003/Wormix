# Moves the 1.25.0 achievements room into the 1.05.0 layout:
# everything 228.4 px left (same offset as the race room between versions), "bosses" tab removed,
# awards/statistics tabs moved up into its place.
import re, sys
src, dst = sys.argv[1], sys.argv[2]
s = open(src).read()
m = re.search(r'(<item type="DefineSpriteTag"[^>]*spriteId="281".*?<subTags>)(.*?)(</subTags>)', s, re.S)
sub = m.group(2)
DX = -4568            # twips (228.4 px)
DY = {'awardBtn': 243.55 - 300.05, 'statisticBtn': 300.05 - 356.35}

def bits(*vals):
    need = 1
    for v in vals:
        n = (v if v >= 0 else -v - 1).bit_length() + 1
        need = max(need, n)
    return need

def fix_place(tag):
    if 'characterId="254"' in tag:          # bossesBtn
        return ''
    name = re.search(r' name="([^"]*)"', tag)
    name = name.group(1) if name else None
    def fix_matrix(mm):
        mt = mm.group(0)
        tx = int((re.search(r'translateX="(-?\d+)"', mt) or [0, 0])[1]) + DX
        ty = int((re.search(r'translateY="(-?\d+)"', mt) or [0, 0])[1]) + round(DY.get(name, 0) * 20)
        mt = re.sub(r'translateX="-?\d+"', 'translateX="%d"' % tx, mt)
        mt = re.sub(r'translateY="-?\d+"', 'translateY="%d"' % ty, mt)
        mt = re.sub(r'nTranslateBits="\d+"', 'nTranslateBits="%d"' % bits(tx, ty), mt)
        return mt
    return re.sub(r'<matrix [^>]*?/?>', fix_matrix, tag, count=1)

tags = re.split(r'(?=<item type="[A-Za-z0-9]+Tag")', sub)
removed = sum(1 for t in tags if t.startswith('<item type="PlaceObject') and 'characterId="254"' in t)
assert removed == 1, removed
out = [fix_place(t) if t.startswith('<item type="PlaceObject') else t for t in tags]
s = s[:m.start(2)] + ''.join(out) + s[m.end(2):]
open(dst, 'w').write(s)
print('ok')
