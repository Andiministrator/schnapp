# SPDX-License-Identifier: GPL-3.0-or-later
"""Baut das Logo-Paket (gewählt: Runde 6 „e6“ = e4 mit „SCREENSHOTS FOR LINUX“, Andi 19.11.2026).

Aufruf: python3 marke/werkzeug/paket.py  → marke/logo/
Braucht: Python mit fontTools, Open Sans SemiBold (Unterzeile wird zu Pfaden), rsvg-convert, magick (ImageMagick).
Alle Dateien werden aus dem Code erzeugt — nie von Hand ändern, sondern hier und neu bauen.
"""
import subprocess
from pathlib import Path

from runde1 import LAUF, W
from runde4 import DUNKEL, HELL
from runde5 import s_mit_auge
from runde6 import NEBEN, schriftzug, text_als_pfad

ZIEL = Path(__file__).resolve().parent.parent / "logo"
VERSATZ = {"h": 30, "p": -31}           # h tiefer, pp höher: ragen oben und unten gleich weit heraus
FUSS, GROESSE = 241, 24                  # Grundlinie und Größe der Unterzeile
SC = (40 - 11, 40 + LAUF + W + 11)        # Außenkanten von s und c
NA = (40 + 3 * LAUF - 11, 40 + 4 * LAUF + W + 11)  # Außenkanten von n und a
RAHMEN = "19 48 814 204"                 # Ausschnitt Logo: Inhalt x 29–823, y 58–242, 10 Rand
SYMBOL_GROESSEN = (16, 22, 24, 32, 48, 64, 128, 180, 192, 256, 512, 1024)
LOGO_BREITEN = (300, 600, 1200, 2400)

FASSUNG = {"dunkel": DUNKEL, "hell": HELL}  # dunkel = für dunklen Grund (helle Schrift), hell = für hellen Grund
# Einfarbig (Stempel, Druck, Prägung): alles in einer Farbe; das Auge bleibt als Ring mit Pupille erhalten.
EINFARBIG = {
    "schwarz": {"grund": "#FFFFFF", "schrift": "#000000", "akzent": "#000000", "pupille": "#000000", "weiss": "#FFFFFF"},
    "weiss": {"grund": "#000000", "schrift": "#FFFFFF", "akzent": "#FFFFFF", "pupille": "#FFFFFF", "weiss": "none"},
}
UNTERZEILE_FARBE = {**NEBEN, "schwarz": "#000000", "weiss": "#FFFFFF"}


def svg(vb, inhalt, breite=None, hoehe=None):
    groesse = f' width="{breite}" height="{hoehe}"' if breite else ""
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb}"{groesse}>{inhalt}</svg>\n'


def unterzeile(art):
    def gesperrt(text, x0, x1):
        _, nat = text_als_pfad(text, 0, 0, GROESSE, "#000", sperrung=0)
        sp = (x1 - x0 - nat) / (len(text) - 1) / GROESSE
        return text_als_pfad(text, x0, FUSS, GROESSE, UNTERZEILE_FARBE[art], sperrung=sp)[0]
    return gesperrt("SCREENSHOTS", *SC) + gesperrt("FOR LINUX", *NA)


def logo(art, mit_unterzeile, grund=False):
    f = FASSUNG.get(art) or EINFARBIG[art]
    zug, _ = schriftzug(f, versatz=VERSATZ)
    inhalt = zug + (unterzeile(art) if mit_unterzeile else "")
    if grund:
        x, y, w, h = RAHMEN.split()
        inhalt = f'<rect x="{x}" y="{y}" width="{w}" height="{h}" fill="{f["grund"]}"/>' + inhalt
    return svg(RAHMEN, inhalt)


def symbol(art, kachel=True):
    """Quadratisches Symbol: das „s“ mit Auge. kachel=False: nur das s, freigestellt."""
    f = FASSUNG[art]
    s = s_mit_auge(0, f, blick=1.0)
    if not kachel:
        return svg("-16 84 108 132", s)  # s mit Strich: x −11…87, y 89…211, quadratisch genug für Favicons
    grund = f["grund"] if art == "dunkel" else "#ECEAE5"
    return svg("0 0 256 256", f'<rect x="8" y="8" width="240" height="240" rx="56" fill="{grund}"/>'
                              f'<g transform="translate(77,-75) scale(1.35)">{s}</g>')


def sozial():
    """Vorschaubild für GitHub/Links 1280×640: Logo dunkel mittig auf dunkelgrauem Grund."""
    f = DUNKEL
    zug, _ = schriftzug(f, versatz=VERSATZ)
    inhalt = zug + unterzeile("dunkel")
    return svg("0 0 1280 640", f'<rect width="1280" height="640" fill="{f["grund"]}"/>'
                               f'<g transform="translate(150,197) scale(1.2)">'
                               f'<g transform="translate(-19,-48)">{inhalt}</g></g>', 1280, 640)


def schreibe(pfad, text):
    pfad.parent.mkdir(parents=True, exist_ok=True)
    pfad.write_text(text, encoding="utf-8")


def png(svg_pfad, ziel, breite):
    ziel.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(["rsvg-convert", "-w", str(breite), str(svg_pfad), "-o", str(ziel)], check=True)


def main():
    s, p = ZIEL / "svg", ZIEL / "png"
    for art in FASSUNG:
        for mit, name in ((True, "schnapp-logo"), (False, "schnapp-logo-ohne-unterzeile")):
            schreibe(s / f"{name}-{art}.svg", logo(art, mit))
            schreibe(s / f"{name}-{art}-mit-grund.svg", logo(art, mit, grund=True))
            for b in LOGO_BREITEN:
                png(s / f"{name}-{art}.svg", p / f"{name}-{art}-{b}.png", b)
                png(s / f"{name}-{art}-mit-grund.svg", p / f"{name}-{art}-mit-grund-{b}.png", b)
        schreibe(s / f"schnapp-symbol-{art}.svg", symbol(art))
        schreibe(s / f"schnapp-s-{art}.svg", symbol(art, kachel=False))
        for g in SYMBOL_GROESSEN:
            png(s / f"schnapp-symbol-{art}.svg", p / f"schnapp-symbol-{art}-{g}.png", g)
        for g in (64, 256, 512):
            png(s / f"schnapp-s-{art}.svg", p / f"schnapp-s-{art}-{g}.png", g)

    for art in EINFARBIG:
        for mit, name in ((True, "schnapp-logo"), (False, "schnapp-logo-ohne-unterzeile")):
            schreibe(s / f"{name}-einfarbig-{art}.svg", logo(art, mit))
            png(s / f"{name}-einfarbig-{art}.svg", p / f"{name}-einfarbig-{art}-1200.png", 1200)

    fav = ZIEL / "favicon"
    schreibe(fav / "favicon.svg", symbol("dunkel"))
    for g in (16, 32, 48):
        png(fav / "favicon.svg", fav / f"tmp-{g}.png", g)
    subprocess.run(["magick", *(str(fav / f"tmp-{g}.png") for g in (16, 32, 48)), str(fav / "favicon.ico")], check=True)
    for g in (16, 32, 48):  # Zwischenstufen überschreiben statt löschen: in das ico gewandert, als PNG behalten
        (fav / f"tmp-{g}.png").replace(fav / f"favicon-{g}.png")
    png(fav / "favicon.svg", fav / "apple-touch-icon.png", 180)
    png(fav / "favicon.svg", fav / "icon-192.png", 192)
    png(fav / "favicon.svg", fav / "icon-512.png", 512)
    schreibe(fav / "site.webmanifest",
             '{\n  "name": "Schnapp",\n  "short_name": "Schnapp",\n  "icons": [\n'
             '    {"src": "icon-192.png", "sizes": "192x192", "type": "image/png"},\n'
             '    {"src": "icon-512.png", "sizes": "512x512", "type": "image/png"}\n  ],\n'
             f'  "theme_color": "{DUNKEL["grund"]}",\n  "background_color": "{DUNKEL["grund"]}",\n'
             '  "display": "standalone"\n}\n')

    schreibe(ZIEL / "sozial" / "schnapp-sozial-1280x640.svg", sozial())
    png(ZIEL / "sozial" / "schnapp-sozial-1280x640.svg", ZIEL / "sozial" / "schnapp-sozial-1280x640.png", 1280)


if __name__ == "__main__":
    main()
