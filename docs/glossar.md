# Glossar

## Produktbegriffe

| Begriff | Bedeutung |
|---|---|
| **Aufnahme** | Ein Screenshot (Bereich, Fenster, Bildschirm). Landet sofort in der Bibliothek. |
| **Aufnahmeschicht** | Austauschbare Schnittstelle, über die Schnapp Aufnahmen holt (KWin, später Portal). ADR-0007. |
| **Overlay** | Vollbildfenster über allen Monitoren zur Bereichsauswahl, mit Lupe, Pixelraster und Maßanzeige. |
| **Bibliothek** | Automatische Sammlung aller Aufnahmen mit Vorschau, Datum, Quellprogramm, Suche, Schlagworten. |
| **Leiste der letzten Aufnahmen** | Englisch „Recent captures“, bei Snagit „Tray“. Leiste unter dem Bild im Editor mit den Vorschaubildern der neuesten Aufnahmen (neueste links, offene grün, ohne Agenten-Aufnahmen); ein Klick öffnet eine im selben Fenster, nachdem die offene gespeichert ist (**Wechsel**). Ansicht → „Letzte Aufnahmen“, Strg+Umschalt+R, `[library] bar`. ADR-0036, `internal/ui/leiste_unten.go`, Logik `internal/libview`. |
| **Volle Ansicht der Bibliothek** | Englisch „Library“. Die ganze Bibliothek im Editorfenster statt Werkzeugleiste, Leinwand und Leisten (Menüzeile und Fußzeile bleiben): Raster nach Monaten, Suche nach Schlagwort, Zeitraum, Schalter für Agenten-Aufnahmen; Enter/Doppelklick öffnet (**Wechsel**), Entf legt in den Papierkorb, F2 bearbeitet Schlagworte. Datei → „Bibliothek“, Strg+Umschalt+L, Knopf in der Leiste. ADR-0036 Punkte 3 und 4, `internal/ui/bibliothek.go`, Logik `internal/libview/raster.go`. |
| **Arbeitsdatei** | PNG mit eingebettetem SVG-Block (ADR-0008). Überall ein Bild, in Schnapp voll bearbeitbar. |
| **SVG-Block** | Privater PNG-Chunk mit dem vollständigen SVG: Ebenen, Original, Stauch- und Zuschnittangaben. |
| **Original** | Die unbeschnittene, ungestauchte Aufnahme in der Arbeitsdatei. Wird nur durch Schwärzen verändert. |
| **Pixelebene** | Die Aufnahme als Bild im Ebenenmodell. |
| **Vektorebene** | Eine Beschriftung (Pfeil, Kasten, Text …) als SVG-Element. |
| **Ebene** | Jede Beschriftung ist eine Ebene über dem Bild, mit Name, Sichtbarkeit, Sperre und Deckkraft; das Bild ist die unterste, Schwärzungen gehören zu ihm. Ausgeblendete Ebenen fehlen in Export und Zwischenablage, bleiben aber in der Arbeitsdatei (kein Schwärzen). Leiste „Ebenen“ und Menü „Ebene“, ADR-0032. |
| **Schwärzen** | Bereich deckend überdecken; überschreibt die Pixel auch im Original. Endgültig. |
| **Verpixeln** | Bereich grob aufrastern; überschreibt ebenfalls das Original. |
| **Stauchen** | Aus der Mitte eines Bildes einen Streifen ausblenden (Header und Footer bleiben), nicht zerstörend, mit Kante an der Naht (Kantenform). Snagit: „Cut Out". |
| **Stauchnaht** | Die Stelle, an der gestaucht wird; Schnapp kann eine vorschlagen (Bildvergleich). |
| **Zackenkante** | Gezackter Rand an einer Stauchnaht oder Bildkante, zeigt „hier fehlt etwas"; heute eine der drei Kantenformen (`zigzag`). |
| **Kantenform** | Form der Kante an Stauch-Nähten, je Dokument: Abriss (`tear`, Voreinstellung), Welle (`wave`), Zacken (`zigzag`); dazwischen ein durchsichtiger Spalt, die Kanten als Papier-Abriss angedeutet. Einstellung `[capture] seam`, Menüeintrag „Kantenform“ (grau ohne Naht; ADR-0023 Nachträge 6 und 7). |
| **Vollansicht** | Das ganze Original mit denselben Stauch-Streifen, ohne Zuschnitt; nur beim Werkzeug Zuschneiden gezeigt. Die Ansicht liegt darin bei `VollVersatz` (ADR-0023 Nachtrag 5). |
| **Schritt-Nummer** | Nummerierter Kreis, der beim Setzen selbst hochzählt. |
| **Stempel** | Fertiges Symbol aus einer durchsuchbaren Sammlung (Icon-Sets oder eigene). |
| **Schnellstil** | Gespeicherter Satz aus Farbe, Strichstärke, Schrift für ein Werkzeug. Eingebaute stehen fest im Programm, eigene (höchstens 16 je Art) in `styles.ini` (ADR-0037). |
| **Sprechblase** | Text mit Spitze (`document.Text.Spitze`), kein eigenes Objekt; Werkzeug K. Ohne Füllung zwei Schenkel statt Fläche (ADR-0021 Nachtrag 13). |
| **Spitze (Sprechblase)** | Punkt im Original, auf den die Sprechblase zeigt; nicht die Pfeilspitze. |
| **Einstellungstafel** | Tafel über dem Editor (Strg+,), die `settings.ini` liest und nur geänderte Schlüssel schreibt; kein eigenes Fenster (ADR-0034). |
| **Text aus dem Bild** | Texterkennung über ein installiertes `tesseract`, sonst über `schnapp-ocr`, auf dem sichtbaren Bild (nie dem Original); Ergebnis in die Zwischenablage (ADR-0035). |
| **schnapp-ocr** | Eigenes Hilfsprogramm (Modul `ocr/`): tesseract als WebAssembly mit eingebautem Deutsch und Englisch, springt ein, wenn kein `tesseract` installiert ist; langsamer, eine Sprache je Lauf (ADR-0035 Nachtrag 1). |
| **Bereichsmodus** | Einmaliger Zustand nach Strg+Umschalt+T: Ziehen wählt den Bereich der Texterkennung, das Werkzeug bleibt. |
| **Spotlight** | Alles außer einem Bereich abdunkeln. |
| **Lupe (Beschriftung)** | Vergrößerter Ausschnitt, neben die Stelle gesetzt. |
| **Blatt** | Mehrere Bilder auf einer Fläche (Vorher-nachher, Schritt-für-Schritt). |
| **Zeichenpakete** | Die aus einem Schwesterprojekt übernommenen Go-Pakete `svg`, `raster`, `geo` (ADR-0012). |
| **Wechsel** | Öffnen einer anderen Aufnahme im selben Editorfenster (Leiste der letzten Aufnahmen oder volle Ansicht der Bibliothek): erst wird die offene gespeichert wie mit Strg+S, dann die neue geladen; solange ruht das Bearbeiten. Kein Wechsel bei offenem Export, laufender Texteingabe, Zug, Arbeit der Ansicht oder laufendem Kopieren. ADR-0036, `internal/ui/wechsel.go`. |
| **Navigator** | Mini-Übersicht unten rechts auf der Leinwand, nur wenn das Bild bei dieser Zoomstufe nicht ins Fenster passt; Klicken und Ziehen verschiebt die Ansicht. Ansicht → „Navigator“. ADR-0031 Nachtrag 9, `internal/canvas/navigator.go`. |
| **Einrasten** | Englisch „snap“. Verschieben, Griffe und neue Pfeile, Kästen und Nummern rasten an Kanten und Mitten anderer sichtbarer Objekte und des Bilds ein (6 Fensterpixel), orange Hilfslinien zeigen das Ziel; Alt oder Ansicht → „An Objekten einrasten“ schaltet es ab. ADR-0021 Nachtrag 12. |
| **Farbpaar** | Die zwei Farben einer Beschriftung: Vordergrund (Strich bzw. Schrift) und Füllung, je Werkzeug gemerkt, „Keine“ möglich, Tauschen mit Strg+Umschalt+X. ADR-0021 Nachtrag 11, `internal/canvas/farbpaar.go`. |
| **Agenten-Aufnahme** | Aufnahme, die ein Agent über den MCP-Server (`schnapp mcp`) gemacht oder beschriftet hat; sie liegt in der Bibliothek unter `agents/`, fehlt in der Leiste der letzten Aufnahmen und erscheint in der vollen Ansicht nur mit dem Schalter. ADR-0030 (Nachträge 2 und 13). |

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
| **Rettungskopie** | Englisch „rescue copy“. Kleine JSON-Datei, die der Editor nach Änderungen im Hintergrund schreibt, damit ungespeicherte Arbeit einen Absturz, SIGKILL oder Stromausfall übersteht: Beschriftungen mit ihren Texten, Zuschnitt, Stauchstreifen und die Schwärzungen seit dem letzten Speichern, keine Pixel. Ort `$XDG_STATE_HOME/schnapp/rescue/` (ohne die Variable `~/.local/state/schnapp/rescue/`), Ordner 0700, Datei 0600, je Ausgangsdatei eine (Name aus dem SHA-256 ihres Pfads); die Kopie trägt den absoluten Pfad der bearbeiteten Datei samt Größe, Änderungszeit und SHA-256. Gelöscht nach gelungenem Speichern; beim Öffnen derselben unveränderten Datei angewendet, sonst als `….alt.json` beiseitegelegt; Beim Start des Editors fällt jede Kopie weg, deren eigene Änderungszeit mehr als 30 Tage zurückliegt, auch eine beiseitegelegte oder nie angewendete. Ein zweites Fenster auf derselben Datei sichert nicht (Sperrdatei `<SHA>.lock` im selben Ordner). `internal/rescue`, ADR-0029. |
| **Halter** | Prozess ohne Fenster (`wlclip`), der ein kopiertes Bild nach dem Schließen des Editors in der Zwischenablage hält (ADR-0024). |
| **Hängendes Speichern** | Speichern, das seit der Frist (15 s) keinen Fortschritt gemeldet hat. Nur dann gibt eine zweite Schließen-Anfrage oder ein weiteres Signal es auf; beim Zerstören des Fensters gilt zusätzlich höchstens 2 Minuten ab Beginn (ADR-0020 Nachträge 7 und 9). |
