# SPDX-License-Identifier: GPL-3.0-or-later
"""Logo-Entwürfe Runde 2: dunkelgrauer Grund, dezentes Grün und Orange, Krokodil.

Aufruf: python3 marke/werkzeug/runde2.py  → marke/entwuerfe/runde-2/*.svg
Schriftzug und Maße wie Runde 1 (runde1.py).
"""
from pathlib import Path

from runde1 import SW, schriftzug, svg, ecken

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-2"

# Palette (marke/farben.md) — dunkel ist die Hauptfassung, hell für weiße Flächen.
DUNKEL = {"grund": "#1E2124", "schrift": "#ECEAE5", "kroko": "#5FAE8C", "akzent": "#E5934B"}
HELL = {"grund": "#FFFFFF", "schrift": "#1E2124", "kroko": "#3B7A5F", "akzent": "#A85F22"}


def linie(d, farbe, sw=SW):
    return (f'<path d="{d}" fill="none" stroke="{farbe}" stroke-width="{sw}" '
            f'stroke-linecap="round" stroke-linejoin="round"/>')


def s_kroko(x, f):
    """Großes S als Krokodil: oben der Kopf mit weit offenem Maul nach rechts, unten der Schwanz."""
    koerper = (f"M{x+150},40 L{x+62},40 C{x+22},40 {x+8},62 {x+8},88 C{x+8},132 {x+98},114 {x+98},160 "
               f"C{x+98},198 {x+44},208 {x+2},192")
    kiefer = f"M{x+46},66 L{x+142},104"
    hoecker = f"M{x+34},40 C{x+36},14 {x+74},12 {x+78},40"
    zaehne = (f"M{x+96},51 l7,12 l7,-12 M{x+122},51 l7,12 l7,-12 "
              f"M{x+92},77 l7,-11 l7,15 M{x+116},87 l7,-11 l7,15")
    auge = f'<circle cx="{x+56}" cy="{26}" r="8" fill="{f["akzent"]}"/>'
    return (linie(koerper + " " + kiefer + " " + hoecker, f["kroko"])
            + linie(zaehne, f["akzent"], 6) + auge)


def langes_kroko(x0, x1, y, f):
    """Lang gezogenes Krokodil unter dem Schriftzug: Rumpf als Umriss, Maul offen nach rechts."""
    hals = x1 - 170
    ruecken = (f"M{x0},{y} C{x0+140},{y-4} {x0+260},{y-34} {x0+420},{y-34} "
               f"C{hals-120},{y-34} {hals-40},{y-26} {hals},{y-20} "
               f"C{hals+20},{y-44} {hals+56},{y-44} {hals+70},{y-24} L{x1},{y-30}")
    bauch = (f"M{x0},{y} C{x0+160},{y+8} {x0+300},{y+16} {x0+440},{y+16} "
             f"C{hals-100},{y+16} {hals-30},{y+10} {hals+10},{y+2} L{x1-10},{y+40}")
    beine = (f"M{x0+300},{y+12} l-16,30 h-14 M{x0+520},{y+16} l-16,30 h-14 "
             f"M{x0+560},{y+16} l10,30 h14")
    zacken = " ".join(f"M{x},{y-31 if x > x0+330 else y-12-int((x-x0)*0.06)} l8,-11 l8,11"
                      for x in range(x0 + 200, hals - 60, 46))
    zaehne = (f"M{hals+100},{y-27} l7,12 l7,-12 M{hals+132},{y-28} l7,12 l7,-12 "
              f"M{hals+80},{y+18} l7,-11 l7,14 M{hals+116},{y+27} l7,-11 l7,14")
    auge = f'<circle cx="{hals+40}" cy="{y-30}" r="8" fill="{f["akzent"]}"/>'
    return (linie(ruecken + " " + bauch + " " + beine, f["kroko"])
            + linie(zacken, f["kroko"], 9) + linie(zaehne, f["akzent"], 6) + auge)


def kachel(inhalt, f, rand=None):
    ecke = ecken(44, 44, 212, 212, rand, l=40, sw=14) if rand else ""
    return svg(256, 256, f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{f["grund"]}"/>{ecke}{inhalt}')


def bild(breite, hoehe, inhalt, grund, vb):
    """SVG mit Hintergrund über den ganzen Ausschnitt (auch bei negativem Ursprung)."""
    x, y, w, h = vb.split()
    hoehe = round(breite * float(h) / float(w))  # Seitenverhältnis wie der Ausschnitt, sonst Ränder
    return svg(breite, hoehe, f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{grund}"/>{inhalt}', vb=vb)


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)
    for art, f in (("dunkel", DUNKEL), ("hell", HELL)):
        # A — S als Krokodilkopf
        rest, ende = schriftzug("chnapp", 20 + 172, f["schrift"])
        schreibe(f"a-s-kroko-{art}.svg",
                 bild(ende + 60, 300, s_kroko(20, f) + rest, f["grund"], f"-10 -10 {ende + 70} 290"))
        # B — Schriftzug, darunter ein langes Krokodil mit offenem Maul unter dem pp
        zug, ende = schriftzug("schnapp", 40, f["schrift"])
        schreibe(f"b-langes-kroko-{art}.svg",
                 bild(ende + 140, 420, zug + langes_kroko(30, ende + 90, 330, f), f["grund"],
                     f"0 10 {ende + 130} 400"))
    # Symbole (dunkel): S-Krokodil allein und im Auswahlrahmen
    g = '<g transform="translate(52,52) scale(0.95)">{}</g>'
    schreibe("a-symbol.svg", kachel(g.format(s_kroko(0, DUNKEL)), DUNKEL))
    schreibe("a-symbol-rahmen.svg",
             kachel('<g transform="translate(68,66) scale(0.8)">{}</g>'.format(s_kroko(0, DUNKEL)), DUNKEL,
                    rand=DUNKEL["akzent"]))


if __name__ == "__main__":
    main()
