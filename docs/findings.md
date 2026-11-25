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
| F-782 | [K] | P3 | Sprint 13 | **Editor: runde Kästen mit breitem Strich über vielen Kacheln kosten weiter Sekunden je Bild** (Verify Sprint 13 (Fable) V-2): 200 Kästen über 4096 Kacheln, llvmpipe: Strich 1 0,45 s, 12 0,60 s, 40 1,55 s; knapp unter KantenBudget mit Strich ~120 rund 2,6 s. `TestEditorGanzesBildUeberVielenNaehten` prüft nur Strich 1. **Prüfbefehl:** `TestVerifyBreiterStrich` per Overlay `tmp/sprint13/verify/overlay-zeit.json` in `./internal/ui` mit llvmpipe. | eingeplant (Sprint 13) |
| F-783 | [K] | P3 | Sprint 13 | **Editor: dicker Strich an kleinem rundem Kasten zeigt ein Loch, der Renderer füllt** (Verify Sprint 13 (Fable) V-3, Altbestand aus Gios Strich): 40×40, Radius 20, Strich 60 → 364 Pixel Abweichung, Mitte Editor A=2, Renderer A=255; ADR-0023 Nachtrag 18 Punkt 1 („wie der Renderer“, „Trapeze überlappen nicht“) gilt hier nicht. **Prüfbefehl:** `TestVerifyDickerStrich` per Overlay `tmp/sprint13/verify/overlay-kasten.json` in `./internal/ui` mit llvmpipe. | eingeplant (Sprint 13) |
| F-784 | [D] | P3 | Sprint 13 | **Testlücken zu F-765/F-769/F-770** (Verify Sprint 13 (Fable) V-4): grün bleiben ohne `ladeMitRettung` in `oeffnerFuer`, ohne Fremdheitsteil von `passtZu`, mit `ladeVersuche = 2`, ohne `r.ende()` nach gescheitertem `VorgabeKante`, mit `rand := 0` in `strichJeStueck`, ohne Vorschau-Zweig. **Prüfbefehl:** `python3 tmp/sprint13/verify/mut.py A1-oeffner-ohne-warten A5-passt-ohne-fremd A8-versuche-unbegrenzt-1 B6-rand-null B9-vorschau-zweig` (Soll: alle rot). | eingeplant (Sprint 13) |
