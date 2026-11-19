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
