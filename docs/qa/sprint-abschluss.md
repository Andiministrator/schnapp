# Sprint-Abschluss

Maßgeblich ist der Command `/sprint-abschluss` (`~/.claude/commands/sprint-abschluss.md`). Diese Datei nennt, was für
Schnapp zusätzlich gilt.

## Phase 1 — Abhängigkeits-Check
- Welche Karten und Pakete hängen an den Änderungen? Aufnahme ↔ Dokument ↔ Datei ↔ Zeichenpakete ↔ Oberfläche.
- Ändert sich eine öffentliche Schnittstelle von `svg`/`raster`/`geo`? Dann ist das Schwesterprojekt betroffen (ADR-0012).
- Ändert sich das Dateiformat? Dann Formatversion und Lesbarkeit alter Dateien (ADR-0006, ADR-0008).

## Phase 2 — Kritiker-Lauf (Sub-Agent)
Als Sub-Agent mit selbst-enthaltendem Prompt (kein Memory: Dateipfade, Regeln, ADR-Bezüge explizit).
Der Auflagen-Block steht im Command `sprint-abschluss` (Phase 2) und wird von dort in jeden Prompt kopiert, zusammen
mit „Nichts nach /tmp" und dem Ort `tmp/<sprint>/<lauf>/`.

Der Kritiker prüft aktiv:
- **Sicherheit [S]:** Überlebt Geschwärztes irgendwo (sichtbares Bild, SVG-Block, Original, Vorschaubild der
  Bibliothek, Zwischenablage, Exporte SVG/PDF/JPG)? Metadaten mit Pfaden, Benutzer-, Rechner-, Programmnamen? Rechte
  der Bibliotheksdateien? Eingelesene Dateien (SVG, PNG-Block) als Angriffsfläche (übergroße Werte, externe
  Verweise in `<image>`)?
- **Stabilität [K]:** Absturz bei fehlender KWin-Schnittstelle, mehreren Monitoren, HiDPI-Skalierung, leerem Bereich;
  Datenverlust beim Speichern (atomar schreiben).
- **ADR-Verstöße:** X11-Wege, unbegründete Fremdpakete, zerstörendes Bearbeiten außer Schwärzen.
- **Öffentliches Repo:** Kunden-/Firmenbezug, lokale Pfade, Mailadressen, Hausnetz-Angaben, echte Screenshots.
- **Regression und Vollständigkeit:** Rundlauf Datei speichern/laden, Rückgängig über alle neuen Werkzeuge.

## Phase 2b — Verify-Lauf über die Nacharbeit
Nachdem die Kritiker-Findings behoben sind: ein zweiter Sub-Agent mit **frischem Kontext** (nicht dasselbe Modell, das
die Nacharbeit gemacht hat) prüft **nur den Nacharbeits-Diff** — und ob Commit-Message, Findings-Status und
Kartenkommentare vom Code gedeckt sind. Entfällt, wenn der Kritiker nichts gefunden hat oder alle Findings ohne
Code-Änderung triagiert wurden (im Recap vermerken). Ein **zweiter** Durchlauf ist Pflicht, wenn die Behebung der
2b-Findings einen `[S]`- oder `[K]`-Pfad anfasst — sonst nicht.

## Phase 3 — Aufräumen & Triage
- `tmp/` aufgeräumt (nur ausgeschriebene Pfade), Knowledge und Glossar aktuell.
- Alle Findings triagieren: sofort · eingeplant · backlog · akzeptiert. Jedes offene Finding hat eine Board-Karte.

## Freigabekriterien
Alle P0 behoben · alle `[S]` behoben (jede Priorität; Ausnahmepfad unten) · alle `[K]` P0/P1
behoben · alle `[B]` behoben · Sprint-Definition erfüllt. Maßgeblich: `~/.claude/commands/sprint-abschluss.md`
§Sprint-Release-Check.

## Ausnahmepfad für `[S]` P2/P3

**Ausnahmepfad für `[S]` P2/P3** — ein offenes `[S]`-Finding blockiert den Sprint-Release ausnahmsweise nicht, wenn ALLE vier Bedingungen erfüllt sind:

1. Es ist eine bewusste Design-Entscheidung (kein vergessener Fehler) — mit Verweis auf das ADR oder den Nachtrag, der sie trägt.
2. Priorität P2 oder P3. `[S]` P0/P1 blockiert ausnahmslos.
3. Die Begründung steht am Fundort im Register (Status-Spalte des Findings in `docs/findings.md`: „Ausnahmepfad Sprint N“ plus Grund, und bis wann bzw. mit welcher Karte es gelöst wird).
4. Das Finding wird im Recap der 🏁-Sprint-Abschluss-Karte unter „Ausnahmepfad [S]“ namentlich genannt.

Fehlt eine Bedingung, blockiert das Finding.
