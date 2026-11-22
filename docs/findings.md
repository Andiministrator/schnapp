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
| F-98 | [V] | P3 | Sprint 4 | **`--list-screens` nicht maschinenlesbar** (Kritiker B, Sitzungs-QA Sprint 4): übersetzt, mit „ד; `--screen=NAME` lässt sich nur über diese Liste füttern (E-14). **Prüfbefehl:** `schnapp --list-screens --json` liefert stabile Namen. **Widerlegungsbedingung:** — | eingeplant (Sprint 11, Karte #43) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
| F-379 | [D] | P3 | Sprint 11 | **Präfix-Regel in den übrigen Paketen nicht geprüft** (Bericht des Agenten zu Karte #159, Sonnet; ADR-0019 Nachtrag 2): F-362 hat document, fileformat, export und library umgestellt; `render`, `canvas`, `portal`, `settings`, `saveprogress` (und `capture` über F-339 hinaus) wurden nicht auf ungehüllte Rückgaben exportierter Funktionen geprüft. **Prüfbefehl:** je Paket ein Präfix-Test über die exportierten Fehlerwege (Muster `internal/export/praefix_paketgrenze_test.go`, Hülle `internal/errprefix`). | backlog |
| F-387 | [K] | P3 | Sprint 11 | **Formatversion wird erst nach dem Inhalt geprüft, Version 0 wird angenommen** (Bericht des Agenten zu Karte #44, E-15, Sonnet; `internal/fileformat/svg.go` `leseSVG`): Ein neueres Format mit unbekanntem Objekttyp (`s:format="2"`, `s:typ="ellipse"`) meldet `ErrBeschaedigt` statt `ErrFormatNeuer`, weil `l.objekt` schon in der Schleife scheitert und die Versionsprüfung erst nach dem Parsen kommt; `s:format="0"` wird gelesen, obwohl es Version 0 nie gab. Folge nur Genauigkeit der Meldung (Oeffne behandelt beides als fremd), `docs/format.md` §11 verlangt aber die Ablehnung unbekannter Versionen vorab. **Prüfbefehl:** Test mit `pngMitSVG` (`internal/fileformat/format_konform_test.go`): Version 2 mit unbekanntem Typ liefert `ErrFormatNeuer`, Version 0 wird abgelehnt. **Widerlegungsbedingung:** Die Version steht erst nach den Objekten im Strom fest (etwa weil der Leser streamt und `s:format` am Wurzelelement nicht vorab kennt) — dann Prüfung am Wurzelelement nicht möglich und Meldung so lassen. | offen (Backlog, Karte #170) |
| F-392 | [D] | P3 | Sprint 11 | **F-328 gegen einen echten Compositor ungeprüft** (Sitzungs-QA 22.11., Kritiker A, QA-6): Die Widerlegungsbedingung von F-328 (liefert KWin `cancelled` rechtzeitig?) ist offen; zu prüfen auch, ob KWin `cancelled` schickt, wenn es `set_selection` wegen veralteter Seriennummer ablehnt (dann entfiele die Warnung, obwohl nichts kopiert ist). Doku (CLAUDE.md, ADR-0015 Nachtrag 3) im selben Zug nachgezogen. **Prüfbefehl:** Handprobe Andi im echten KWin mit `SCHNAPP_DEBUG=1`: Strg+C ohne Halter (Rückfall), in anderem Programm kopieren, Schließen → keine Warnung; Ergebnis im Register. | eingeplant (Sprint 11, Karte #172; Handprobe Andi im echten KWin) |
