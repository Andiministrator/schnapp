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
| **Prototyp** | Wegwerf-Programm, das entschied, ob Gio trägt (ADR-0003); in Sprint 9 entfernt, Stand in der git-Geschichte (ADR-0027). |
| **Meilenstein** | Erreichter Funktionsstand mit Version (MVP = erste Fassung). |
| **Finding** | Befund in `docs/findings.md` mit Typ, Priorität und Board-Karte. |
| **Schritt** | Eine Karte bzw. ein abgeschlossener Arbeitsgang, danach Schritt-QA. |

## Speichern im Editor

| Begriff | Bedeutung |
|---|---|
| **Abzug** | Unabhängige Kopie eines Dokuments mit gleichem Stand, ohne Rückgängig-Verlauf; wird im Hintergrund gespeichert oder kopiert, während das Bearbeiten ruht (`document`, ADR-0020 Nachtrag 6). |
| **Marke** | Meldung eines erreichten Speicherschritts (Rendern, Kodieren, Schreiben …); zählt als Fortschritt, mit `SCHNAPP_DEBUG` mit Zeitstempel (`internal/saveprogress`, ADR-0020 Nachtrag 9). |
| **Puls** | Fortschrittsmeldung aus einer langen Schleife innerhalb eines Schritts (Rastern, PNG-Kodierung, Lesen, Vorschau), damit ein langer, aber arbeitender Schritt nicht als hängend gilt. |
| **Fortschritt** | Zeitpunkt der letzten Marke oder des letzten Pulses (`saveprogress.Zuletzt`); die Frist für hängendes Speichern zählt ab ihm, frühestens ab Beginn des Speicherns. |
| **Rückfall** | Wird in Schnapp dreierlei genannt: (1) in `library` der Weg über `O_EXCL`, wenn das exklusive Ablegen per Hardlink nicht geht (ADR-0016 Nachtrag 4); (2) im Export das direkte Schreiben eines vorhandenen Ziels, wenn neben ihm keine temporäre Datei entstehen darf, mit Sicherung des alten Inhalts über einen Deskriptor und `ErrZielBeschaedigt`, wenn auch das Wiederherstellen misslingt (ADR-0025 Nachträge 3 und 4); (3) in `ui` die Kopie, die das Fenster selbst hält, weil es keinen Halter gibt — beim Schließen ist das Bild dann aus der Zwischenablage weg, das Fenster warnt einmal (ADR-0024 Nachtrag 6). |
| **Halter** | Prozess ohne Fenster (`wlclip`), der ein kopiertes Bild nach dem Schließen des Editors in der Zwischenablage hält (ADR-0024). |
| **Hängendes Speichern** | Speichern, das seit der Frist (15 s) keinen Fortschritt gemeldet hat. Nur dann gibt eine zweite Schließen-Anfrage oder ein weiteres Signal es auf; beim Zerstören des Fensters gilt zusätzlich höchstens 2 Minuten ab Beginn (ADR-0020 Nachträge 7 und 9). |
