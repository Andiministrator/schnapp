# SPDX-License-Identifier: GPL-3.0-or-later
"""Erzeugt die Logo-Entwürfe der Runde 1 als SVG (Schriftzug selbst gezeichnet, keine Schriftdatei nötig).

Aufruf: python3 brand/werkzeug/runde1.py  → brand/entwuerfe/runde-1/*.svg

Schriftzug: Monoline mit runden Enden; Bögen sind abgerundete Rechtecke (wie ein Auswahlrahmen), nicht Kreise.
Maße: x-Höhe 100 (y 100–200), Oberlänge bis y 40, Unterlänge bis y 262, Strich 22.
"""
from pathlib import Path

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-1"

TINTE = "#1D2B4F"    # Nachtblau: Schriftzug
PETROL = "#0E8C83"   # Petrol: Bildzeichen
GELB = "#FFC23D"     # Markergelb: Akzent (Textmarker, Beschriftung)
HELL = "#F6F4EE"     # heller Hintergrund

SW = 22   # Strichstärke
W = 76    # Breite eines Bogens (Mittellinie)
R = 30    # Eckenradius der Bögen
LAUF = W + 40  # Vorschub je Buchstabe


def kasten(x, oben=100, unten=200, w=W, r=R):
    """Geschlossener Bogen als abgerundetes Rechteck."""
    return (f"M{x+r},{oben} H{x+w-r} A{r},{r} 0 0 1 {x+w},{oben+r} V{unten-r} "
            f"A{r},{r} 0 0 1 {x+w-r},{unten} H{x+r} A{r},{r} 0 0 1 {x},{unten-r} V{oben+r} "
            f"A{r},{r} 0 0 1 {x+r},{oben} Z")


def buchstabe(z, x):
    if z == "s":
        q = 25
        return [f"M{x+W},{100} H{x+q} A{q},{q} 0 0 0 {x+q},{150} H{x+W-q} A{q},{q} 0 0 1 {x+W-q},{200} H{x}"]
    if z == "c":
        return [f"M{x+W},{100} H{x+R} A{R},{R} 0 0 0 {x},{100+R} V{200-R} A{R},{R} 0 0 0 {x+R},{200} H{x+W}"]
    if z == "h":
        return [f"M{x},40 V200", f"M{x},{100+R} A{R},{R} 0 0 1 {x+R},100 H{x+W-R} A{R},{R} 0 0 1 {x+W},{100+R} V200"]
    if z == "n":
        return [f"M{x},200 V100 M{x},{100+R} A{R},{R} 0 0 1 {x+R},100 H{x+W-R} A{R},{R} 0 0 1 {x+W},{100+R} V200"]
    if z == "a":
        return [kasten(x), f"M{x+W},100 V200"]
    if z == "p":
        return [kasten(x), f"M{x},100 V262"]
    raise ValueError(z)


def schriftzug(text, x0=0, farbe=TINTE):
    teile, x = [], x0
    for z in text:
        for d in buchstabe(z, x):
            teile.append(f'<path d="{d}"/>')
        x += LAUF
    g = (f'<g fill="none" stroke="{farbe}" stroke-width="{SW}" stroke-linecap="round" '
         f'stroke-linejoin="round">{"".join(teile)}</g>')
    return g, x - 40  # rechter Rand der letzten Mittellinie


def s_pfeil(x, farbe):
    """Großes S als gebogener Pfeil (Kernfunktion: Pfeile biegen). Spitze unten links."""
    kurve = (f"M{x+92},62 C{x+66},34 {x+8},38 {x+8},84 C{x+8},128 {x+98},112 {x+98},160 "
             f"C{x+98},200 {x+40},208 {x+6},186")
    spitze = f"M{x+46},{168} L{x+6},186 L{x+38},{218}"
    return (f'<g fill="none" stroke="{farbe}" stroke-width="{SW}" stroke-linecap="round" '
            f'stroke-linejoin="round"><path d="{kurve}"/><path d="{spitze}"/></g>')


def ecken(x0, y0, x1, y1, farbe, l=34, sw=14):
    d = (f"M{x0},{y0+l} V{y0} H{x0+l} M{x1-l},{y0} H{x1} V{y0+l} "
         f"M{x1},{y1-l} V{y1} H{x1-l} M{x0+l},{y1} H{x0} V{y1-l}")
    return (f'<path d="{d}" fill="none" stroke="{farbe}" stroke-width="{sw}" '
            f'stroke-linecap="round" stroke-linejoin="round"/>')


def svg(breite, hoehe, inhalt, hintergrund=None, vb=None):
    vb = vb or f"0 0 {breite} {hoehe}"
    bg = f'<rect width="100%" height="100%" fill="{hintergrund}"/>' if hintergrund else ""
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb}" width="{breite}" height="{hoehe}">'
            f'{bg}{inhalt}</svg>\n')


def kachel(inhalt, grund=TINTE):
    """App-Symbol: abgerundetes Quadrat 256×256."""
    return svg(256, 256, f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{grund}"/>{inhalt}')


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)

    # A — Bogenpfeil-S: das S ist ein gebogener Pfeil, Rest Schriftzug.
    s = s_pfeil(20, PETROL)
    rest, ende = schriftzug("chnapp", 20 + LAUF + 12)
    schreibe("a-bogenpfeil.svg", svg(ende + 40, 300, s + rest, HELL, f"-10 10 {ende + 50} 280"))
    schreibe("a-bogenpfeil-symbol.svg",
             kachel(f'<g transform="translate(76,2) scale(1.0)">{s_pfeil(0, GELB)}</g>'))

    # B — Auswahl: Schriftzug im Auswahlrahmen (vier Ecken), die Ecken als Bildzeichen.
    zug, ende = schriftzug("schnapp", 60)
    rahmen = ecken(20, 20, ende + 40, 240, PETROL)
    schreibe("b-auswahl.svg", svg(ende + 70, 300, rahmen + zug, HELL, f"0 0 {ende + 60} 290"))
    s_klein, _ = schriftzug("s", 0, GELB)
    schreibe("b-auswahl-symbol.svg",
             kachel(ecken(48, 48, 208, 208, PETROL, l=44, sw=16)
                    + f'<g transform="translate(83,-22) scale(0.95)">{s_klein}</g>'))

    # C — Schnapper: Maskottchen-Kopf aus zwei Kiefern, die wie Auswahlecken zuschnappen.
    def kopf(f_linie, f_auge, f_zahn):
        # Oberkiefer: Hinterkopf → Stirn → Augenhöcker → Schnauze → Nasenhöcker → Spitze → Unterkante zurück
        oben = ("M44,150 C34,118 44,84 80,78 L118,74 C126,52 156,52 162,72 L214,78 "
                "C222,70 236,72 238,86 C240,98 232,104 220,104 L96,124")
        # Unterkiefer, schräg geöffnet
        unten = "M58,162 L206,196 C220,199 228,190 224,180"
        zaehne = ("M124,118 l7,14 l7,-17 M156,111 l7,14 l7,-16 M188,106 l7,13 l7,-15 "
                  "M122,175 l7,-13 l7,16 M154,183 l7,-13 l7,16")
        auge = '<circle cx="140" cy="66" r="9" fill="{0}"/>'
        return (f'<g fill="none" stroke="{f_linie}" stroke-width="{SW}" stroke-linecap="round" '
                f'stroke-linejoin="round"><path d="{oben}"/><path d="{unten}"/></g>'
                f'<path d="{zaehne}" fill="none" stroke="{f_zahn}" stroke-width="7" stroke-linecap="round" '
                f'stroke-linejoin="round"/>' + auge.format(f_auge))
    zug, ende = schriftzug("schnapp", 300)
    schreibe("c-schnapper.svg", svg(ende + 40, 300, kopf(PETROL, TINTE, GELB) + zug, HELL,
                                    f"0 0 {ende + 50} 290"))
    schreibe("c-schnapper-symbol.svg",
             kachel(f'<g transform="translate(6,6) scale(0.95)">{kopf(GELB, "#FFFFFF", "#FFFFFF")}</g>'))


if __name__ == "__main__":
    main()
