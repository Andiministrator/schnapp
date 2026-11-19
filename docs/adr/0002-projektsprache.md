# ADR-0002: Projektsprache

## Status
Akzeptiert (19.11.2026)

## Kontext
Andi arbeitet auf Deutsch (maschinenweite Festlegung). Schnapp wird öffentlich (GitHub, GPL-3.0); Beiträge von
außen und eine englische Oberfläche sind später wahrscheinlich.

## Entscheidung
- Doku, ADRs, Commits, Code-Kommentare und die erste Oberflächensprache: **Deutsch**.
- Go-Bezeichner dürfen deutsch oder englisch sein, innerhalb eines Pakets einheitlich. Fremd-APIs (Gio, KWin, D-Bus,
  Standardbibliothek) behalten ihre Namen.
- Oberflächentexte werden von Anfang an an einer Stelle gesammelt (übersetzbar), nicht im Code verstreut.
- Eine englische README und Übersetzung der Oberfläche folgen vor der ersten Veröffentlichung (offene Entscheidung
  E-07).

## Konsequenzen
- Neue Pakete legen beim Anlegen fest, welche Sprache ihre Bezeichner haben.
- Die übernommenen Zeichenpakete behalten ihre bestehende Benennung.
