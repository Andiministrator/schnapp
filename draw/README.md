# draw

Go-Modul `github.com/Andiministrator/schnapp/draw` — SVG lesen und rastern, nur Standardbibliothek.

- `svg` — SVG-Dateien und Pfaddaten lesen, Pfade in dichte Punktfolgen zerlegen.
- `raster` — eigener Rasterer mit Kantenglättung: Flächen, Striche (Enden, Ecken, Strichelung), Masken, clipPath,
  Bildvergleich. Was er nicht kann (Text, Verläufe, Filter, Bilder), ist ein Fehler, kein stilles Übergehen.
- `geo` — Rechenhilfen mit Python-gleichen Ergebnissen (hypot, sum, Modulo, Rundung, `Repr`).

Testen: `go vet ./... && go test ./...`
