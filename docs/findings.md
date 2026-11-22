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
| F-406 | [K] | P3 | Sprint 11 | **Mutant S2 in `rescue` („ältere überschreibt jüngere“) schwankt** (Bericht des Agenten zu Karte #96): `TestSicherungReihenfolge` erkennt ihn nur zeitabhängig, in 4 Läufen 2-mal überlebt. Vorhanden, nicht von #96 verursacht. **Prüfbefehl:** `cd scripts/mutanten && for i in 1 2 3 4 5; do GOFLAGS= go run . -wurzel ../.. -nur "S2 ältere überschreibt jüngere"; done` — erwartet fünfmal rot. **Widerlegungsbedingung:** Ist die Reihenfolge ohne eingespeiste Uhr oder Nummer nicht festzumachen, weil das Produkt selbst nach Zeit entscheidet, dann die Entscheidung im Produkt auf eine Seriennummer umstellen. | offen (Karte „Kanten-Reste #96“) |
| F-407 | [K] | P3 | Sprint 11 | **Editor mischt die halbdurchsichtigen Kantenschatten womöglich anders als der Renderer** (Bericht des Agenten zu Karte #96): Gio mischt vermutlich linear, `render` in sRGB; bis #96 gab es an der Naht nur deckende Bänder. Nicht gemessen. **Prüfbefehl:** Editor-Foto aus `scripts/drehbuecher/stauchkante-formen-96.txt` bei Zoom 100 % gegen das gerenderte PNG derselben Datei an der Naht pixelweise vergleichen; erwartet Abweichung ≤ 2 je Kanal. **Widerlegungsbedingung:** Liegt die Abweichung an der Skalierung des Fotos statt an der Mischung, dann mit 1:1 und Skala 1 wiederholen, bevor Deckkräfte angepasst werden. | offen (Karte „Kanten-Reste #96“) |
| F-408 | [K] | P3 | Sprint 11 | **Rechenbudget bei vielen Abriss-Nähten** (Bericht des Agenten zu Karte #96): 10 Schichten mit 1-px-Stützpunkten kosten je Naht auf 8192 px rund 1,6e7 Kanteneinheiten; ab etwa 600 Nähten auf 8192 px lehnt `KantenBudget` (1e10) eine Datei beim Öffnen ab (Speichern bricht nie ab). Real kaum erreichbar. **Prüfbefehl:** Test mit 600 Abriss-Nähten auf 8192 px: `render.Pruefe` lehnt nicht ab. **Widerlegungsbedingung:** Lässt sich der Aufwand nicht teilen (je Achse gemeinsame Punkte), dann die Grenze der Nahtzahl in `PruefeInhalt` dokumentieren statt das Budget zu heben. | offen (Karte „Kanten-Reste #96“) |
| F-396 | [S] | P3 | Sprint 11 | **Temporäre Exportdatei `.schnapp-export-*` bleibt bei Abbruch von `--output`/`--export` liegen** (Kritiker A KA-3). Teilweise behoben in f25af93: erst vollständig im Speicher rendern, dann schreiben; `docs/agents.md` nennt den Rest. Offen: SIGTERM während des kurzen Schreibens. **Prüfbefehl:** Test mit Signalwache bzw. Rückruf, der nach Abbruch keinen Rest im Zielordner findet. **Widerlegungsbedingung:** Eine Signalwache in der Kommandozeile stört den Editor-Weg (Signale dort schon gefangen) — dann den Rest nur dokumentieren. | offen (Karte „Kritik-Reste 22.11.“) |
| F-405 | [V] | P3 | Sprint 11 | **Rechte 0600 von `--output` nur im Unit-Test, nicht im Echtlauf geprüft** (Bericht des Behebungs-Agenten zu F-397). **Prüfbefehl:** Drehbuch `agenten-kommandozeile.sh` prüft `stat -c %a` einer frisch benannten Ausgabe = 600. | offen (Karte „Kritik-Reste 22.11.“) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
