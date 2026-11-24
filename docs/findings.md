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
| F-706 | [K] | P3 | Sprint 12 | **Zug in der Ebenenliste überlebt den Wechsel der Aufnahme** (Abschluss-Kritik C-2): `uebernimm` setzt `e.ebenen.druck` nicht zurück, das Loslassen ordnet die neue Aufnahme um. **Prüfbefehl:** Gegenprobe `tmp/sprint12/abschluss/C/` (`TestKritikCEbenenZugUeberWechsel`). **Widerlegungsbedingung:** Gio bricht den Zeigerzug beim Tastendruck selbst ab. | backlog |
| F-708 | [K] | P3 | Sprint 12 | **Schließen wartet nicht auf Schlagworte/Papierkorb der Bibliotheksansicht** (Abschluss-Kritik C-4, gelesen): `zerstoert` wartet nicht auf `e.bibl.arbeitLaeuft`. **Prüfbefehl:** Test mit gehaltener Arbeit: Schließen wartet (mit Frist). **Widerlegungsbedingung:** `schliessen()` hält das Fenster bei laufender Arbeit doch offen. | backlog |
| F-710 | [D] | P3 | Sprint 12 | **„Copied to clipboard“ nach einem Wechsel bezieht sich auf die vorige Aufnahme** (Abschluss-Kritik C-6, gelesen): der Wechsel wartet nicht auf `kopierLaeuft`, die Antwort vergleicht mit dem Stand der neuen Sitzung. **Prüfbefehl:** Test: Kopieren halten, wechseln, freigeben → Meldung nennt die vorige oder kein Wechsel. | backlog |
| F-723 | [D] | P3 | Sprint 12 | **`docs/open-decisions.md` endet mit Sprint 10** (Abschluss-Kritik D-5): „Umsetzung offen (F-144)“ ist umgesetzt; Entscheidungen aus Sprint 11 und 12 fehlen. **Prüfbefehl:** `grep -c 'Sprint 12' docs/open-decisions.md` → ≥ 1. | backlog |
| F-724 | [D] | P3 | Sprint 12 | **Glossar: Begriffe aus Sprint 12 fehlen** (Abschluss-Kritik D-6): Wechsel, Navigator, Einrasten, Farbpaar, Agenten-Aufnahme. **Prüfbefehl:** fünf Einträge vorhanden. | backlog |
| F-725 | [D] | P3 | Sprint 12 | **SPDX-Köpfe fehlen in 17 Dateien, 36 Testdateien heißen nach Kritik oder Karte** (Abschluss-Kritik D-7) gegen `CONTRIBUTING.md`. **Prüfbefehl:** Dateien ohne SPDX → 0; oder Regel in beiden CONTRIBUTING-Fassungen ändern. | backlog |
| F-743 | [D] | P3 | Sprint 12 | **Prüfbefehle und Belege offener Befunde zeigen auf `tmp/sprint12/abschluss/`** (Sitzungs-QA Q-6): F-695, F-700, F-701, F-705 bis F-707, F-709; `tmp/` ist nicht versioniert. **Prüfbefehl:** `grep -c 'tmp/sprint12/abschluss' docs/findings.md` → 0 nach Umsetzung. | backlog (Karte #280: Probetests bei Umsetzung der Karten als Regressionstests ins Repo; bis dahin `tmp/sprint12/abschluss/{A,B,C,D,V,V2,V3}` nicht räumen) |
| F-745 | [D] | P3 | Sprint 13 | **Pfeile über einer Stauch-Naht: Renderer und Editor stauchen, SVG-Export schneidet ab** (Karte #275): Pfeile bilden Renderer und Editor über 48 Stützpunkte ab (ADR-0023 Punkt 7), der SVG-Export zeigt sie kachelweise abgeschnitten; seit Nachtrag 13/14 schneiden Text, Dreieck und Lupen-Linie überall ab. **Prüfbefehl:** Renderbild eines Pfeils über einer Naht gegen das ungestauchte Bild (wie `schraege_naht_test.go`). **Widerlegungsbedingung:** ein ADR-Nachtrag hält das Stauchen von Pfeilen in allen Ausgängen fest und der SVG-Export folgt. | eingeplant (Sprint 13) |
