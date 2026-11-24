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
| F-600 | [K] | P3 | Sprint 12 | **Alt+Ziehen in einer echten Plasma-6-Sitzung ungeprüft** (Kritik #196): ADR-0021 Nachtrag 12 nimmt an, dass KWin unter Plasma 6 Meta zum Fensterziehen nutzt und Alt bei Schnapp ankommt. **Prüfbefehl:** Echtlauf (Andi, gebündelt): Kasten an einen anderen heranziehen, dabei Alt halten → kein Einrasten, Fenster bleibt stehen. **Widerlegungsbedingung:** Fängt KWin Alt ab, zweite Taste wählen (ADR-0026). | offen |
| F-622 | [K] | P3 | Sprint 12 | **sRGB-Mischung des Editors auf echter Grafik ungeprüft** (Karte #215, ADR-0020 Nachtrag 14): geprüft nur mit Mesa-Software (headless, virtueller KWin); die EGL-Fläche ohne sRGB-Farbraum und `FRAMEBUFFER_SRGB` aus könnten auf echter Hardware anders wirken. **Prüfbefehl:** Echtlauf (Andi, gebündelt am Sprint-Ende): Editor starten, Oberfläche normal hell? Kasten mit Füllung 50 % und Ebene 50 % gegen den PNG-Export desselben Bilds — gleich hell; dabei auch eine Ebene mit Deckkraft 50 % über zwei sich überlappenden halbdurchsichtigen Kästen (Zwischenebene, F-621). **Widerlegungsbedingung:** Ist das Fenster zu hell oder zu dunkel, den EGL-Teil des Patches zurücknehmen und halbdurchsichtige Beschriftungen vorrastern (ADR-0020 Nachtrag 14). | offen |
| F-623 | [V] | P3 | Sprint 12 | **Fehler des Einstellungs-Schreibers und des Betriebssystems erscheinen englisch in der deutschen Oberfläche** (Bericht des Agenten zu Karte #175): Die Tafel zeigt „Einstellungen nicht gespeichert: the settings file is not a regular file“; beim Export dasselbe Muster (deutscher Satzanfang, englischer Grund). **Prüfbefehl:** Sichtprobe `scripts/drehbuecher/einstellungen-175-fehler.txt` mit deutscher Oberfläche — der Grund steht deutsch. **Widerlegungsbedingung:** Lassen sich Fehler des Betriebssystems nicht sinnvoll übersetzen, nur die eigenen Fehlerwerte (`settings.Err…`, `export.Err…`) über `errors.Is` auf übersetzte Texte abbilden und den Rest englisch lassen. | offen |
| F-624 | [V] | P3 | Sprint 12 | **Ebenenname einer Sprechblase lautet wie beim Text „Text „…““** (Bericht des Agenten zu Karte #194 Teil 2): Leiste, Schnellstile und Kopf der rechten Leiste kennen die Art Sprechblase schon (`objektWerkzeug`), nur `ebenenName` nicht. **Prüfbefehl:** Sprechblase anlegen, Ebenenliste zeigt „Sprechblase „…““ bzw. englisch „Speech bubble „…““; Test in `internal/canvas` oder `internal/ui`. | offen |