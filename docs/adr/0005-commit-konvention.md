# ADR-0005: Commit-Konvention

## Status
Akzeptiert (19.11.2026)

## Kontext
Ein öffentliches Repo braucht lesbare, maschinell auswertbare Commit-Nachrichten; Sub-Agent-Arbeit muss nachprüfbar
zugeordnet sein.

## Entscheidung
Conventional Commits mit deutscher Beschreibung: `<typ>(<bereich>): <beschreibung>`.
Typen: `feat` · `fix` · `docs` · `refactor` · `chore` · `style` · `test` · `perf`.
Bereiche nach Paket bzw. Schicht, z. B. `aufnahme`, `dokument`, `datei`, `raster`, `svg`, `oberflaeche`, `bibliothek`.
`Co-Authored-By` nennt jedes Modell, das gearbeitet oder geprüft hat (Tabelle in `CLAUDE.md`).

## Konsequenzen
- Commit-Nachrichten enthalten keine Kunden-, Firmen- oder lokalen Pfadangaben (öffentliches Repo).

## Nachtrag 1 (20.11.2026): Bereiche englisch
Seit ADR-0019 heißen die Bereiche wie die Pakete (`capture`, `document`, `fileformat`, `render`, `canvas`, `library`,
`ui`, `cmd`, `draw`); zusätzlich `editor` (paketübergreifend ui/canvas/cmd), `findings`, `adr`, `qa`, `knowledge`. Ohne
Bereich nur bei paketübergreifenden Tests oder Doku über viele Stellen.

## Nachtrag 2 (24.11.2026): Typ `ci` (F-727)

**Status:** Akzeptiert (Claude Opus 5.5 beim Sprint-Abschluss 12, im Auftrag Andis: „arbeite selbstständig“).

Änderungen nur an `.github/workflows/` bekommen den Typ `ci` (wie in Conventional Commits üblich); er war in Sprint 12
zweimal in Gebrauch (760d183, 9109173), ohne in der Liste zu stehen. Die Liste lautet damit `feat` · `fix` · `docs` ·
`refactor` · `chore` · `style` · `test` · `perf` · `ci`. Zwei Commits desselben Sprints tragen kein `Co-Authored-By`
(fa0f40d, 6f50256); die Geschichte bleibt, wie sie ist.
