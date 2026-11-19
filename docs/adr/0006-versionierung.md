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
