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
| F-747 | [K] | P1 | Sprint 13 | **Zeilen × Bereiche: Prüfen beim Öffnen und jedes Editor-Bild werden über vielen Nähten um Größenordnungen langsamer, das Budget zählt nichts** (Sitzungs-Kritik 25.11. (Fable) A-2): 20 Texte mit je 10 000 Zeilen über 1024 Zeilenstreifen: `Pruefe` 5,2 s statt 22 ms, Editor 5,3 s je Bild. **Prüfbefehl:** Zeittest wie `TestPruefeVieleStreifenUndPfeile` mit Text (Vorlage `TestQAVieleZeilenVieleStuecke`). **Widerlegungsbedingung:** Zeilen je senkrechtem Abschnitt eingegrenzt, Zeittest unter 1 s. | eingeplant (Sprint 13) |
| F-748 | [K] | P3 | Sprint 13 | **Pfeil über einer Naht weicht bei langen Pfeilen, breiten Strichen und Spitze im Streifen auch außerhalb des Kantenbands vom SVG-Export ab** (Sitzungs-Kritik 25.11. (Fable) A-4, C-4): 48 Stützpunkte liegen weiter auseinander als das Band. **Prüfbefehl:** Renderbild eines Pfeils (20,20)→(3100,3100) bzw. Strich 32 über einem Streifen gegen das ungestauchte Bild. | eingeplant (Sprint 13) |
| F-749 | [D] | P3 | Sprint 13 | **Kasten mit Eckradius wird an einer Naht neu gerundet statt abgeschnitten** (Sitzungs-Kritik 25.11. (Fable) A-5): Eckradius 60, rechte Kante im Streifen: links der Naht 1186 Pixel anders als im SVG. **Prüfbefehl:** runder Kasten über Streifen gegen das ungestauchte Bild. **Widerlegungsbedingung:** Ausnahme im ADR-0023 benannt. | eingeplant (Sprint 13) |
| F-750 | [K] | P2 | Sprint 13 | **Glyphen werden je Bereich neu gerastert, das Budget zählt sie einmal** (Sitzungs-Kritik 25.11. (Fable) A-3): bei 1023 Spaltenstreifen Faktor 141 (Bild 4,6 s statt 33 ms) bei gleicher Zählung; Zeitversprechen von GlyphBudget (ADR-0020 Nachträge 12/13) nicht gedeckt; ebenso 1000 Sprechblasen über 4096 Kacheln 25,7 s. **Prüfbefehl:** Vorlage `TestQAGlyphenJeStueck`. | eingeplant (Sprint 13) |
| F-751 | [V] | P3 | Sprint 13 | **Naht-Tests im Editor nur mit einem Spaltenstreifen** (Sitzungs-Kritik 25.11. (Fable) A-6): Renderer hat seit 26b5f81 Zeilen-Naht, Kreuz, Zuschnitt, Deckkraft; der Editor-Test nicht. **Prüfbefehl:** headless-Gegenstück zu `TestKreuzendeNaehteMitZuschnitt`. | eingeplant (Sprint 13) |
| F-752 | [K] | P3 | Sprint 13 | **Texteingabe an einem Text, dessen Ende im Streifen liegt: getippte Zeichen sind unsichtbar** (Sitzungs-Kritik 25.11. (Fable) A-9, Folge von F-746). **Widerlegungsbedingung:** Entscheidung, Eingabe zu sperren oder während der Eingabe ungeschnitten zu zeigen, im ADR. | eingeplant (Sprint 13) |
| F-755 | [K] | P2 | Sprint 13 | **Abgleich und Ablegen eines zweiten Prozesses erzeugen eine doppelte Indexzeile** (Sitzungs-Kritik 25.11. (Fable) B-3): Ergänzen zwischen Datei und Zeile. **Prüfbefehl:** Test mit zwei Bibliotheken (Vorlage `TestQAB_AbgleichUndAblegenZweiterProzessDoppelteZeile`). | eingeplant (Sprint 13) |
| F-756 | [K] | P2 | Sprint 13 | **Ablegen an fehlenden Index heilt alle Vorschaubilder im Vordergrund der Aufnahme** (Sitzungs-Kritik 25.11. (Fable) B-4): 41 alte Aufnahmen 1,58 s statt 53 ms. **Prüfbefehl:** Vorlage `TestQAB_AblegenOhneIndexHeiltAllesImVordergrund`. | eingeplant (Sprint 13) |
| F-757 | [K] | P3 | Sprint 13 | **`rescue.UmDatei` prüft „nur Metadaten“ nicht, und ein gleichzeitig öffnendes Fenster wird fremd** (Sitzungs-Kritik 25.11. (Fable) B-5). **Prüfbefehl:** Vorlage `rescue_qa_b_test.go`. | eingeplant (Sprint 13) |
| F-758 | [K] | P3 | Sprint 13 | **MCP-Abbruch: Kennungsvergleich über float64 trifft andere Anfragen (über 2^53), `initialize` streichbar** (Sitzungs-Kritik 25.11. (Fable) B-6). **Prüfbefehl:** Vorlage `mcp_qa_b_test.go`. | eingeplant (Sprint 13) |
| F-759 | [K] | P3 | Sprint 13 | **Wechsel der Aufnahme während des Kopierens gilt still nicht** (Sitzungs-Kritik 25.11. (Fable) B-7): ohne Hinweis in der Statuszeile. | eingeplant (Sprint 13) |
