# SPDX-License-Identifier: GPL-3.0-or-later
"""Logo-Entwürfe Runde 5: nur der Schriftzug, das orange „s“ mit Auge als einzige Besonderheit.

Aufruf: python3 brand/werkzeug/runde5.py  → brand/entwuerfe/runde-5/*.svg
Varianten: a Auge geradeaus, b Pupille schaut nach rechts auf „chnapp“, c zwei Zähnchen im unteren Bogen,
d Symbol im Auswahlrahmen.
"""
from pathlib import Path

from runde1 import LAUF, SW, buchstabe, ecken, svg
from runde4 import DUNKEL, HELL

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-5"


def zug(d, farbe):
    return (f'<path d="{d}" fill="none" stroke="{farbe}" stroke-width="{SW}" '
            f'stroke-linecap="round" stroke-linejoin="round"/>')


def s_mit_auge(x, f, blick=0.0, zaehne=False):
    """Das orange s; blick verschiebt die Pupille nach rechts (0 = mittig, 1 = ganz rechts)."""
    teile = [zug(d, f["akzent"]) for d in buchstabe("s", x)]
    teile.append(f'<circle cx="{x+30}" cy="125" r="11" fill="{f["weiss"]}" stroke="{f["akzent"]}" stroke-width="2"/>'
                 f'<circle cx="{x+27 + 7 * blick:.1f}" cy="125" r="5.5" fill="{f["pupille"]}"/>')
    if zaehne:  # zwei kleine Zähne an der Oberkante des unteren Bogens, nach unten in den Innenraum
        teile.append(f'<path d="M{x+30},161 l5,9 l5,-9 Z M{x+42},161 l5,9 l5,-9 Z" fill="{f["weiss"]}"/>')
    return "".join(teile)


def schriftzug(f, **art):
    teile, x = [s_mit_auge(40, f, **art)], 40 + LAUF
    for z in "chnapp":
        teile += [zug(d, f["schrift"]) for d in buchstabe(z, x)]
        x += LAUF
    return "".join(teile), x - 40


def bild(breite, inhalt, grund, vb):
    x, y, w, h = vb.split()
    hoehe = round(breite * float(h) / float(w))
    return svg(breite, hoehe, f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{grund}"/>{inhalt}', vb=vb)


def symbol(f, rahmen=False, **art):
    s = s_mit_auge(0, f, **art)
    if rahmen:
        inhalt = ecken(44, 44, 212, 212, f["schrift"], l=36, sw=12) + f'<g transform="translate(88,-30) scale(1.05)">{s}</g>'
    else:
        inhalt = f'<g transform="translate(77,-75) scale(1.35)">{s}</g>'
    return svg(256, 256, f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{f["grund"]}"/>{inhalt}')


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)
    varianten = {"a-geradeaus": {}, "b-blick": {"blick": 1.0}, "c-zaehne": {"zaehne": True}}
    for name, art in varianten.items():
        for hell_dunkel, f in (("dunkel", DUNKEL), ("hell", HELL)):
            zugtext, ende = schriftzug(f, **art)
            schreibe(f"{name}-{hell_dunkel}.svg", bild(640, zugtext, f["grund"], f"0 20 {ende + 60} 270"))
        schreibe(f"{name}-symbol.svg", symbol(DUNKEL, **art))
    schreibe("d-symbol-rahmen.svg", symbol(DUNKEL, rahmen=True, blick=1.0))


if __name__ == "__main__":
    main()
