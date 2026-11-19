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
| F-29 | [V] | P3 | — | **`marke/werkzeug/cd.py` `pdf()` kann ein veraltetes Roh-PDF übernehmen** (zweiter Verify-Lauf): `tmp/marke/corporate-design.roh.pdf` wird vor dem Chromium-Lauf nicht geprüft; endet Chromium mit Exit 0 ohne Datei, wird das PDF des Vorlaufs bereinigt und übernommen (sauber, aber inhaltlich alt). **Prüfbefehl:** nach dem Chromium-Aufruf Existenz und mtime ≥ Laufbeginn prüfen; Test mit gestubbtem `chromium`, der nichts schreibt → Fehler erwartet. **Widerlegungsbedingung:** — | backlog (Karte) |
| F-42 | [V] | P3 | Sprint 2 | **Gio-Patch: Restpunkte vor der Einreichung bei Gio** (Kritiker D-7, `extern/gio/app/os_wayland.go`): `viewporter`/`fracScaleMgr` werden in `destroy` nicht zerstört; `WaylandOutputs()` öffnet je Aufruf eine ganze Verbindung; `sortedOutputs` sortiert lexikographisch (`DP-10` < `DP-2`); `log.Printf` aus der Bibliothek ist für Upstream fraglich. **Prüfbefehl:** `grep -n 'viewporter\|fracScaleMgr' extern/gio/app/os_wayland.go` zeigt Destroy-Aufrufe. **Widerlegungsbedingung:** Upstream übernimmt eine eigene Fassung | backlog (Karte #17) |
| F-43 | [V] | P3 | Sprint 2 | **CI-Aktionen nur per Tag statt Commit-SHA gepinnt** (Kritiker D-8, `.github/workflows/test.yml`: `actions/checkout@v4`, `actions/setup-go@v5`) — Lieferkette bei öffentlichem Repo. Die übrigen Punkte aus D-8 sind erledigt bzw. eingeplant: CI lief grün (Lauf zu 9330093), das Wurzelmodul bekommt `replace gioui.org` mit dem Programm (#10, ADR-0015). **Prüfbefehl:** `grep -n 'uses:' .github/workflows/test.yml` zeigt 40-stellige SHAs. **Widerlegungsbedingung:** — | backlog (Karte) |
| F-48 | [K] | P3 | Sprint 3 | **Bibliothek: zwei gleichzeitige Schnapp-Prozesse können sich beim Ablegen überschreiben** (Sub-Agent #33, `bibliothek/bibliothek.go` Ablegen): Die Kollisionsprüfung ist Lstat plus Umbenennen, kein exklusives Anlegen; die Sperre (`sync.Mutex`) gilt nur im Prozess. Zwei Aufnahmen in derselben Sekunde aus zwei Prozessen (z. B. zweimal Druck) → die zweite ersetzt die erste (Datenverlust). **Prüfbefehl:** Test mit zwei Bibliothek-Instanzen auf derselben Ablage und gleicher Zeit → zwei Dateien. **Widerlegungsbedingung:** — | backlog (Karte) |
| F-56 | [V] | P3 | Sprint 3 | **Echter D-Bus-Transport ungetestet, memfd-Test prüfte die Größenprüfung nicht, FD-Leck bei Antwortfehler** (Kritiker B, `aufnahme/dbus.go`). **Prüfbefehl:** `go test ./aufnahme -run TestMemfdQuelle`; Test für `capture` mit Attrappe auf FD-Ebene. **Widerlegungsbedingung:** — | teilweise behoben 19.11.2026: FD wird bei Antwortfehler geschlossen, memfd-Test prüft die eigene Meldung; offen: Test für capture (Pipe/memfd) auf FD-Ebene — Karte |
| F-58 | [D] | P3 | Sprint 3 | **`--screen` ohne Wert: Monitorliste oder Monitor unter dem Zeiger?** (Kritiker B): `argumente.go` legt „Liste“ fest, ADR-0017 Nachtrag 1 kündigt an, dass derselbe Aufruf später den Monitor wählt; die Desktop-Aktion „Screen“ zeigt bis dahin nichts Sichtbares. Dazu: die `sed`-Vorlage für `@EXEC@` quotet Pfade mit Leerzeichen nicht. **Prüfbefehl:** Entscheidung in ADR-0017. **Widerlegungsbedingung:** — | offen — Entscheidung Andi (Vorschlag: `--list-screens` für die Liste, `--screen` nimmt den Monitor unter dem Zeiger), Karte |
| F-64 | [V] | P3 | Sprint 3 | **Strengere Leser und Speicherschulden** (Kritiker A): doppelte Blöcke und Daten nach IEND still akzeptiert; Fremd-PNG/LeseBild halten die Datei dreifach; Verlauf unbegrenzt mit Vollkopien je Schwärzung, SchreibeArbeitsdatei leert ihn nicht. **Prüfbefehl:** `go test ./datei -run TestBeschaedigt`. **Widerlegungsbedingung:** — | teilweise behoben 19.11.2026: doppelte Blöcke und Daten nach IEND sind Fehler (Gegenprobe rot); offen: Speicherkopien und Verlaufsgrenze — Karte |
| F-70 | [K] | P3 | Sprint 3 | **Objektfilter im SVG-Export kann sichtbare Teile weglassen** (zweiter Verify-Lauf, `datei/export.go` inAnsicht, `dokument/objekte.go` Grenzen): Die Grenzen sind eine Schätzung, kein Obermaß — die Pfeilspitze ragt bis max(8, 4·Breite) über die Hülle aus A, B, K hinaus, und Text-Grenzen kappen bei `BreiteGrenze` und schätzen 0,6·Grad je Zeichen, während der Export nicht umbricht. Ein Pfeil knapp über der Zuschnittkante oder ein langer Text links davon fällt ganz weg, obwohl ein Stück sichtbar wäre. **Prüfbefehl:** Test mit Pfeilspitze 1,5 px in der Ansicht und Text mit BreiteGrenze, der hineinragt → beide im Export. **Widerlegungsbedingung:** — | backlog (Karte) |
