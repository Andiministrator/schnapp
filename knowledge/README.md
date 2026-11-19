# Wissensbasis Schnapp

- [`spurensicherung.md`](spurensicherung.md) — wie Recherchen und Quellen abgelegt werden.
- [`sources-visited.md`](sources-visited.md) — Log aller besuchten URLs.
- `sources/` — eine Datei je genutzter Quelle.

## Topics

| Datei | Kernaussage | Lesen, wenn … |
|---|---|---|
| [`topics/projekt-konventionen.md`](topics/projekt-konventionen.md) | Kompaktblatt aller Konventionen für Sub-Agents und neue Sitzungen | ein Sub-Agent-Prompt gebaut wird oder die Sitzung ohne Memory startet |
| [`topics/lehren.md`](topics/lehren.md) | Regeln aus Befunden: öffentliche Geschichte prüfen, Internes in CLAUDE.local.md, Go-Konstanten in Tests, Sollwerte statt Rauchtest | vor einem Push, vor neuen Tests, beim Sitzungsstart |
| [`topics/kwin-screenshot.md`](topics/kwin-screenshot.md) | ScreenShot2 über D-Bus, Pixel per Pipe, je Bildschirm einzeln aufnehmen; Berechtigung laut Doku über installierte .desktop mit absolutem Exec — im Prototyp ging es ohne (F-15); erst aufnehmen, dann im eingefrorenen Bild wählen | Aufnahme, Overlay, Installation |
| [`topics/gio-wayland.md`](topics/gio-wayland.md) | Gio v0.10.3 kann unter Wayland weder Monitor wählen noch gebrochene Skalierung — der Patch rüstet beides nach; cgo, Build-Tags, Eingabe-API, Messen über den Stau statt die Bildrate | Oberfläche, Overlay, Gio-Update, Patch |
| [`topics/grafik-werkzeuge.md`](topics/grafik-werkzeuge.md) | Logo/CD-Kette: SVG-Pfade, Schrift als Umrisse, rsvg-convert, Chromium mit eigenem Profil, qpdf gegen Metadaten; Fallen (Bytecode, viewBox, Kontrast-Matrix) | Logo, Grafiken, PDFs erzeugen |
| [`topics/plattformen-und-ki-regeln.md`](topics/plattformen-und-ki-regeln.md) | Codeberg verbietet überwiegend KI-geschriebene Projekte; Flathub verlangt Offenlegung, Manifest und Einreichung von Hand | ein Repo, eine Paketquelle oder eine Veröffentlichung ansteht |
