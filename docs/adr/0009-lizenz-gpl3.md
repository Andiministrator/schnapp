# ADR-0009: Lizenz GPL-3.0

## Status
Akzeptiert (Andi, 18.11.2026; übernommen am 19.11.2026)

## Kontext
Schnapp wird öffentlich. Andi will, dass Weiterentwicklungen offen bleiben.

## Entscheidung
Schnapp steht unter der **GPL-3.0** als **`GPL-3.0-or-later`** (Andi, 19.11.2026: folgt der Empfehlung). `LICENSE` enthält den unveränderten
Text von gnu.org; jede Go-Datei beginnt mit `// SPDX-License-Identifier: GPL-3.0-or-later`.

## Konsequenzen
- Fremdpakete und Stempelquellen müssen GPL-3.0-verträglich sein (MIT, BSD, Apache-2.0, ISC: ja; CC BY-SA bei Stempeln
  als Daten mit Namensnennung, getrennt ausgewiesen). Vor jeder Übernahme prüfen (ADR-0011).
- Die übernommenen Zeichenpakete stehen ebenfalls unter GPL-3.0-or-later (ADR-0012).
