# Glossar

## Produktbegriffe

| Begriff | Bedeutung |
|---|---|
| **Aufnahme** | Ein Screenshot (Bereich, Fenster, Bildschirm). Landet sofort in der Bibliothek. |
| **Aufnahmeschicht** | Austauschbare Schnittstelle, über die Schnapp Aufnahmen holt (KWin, später Portal). ADR-0007. |
| **Overlay** | Vollbildfenster über allen Monitoren zur Bereichsauswahl, mit Lupe, Pixelraster und Maßanzeige. |
| **Bibliothek** | Automatische Sammlung aller Aufnahmen mit Vorschau, Datum, Quellprogramm, Suche, Schlagworten. |
| **Arbeitsdatei** | PNG mit eingebettetem SVG-Block (ADR-0008). Überall ein Bild, in Schnapp voll bearbeitbar. |
| **SVG-Block** | Privater PNG-Chunk mit dem vollständigen SVG: Ebenen, Original, Stauch- und Zuschnittangaben. |
| **Original** | Die unbeschnittene, ungestauchte Aufnahme in der Arbeitsdatei. Wird nur durch Schwärzen verändert. |
| **Pixelebene** | Die Aufnahme als Bild im Ebenenmodell. |
| **Vektorebene** | Eine Beschriftung (Pfeil, Kasten, Text …) als SVG-Element. |
| **Schwärzen** | Bereich deckend überdecken; überschreibt die Pixel auch im Original. Endgültig. |
| **Verpixeln** | Bereich grob aufrastern; überschreibt ebenfalls das Original. |
| **Stauchen** | Aus der Mitte eines Bildes einen Streifen ausblenden (Header und Footer bleiben), nicht zerstörend, mit Zackenkante. Snagit: „Cut Out". |
| **Stauchnaht** | Die Stelle, an der gestaucht wird; Schnapp kann eine vorschlagen (Bildvergleich). |
| **Zackenkante** | Gezackter Rand an einer Stauchnaht oder Bildkante, zeigt „hier fehlt etwas". |
| **Schritt-Nummer** | Nummerierter Kreis, der beim Setzen selbst hochzählt. |
| **Stempel** | Fertiges Symbol aus einer durchsuchbaren Sammlung (Icon-Sets oder eigene). |
| **Schnellstil** | Gespeicherter Satz aus Farbe, Strichstärke, Schrift für ein Werkzeug. |
| **Spotlight** | Alles außer einem Bereich abdunkeln. |
| **Lupe (Beschriftung)** | Vergrößerter Ausschnitt, neben die Stelle gesetzt. |
| **Blatt** | Mehrere Bilder auf einer Fläche (Vorher-nachher, Schritt-für-Schritt). |
| **Zeichenpakete** | Die aus einem Schwesterprojekt übernommenen Go-Pakete `svg`, `raster`, `geo` (ADR-0012). |

## Entwicklungsbegriffe

| Begriff | Bedeutung |
|---|---|
| **Sprint** | Arbeitseinheit auf dem Board (`kind:sprint`), endet mit `/sprint-abschluss`. |
| **Fassung** | Themenbündel aus den Anforderungen (erste, zweite, dritte Fassung); kein Prozessbegriff, wird in Sprints abgearbeitet. |
| **Prototyp** | Wegwerf-Programm, das entscheidet, ob Gio trägt (ADR-0003). |
| **Meilenstein** | Erreichter Funktionsstand mit Version (MVP = erste Fassung). |
| **Finding** | Befund in `docs/findings.md` mit Typ, Priorität und Board-Karte. |
| **Schritt** | Eine Karte bzw. ein abgeschlossener Arbeitsgang, danach Schritt-QA. |
