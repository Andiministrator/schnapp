# ADR-0037: Schnellstile — gespeicherte Werkzeug-Einstellungen mit einem Klick

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #192 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Gestaltung nach Muster `tmp/sprint12/muster/bilder/192-schnellstile.png`; Andi nimmt sie mit der fertigen Umsetzung
ab (gebündelt am Sprint-Ende). Grundlage: Wunsch Andis vom 22.11.2026 (Ideenliste, Idee 1). Baut auf ADR-0021
Nachträgen 7, 10 und 11 (Strichstärke mit Bildgröße, Schrift, Farbpaar je Werkzeug) auf; ergänzt ADR-0018 (Ort der
Konfiguration) und ADR-0031 (rechte Leiste).

## Kontext
Wer oft dieselben Kombinationen nutzt (schwarze Schrift auf gelbem Schild, dicker roter Rahmen, halbdurchsichtige
gelbe Markierung), stellt heute Farbe, Füllung, Strich und Schrift jedes Mal einzeln ein. Snagit hat dafür
„Quick Styles“.

## Entscheidung

1. **Ein Stil** gehört zu einer Objektart und trägt: Vordergrund, Füllung (bei Kasten, Text und Sprechblase), Strichstärke
   (Pfeil, Kasten, Lupe), beim Text dazu Schrift, Schriftgrad, Fett, Kursiv, Unterstrichen, Durchgestrichen und
   Ausrichtung, bei der Nummer den Radius. Größen sind auf **Faktor 1** gespeichert und wirken mal
   `document.KantenFaktor` des Bilds (wie die Voreinstellung, ADR-0021 Nachträge 7 und 8), damit ein Stil auf einer
   4K-Aufnahme gleich aussieht wie auf Full HD.
2. **Abschnitt „Quick styles“/„Schnellstile“** oben in der rechten Leiste (über den Eigenschaften), sichtbar bei einem
   Werkzeug mit Stilen (Pfeil, Kasten, Text, Sprechblase, Nummer, Lupe) oder einem gewählten Objekt dieser Art; einklappbar
   wie „Farbe“. Kacheln 40 × 40 dp in Reihen zu sechs, auf hellem Schachbrett (Schwarz, Weiß und Durchsichtiges bleiben
   sichtbar), je Kachel eine Mini-Darstellung des Objekts im Stil. Orange umrandet die Kachel, die zum aktuellen Stand
   passt (gleiche Werte), sonst keine.
3. **Anwenden:** Klick (oder Leertaste mit Fokus): ohne gewähltes Objekt wird der Stil zum Stand des Werkzeugs (sein
   Farbpaar und seine Maße, ADR-0021 Nachtrag 11 Punkt 2); mit gewähltem Objekt ändert er das Objekt in **einem**
   Rückgängig-Schritt und, wie eine Farbe am Objekt, auch den Stand des Werkzeugs dieser Art.
4. **Eingebaute Stile** (fest, übersetzt benannt im Tooltip/Statustext): Pfeil und Kasten je Rot, Blau, Grün, Schwarz,
   Rot dick, Weiß auf Schwarz (Kasten: weiße Linie mit schwarzer Füllung), Kasten zusätzlich „Marker“ (gelbe Linie,
   gelbe Füllung 40 %); Text Weiß auf Rot (Voreinstellung), Schwarz auf Gelb, Weiß auf Schwarz, Weiß auf Blau, Rot ohne
   Schild, Schwarz ohne Schild; Nummer Rot, Blau, Grün, Schwarz; Lupe Rot, Schwarz, Weiß. Die Werte stehen in einer
   Tabelle in `internal/canvas/schnellstile.go`.
5. **Eigene Stile:** Knopf „+“ hinter den Kacheln speichert den aktuellen Stand (des Werkzeugs bzw. des gewählten
   Objekts) als eigenen Stil am Ende der Reihe; eigene Kacheln tragen eine graue Ecke. Rechtsklick oder Entf auf einer
   eigenen Kachel mit Fokus entfernt sie (ohne Rückfrage, Statuszeile „Style removed“; kein Rückgängig, weil Stile
   nicht zum Dokument gehören). Höchstens 16 eigene je Objektart; ist die Grenze erreicht, ist „+“ grau.
   Umbenennen und Umordnen gibt es nicht.
6. **Speicherort:** `$XDG_CONFIG_HOME/schnapp/styles.ini` (sonst `~/.config/schnapp/styles.ini`), Ordner 0700, Datei
   0600, atomar geschrieben wie `settings.ini` (ADR-0034 Punkt 5), aber als **Datei des Programms**: Schnapp schreibt sie
   jeweils ganz neu (Kopfkommentar sagt das). Ein Abschnitt je Stil, Name `[<art>]` (`arrow`, `box`, `text`, `callout`,
   `step`, `magnifier`), Reihenfolge der Abschnitte = Reihenfolge der Kacheln; Schlüssel englisch (`color`, `fill`,
   `width`, `font`, `size`, `bold`, `italic`, `underline`, `strike`, `align`, `radius`), Farben `#RRGGBBAA`. Unbekannte
   Schlüssel und ungültige Werte: der Stil fällt weg, mit Warnung auf stderr wie bei `settings.ini`; die Datei ist
   höchstens 64 KiB. Gelesen beim Öffnen des Editors, geschrieben bei jedem „+“ und Entfernen. Eigene Stile gehören
   nicht in die Einstellungstafel (ADR-0034 Punkt 2 betrifft `settings.ini`).
7. **Tastatur:** Tab erreicht die erste Kachel, Pfeiltasten wandern, Leertaste/Enter wendet an, Entf entfernt einen eigenen
   Stil; kein globales Kürzel je Stil (Ziffern bleiben frei).
8. **Nicht in diesem Schritt:** Stile für Schwärzen/Verpixeln (haben keine Farbe), Stile über mehrere Objektarten,
   Austausch von Stil-Dateien, Stile im Dateiformat.
9. **Umsetzung:** Logik ohne Gio in `internal/canvas/schnellstile.go` (Tabelle, Anwenden als ein Schritt, Passt-zu,
   Faktor), Lesen/Schreiben in `internal/settings/stile.go` (Tests mit eigenem `XDG_CONFIG_HOME`), Leiste in
   `internal/ui/schnellstile.go`; Texte über `i18n`; Mutanten-Gruppe `schnellstile`; Sichtprobe per Drehbuch.

## Erwogene Alternativen
- **Eigene Stile in `settings.ini`:** eine Datei weniger, aber eine Liste von Abschnitten passt nicht zu „jeder Schlüssel
  hat eine Tafelzeile“ und vermischt Handbearbeitetes mit Programmdaten.
- **Stile im Zustand (`$XDG_STATE_HOME`):** Stile sind bewusst gewählt und gehören damit nach ADR-0018 zur Konfiguration.

## Konsequenzen
- Neue Datei im Konfigurationsordner (README nennt sie neben `settings.ini`).
- Die Sprechblase (ADR-0021 Nachtrag 13) bekommt eigene Stile; bis sie umgesetzt ist, entfällt die Art `callout`.
