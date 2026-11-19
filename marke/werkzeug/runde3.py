# SPDX-License-Identifier: GPL-3.0-or-later
"""Logo-Entwürfe Runde 3: das S als Krokodil mit Maul nach links, ausgemaltes Krokodil, Palette nachgeschärft.

Aufruf: python3 marke/werkzeug/runde3.py  → marke/entwuerfe/runde-3/*.svg
"""
from pathlib import Path

from runde1 import SW, schriftzug, svg, ecken

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-3"

# Palette Runde 3 (marke/farben.md): Grün dunkler/ruhiger, Orange heller/markanter.
DUNKEL = {"grund": "#1E2124", "schrift": "#ECEAE5", "kroko": "#4F9479",
          "akzent": "#F5A65B", "pupille": "#1E2124", "weiss": "#FFFFFF"}
HELL = {"grund": "#FFFFFF", "schrift": "#1E2124", "kroko": "#3B7A5F",
        "akzent": "#E0893A", "pupille": "#1E2124", "weiss": "#FFFFFF"}


def linie(d, farbe, sw=SW):
    return (f'<path d="{d}" fill="none" stroke="{farbe}" stroke-width="{sw}" '
            f'stroke-linecap="round" stroke-linejoin="round"/>')


def flaeche(d, farbe):
    return f'<path d="{d}" fill="{farbe}"/>'


def s_kroko(x, f):
    """Großes S als Krokodil, Blick nach links.

    Oberer Bogen = Kopf (Auge im Bogen), Mittelstrich = Oberkiefer mit Nasenhöcker vorn links,
    unterer Bogen = Unterkiefer; das offene Innere des unteren Bogens ist das Maul (offen nach links).
    """
    kopf = f"M{x+96},62 C{x+72},34 {x+12},36 {x+12},80 C{x+12},96 {x+20},104 {x+34},108"
    oberkiefer = f"M{x-14},106 L{x+34},108 C{x+70},110 {x+100},122 {x+100},158"
    unterkiefer = f"M{x+100},158 C{x+100},198 {x+50},212 {x-6},196"
    nase = f"M{x-14},106 C{x-16},94 {x-2},90 {x+4},100"
    zaehne = (f"M{x+4},116 l6,11 l6,-11 M{x+30},118 l6,11 l6,-11 M{x+56},123 l6,10 l6,-10 "
              f"M{x+4},188 l6,-11 l6,11 M{x+30},191 l6,-11 l6,10 M{x+56},189 l6,-11 l6,9")
    auge = (f'<circle cx="{x+56}" cy="{70}" r="13" fill="{f["weiss"]}" stroke="{f["kroko"]}" stroke-width="3"/>'
            f'<circle cx="{x+50}" cy="{70}" r="6.5" fill="{f["pupille"]}"/>')
    return (linie(zaehne, f["akzent"], 6)
            + linie(kopf + " " + oberkiefer + " " + unterkiefer + " " + nase, f["kroko"]) + auge)


def kroko_gemalt(f):
    """Ausgemaltes, rundliches Krokodil im Profil, Blick nach links, Maul offen. Rahmen 0..800 × 0..150."""
    maul = "M180,88 C140,82 80,80 40,80 L44,130 C90,118 150,104 186,96 Z"
    koerper = ("M22,64 C22,54 40,50 70,50 L128,46 C134,26 170,24 178,44 C200,46 230,50 260,54 "
               "C380,56 520,58 620,70 C700,78 760,88 794,98 C770,106 700,106 620,104 "
               "C520,110 380,116 290,114 C240,113 208,106 190,96 L180,88 "
               "C140,82 80,80 40,80 C28,80 22,74 22,64 Z")
    unterkiefer = ("M186,96 C150,104 90,118 44,130 C32,133 26,142 34,147 "
                   "C80,146 150,130 204,110 Z")
    buckel = "".join(f'<circle cx="{x}" cy="{54 + (x - 260) * 16 / 360 - 3:.1f}" r="10" fill="{f["kroko"]}"/>'
                     for x in range(286, 600, 40))
    beine = ("M296,110 C298,128 294,140 282,146 L322,146 C330,138 334,124 334,112 Z "
             "M516,106 C518,124 514,136 502,142 L542,142 C550,134 554,120 554,106 Z")
    zaehne_o = "".join(f"M{x},81 l6,11 l6,-11 Z " for x in (56, 82, 108, 134))
    zaehne_u = " ".join(f"M{x},{96 + (186 - x - 6) * 34 / 142 + 1:.1f} l6,-11 l6,11 Z" for x in (70, 98, 126, 154))
    auge = (f'<circle cx="156" cy="40" r="10" fill="{f["weiss"]}"/>'
            f'<circle cx="152" cy="40" r="5" fill="{f["pupille"]}"/>'
            f'<circle cx="36" cy="60" r="3.5" fill="{f["pupille"]}"/>')
    return (flaeche(maul, f["akzent"]) + flaeche(zaehne_o + zaehne_u, f["weiss"]) + buckel
            + flaeche(koerper, f["kroko"]) + flaeche(unterkiefer, f["kroko"])
            + flaeche(beine, f["kroko"]) + auge)


def bild(breite, inhalt, grund, vb):
    x, y, w, h = vb.split()
    hoehe = round(breite * float(h) / float(w))
    return svg(breite, hoehe, f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{grund}"/>{inhalt}', vb=vb)


def kachel(inhalt, f, rand=None):
    ecke = ecken(44, 44, 212, 212, rand, l=40, sw=14) if rand else ""
    return svg(256, 256, f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{f["grund"]}"/>{ecke}{inhalt}')


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)
    for art, f in (("dunkel", DUNKEL), ("hell", HELL)):
        rest, ende = schriftzug("chnapp", 40 + 150, f["schrift"])
        schreibe(f"a-s-kroko-{art}.svg", bild(640, s_kroko(40, f) + rest, f["grund"], f"0 10 {ende + 50} 270"))
        zug, ende = schriftzug("schnapp", 40, f["schrift"])
        skala = (ende + 10) / 800
        tier = f'<g transform="translate(20,282) scale({skala:.3f})">{kroko_gemalt(f)}</g>'
        schreibe(f"b-kroko-gemalt-{art}.svg",
                 bild(640, zug + tier, f["grund"], f"0 20 {ende + 60} {282 + 150 * skala + 10:.0f}"))
    schreibe("a-symbol.svg", kachel(f'<g transform="translate(88,10) scale(0.95)">{s_kroko(0, DUNKEL)}</g>', DUNKEL))
    schreibe("a-symbol-rahmen.svg",
             kachel(f'<g transform="translate(95,31) scale(0.78)">{s_kroko(0, DUNKEL)}</g>', DUNKEL,
                    rand=DUNKEL["akzent"]))
    schreibe("b-kroko-allein.svg", bild(640, kroko_gemalt(DUNKEL), DUNKEL["grund"], "-10 10 820 150"))


if __name__ == "__main__":
    main()
