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
| F-887 | [K] | P2 | Sprint 14 | **Papierkorb mehrerer: „mitnehmen“ wird bei der Übernahme nicht mehr gegen Markierung und Gezeigtes geprüft** (Kritik Fable 27.11. zu #361, K-1): `papierkorbMehrere` friert die übrigen markierten beim Auslösen ein, `wechselAbholen` wirft sie nach dem Laden ungeprüft weg. Esc, Strg+Klick oder eine geänderte Suche während Speichern und Laden ändern nichts mehr. **Prüfbefehl:** Test „Esc während des Wechsels“: offene und eine weitere markiert, Entf, Esc vor der Übernahme → Soll: die weitere bleibt. **Widerlegungsbedingung:** die Markierung lässt sich während des Wechsels nicht ändern. | offen (Karte Sitzungs-QA (10)) |
| F-888 | [K] | P3 | Sprint 14 | **Tasten der Markierung in der Leiste nehmen `Eintraege()[i]`, der Klick `gezeigt[i]`** (K-2): liest die Leiste zwischen Zeichnen und Taste neu, markiert Strg+Leertaste/Umschalt+Pfeil die Nachbarin. **Prüfbefehl:** Test mit `Setze` zwischen Zeichnen und Strg+Leertaste → Soll: die gezeigte ist markiert. | offen (Karte Sitzungs-QA (10)) |
| F-889 | [K] | P3 | Sprint 14 | **Papierkorb vieler: Index je Aufnahme ganz gelesen und geschrieben, ohne Fortschritt, ohne Rückfrage; ShowItems ohne Obergrenze** (K-3). Strg+A und Entf in der ungefilterten Ansicht legen die ganze Bibliothek ohne Rückfrage weg (O(n·m)); Schließen wartet 5 s. **Prüfbefehl:** Messung `papierkorbArbeit` über 2000 Einträge in `t.TempDir()`. Entscheidung offen: Rückfrage ab welcher Zahl. | offen (Karte #373) |
| F-890 | [D] | P3 | Sprint 14 | **Esc hebt die Markierung der Leiste nur mit Tastaturfokus an einer Vorschau auf** (D-1), ADR-0036 Zusatz #361 Punkt 3 und README sagen es ohne Bedingung; eine stehen gebliebene Markierung nimmt ein späteres Entf mit. **Prüfbefehl:** Markieren, Fokus Leinwand, Esc → Soll: Markierung leer. | offen (Karte Sitzungs-QA (10)) |
| F-891 | [D] | P3 | Sprint 14 | **Doku #361 lückenhaft** (D-2): F1 nennt nur „Ctrl/Shift+Click“; Hinweis „Del moves them…“ auch beim bloßen Überfahren (Entf wirkt nur mit Fokus an der Vorschau); README verschweigt, dass der Klick ohne Taste in der Leiste zugleich öffnet; Glossar ohne „Markierung“; CLAUDE.md ohne Sichtprobe. **Prüfbefehl:** `grep -n "Markierung" docs/glossar.md`. | offen (Karte Sitzungs-QA (10)) |
| F-892 | [V] | P2 | Sprint 14 | **Testlücken #361: 17 von 19 Mutanten der Kritik bleiben grün** (V-1): Kette der Nachbarinnen mit mitnehmen, gefilterte Ansicht, Sperren, Teilfehler, Abgleich, Verlauf je Aufnahme, Meldungen in der Mehrzahl, Rechtsklick der Ansicht, Hinweis der Leiste, Anker der Ansicht, Umschalt+Pfeil der Ansicht, loeschRuht im Kontextmenü, Zahl beim Auslösen, ✕ der Ansicht, Nachbarsuche weiter als 1, loeschRuhe beim Wechsel; Umschalt+F10 an markierter ohne Test. **Prüfbefehl:** Mutanten A–T der Kritik in `scripts/mutanten/markierung.json` → Soll: alle rot. | offen (Karte Sitzungs-QA (10)) |
| F-893 | [V] | P3 | Sprint 14 | **Reste #361** (V-2): unerreichbarer Zweig `len(rels)==0 && offenBleibt` in `papierkorbArbeit`; `wechselNachSpeichern` leert bei Abbruch nur `ziel`, `papierkorb`/`weitere`/`mitnehmen` bleiben stehen. **Prüfbefehl:** Code lesen. | offen (Karte Sitzungs-QA (10)) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
