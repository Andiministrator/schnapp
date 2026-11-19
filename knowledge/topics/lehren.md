# Lehren aus der Arbeit an Schnapp

Stand: 2026-11-19. Kurze Regeln aus Fehlern und Befunden; Einzelheiten in `docs/findings.md`.

## Öffentliches Repo

- **Vor dem ersten öffentlichen Push die ganze Geschichte prüfen**, nicht nur den Arbeitsbaum: Autor-Mailadresse
  (`git log --format=%ae`), Namen, Pfade, Hausnetz (F-1). Ein privates Zweit-Repo mit Schreibsperre gegen Force-Push
  macht spätere Korrekturen unmöglich — dort erst pushen, wenn die Geschichte veröffentlichungsreif ist.
- **Interne Angaben gehören in `CLAUDE.local.md`** (gitignoriert), nicht in versionierte Dateien. Auch Namen anderer
  eigener Projekte zählen dazu (F-6).
- **`git filter-branch` hinterlässt `refs/original/…`** mit der alten Geschichte (F-2). Push-Varianten `--all`,
  `--mirror`, `--tags`, Force und Refspecs sind in `.claude/settings.json` gesperrt (F-3).

## Go-Tests

- **Konstante Ausdrücke rechnet Go exakt:** `0.1 + 0.2` ist im Test `0.3`, nicht `0.30000000000000004`; ebenso
  `-0.0` ist `0`. Für Gleitkomma-Grenzfälle den Wert direkt als Literal oder per `math.Copysign` angeben.
- **Ein Rauchtest beweist wenig:** Für Raster-Ergebnisse Sollwerte (Pixelzahl, Kanalsummen) festhalten und mit einem
  verfälschten Sollwert gegenprüfen, dass der Test rot wird (F-12).

## Recherche

- **KWin ScreenShot2** gibt Pixel roh über eine Pipe; die Berechtigung hängt an einer installierten `.desktop`-Datei
  mit absolutem `Exec=` (`topics/kwin-screenshot.md`).
- **Plattformregeln zu KI-Code ändern sich schnell** (Codeberg Verbot Juli 2026, Flathub Offenlegung Okt. 2026) — vor
  jeder neuen Plattform aktuell prüfen (`topics/plattformen-und-ki-regeln.md`).
