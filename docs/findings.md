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
| F-827 | [K] | P3 | Sprint 14 | **✕ an der offenen Aufnahme: kaputte Nachbarin blockiert, gefilterte Ansicht meldet „einzige Aufnahme“** (Sitzungs-Kritik 26.11., K-9): `nachbarStelle` nimmt immer i+1; ist diese beschädigt oder über dem Budget, scheitert jedes ✕. In der vollen Ansicht mit Suche/Zeitraum zählt nur die gefilterte Liste. **Prüfbefehl:** Test mit scheiterndem Laden für i+1 und vorhandenem i−1. **Widerlegungsbedingung:** der Test wirft trotzdem weg. | offen (Karte #350) |
| F-828 | [K] | P3 | Sprint 14 | **Datei → Öffnen… einer schon übernommenen, gerade offenen Datei lädt dieselbe Aufnahme neu** (Sitzungs-Kritik 26.11., K-10): `wechselAnfrage` prüft `istOffen` mit dem Pfad von außen; erst die Ladefunktion bildet auf die Aufnahme ab. Folge: Speichern, Neuladen, Rückgängig-Verlauf weg. **Prüfbefehl:** zweimal dieselbe Datei über `oeffnenDialog`. **Widerlegungsbedingung:** der Verlauf bleibt. | offen (Karte #350) |
| F-831 | [K] | P3 | Sprint 14 | **Der Standardweg mit Übernahme ist schmal getestet** (Sitzungs-Kritik 26.11., K-8; nach ADR-0016 Nachtrag 11 neu gefasst): `TestMain` schaltet `importBeimOeffnen` ab; mit Übernahme gibt es `TestOeffnenUebernimmtErstBeimSpeichern`, `TestUebernahmeWarntVorUngeschwaerztemOriginal`, `TestUebernahmeNurRegulaereDatei` und die UI-Tests `TestUebernahme*`. Es fehlen: `--ignore-limit` mit Übernahme, ein gescheitertes erstes Speichern (danach gewöhnliches Strg+S/Schließen), Rettungskopie der Datei von außen vor und nach der ersten Änderung (Nachtrag 11 Punkt 4). **Prüfbefehl:** `grep -n importLauf cmd/schnapp/*_test.go`. **Widerlegungsbedingung:** Tests für die drei Fälle. | offen (Karte #350) |
| F-832 | [V] | P2 | Sprint 14 | **Übernahme bei der ersten Änderung: Verdrahtung und Bedingungen ohne Test/Mutant** (Sitzungs-Kritik 26.11. Code, K-5): Der Aufruf `uebernahmePruefen` in `nachEreignis` und `aussenSetzen` in `EditorMit` können entfallen, ohne dass ein Test rot wird; für `TextBereichLaeuft`, `SpeichernLaeuft`, `Ruht`, `SignalVorgemerkt`, `speichernAufgegeben`, `erzwungen`, `wechsel.ziel` gibt es weder Test noch Mutant (ADR-0016 Nachtrag 11 Punkt 2 behauptet sie); der Drehbuch-Lauf schaltet die Übernahme ab, eine Sichtprobe fehlt. **Prüfbefehl:** Mutant „Aufruf in nachEreignis entfernt“ in `scripts/mutanten/import-oeffnen.json`, erwartet ROT. **Widerlegungsbedingung:** Test über `nachEreignis` plus ein Mutant je Bedingung (oder Streichung äquivalenter Bedingungen) sind rot. | offen (Karte #350) |
