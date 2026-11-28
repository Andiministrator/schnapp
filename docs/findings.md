# Findings

Typen: `[S]` Sicherheit · `[K]` Stabilität · `[B]` Blockierend · `[V]` Verbesserung · `[D]` Konsistenz.
Priorität: P0 (sofort) · P1 (diesen Sprint) · P2 (nächsten Sprint) · P3 (irgendwann).
Schulden-Grenze: > 10 offene `[K]`-Findings → Hygiene-Sprint einplanen.

**Jedes offene Finding hat eine offene Board-Karte** mit seiner F-Nummer in Titel oder Beschreibung.

**Pflichtangaben für offen eingetragene Befunde:**
- **Prüfbefehl** — ein Aufruf mit erwartetem Wert, an dem sich ablesen lässt, ob die Lücke noch besteht. Am besten
  eine Verhaltensprobe (Test, Aufruf, Messung); ein Wort-`grep` nur für reine Doku-Befunde.
- **Widerlegungsbedingung** — nur bei Befunden, deren Fix-Richtung Code ändert: woran sich zeigt, dass der notierte Weg
  nicht trägt. Für `[D]` freiwillig.

Im selben Zug gefundene und behobene Befunde brauchen beides nicht; dort steht der Test im Fix-Commit.

## Offen

| ID | Typ | P | Sprint | Beschreibung | Status |
|---|---|---|---|---|---|
| F-1045 | [K] | P3 | Sprint 14 | **Freihand-Punkte stehen doppelt in der Arbeitsdatei** (Nacharbeit F-1036): `s:punkte` und Darstellungspfad, daher mindestens rund 25 Byte je Punkt. Abhilfe vor einer Auslieferung von Version 3: `s:punkte` relativ schreiben oder die Darstellung kürzen. **Prüfbefehl:** `GOFLAGS=-tags=novulkan,nox11 go test ./internal/fileformat -run TestFreihandGroesseJeStrich -v` (Bytes je Punkt). **Widerlegungsbedingung:** Betrachter brauchen den Pfad. | offen (Karte #403) |
| F-1034 | [V] | P3 | Sprint 14 | **Menü „Werkzeuge“ deckt im kleinsten Fenster Menü- und Fußzeile ab** (Kritik #329 K-6): Mit vierzehn Einträgen (seit #333) füllt die Liste 480 × 320 ganz aus, Auswählen y 16 … Stauchen y 302; Menülisten rollen nicht. **Prüfbefehl:** `scripts/probe.sh --drehbuch scripts/drehbuecher/rueckfall-menue-klein.txt --bild scripts/drehbuecher/testbild-1200x800.png --breite 480 --hoehe 320`, Foto 01: Liste über der Menüzeile. **Widerlegungsbedingung:** Werkzeug-Gruppen (#223) kürzen die Liste nicht. | offen (Karte #223) |
| F-899 | [V] | P3 | Sprint 14 | **Papierkorb mehrerer: Zweig `ErrIndex` ohne Test** (Nacharbeit #361, Kritik-Mutant F): Indexfehler nach gelungenem Verschieben zählt als Erfolg mit Abgleich; kein Test fährt ihn. **Prüfbefehl:** Mutant „ErrIndex-Zweig entfernt“ in `markierung.json` → Soll rot; braucht eine Naht für `InPapierkorb`. | offen (Karte #373) |
| F-889 | [K] | P3 | Sprint 14 | **Papierkorb vieler: Index je Aufnahme ganz gelesen und geschrieben, ohne Fortschritt, ohne Rückfrage; ShowItems ohne Obergrenze** (K-3). Strg+A und Entf in der ungefilterten Ansicht legen die ganze Bibliothek ohne Rückfrage weg (O(n·m)); Schließen wartet 5 s. **Prüfbefehl:** Messung `papierkorbArbeit` über 2000 Einträge in `t.TempDir()`. Entscheidung offen: Rückfrage ab welcher Zahl. | offen (Karte #373) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
