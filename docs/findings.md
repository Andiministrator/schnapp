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
| F-787 | [K] | P2 | Sprint 13 | **Editor-Bildzeit über vielen Kacheln nicht im Budget** (Karte #297 (Opus), 25.11., N-1): `render.Pruefe` nimmt bei 4096 Kacheln mindestens 2000 runde Kästen mit Strich 1 an; ein Editorbild braucht dann rund 30 s (1000 Kästen 4,8 s, 950 mit Strich 12 3,7 s; belasteter Rechner). Das Kostenmodell schützt den Export, nicht das Editorbild. **Prüfbefehl:** Zeittest wie `TestEditorGanzesBildUeberVielenNaehten` mit 1000 und 2000 Kästen, llvmpipe. **Widerlegungsbedingung:** Editor-Budget je Bild (Objekte × Bereiche) oder zwischengespeicherte Objektebene, Bild unter 2 s an der Grenze. | eingeplant (Sprint 13) |
| F-788 | [K] | P3 | Sprint 13 | **Renderer addiert die Kantenglättung an Strichgelenken** (Karte #297 (Opus), 25.11., N-2): Rechtecke und Kreise einer Füllung addieren sich; äußere Randpixel einer Rundung voll statt halb gedeckt (leichte Treppe). **Prüfbefehl:** Randpixel eines runden Kastens mit Strich 24 gegen analytische Deckung. | eingeplant (Sprint 13) |
| F-789 | [D] | P3 | Sprint 13 | **Restlücken nach #297** (Karte #297 (Opus), 25.11.): Vorschau der Ebenenliste zeichnet runde Kästen mit Gios Strich und zeigt bei Strich > 2·Radius dasselbe Loch wie F-783; Mutant B8 („Fächer als einzelne Dreiecke“) überlebt nach dem Zusammenführen. **Prüfbefehl:** `cd scripts/mutanten && go run . -gruppe text-naht -nur "B8 Editor: Fächer einer Rundung als einzelne Dreiecke (F-783)"` (Soll ROT). | eingeplant (Sprint 13) |
