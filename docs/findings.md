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

| ID | Typ | P | Sprint | Beschreibung | Status |
|---|---|---|---|---|---|
| F-1 | [S] | P1 | — | **Git-Geschichte nicht veröffentlichungsreif** (Karte #8): private Mailadresse als Autor, interne Firmen-, Projekt- und Hausnetz-Angaben in Docs und Commits. Behoben 19.11.2026 (Variante A, Andi): Docs neutralisiert, Internes in `CLAUDE.local.md` (gitignoriert), Geschichte neu begonnen, Autor GitHub-noreply. **Prüfbefehl:** `git log --format=%ae \| sort -u` (erwartet nur die noreply-Adresse) und `git log -p \| grep -ciE "<Liste in CLAUDE.local.md>"` (erwartet 0). | behoben |
