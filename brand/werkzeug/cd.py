# SPDX-License-Identifier: GPL-3.0-or-later
"""Baut die Bilder des Corporate Designs und das PDF aus brand/corporate-design.md.

Aufruf: python3 brand/werkzeug/cd.py
  → brand/cd/*.svg|png (Farbfelder, Schutzraum, Oberflächen-Beispiel)
  → brand/corporate-design.pdf (über HTML, gedruckt mit Chromium im Headless-Modus, eigenes Profil unter tmp/;
    Metadaten danach mit qpdf entfernt)
Braucht: chromium, qpdf, rsvg-convert, Python mit fontTools, Open Sans SemiBold (SCHNAPP_SCHRIFT).

Der Markdown-Umwandler kann genau das, was corporate-design.md benutzt: Überschriften, Absätze, Listen,
Tabellen, Bilder, Zitate, Code, fett, Links.
"""
import html
import re
import subprocess
from pathlib import Path

from paket import RAHMEN, logo
from runde6 import text_als_pfad

MARKE = Path(__file__).resolve().parent.parent
WURZEL = MARKE.parent
CD = MARKE / "cd"

FARBEN = [  # Name, Hex, Schrift darauf
    ("grau-950", "#16181B", "#ECEAE5"), ("grau-900", "#1E2124", "#ECEAE5"), ("grau-800", "#2A2E33", "#ECEAE5"),
    ("grau-700", "#3A3F45", "#ECEAE5"), ("grau-400", "#92989F", "#1E2124"), ("grau-100", "#ECEAE5", "#1E2124"),
    ("gruen-300", "#86C2A6", "#1E2124"), ("gruen-500", "#4F9479", "#1E2124"), ("gruen-700", "#3B7A5F", "#FFFFFF"),
    ("orange-300", "#F9C894", "#1E2124"), ("orange-500", "#F5A65B", "#1E2124"), ("orange-700", "#A85F22", "#FFFFFF"),
    ("rot-500", "#E67873", "#1E2124"), ("rot-700", "#B4433E", "#FFFFFF"),
]


def farbfelder():
    feld, abstand, je_zeile = 150, 12, 7
    teile = []
    for i, (name, hexwert, schrift) in enumerate(FARBEN):
        x, y = (i % je_zeile) * (feld + abstand), (i // je_zeile) * (feld + abstand)
        teile.append(f'<rect x="{x}" y="{y}" width="{feld}" height="{feld}" rx="10" fill="{hexwert}" '
                     f'stroke="#3A3F45" stroke-width="1"/>')
        teile.append(text_als_pfad(name, x + 12, y + feld - 36, 17, schrift)[0])
        teile.append(text_als_pfad(hexwert, x + 12, y + feld - 14, 17, schrift)[0])
    b = je_zeile * (feld + abstand) - abstand
    h = ((len(FARBEN) - 1) // je_zeile + 1) * (feld + abstand) - abstand
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {b} {h}" width="{b}" height="{h}">{"".join(teile)}</svg>\n'


def schutzraum():
    """Logo dunkel mit gestricheltem Schutzraum: ringsum eine halbe x-Höhe (50 Einheiten)."""
    x, y, w, h = (float(v) for v in RAHMEN.split())
    innen = (x + 10, y + 10, w - 20, h - 20)              # Inhalt ohne Rand
    rand = 50
    aussen = (innen[0] - rand, innen[1] - rand, innen[2] + 2 * rand, innen[3] + 2 * rand)
    logo_inhalt = re.sub(r"^<svg[^>]*>|</svg>\s*$", "", logo("dunkel", True))
    vb = f"{aussen[0] - 20} {aussen[1] - 20} {aussen[2] + 40} {aussen[3] + 40}"
    return (f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{vb}">'
            f'<rect x="{aussen[0] - 20}" y="{aussen[1] - 20}" width="{aussen[2] + 40}" height="{aussen[3] + 40}" fill="#1E2124"/>'
            f'<rect x="{aussen[0]}" y="{aussen[1]}" width="{aussen[2]}" height="{aussen[3]}" fill="none" '
            f'stroke="#F5A65B" stroke-width="3" stroke-dasharray="12 8"/>'
            f'<rect x="{innen[0]}" y="{innen[1]}" width="{innen[2]}" height="{innen[3]}" fill="none" '
            f'stroke="#92989F" stroke-width="1.5" stroke-dasharray="4 6"/>'
            f'{logo_inhalt}</svg>\n')


UI_HTML = """<!doctype html><html><head><meta charset="utf-8"><style>
body{margin:0;background:#1E2124;color:#ECEAE5;font:15px/1.5 'Open Sans',sans-serif;padding:20px}
.ui{background:#2A2E33;border:1px solid #3A3F45;border-radius:10px;padding:16px;width:600px}
.l{display:flex;gap:8px;margin-bottom:14px;align-items:center}
.k{border:0;border-radius:6px;padding:7px 14px;font:inherit}
.haupt{background:#4F9479;color:#1E2124;font-weight:600}.zweit{background:none;color:#ECEAE5;border:1px solid #3A3F45}
.wz{background:#1E2124;color:#ECEAE5;border:1px solid #3A3F45}.wz.a{border-color:#4F9479;color:#86C2A6}
.fokus{outline:2px solid #F5A65B;outline-offset:2px}
.gefahr{background:none;color:#E67873;border:1px solid #E67873}
.h{border-left:4px solid #F5A65B;background:#1E2124;padding:8px 12px;border-radius:4px}.h b{color:#F9C894}
.aw{position:relative;height:100px;margin:14px 0;background:#16181B;border-radius:6px}
.aw div{position:absolute;left:30%;top:20px;width:40%;height:58px;border:2px solid #4F9479}
.aw i{position:absolute;width:12px;height:12px;border:3px solid #F5A65B}
.n{color:#92989F;margin-left:auto}
</style></head><body><div class="ui">
<div class="l"><button class="k wz a">Pfeil</button><button class="k wz">Kasten</button><button class="k wz fokus">Text</button>
<button class="k gefahr">Schwärzen</button><span class="n">Fokus: oranger Ring</span></div>
<div class="aw"><div></div>
<i style="left:calc(30% - 6px);top:14px;border-right:0;border-bottom:0"></i>
<i style="left:calc(70% - 8px);top:14px;border-left:0;border-bottom:0"></i>
<i style="left:calc(30% - 6px);top:74px;border-right:0;border-top:0"></i>
<i style="left:calc(70% - 8px);top:74px;border-left:0;border-top:0"></i></div>
<div class="h"><b>Hinweis:</b> Geschwärzte Bereiche lassen sich nach dem Speichern nicht wiederherstellen.</div>
<div class="l" style="margin:14px 0 0"><button class="k haupt">Speichern</button><button class="k zweit">Abbrechen</button>
<span class="n">1920 × 1080 · 100 %</span></div></div></body></html>"""


def chromium(*args):
    profil = WURZEL / "tmp" / "chromium-profil"  # eigenes, leeres Profil — nie das des Nutzers
    profil.mkdir(parents=True, exist_ok=True)
    subprocess.run(["chromium", "--headless", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
                    f"--user-data-dir={profil}", *args], check=True, capture_output=True)


# ---- Markdown → HTML (nur der benutzte Teil) ----

def inline(t):
    t = html.escape(t, quote=False)
    t = re.sub(r"`([^`]+)`", r"<code>\1</code>", t)
    t = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", t)
    t = re.sub(r"!\[([^\]]*)\]\(([^)]+)\)", r'<img src="\2" alt="\1">', t)
    t = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', t)
    return t


def md_zu_html(md):
    zeilen, aus, i = md.split("\n"), [], 0
    while i < len(zeilen):
        z = zeilen[i]
        if z.startswith("```"):
            j = i + 1
            while j < len(zeilen) and not zeilen[j].startswith("```"):
                j += 1
            if j == len(zeilen):
                raise ValueError(f"Code-Zaun ab Zeile {i + 1} wird nicht geschlossen")
            aus.append("<pre><code>" + html.escape("\n".join(zeilen[i + 1:j])) + "</code></pre>")
            i = j + 1
        elif m := re.match(r"(#{1,4}) (.*)", z):
            n = len(m.group(1))
            aus.append(f"<h{n}>{inline(m.group(2))}</h{n}>")
            i += 1
        elif z.startswith("|"):
            kopf = [c.strip() for c in z.strip("|").split("|")]
            i += 2
            reihen = []
            while i < len(zeilen) and zeilen[i].startswith("|"):
                reihen.append([c.strip() for c in zeilen[i].strip("|").split("|")])
                i += 1
            aus.append("<table><thead><tr>" + "".join(f"<th>{inline(c)}</th>" for c in kopf) + "</tr></thead><tbody>"
                       + "".join("<tr>" + "".join(f"<td>{inline(c)}</td>" for c in r) + "</tr>" for r in reihen)
                       + "</tbody></table>")
        elif re.match(r"(- |\d+\. )", z):
            tag = "ol" if re.match(r"\d+\. ", z) else "ul"
            punkte = []
            while i < len(zeilen) and (re.match(r"(- |\d+\. )", zeilen[i]) or zeilen[i].startswith("  ")):
                if zeilen[i].startswith("  "):
                    punkte[-1] += " " + zeilen[i].strip()
                else:
                    punkte.append(re.sub(r"^(- |\d+\. )", "", zeilen[i]))
                i += 1
            aus.append(f"<{tag}>" + "".join(f"<li>{inline(p)}</li>" for p in punkte) + f"</{tag}>")
        elif z.startswith(">"):
            teil = []
            while i < len(zeilen) and zeilen[i].startswith(">"):
                teil.append(zeilen[i].lstrip("> "))
                i += 1
            aus.append("<blockquote>" + inline(" ".join(teil)) + "</blockquote>")
        elif z.strip() == "---":
            aus.append('<hr>')
            i += 1
        elif z.strip():
            teil = []
            while i < len(zeilen) and zeilen[i].strip() and not re.match(r"(#|\||- |\d+\. |>|```)", zeilen[i]):
                teil.append(zeilen[i])
                i += 1
            aus.append("<p>" + inline(" ".join(teil)) + "</p>")
        else:
            i += 1
    return "\n".join(aus)


CSS = """
@page { size: A4; margin: 18mm 16mm; }
body { font: 10.5pt/1.5 'Open Sans', sans-serif; color: #1E2124; }
h1 { font-size: 22pt; margin: 0 0 4mm; } h2 { font-size: 15pt; margin: 8mm 0 3mm; border-bottom: 2px solid #4F9479;
padding-bottom: 1mm; break-after: avoid; } h3 { font-size: 12pt; margin: 5mm 0 2mm; break-after: avoid; }
table { border-collapse: collapse; width: 100%; margin: 2mm 0 4mm; font-size: 9.5pt; break-inside: avoid; }
th, td { border: 1px solid #C9CCD0; padding: 1.5mm 2mm; text-align: left; vertical-align: top; }
th { background: #ECEAE5; } code { font: 9pt monospace; background: #F1F0EC; padding: 0 1mm; border-radius: 2px; }
pre { background: #F1F0EC; padding: 3mm; border-radius: 3px; } pre code { background: none; padding: 0; }
img { max-width: 100%; display: block; margin: 3mm 0; break-inside: avoid; }
blockquote { border-left: 4px solid #F5A65B; margin: 3mm 0; padding: 1mm 4mm; background: #FBF6EF; }
a { color: #3B7A5F; } hr { border: 0; border-top: 1px solid #C9CCD0; margin: 6mm 0; }
"""


def pdf():
    md = (MARKE / "corporate-design.md").read_text(encoding="utf-8")
    seite = (f'<!doctype html><html lang="de"><head><meta charset="utf-8"><title>Schnapp Corporate Design</title>'
             f'<style>{CSS}</style></head><body>{md_zu_html(md)}</body></html>')
    zwischen = MARKE / "corporate-design.tmp.html"   # liegt neben der md, damit relative Bildpfade stimmen
    zwischen.write_text(seite, encoding="utf-8")
    ablage = WURZEL / "tmp" / "brand"
    ablage.mkdir(parents=True, exist_ok=True)
    roh, sauber = ablage / "corporate-design.roh.pdf", ablage / "corporate-design.pdf"
    try:
        chromium("--no-pdf-header-footer", f"--print-to-pdf={roh}", zwischen.as_uri())
        # Keine Spuren (Kernprinzip 3): Chromium schreibt Programm- und Systemnamen in Creator/Producer — entfernen.
        subprocess.run(["qpdf", "--remove-info", "--remove-metadata", str(roh), str(sauber)], check=True)
        sauber.replace(MARKE / "corporate-design.pdf")  # erst ersetzen, wenn alles geklappt hat
    finally:
        zwischen.replace(ablage / "corporate-design.html")  # Zwischenstand aufheben statt löschen, auch bei Fehler


def main():
    CD.mkdir(exist_ok=True)
    for name, inhalt in (("farben", farbfelder()), ("schutzraum", schutzraum())):
        (CD / f"{name}.svg").write_text(inhalt, encoding="utf-8")
        subprocess.run(["rsvg-convert", "-w", "1100", str(CD / f"{name}.svg"), "-o", str(CD / f"{name}.png")], check=True)
    ui = WURZEL / "tmp" / "brand" / "ui-beispiel.html"
    ui.parent.mkdir(parents=True, exist_ok=True)
    ui.write_text(UI_HTML, encoding="utf-8")
    chromium("--hide-scrollbars", "--window-size=680,330", f"--screenshot={CD / 'ui-beispiel.png'}", ui.as_uri())
    pdf()


if __name__ == "__main__":
    main()
