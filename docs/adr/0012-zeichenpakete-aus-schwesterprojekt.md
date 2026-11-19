# ADR-0012: Zeichenpakete aus einem Schwesterprojekt

## Status
Akzeptiert (Andi, 18. und 19.11.2026)

## Kontext
Ein privates Schwesterprojekt des Maintainers hatte die Go-Pakete `svg` (SVG lesen, Pfade zerlegen), `raster` (eigener
Rasterer mit Kantenglättung, Striche, Masken, Bildvergleich) und `geo` (Rechenhilfen mit Python-gleichen
Ergebnissen). Sie sind genau das, was Schnapp zum Zeichnen und Exportieren braucht. Die Rechte daran liegen beim
Maintainer; die Veröffentlichung unter GPL-3.0 ist erlaubt (Andi, 19.11.2026).

## Entscheidung
- Die drei Pakete leben im **eigenen Go-Modul** `github.com/Andiministrator/schnapp/zeichnen` (Ordner `zeichnen/`),
  Pakete `…/zeichnen/svg`, `…/zeichnen/raster`, `…/zeichnen/geo` — öffentlich importierbar. Das Schwesterprojekt
  bindet sie von hier ein.
- Das Programm Schnapp bekommt später ein eigenes Modul im Wurzelordner und bindet `zeichnen` ein.
- Nichts anderes aus dem Schwesterprojekt kommt ins Repo; seine Git-Geschichte wird nicht übernommen.
- Ergänzt werden nach Bedarf: Schrift (z. B. `golang.org/x/image`), eingebettete Bilder (`<image>`), Weichzeichnen,
  Verpixeln.

## Konsequenzen
- **Öffentliche Schnittstellen von `zeichnen` nur mit Abstimmung ändern** — ein anderes Projekt hängt daran und prüft
  nach jeder Änderung, dass seine Ausgaben byte-genau gleich bleiben.
- Umzug am 19.11.2026: Importpfade angepasst, Kommentare ohne Bezug zum Schwesterprojekt, zwei Vergleichstests gegen
  dessen Python-Werkzeuge entfernt, Testfarben neutralisiert. Eine Zahlformatierung (`geo.Repr`, wie Python `repr()`)
  ersetzt eine Abhängigkeit; an 4 Mio. Zahlen ohne Abweichung zur Vorgängerfunktion geprüft. Sonst ist der Code
  unverändert; das Schwesterprojekt hat den byte-gleichen Neubau aller Ausgaben bestätigt.
- Die 13 Randfall-Proben liegen unter `zeichnen/raster/testdata/rasterproben/`; `TestRasterproben` zeichnet jede.
