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
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
| F-803 | [K] | P3 | Sprint 13 | **Runde Kappe fehlt an einer exakten Kehrtwende in zwei von vier Richtungen** (Verify 4, Fable, V4-1; Regression durch 88200e9): `internal/render/linienstrich.go` — bei achsparalleler Kehrtwende ist `sin` ±0, `sin < 0` und `Atan2(−0, −1) = −π` passen nicht zusammen, der Fächer dreht rückwärts. Pfeil A=(300,100), K=(−150,100), B=(250,100), Strich 12: Ende flach statt rund (Abweichung abseits der Kante 1,0; alter Renderer 0). `TestStrichGefaltetOhneLoch` prüft nur die Richtung, die stimmt; ADR-0023 Nachtrag 22 Punkt 2 behauptet jede Kehrtwende. **Prüfbefehl:** Kehrtwende in allen vier Richtungen und als Pfeil im analytischen Deckungstest. **Widerlegungsbedingung:** alle vier Richtungen mit Abweichung abseits der Kante ≤ 0,01. | eingeplant (Sprint 13) |
| F-804 | [K] | P3 | Sprint 13 | **`mittelpunktFang` verschiebt die innere Strichkante um bis zu 0,25 px** (Verify 4, Fable, V4-2): `internal/document/kastenstrich.go` — der Fang greift auch an den Tangentenpunkten; dann rückt die ganze innere Kante jeder Seite um Radius − hw nach innen. Kasten (30,30)–(150,110), Radius 20, Strich 39,5: Abweichung zur analytischen Deckung 0,25 (ohne Fang 0,028, Grenze 0,084); Radius 3/Strich 5,5: 0,30. Betroffen ist ein Band von rund 0,5 px Strichbreite knapp unter 2·Radius; nur der SVG-Export weicht ab, Editor und Renderer füllen dieselben Teile. Kommentar und ADR („Keil“) falsch; kein Test im Band, `mittelpunktFang = 2.0` bleibt grün. **Prüfbefehl:** Fälle Strich 39,42/39,5/39,6 bei Radius 20 und 5,5 bei Radius 3 in `TestKastenStrichDecktWieAnalytisch`. **Widerlegungsbedingung:** alle unter der Grenze des Tests, ohne dass Strich 39,9/Radius 20 in der CI wieder Splitter zeigt (F-798). | eingeplant (Sprint 13) |
| F-805 | [V] | P3 | Sprint 13 | **Die drei Aufrufstellen von `frei` hat kein Test** (Verify 4, Fable, V4-3): `editor.go` (unter der Grenze), `wechsel.go` (`uebernimm`), `objektebene.go` (Fehler in `baue`); jede einzeln zurückgebaut, `./internal/ui` bleibt grün. **Prüfbefehl:** Rückbau je Stelle per Overlay, Soll rot. | eingeplant (Sprint 13) |
| F-806 | [D] | P3 | Sprint 13 | **Doku nach Verify 4** (Fable, V4-4): `BenchmarkAufwandNaheKantenBudget` heißt „knapp unter KantenBudget“, hat aber 171 Pfeile (jetzt 39,8 %, Grenze 429); „275 MiB“ sind 262,5 MiB (`objektebene.go`, ADR-0023 Nachtrag 20 Zusatz, Archiv F-799); „Keil“ (F-804) und „jede Kehrtwende“ (F-803) im ADR; Archiv F-796 „0,18–0,63“ statt „0,15 bis 0,63“; Kopfzeile `lehren.md`. Hinweis ohne Handlung: Commit abad68c trägt nur den Sonnet-Trailer. **Prüfbefehl:** `grep -n '275 MiB' internal/ui/objektebene.go docs/adr/0023-zuschneiden-und-stauchen.md` (Soll leer). | eingeplant (Sprint 13) |
