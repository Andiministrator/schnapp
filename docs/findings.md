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
| F-517 | [V] | P3 | Sprint 12 | **Mutanten U1 bis U12 des Kritikers B liegen nur in `tmp/sprint12/qa-sitzung2/b/mutanten.py`** (Bericht des Agenten zu Karte #213): außerhalb von `scripts/mutanten` veralten sie mit jeder Codeänderung (U2, U9 mussten schon angepasst werden) und laufen in keiner Gruppe. **Prüfbefehl:** `cd scripts/mutanten && go run . -wurzel ../.. -gruppe ui-oberflaeche` enthält U1–U12 und alle sind rot (Prüfbefehl korrigiert nach Sitzungs-QA B-9: `go run ./scripts/mutanten` aus der Wurzel geht nicht, eigenes Modul). | eingeplant (Sprint 12) |
| F-518 | [V] | P3 | Sprint 12 | **`TestNahtPruefsumme` vergleicht feste SHA-256 über Fließkomma-Rastern** (Bericht des Agenten zu Karte #213): auf einer anderen Architektur (arm64 mit FMA) könnte der Wert abweichen, ohne dass etwas kaputt ist; die CI läuft nur auf amd64. **Prüfbefehl:** `GOARCH=arm64` (qemu) oder Prüfsumme nur unter `runtime.GOARCH == "amd64"`, sonst Vergleich gegen ein Referenzbild mit Toleranz. **Widerlegungsbedingung:** Go garantiert für die betroffenen Rechnungen bitgleiche Ergebnisse auf allen Architekturen (kein FMA-Fusing ohne explizite Klammern). | eingeplant (Sprint 12) |
| F-519 | [V] | P3 | Sprint 12 | **Außenkante gerundeter Kästen (Radius > 0) nicht über Renderer, SVG und Editor verglichen** (Bericht des Agenten zu Karte #211): `TestKastenEckeRund` prüft nur den Renderer; Gleichheit hängt allein am gemeinsamen `WirksamerEckradius`. **Prüfbefehl:** Test mit Kasten Radius 6, Strich 8: Eckpixel innen/außen der Rundung im Renderer, im gerasterten SVG-Pfad und im Editor-Umriss gleich. | eingeplant (Sprint 12) |
| F-520 | [K] | P3 | Sprint 12 | **Gio mischt halbdurchsichtige Flächen linear, Renderer und Export in sRGB** (Bericht des Agenten zu Karte #212): nach F-407 betrifft das noch alle übrigen halbdurchsichtigen Gio-Zeichnungen im Editor (z. B. Kasten mit Deckkraft < 100 %, Schatten der Nähte, Hinweisflächen). **Prüfbefehl:** Verfahren von `TestEditorFotoGegenRenderer` mit einem halbdurchsichtigen Kasten — Abweichung ≤ 2 je Kanal. **Widerlegungsbedingung:** Lässt sich die Mischung im Gio-Patch nicht auf sRGB umstellen, ohne andere Oberflächenteile zu verfälschen, halbdurchsichtige Beschriftungen wie die Naht vorrastern. | eingeplant (Sprint 12) |
