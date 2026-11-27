# ADR-0043: Mauszeiger mit aufnehmen

## Status
Akzeptiert für Teil 1 (Claude Opus 5.5, 27.11.2026, Karte #343 in Sprint 14; Andi 25.11.2026: „wir brauchen eine
schnell-setting – ob man den mauszeiger mit aufnehmen will“; Andi 27.11.: „arbeite selbstständig“). Ergänzt ADR-0007
(KWin-Schnittstelle), ADR-0017 (Aufruf) und ADR-0018 (Einstellungen).

## Kontext
KWin ScreenShot2/3 kennen die Option `include-cursor`; `capture.Optionen.Mauszeiger` reicht sie schon durch, gesetzt
wurde sie bisher nie. Für Anleitungen will man den Zeiger oft im Bild haben, für die meisten Bildschirmfotos nicht.

## Entscheidung
1. **Einstellung `[capture] cursor`** (yes/no, auch true/false; Voreinstellung no), Zeile „Mauszeiger — Mit aufnehmen“
   im Reiter Capture der Einstellungstafel.
2. **Befehlszeile `--cursor` / `--no-cursor`** für eine Aufnahme, vor der Einstellung; beide zusammen sind ein Fehler,
   ebenso mit einer Datei oder ohne Aufnahme.
3. **Gilt für jede Aufnahmeart**: Bildschirm, alle Monitore, Fenster, Wiederholen und Bereich. Beim Bereich steht der
   Zeiger im eingefrorenen Bild dort, wo er beim Auslösen war (KWin nimmt vor dem Overlay auf).
4. **Teil 2, offen:** ein Schalter im Overlay (Taste, Hinweis unten) und im Menü „Neue Aufnahme“ (#341). Der Overlay-
   Schalter bräuchte je Monitor zwei Aufnahmen (mit und ohne Zeiger) vor dem Overlay, also doppelte Zeit und doppelten
   Speicher, bevor das Overlay erscheint — erst in der echten Sitzung messen, dann entscheiden. Der MCP-Server nimmt
   weiter ohne Zeiger auf.

## Konsequenzen
`internal/settings` (`Einstellungen.Mauszeiger`), `cmd/schnapp` (`Aufruf.Mauszeiger`, `Auftrag.Mauszeiger`,
`capture.Optionen` in `aufnehmen`), Tafel `internal/ui/einstellungen_logik.go`; Tests, Mutanten
`scripts/mutanten/mauszeiger.json`. Echtlauf (Zeiger wirklich im Bild) nur in der Plasma-Sitzung prüfbar.
