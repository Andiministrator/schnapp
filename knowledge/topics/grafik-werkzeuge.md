# Grafik-Werkzeuge für Logo und Corporate Design

Stand: 2026-11-19. Gilt für `brand/werkzeug/` (Python, wird später durch Go oder Schnapp selbst ersetzt — Karte
im Board, CLAUDE.md „Was zu vermeiden ist“).

## Kette

- **SVG von Hand im Code** (Monoline-Pfade, `stroke-linecap/linejoin="round"`), Schrift für die Unterzeile als
  **Umrisse** (fontTools `SVGPathPen` + `TransformPen`, y gespiegelt) — das SVG braucht dann keine Schriftdatei.
- **Rendern:** `rsvg-convert -w <breite>`; Prüfbilder mit ImageMagick `magick … -append/+append` nach `tmp/`.
- **ICO:** `magick a-16.png a-32.png a-48.png favicon.ico`.
- **PDF aus Markdown:** eigener kleiner Umwandler → HTML → `chromium --headless --print-to-pdf` mit **eigenem
  Profil** unter `tmp/` (`--user-data-dir`, nie das Profil des Nutzers) → `qpdf --remove-info --remove-metadata`.
- **Bildschirmfoto einer HTML-Seite:** `chromium --headless --screenshot=… --window-size=B,H`.

## Fallen

- **Chromium schreibt Programm- und Systemnamen in die PDF-Metadaten** (Creator „HeadlessChrome … X11; Linux“,
  Producer „Skia/PDF“) — Kernprinzip 3 verletzt, deshalb qpdf (F-23). qpdf setzt selbst nur `/ModDate` (F-30).
- **Python-Bytecode trägt absolute Pfade:** `__pycache__/` gehört in `.gitignore`; Skripte mit `python3 -B` starten
  (F-16).
- **Seitenverhältnis von `viewBox` und `width/height` müssen übereinstimmen**, sonst entstehen Ränder (Letterbox);
  Hintergrund-Rechteck über den ganzen Ausschnitt legen, auch bei negativem Ursprung.
- **Zentrieren im App-Symbol:** Mittelpunkt des Inhalts samt halber Strichstärke rechnen, dann
  `translate(128 − mx·s, 128 − my·s) scale(s)`; bei 16 px prüfen, ob das Zeichen noch lesbar ist.
- **ImageMagick `-splice` mit `-gravity center`** fügt die Lücke in der Bildmitte ein — für Prüfbilder `-border`
  nehmen.
- **Kontraste als Matrix rechnen**, nicht nur die genannten Paare: jede Text- und Akzentfarbe gegen jede Fläche
  (F-22: grau-400 auf grau-800 lag mit 4,46:1 knapp unter AA).
