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


def schriftzug(f, h_haengt=False, h_tief=False, versatz=None):
    """versatz: {buchstabe: dy} verschiebt einzelne Buchstaben senkrecht (überschreibt h_tief)."""
    teile, x = [s_mit_auge(40, f, blick=1.0)], 40 + LAUF
    for z in "chnapp":
        pfade = h_unten(x) if (z == "h" and h_haengt) else buchstabe(z, x)
        neu = [zug(d, f["schrift"]) for d in pfade]
        dy = (versatz or {}).get(z, 62 if (z == "h" and h_tief) else 0)
        if dy:  # z. B. ganzes h tiefer: Fuß auf Höhe der p-Unterlängen
            neu = [f'<g transform="translate(0,{dy})">{"".join(neu)}</g>']
        teile += neu
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
        zugtext, ende = schriftzug(f, h_tief=True)
        schreibe(f"e2-h-tief-{art}.svg", bild(640, zugtext, f["grund"], f"0 20 {ende + 60} 270"))
        # e3: Unterzeile in zwei Teilen links und rechts des tiefen h: „SCREENSHOTS“ unter „sc“, „FOR LINUX“ unter „na“.
        sc = (40 - 11, 40 + LAUF + W + 11)              # Außenkanten von s und c
        na = (40 + 3 * LAUF - 11, 40 + 4 * LAUF + W + 11)  # Außenkanten von n und a
        groesse = 28
        for wort in ("SHOTS", "CAPTURE"):
            _, nat1 = text_als_pfad(wort, 0, 0, groesse, "#000", sperrung=0)
            t1, _ = text_als_pfad(wort, sc[0], 262, groesse, NEBEN[art],
                                  sperrung=(sc[1] - sc[0] - nat1) / (len(wort) - 1) / groesse)
            _, nat2 = text_als_pfad("FOR LINUX", 0, 0, groesse, "#000", sperrung=0)
            t2, _ = text_als_pfad("FOR LINUX", na[0], 262, groesse, NEBEN[art],
                                  sperrung=(na[1] - na[0] - nat2) / 8 / groesse)
            schreibe(f"e3-{wort.lower()}-{art}.svg", bild(640, zugtext + t1 + t2, f["grund"], f"0 20 {ende + 60} 270"))
        # e4: ohne Unterzeile; h um 30 tiefer, pp um 31 höher → beide ragen oben und unten gleich weit heraus
        mitte, ende4 = schriftzug(f, versatz={"h": 30, "p": -31})
        schreibe(f"e4-mittig-{art}.svg", bild(640, mitte, f["grund"], f"0 20 {ende4 + 60} 250"))
        for name, gleich in (("e3-voll", False), ("e3-gleich", True)):
            teile = []
            _, nat1 = text_als_pfad("SCREENSHOTS", 0, 0, groesse, "#000", sperrung=0)
            sp1 = (sc[1] - sc[0] - nat1) / 10 / groesse
            t1, _ = text_als_pfad("SCREENSHOTS", sc[0], 262, groesse, NEBEN[art], sperrung=sp1)
            _, nat2 = text_als_pfad("FOR LINUX", 0, 0, groesse, "#000", sperrung=0)
            if gleich:
                breite2 = nat2 + 8 * sp1 * groesse
                t2, _ = text_als_pfad("FOR LINUX", na[0] + (na[1] - na[0] - breite2) / 2, 262, groesse, NEBEN[art],
                                      sperrung=sp1)
            else:
                t2, _ = text_als_pfad("FOR LINUX", na[0], 262, groesse, NEBEN[art],
                                      sperrung=(na[1] - na[0] - nat2) / 8 / groesse)
            schreibe(f"{name}-{art}.svg", bild(640, zugtext + t1 + t2, f["grund"], f"0 20 {ende + 60} 270"))
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
