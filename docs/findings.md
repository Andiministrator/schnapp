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
| F-943 | [K] | P2 | Sprint 14 | **Schatten bei mehreren merkt auch Werkzeuge unveränderter Objekte** (Kritik #377 K-1): Pfeil mit, Text ohne Schatten gewählt, Richtung gedreht → Text bleibt richtig ohne Schatten, aber der Text-Stil (und `tools.ini`) bekommt den Winkel; ebenso der Schattenzug und „alle ein“. Widerspricht ADR-0039 Nachtrag 4 Punkt 2. **Prüfbefehl:** `TestMehrereSchattenNurGeaenderte` in `internal/canvas/mehrfachstil_test.go` (Kasten-Stil bleibt). **Widerlegungsbedingung:** ein Werkzeug soll Werte auch ohne geändertes Objekt merken. | offen (Karte #389) |
| F-944 | [K] | P2 | Sprint 14 | **Auswahlabfragen kubisch bei vielen gewählten Objekten** (Kritik #377 K-2): `Ebenen()` ruft `IstGewaehlt` je Zeile, die Leiste `Gewaehlt()` Dutzende Male je Bild, jedes mit linearer Suche und Sortieren; Strg+A bei 2000 Objekten macht den Editor zäh. **Prüfbefehl:** `TestMehrereVieleObjekteSchnell` (2000 Kästen, 20 Bilder unter der Grenze). **Widerlegungsbedingung:** Merker liefert nach Änderungen eine veraltete Auswahl. | offen (Karte #389) |
| F-945 | [V] | P3 | Sprint 14 | **Ebenenliste: Hauptauswahl unter mehreren nicht erkennbar** (Kritik #377 K-3); `zeigeGewaehlte` blättert zur untersten gewählten statt zur Hauptauswahl. **Prüfbefehl:** UI-Test Zeilenfarbe Haupt ≠ weitere. | offen (Karte #389) |
| F-946 | [V] | P3 | Sprint 14 | **Umschalt+Klick in der Ebenenliste kann den Bereich nur vergrößern** (Kritik #377 K-4): die geklickte wird Anker, die bisherige Auswahl bleibt. **Prüfbefehl:** Test Klick 1, Umschalt 3, Umschalt 2 → {1,2}. | offen (Karte #389) |
| F-947 | [K] | P3 | Sprint 14 | **Schattenregler bei gemischten Werten setzt den Ausgangswert nicht auf alle** (Kritik #377 K-5): Länge 8/3, ziehen 8→9→8 → B wieder 3. **Prüfbefehl:** Test Zug 9 dann 8, Soll beide 8. | offen (Karte #389) |
| F-948 | [V] | P3 | Sprint 14 | **„Keine“ Füllung bei mehreren überschreibt den Farbton aller Füllungen** (Kritik #377 K-6). **Prüfbefehl:** Test RGB je Objekt nach `KeineFuellung` unverändert. | offen (Karte #389) |
| F-949 | [V] | P3 | Sprint 14 | **Schnellstil trifft Text und Sprechblase nicht gemeinsam** (Kritik #377 K-7), obwohl sie sich einen Werkzeug-Stil teilen. **Prüfbefehl:** Test Text + Sprechblase, beide in Stilfarbe. | offen (Karte #389) |
| F-950 | [K] | P3 | Sprint 14 | **Feld „Schatten“ fällt nach „alle ein“ still auf den Vordergrund zurück** (Kritik #377 K-8): `feldSchattenID` zeigt auf den alten Träger; der Farbwähler färbt dann Strich/Schrift aller gewählten. **Prüfbefehl:** Test Ablauf, Soll `AktivesFeld() == FeldSchatten`. | offen (Karte #389) |
| F-951 | [V] | P3 | Sprint 14 | **Zug über mehrere ohne Träger fällt in den Einzelweg; Testlücken** (Kritik #377 K-9): `beginneMehrZug` ohne Träger lässt den Zug laufen und ändert den Stil der Hauptauswahl ohne Merker; ohne Test: Umschalter/Ausrichtung bei mehreren, `KeineFuellung`, Abbruch des Schattenzugs über mehrere. **Prüfbefehl:** Tests in `mehrfachstil_test.go`. | offen (Karte #389) |
| F-899 | [V] | P3 | Sprint 14 | **Papierkorb mehrerer: Zweig `ErrIndex` ohne Test** (Nacharbeit #361, Kritik-Mutant F): Indexfehler nach gelungenem Verschieben zählt als Erfolg mit Abgleich; kein Test fährt ihn. **Prüfbefehl:** Mutant „ErrIndex-Zweig entfernt“ in `markierung.json` → Soll rot; braucht eine Naht für `InPapierkorb`. | offen (Karte #373) |
| F-889 | [K] | P3 | Sprint 14 | **Papierkorb vieler: Index je Aufnahme ganz gelesen und geschrieben, ohne Fortschritt, ohne Rückfrage; ShowItems ohne Obergrenze** (K-3). Strg+A und Entf in der ungefilterten Ansicht legen die ganze Bibliothek ohne Rückfrage weg (O(n·m)); Schließen wartet 5 s. **Prüfbefehl:** Messung `papierkorbArbeit` über 2000 Einträge in `t.TempDir()`. Entscheidung offen: Rückfrage ab welcher Zahl. | offen (Karte #373) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
