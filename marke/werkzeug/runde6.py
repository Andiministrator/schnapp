# SPDX-License-Identifier: GPL-3.0-or-later
"""Logo-Entwürfe Runde 6 (Grundlage Runde 5 b: oranges s, Blick nach rechts).

e: das „h“ mit Unterlänge statt Oberlänge (hängt wie die „pp“),
f: Unterzeile unter „schna“, als Pfade aus Open Sans (fontTools; keine Schriftdatei im SVG nötig).

Aufruf: python3 marke/werkzeug/runde6.py  → marke/entwuerfe/runde-6/*.svg
"""
from pathlib import Path

from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont

from runde1 import LAUF, R, W, buchstabe
from runde4 import DUNKEL, HELL
from runde5 import bild, s_mit_auge, zug

ZIEL = Path(__file__).resolve().parent.parent / "entwuerfe" / "runde-6"
SCHRIFT = Path("/usr/share/fonts/TTF/OpenSans-SemiBold.ttf")  # OFL; nur Umrisse landen im SVG
NEBEN = {"dunkel": "#8E949B", "hell": "#5E646B"}  # grau-400 auf dunkel; auf Weiß dunkler für Kontrast


def h_unten(x):
    """h, dessen Stamm nach unten hängt (bis zur Unterlänge der p)."""
    return [f"M{x},100 V262",
            f"M{x},{100+R} A{R},{R} 0 0 1 {x+R},100 H{x+W-R} A{R},{R} 0 0 1 {x+W},{100+R} V200"]


def schriftzug(f, h_haengt=False):
    teile, x = [s_mit_auge(40, f, blick=1.0)], 40 + LAUF
    for z in "chnapp":
        pfade = h_unten(x) if (z == "h" and h_haengt) else buchstabe(z, x)
        teile += [zug(d, f["schrift"]) for d in pfade]
        x += LAUF
    return "".join(teile), x - 40


def text_als_pfad(text, x, grundlinie, groesse, farbe, sperrung=0.04):
    """Setzt text als Pfad (Glyphenumrisse), linksbündig ab x."""
    font = TTFont(SCHRIFT)
    einheiten = font["head"].unitsPerEm
    glyphen, cmap, hmtx = font.getGlyphSet(), font.getBestCmap(), font["hmtx"]
    s = groesse / einheiten
    pfade, lauf = [], 0.0
    for zeichen in text:
        name = cmap[ord(zeichen)]
        pen = SVGPathPen(glyphen)
        glyphen[name].draw(TransformPen(pen, (s, 0, 0, -s, x + lauf, grundlinie)))
        if pen.getCommands():
            pfade.append(pen.getCommands())
        lauf += hmtx[name][0] * s + sperrung * groesse
    return f'<path d="{" ".join(pfade)}" fill="{farbe}"/>', lauf


def schreibe(name, text):
    (ZIEL / name).write_text(text, encoding="utf-8")


def main():
    ZIEL.mkdir(parents=True, exist_ok=True)
    for art, f in (("dunkel", DUNKEL), ("hell", HELL)):
        zugtext, ende = schriftzug(f, h_haengt=True)
        schreibe(f"e-h-haengt-{art}.svg", bild(640, zugtext, f["grund"], f"0 20 {ende + 60} 270"))
        varianten = (("linux screenshots", "f-klein", 34), ("LINUX SCREENSHOTS", "f-versal", 28),
                     ("screenshots for linux", "f-for-linux", 32))
        for text, name, groesse in varianten:
            zugtext, ende = schriftzug(f)
            # Unterzeile unter „schna“: von der linken Kante des s bis vor den ersten p-Stamm,
            # Sperrung so gewählt, dass die Zeile genau diese Breite füllt.
            breite_ziel = (40 + 5 * LAUF - 30) - (40 - 11)
            _, natuerlich = text_als_pfad(text, 0, 0, groesse, "#000", sperrung=0)
            sperrung = (breite_ziel - natuerlich) / (len(text) - 1) / groesse
            unter, _ = text_als_pfad(text, 40 - 11, 262, groesse, NEBEN[art], sperrung=sperrung)
            schreibe(f"{name}-{art}.svg", bild(640, zugtext + unter, f["grund"], f"0 20 {ende + 60} 270"))


if __name__ == "__main__":
    main()
