# ADR-0020: Editor — wann er öffnet, wie er gebaut ist, wer die Pixel rendert

## Status
Akzeptiert (Andi, 19.11.2026: „ja zu deinem vorschlag“)

## Kontext
Schnapp nimmt auf und legt in der Bibliothek ab (Sprint 4); der Editor (Karte #35) fehlt. Offen sind drei Dinge:

1. **Wann öffnet der Editor?** Snagit öffnet ihn nach jeder Aufnahme. Schnapp gibt heute den Pfad auf stdout aus und
   beendet sich (für Skripte und Agenten, E-14, wichtig). Eine vorhandene Datei lässt sich noch gar nicht öffnen.
2. **Aufbau:** Wie im Overlay soll die Logik ohne Gio testbar sein (ADR-0003, Karte #34).
3. **Pixel der Datei:** Die Arbeitsdatei trägt ein fertiges Bild (ADR-0008), Exporte PNG/JPG und die Zwischenablage
   ebenso. Dafür müssen die Objekte (Pfeil, Kasten, Text, Nummer) in Pixel gerendert werden. Der eigene Rasterer
   `zeichnen/raster` kann Pfade, Rechtecke, Kreise und Linien, aber weder Text noch `<image>`.

## Entscheidung
1. **Ablauf:**
   - Nach einer interaktiven Aufnahme (`--region`, `--screen`, `--window`) öffnet der Editor im selben Prozess,
     nachdem die Aufnahme schon in der Bibliothek liegt und ihr Pfad auf stdout steht. Abschaltbar mit der
     Einstellung `[capture] open-editor = false` (Voreinstellung `true`) und für einen Aufruf mit `--no-editor`.
     `--repeat` öffnet ihn wie die anderen Modi.
   - `schnapp DATEI.png` öffnet eine vorhandene Datei im Editor (Arbeitsdatei mit Ebenen oder fremdes PNG/JPG).
   - Speichern schreibt eine Bibliotheksaufnahme an ihren Platz zurück (Bearbeitet = wahr). Eine Datei außerhalb
     der Bibliothek wird beim ersten Speichern als neue Aufnahme in die Bibliothek gelegt; das Original bleibt
     unberührt.
2. **Aufbau:** Paket `canvas` ohne Gio (Abbildung, Zoom zum Zeiger, Treffer, Griffe, Ziehen); Änderungen nur über
   Befehle und Züge von `document` (Rückgängig ohne Grenze, F-18: nicht während eines Zugs). `ui` zeichnet Bild und
   Objekte mit Gio-Vektorpfaden und übersetzt Eingaben. Der Editor öffnet erst, wenn alle Overlay-Fenster zu sind
   (F-82).
3. **Rendern:** ein Go-Renderer ohne GPU, der Ansichtsbild und Objekte zu einem `image.NRGBA` zusammensetzt — für
   Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Formen über `golang.org/x/image/vector`, Schrift über
   `golang.org/x/image/font` mit den Go-Schriften (beides steckt über Gio schon im Abhängigkeitsbaum, BSD-Lizenz,
   ADR-0011). *Geändert bei der Umsetzung (#58):* nicht `zeichnen/raster` — das Wurzelmodul müsste dafür das eigene
   Modul `zeichnen` einbinden (require + replace), kurz vor dessen Umbenennung (#57), und dessen Strich-Funktionen
   sind nicht öffentlich. Das
   Bild auf dem Bildschirm zeichnet Gio; Abweichungen in Kantenglättung sind erlaubt, Lage und Größe nicht (Test:
   Treffer und Grenzen gleich).

## Konsequenzen
- Neue Karte „Renderer“ vor dem Speichern mit Objekten; der Editor kann vorher öffnen, zoomen und auswählen.
- `--no-editor` gehört zur Kommandozeile (ADR-0017), `[capture] open-editor` zu den Einstellungen (ADR-0018); beide
  kommen mit dem Editor (#35).
- Für Skripte bleibt der Pfad auf stdout das Erste, was Schnapp ausgibt; ohne Bildschirm (kein Wayland) öffnet kein
  Editor.

## Nachtrag 1 (19.11.2026): Speichern beim Schließen, Ausweichen (Kritik B-8, A-1, A-4)

- Ungespeicherte Änderungen werden beim Schließen ohne Rückfrage gespeichert (Claudes Entscheidung, Andi informiert)
  — auch wenn das Fenster mit Fehler endet. Bei einer Datei außerhalb der Bibliothek entsteht dabei eine neue
  Aufnahme der Bibliothek.
- Fehlt die Bibliotheksdatei beim Speichern, ist sie fremd oder neuer, weicht Speichern auf eine neue Aufnahme aus,
  statt die Bearbeitung zu verlieren.
- Bildschirm und Renderer nutzen dieselben Go-Schriften (keine Systemschriften); Tabulator wird als vier Leerzeichen
  gezeichnet. Schriftgrad und Nummer-Radius sind auf `document.MaxSchrift` = 4096 begrenzt ([S], Speicher).
- Offen (eigene Karten): Speichern bei Signal/Absturz und Konflikt zweier Editoren (F-107, F-110), Angleichen der
  Maße an den SVG-Export (F-108).
