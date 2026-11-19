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
| F-17 | [K] | P2 | — | **`prototyp/format.go`: Überlauf bei `breite*4`** (Kritiker K-1): `rohZuRGBA` mit unplausibler Breite (z. B. 1<<62) besteht alle Prüfungen und stürzt in `image.NewRGBA` ab; keine Obergrenze für Breite/Höhe. Dazu in `prototyp/aufnahme.go`: D-Bus-Verbindung je Aufruf nie geschlossen, `Call` ohne Zeitgrenze, `io.ReadAll` ohne Grenze, `int32`→`uint64` unsauber. Quelle ist KWin (vertraut) — vor einer Übernahme in Produktcode beheben (ADR-0003 Nachtrag 2 nennt `format.go` übernehmbar). **Prüfbefehl:** Test mit Breite 1<<62 in `format_test.go` liefert Fehler statt Panic. **Widerlegungsbedingung:** Produktcode übernimmt `format.go` nicht. | backlog (Karte: Prototyp-Code vor Übernahme härten) |
| F-18 | [K] | P3 | — | **Prototyp: Strg+Z während des Ziehens → Index-Panic** (K-2, `prototyp/oberflaeche.go` `rueckgaengig`/`zieh`/`loslassen`): Rückgängig verkürzt `a.pfeile`, während `a.aktiv` auf den neuen Pfeil zeigt. Wegwerf-Code; die Zustandsmaschine des Editors im Produktcode von vornherein so bauen, dass Rückgängig während eines Zugs gesperrt ist. **Prüfbefehl:** Test der Editor-Zustandsmaschine (Produktcode): Rückgängig während Zug ändert nichts. **Widerlegungsbedingung:** — | backlog (Karte: Prototyp-Code vor Übernahme härten) |
| F-29 | [V] | P3 | — | **`marke/werkzeug/cd.py` `pdf()` kann ein veraltetes Roh-PDF übernehmen** (zweiter Verify-Lauf): `tmp/marke/corporate-design.roh.pdf` wird vor dem Chromium-Lauf nicht geprüft; endet Chromium mit Exit 0 ohne Datei, wird das PDF des Vorlaufs bereinigt und übernommen (sauber, aber inhaltlich alt). **Prüfbefehl:** nach dem Chromium-Aufruf Existenz und mtime ≥ Laufbeginn prüfen; Test mit gestubbtem `chromium`, der nichts schreibt → Fehler erwartet. **Widerlegungsbedingung:** — | backlog (Karte) |
