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
| F-673 | [S] | P1 | Sprint 12 | **`SetzeSchlagworte` vergleicht den Dateistand nicht und kann eine gespeicherte Schwärzung überschreiben** (Sitzungs-QA 24.11. abends, Kritiker A-1, gemessen): liest, kodiert neu, benennt um ohne Standvergleich; speichert ein zweites Fenster dazwischen eine Schwärzung, ist der ungeschwärzte Stand zurück. **Prüfbefehl:** Test mit zweitem Bibliothekswert und Haken bei `bild-kodiert`: `ErrGeaendert`, Pixel bleibt schwarz. **Widerlegungsbedingung:** — | offen |
| F-675 | [S] | P2 | Sprint 12 | **`InPapierkorb` und `SetzeSchlagworte` folgen einem Monatsordner, der als Verweis aus der Ablage führt** (Kritiker A-3, gemessen): nur `gueltig()` und Lstat des letzten Pfadteils, nicht `Relativ(voll) == rel` wie `Speichern`. **Prüfbefehl:** Verweis-Ordner in der Ablage: beide `ErrPfad`. **Widerlegungsbedingung:** — | offen |
| F-678 | [B] | P3 | Sprint 12 | **Ansicht der Bibliothek: vier kleine Zustandsfehler** (Kritiker A-6, gelesen): (1) Schlagwort-Eingabe geht bei laufendem Speichern still verloren, (2) Papierkorb meldet Fehlschlag, wenn nur der Index scheiterte, (3) Strg+C an der Sperre vorbei während eines Speicherns, (4) Ansicht schließt, wenn sie während eines Wechsels geöffnet wird. **Prüfbefehl:** je ein Test nach den Wegen. **Widerlegungsbedingung:** — | offen |
| F-684 | [V] | P3 | Sprint 12 | **Einfarbige Logos: Ring des größeren Auges liegt oben und unten auf dem Strich des „s“** (Kritiker C-6, gerechnet): außen r 16 gegen 14 Innenraum; in „einfarbig weiß“ bleiben vom Ring nur Bögen. **Prüfbefehl:** Andi sieht die beiden einfarbigen Fassungen auf dunklem Grund an. **Widerlegungsbedingung:** Andi findet es in Ordnung. | offen |
| F-685 | [B] | P1 | Sprint 12 | **`Abgleich` leert den ganzen Index, wenn die Ablage fehlt oder leer ist** (Kritiker B-1, gemessen): jedes ENOENT gilt als gelöscht; Ablage nicht eingehängt → alle Einträge und Vorschauen weg, Neuaufbau ruft das Programm nicht. **Prüfbefehl:** Ablage umbenennen, Abgleich, zurück: Einträge bleiben. **Widerlegungsbedingung:** — | offen |
| F-686 | [B] | P2 | Sprint 12 | **Index ohne Sperre über Prozesse: Abgleich/Speichern eines Prozesses verschlucken frische Zeilen eines zweiten** (Kritiker B-2, gemessen: 11 von 60 fehlten): `b.mu` gilt nur im Prozess. **Prüfbefehl:** zwei Bibliothekswerte, Abgleich gegen Ablegen: 0 fehlen. **Widerlegungsbedingung:** — | offen |
| F-689 | [V] | P3 | Sprint 12 | **Teilwort faltet anders als Schlagwort (ToLower statt EqualFold), Leerraum-Teilwort trifft alles** (Kritiker B-5, gemessen). **Prüfbefehl:** „ΟΔΟΣ“/„οδος“, „Meſſe“/„messe“ bei beiden gleich. **Widerlegungsbedingung:** — | offen |
| F-690 | [V] | P3 | Sprint 12 | **`Abgleich` hält `b.mu` über alle Lstat und das Schreiben** (Kritiker B-6, gelesen): auf langsamer Ablage warten Speichern und Vorschau. **Prüfbefehl:** Benchmark `Abgleich` mit 10 000 Einträgen. **Widerlegungsbedingung:** unter 50 ms auf der Zielplatte. | offen |
