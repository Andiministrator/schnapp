# ADR-0001: ADR-Format für Entscheidungen

## Status
Akzeptiert (19.11.2026)

## Kontext
Schnapp wird über viele Sitzungen und mit Sub-Agents ohne Gedächtnis entwickelt. Entscheidungen, die nur im Chat
fallen, gehen verloren oder werden neu verhandelt.

## Entscheidung
Architektur- und Produktentscheidungen werden als ADR unter `docs/adr/NNNN-kurzer-titel.md` festgehalten, mit den
Abschnitten Status, Kontext, Entscheidung, Konsequenzen. Status: `Vorgeschlagen` · `Akzeptiert` · `Abgelehnt` ·
`Überholt`. Erst ADR, dann umsetzen. Die nächste Nummer kommt aus dem Verzeichnis, nicht aus einem Kartentext.

## Konsequenzen
- Wer Verhalten ändert, sucht vorher nach einer ADR, die das alte Verhalten festschreibt (ADR-Gegenrichtung,
  `docs/qa/schritt-qa.md`).
- Nachträge stehen als „Nachtrag N (Datum)" in der ADR selbst; wird eine Entscheidung ersetzt, bekommt die alte den
  Status `Überholt` mit Verweis.
