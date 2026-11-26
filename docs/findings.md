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
| F-834 | [S] | P1 | Sprint 14 | **#356: gemerktes Dokument überschreibt einen fremden Stand** (Sitzungs-Kritik 26.11. (4), Fable, K-1): `merkeVerlassene` nahm den Dateistand frisch von der Platte statt den Stand der eigenen Speicherfunktion; war die Aufnahme unverändert (kein Speichern vor dem Wechsel) und hatte ein anderes Fenster/`--annotate` die Datei inzwischen geschwärzt, nahm der Rückwechsel das alte Dokument, und das nächste Speichern schrieb das Ungeschwärzte über die Datei (F-110 ausgehebelt). **Prüfbefehl:** `go test ./internal/ui -run TestMerkeVerlasseneNichtBeiFremdemStand` und `go test ./cmd/schnapp -run TestFortsetzen`. | in Arbeit (Karte Sitzungs-Kritik 26.11. (4)) |
| F-835 | [K] | P2 | Sprint 14 | **Testlücken #356/#364** (Kritik K-2, K-4): Mutant V2 (Schwärzung im Verlauf) fälschlich äquivalent — Schwärzen + Rückgängig ohne Speichern ist ungeändert mit Schwärzung im Verlauf; Sperre, Verweis und Kürzung von colors.ini ohne Test/Mutant. **Prüfbefehl:** Gruppen `verlaufmerker`, `farbfelder` scharf, alle rot. | in Arbeit |
| F-836 | [K] | P2 | Sprint 14 | **Farbfelder: Index hinter dem Ende nach Entfernen im selben Bild** (K-3): `farbfelderAuswerten` lief nach `farbfeldEntferne` mit der alten Länge weiter (Panik möglich, Klick setzt Nachbarfarbe). **Prüfbefehl:** `go test ./internal/ui -run TestFarbfelderEntfernenUndKlickImSelbenBild`. | in Arbeit |
| F-837 | [D] | P3 | Sprint 14 | **colors.ini: Fehlertext nennt „styles file“, „#“-Kommentar gilt als Fehler** (K-5, K-6). **Prüfbefehl:** `go test ./internal/ui -run TestFehlerGrundUebersetzt`, `go test ./internal/settings -run TestParseFarben`. | in Arbeit |
| F-838 | [K] | P3 | Sprint 14 | **#356: Eintrag schon beim Start des Ladens verbraucht; weggeworfene gemerkte Aufnahme bleibt im Speicher** (K-7). **Prüfbefehl:** `go test ./internal/ui -run 'TestWechselZurueck\|TestPapierkorbNichtOffeneVergisst'`. | in Arbeit |
| F-839 | [D] | P2 | Sprint 14 | **Doku nach #353–#364** (Kritik Doku D-1 bis D-12): „colour“ statt „color“ in Oberflächentexten, README en/de veraltet (vier zuletzt benutzte, colors.ini fehlt, Punkt, Verlauf), geänderte Altstellen in ADR-0021/0031/0034/0036 nicht markiert, Begriff „Farbfeld“ doppelt belegt und nicht im Glossar, de.po-Begriffe („ungesichert“, „ignoriert“), Hinweis am „+“, CLAUDE.md-Satz zerrissen, Schwelle `farbwaehlerHoeheDp` und vier Drehbücher nicht nachgezogen, Farbliste der Fußzeile undokumentiert, veraltete Kommentare, Lehren fehlen. **Prüfbefehl:** `git grep -n -i colour -- internal` leer; README nennt colors.ini. | in Arbeit |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
