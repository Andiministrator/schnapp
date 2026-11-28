# Anforderungen

Stand 18.11.2026 (Andi), übernommen am 19.11.2026.
Entscheidungen dazu: ADR-0003 und ADR-0007 bis ADR-0013. Begriffe: [`glossar.md`](glossar.md).

## Ziel

Ein Screenshot- und Beschriftungswerkzeug für Linux/Wayland, Vorbild Snagit (Bedienung und Teile der Funktionen),
das auch als einfaches Bildbearbeitungsprogramm taugt. ksnip und Spectacle sind zu umständlich und können zu wenig.

## Wichtig, leicht übersehen

- **A-01** Schwärzen und Verpixeln überschreiben die Pixel im mitgespeicherten Original; Weitergeben wahlweise ohne
  Bearbeitungsdaten. Sonst steht das Geschwärzte im eingebetteten Original im Klartext. (ADR-0008)
- **A-02** Jede Aufnahme sofort automatisch in einer Bibliothek (Vorschau, Datum, Quellprogramm, Suche, Schlagworte).
- **A-03** Keine Metadaten mit lokalen Pfaden, Programm- oder Rechnernamen; Autor einstellbar.

## Aufnahme

- **A-10** Bereich (Voreinstellung; Vollbild-Overlay mit Lupe, Pixelraster, Maßanzeige), ganzer Bildschirm mit Wahl
  des Monitors, Fenster per Klick.
- **A-11** Verzögerung, Mauszeiger an/aus, letzten Bereich wiederholen, globales Tastenkürzel.
- **A-12** Lange Webseiten: zuerst über den Browser (Chrome-Fernsteuerung bzw. kleine Browser-Erweiterung, Firefox
  kann ganze Seiten), später Scrollen und Zusammensetzen für beliebige Programme.

## Beschriften

- **A-20** Pfeile gerade und **gebogen** (um etwas herum), Kästen, Ellipsen, Linien, Textmarker, Freihand. (Linien: umgesetzt als Pfeil ohne Spitze, ADR-0021 Nachträge 19 und 20. Ellipse: umgesetzt, ADR-0047. Textmarker gerade/gebogen: umgesetzt, ADR-0048; Freihand und freier Textmarker: umgesetzt, ADR-0049.)
- **A-21** Schritt-Nummern, die selbst hochzählen; Sprechblasen; Text mit Hintergrund.
- **A-22** Lupe (Ausschnitt vergrößert daneben), Spotlight (Rest abdunkeln), Schnellstile (gespeicherte Farb- und
  Strichsätze).
- **A-23** **Stauchen:** aus der Mitte einen Teil herausschneiden (lange Website: Header und Footer zeigen), nicht
  zerstörend (Original bleibt, Schnitt ist eine Angabe), mit Zackenkante.

## Stempel

- **A-30** Sammlung mit Suche nach Themen und Stichworten, wie bei Snagit. Quellen: offene Icon-Sets mit Stichwörtern
  (Tabler, Lucide; OpenMoji mit Namensnennung und CC BY-SA), Lizenzen vor Übernahme prüfen.
- **A-31** Eigene Stempel per Drag & Drop mit eigenen Stichwörtern.

## Einfache Bildbearbeitung

- **A-40** Zuschneiden, drehen, spiegeln, Größe, Leinwand erweitern; Rahmen, Schatten, Zackenkante, runde Ecken.
- **A-41** Farbpipette, Helligkeit/Kontrast, Hintergrund transparent.
- **A-42** Mehrere Bilder zu einem Blatt (Vorher-nachher, Schritt-für-Schritt).
- **A-43** Unbegrenztes Rückgängig, Ebenen ausrichten und anordnen.

## Teilen

- **A-50** Zwischenablage (Wayland), Drag & Drop in andere Programme, Exporte (SVG, PDF, PNG/JPG), Dateinamen-Schema.

## Später

- **A-60** Texterkennung (Tesseract): Text kopieren, Mailadressen automatisch finden und schwärzen.
- **A-61** Bildschirmvideo und GIF; inhaltsbasiertes Verschieben (Snagit „Smart Move").

## Reihenfolge (Fassungen)

1. **Prototyp Gio** (ADR-0003).
2. **Erste Fassung:** Aufnahme (KDE), Bibliothek mit Autospeichern, Pfeile, Kästen, Text, Schritt-Nummern, Schwärzen,
   Zuschneiden, Stauchen, Dateiformat, Zwischenablage. (A-01–A-03, A-10, A-11 teilweise, A-20 teilweise, A-21
   teilweise, A-23, A-40 Zuschneiden, A-50 Zwischenablage)
3. **Zweite Fassung:** Stempel mit Suche, Schnellstile, Lupe, Spotlight, Blätter, einfache Bildbearbeitung.
4. **Dritte Fassung:** Texterkennung, lange Webseiten, Portal-Weg für andere Desktops, eventuell Video.

Grobe Schätzung erste brauchbare Fassung: 60 bis 100 Stunden, große Unsicherheit (vor allem Oberfläche).
