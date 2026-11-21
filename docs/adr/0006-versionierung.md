# ADR-0006: Versionierung

## Status
Akzeptiert (19.11.2026)

## Kontext
Schnapp wird als Programm ausgeliefert (später ggf. Flatpak/Paketquellen). Das Dateiformat (ADR-0008) braucht eine
eigene Versionsangabe, damit alte Dateien lesbar bleiben.

## Entscheidung
- SemVer `MAJOR.MINOR.PATCH`, Start bei `v0.1.0`, Tags mit `v`-Präfix.
- Ein `v*`-Tag nur nach bestandenem `/sprint-abschluss` und auf Andis ausdrückliche Ansage.
- Das Dateiformat trägt eine **eigene** Formatversion im SVG-Block, unabhängig von der Programmversion.

## Konsequenzen
- Vor `v1.0.0` darf sich das Dateiformat ändern; ab `v1.0.0` muss jede Programmversion alle früheren Formatversionen
  lesen können.

## Nachtrag 1 (21.11.2026): Programmversion ohne Tag (F-113, F-327)

**Status:** Akzeptiert (Claude, Andi informiert). Umgesetzt in Sprint 10 (`install.sh`, Commit 754016e), getestet von
`scripts/install-test.sh` (F-324).

- Die Programmversion kommt beim Installieren aus `git describe --tags --dirty`, ohne führendes `v` (`v0.4.0` →
  `0.4.0`, nach einem Tag mit weiteren Commits etwa `0.4.0-3-gabc1234`).
- **Ohne erreichbaren Tag** (flacher Klon, Arbeitsstand vor dem ersten Tag) entsteht eine gültige SemVer-Fassung:
  `0.0.0-dev+<Kurzhash>`, bei geänderten versionierten Dateien `0.0.0-dev+<Kurzhash>.dirty`. Ohne git (Tarball) gilt
  `0.0.0-dev`, wie der Vorgabewert im Programm (`cmd/schnapp/main.go`).
- **Nur ein Tag dieses Repos zählt:** Liegt der Ordner in einem fremden Repo (etwa ein ausgepackter Tarball in einem
  versionierten Home-Ordner), gilt dessen Stand nicht; dann ebenfalls `0.0.0-dev`.
- `schnapp --version` nennt diese Zeichenkette; die Formatversion der Arbeitsdatei bleibt davon unabhängig.
