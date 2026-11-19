# SPDX-License-Identifier: GPL-3.0-or-later
"""Logo-Entwürfe Runde 4: Schriftzug mit orangem „s“ samt Krokodilauge, darunter das ausgemalte Krokodil.

Aufruf: python3 brand/werkzeug/runde4.py  → brand/entwuerfe/runde-4/*.svg
Krokodil-Varianten für das Orange: Buckel, Füße oder gar keins. Das Maul innen ist dunkelgrün, nicht orange.
"""
from pathlib import Path

from runde1 import LAUF, SW, buchstabe, svg

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-4"

DUNKEL = {"grund": "#1E2124", "schrift": "#ECEAE5", "kroko": "#4F9479", "maul": "#2C5545",
          "akzent": "#F5A65B", "pupille": "#1E2124", "weiss": "#FFFFFF"}
HELL = {"grund": "#FFFFFF", "schrift": "#1E2124", "kroko": "#3B7A5F", "maul": "#24473A",
        "akzent": "#E0893A", "pupille": "#1E2124", "weiss": "#FFFFFF"}


def flaeche(d, farbe):
    return f'<path d="{d}" fill="{farbe}"/>'


def zug(d, farbe):
    return (f'<path d="{d}" fill="none" stroke="{farbe}" stroke-width="{SW}" '
            f'stroke-linecap="round" stroke-linejoin="round"/>')


def schriftzug_s_auge(x0, f):
    """„schnapp“; das s in Orange mit Krokodilauge im oberen Bogen."""
    teile = [zug(d, f["akzent"]) for d in buchstabe("s", x0)]
    # Oberer Bogen des s: Halbkreis Mitte (x0+25, 125), Radius 25 — das Auge sitzt darin.
    teile.append(f'<circle cx="{x0+30}" cy="125" r="11" fill="{f["weiss"]}" stroke="{f["akzent"]}" stroke-width="2"/>'
                 f'<circle cx="{x0+27}" cy="125" r="5.5" fill="{f["pupille"]}"/>')
    x = x0 + LAUF
    for z in "chnapp":
        teile += [zug(d, f["schrift"]) for d in buchstabe(z, x)]
        x += LAUF
    return "".join(teile), x - 40


def kroko(f, orange):
    """Ausgemaltes Krokodil (Runde 3), Blick nach links. orange: "buckel", "fuesse" oder None."""
    maul = "M180,88 C140,82 80,80 40,80 L44,130 C90,118 150,104 186,96 Z"
    koerper = ("M22,64 C22,54 40,50 70,50 L128,46 C134,26 170,24 178,44 C200,46 230,50 260,54 "
               "C380,56 520,58 620,70 C700,78 760,88 794,98 C770,106 700,106 620,104 "
               "C520,110 380,116 290,114 C240,113 208,106 190,96 L180,88 "
               "C140,82 80,80 40,80 C28,80 22,74 22,64 Z")
    unterkiefer = "M186,96 C150,104 90,118 44,130 C32,133 26,142 34,147 C80,146 150,130 204,110 Z"
    f_buckel = f["akzent"] if orange == "buckel" else f["kroko"]
    f_beine = f["akzent"] if orange == "fuesse" else f["kroko"]
    buckel = "".join(f'<circle cx="{x}" cy="{54 + (x - 260) * 16 / 360 - 3:.1f}" r="10" fill="{f_buckel}"/>'
                     for x in range(286, 600, 40))
    beine = ("M296,110 C298,128 294,140 282,146 L322,146 C330,138 334,124 334,112 Z "
             "M516,106 C518,124 514,136 502,142 L542,142 C550,134 554,120 554,106 Z")
    zaehne = ("".join(f"M{x},81 l6,11 l6,-11 Z " for x in (56, 82, 108, 134))
              + " ".join(f"M{x},{96 + (186 - x - 6) * 34 / 142 + 1:.1f} l6,-11 l6,11 Z" for x in (70, 98, 126, 154)))
    auge = (f'<circle cx="156" cy="40" r="10" fill="{f["weiss"]}"/>'
            f'<circle cx="152" cy="40" r="5" fill="{f["pupille"]}"/>'
            f'<circle cx="36" cy="60" r="3.5" fill="{f["pupille"]}"/>')
    # Buckel liegen hinter dem Körper (nur die obere Hälfte schaut heraus); Beine vor dem Körper.
    return (flaeche(maul, f["maul"]) + flaeche(zaehne, f["weiss"]) + buckel + flaeche(koerper, f["kroko"])
            + flaeche(unterkiefer, f["kroko"]) + flaeche(beine, f_beine) + auge)


def bild(breite, inhalt, grund, vb):
    x, y, w, h = vb.split()
    hoehe = round(breite * float(h) / float(w))
    return svg(breite, hoehe, f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{grund}"/>{inhalt}', vb=vb)


def logo(f, orange):
    schrift, ende = schriftzug_s_auge(40, f)
    skala = (ende + 10) / 800
    tier = f'<g transform="translate(20,282) scale({skala:.3f})">{kroko(f, orange)}</g>'
    return bild(640, schrift + tier, f["grund"], f"0 20 {ende + 60} {282 + 150 * skala + 10:.0f}")


def symbol(f):
    """App-Symbol: das orange s mit Auge allein."""
    nur_s = "".join(zug(d, f["akzent"]) for d in buchstabe("s", 0)) + (
        f'<circle cx="30" cy="125" r="11" fill="{f["weiss"]}" stroke="{f["akzent"]}" stroke-width="2"/>'
        f'<circle cx="27" cy="125" r="5.5" fill="{f["pupille"]}"/>')
    return svg(256, 256, f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{f["grund"]}"/>'
                         f'<g transform="translate(77,-75) scale(1.35)">{nur_s}</g>')


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)
    for art, f in (("dunkel", DUNKEL), ("hell", HELL)):
        for orange in ("buckel", "fuesse", None):
            schreibe(f"logo-{orange or 'ohne'}-{art}.svg", logo(f, orange))
    schreibe("symbol-s-auge.svg", symbol(DUNKEL))


if __name__ == "__main__":
    main()
