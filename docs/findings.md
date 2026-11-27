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
| F-934 | [K] | P2 | Sprint 14 | **Nach Zuschnitt/Stauchen landen Schnellstil und Regler mit falschem Faktor in tools.ini** (Verify #383, Fable, N-1; Folge von F-927): `WendeSchnellstilAn`, `passtGenau`, `SchnellstilAusStand`, Regler und − / + rechnen mit `s.faktor()`, `WerkzeugStile` teilt durch `stilFaktor`. 4K öffnen, zuschneiden, Pfeil-Schnellstil 6 px → `width = 3`. **Entscheidung (Claude): Variante A** — ändert sich der Faktor des Dokuments, werden alle Stile der Sitzung und `stilFaktor` gemeinsam umgerechnet (ein Faktor je Sitzung; die Strichstärke des Werkzeugs folgt dem Zuschnitt wie die Voreinstellung). **Prüfbefehl:** `TestVerifySchnellstilNachZuschnitt`, `TestVerifyBreiteNachZuschnitt`, `TestVerifySchnellstilAusStandNachZuschnitt` aus `tmp/s14-383/verify/bericht.md` übernehmen → heute rot. **Widerlegungsbedingung:** Die drei Tests bleiben grün, obwohl `SetzeZuschnitt` den Faktor von 2 auf 1 ändert. | offen (Karte „Sitzungs-QA 27.11. (17)“) |
| F-935 | [K] | P3 | Sprint 14 | **Rundung auf Tausendstel verfälscht ganze Pixelwerte bei Faktor 3, 5, 7** (Verify #383 N-2): 7 px → `width = 2.333` → nächste Sitzung 6,999 px. **Prüfbefehl:** `TestVerifyRundungFaktor3` aus dem Verify-Bericht. **Widerlegungsbedingung:** grün ohne Änderung an `SetzeWerkzeugStile`/`gerundet` (Vorschlag: nach dem Malnehmen auf Hundertstel runden). | offen (Karte „Sitzungs-QA 27.11. (17)“) |
| F-936 | [V] | P3 | Sprint 14 | **Testlücken Stil je Werkzeug** (Verify #383 N-3): `standardStilFuer` mit Dokumentfaktor und `merkeZugStil` mit aktivem statt Ziel-Werkzeug überleben; F-932 (Probe-Zweig) ohne Test, Sichtprobe zweimal fahren. **Prüfbefehl:** `TestVerifyVoreinstellungNachZuschnitt`, `TestVerifyAbbruchFremdesWerkzeug` übernehmen, Mutanten WS28/WS29 rot; `stil-je-werkzeug-383.txt` zweimal, Foto 01 gleich. | offen (Karte „Sitzungs-QA 27.11. (17)“) |
| F-937 | [D] | P3 | Sprint 14 | **ADR-0041: Esc in der Texteingabe und veraltete Kommentare** (Verify #383 N-4): ein mit Esc verworfener Text lässt die geänderte Schrift im Stil des Werkzeugs (nicht festgelegt); Kommentar `werkzeugstil.go` zu Grenzen veraltet, `schnellstile.go` „mal Faktor des Bilds“. **Prüfbefehl:** `grep -n 'Voreinstellung' internal/canvas/werkzeugstil.go`, ADR-0041 Nachtrag nennt die Texteingabe. | offen (Karte „Sitzungs-QA 27.11. (17)“) |
| F-899 | [V] | P3 | Sprint 14 | **Papierkorb mehrerer: Zweig `ErrIndex` ohne Test** (Nacharbeit #361, Kritik-Mutant F): Indexfehler nach gelungenem Verschieben zählt als Erfolg mit Abgleich; kein Test fährt ihn. **Prüfbefehl:** Mutant „ErrIndex-Zweig entfernt“ in `markierung.json` → Soll rot; braucht eine Naht für `InPapierkorb`. | offen (Karte #373) |
| F-889 | [K] | P3 | Sprint 14 | **Papierkorb vieler: Index je Aufnahme ganz gelesen und geschrieben, ohne Fortschritt, ohne Rückfrage; ShowItems ohne Obergrenze** (K-3). Strg+A und Entf in der ungefilterten Ansicht legen die ganze Bibliothek ohne Rückfrage weg (O(n·m)); Schließen wartet 5 s. **Prüfbefehl:** Messung `papierkorbArbeit` über 2000 Einträge in `t.TempDir()`. Entscheidung offen: Rückfrage ab welcher Zahl. | offen (Karte #373) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
