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
