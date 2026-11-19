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
   Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Formen über `zeichnen/raster` (Füllen, Strich), Schrift über
   `golang.org/x/image/font` mit den Go-Schriften (steckt über Gio schon im Abhängigkeitsbaum, ADR-0011). Das
   Bild auf dem Bildschirm zeichnet Gio; Abweichungen in Kantenglättung sind erlaubt, Lage und Größe nicht (Test:
   Treffer und Grenzen gleich).

## Konsequenzen
- Neue Karte „Renderer“ vor dem Speichern mit Objekten; der Editor kann vorher öffnen, zoomen und auswählen.
- `--no-editor` gehört zur Kommandozeile (ADR-0017), `[capture] open-editor` zu den Einstellungen (ADR-0018); beide
  kommen mit dem Editor (#35).
- Für Skripte bleibt der Pfad auf stdout das Erste, was Schnapp ausgibt; ohne Bildschirm (kein Wayland) öffnet kein
  Editor.
