# ADR-0045: Objekte kopieren und einfügen, neue Aufnahme aus der Zwischenablage

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #335 in Sprint 14; Vorschlag Claude, von Andi am 25.11.2026 übernommen;
Andi 28.11.: „arbeite selbstständig“). Ergänzt ADR-0024 (Halter der Zwischenablage), ADR-0026 (Menü und Kürzel),
ADR-0036 (Bibliothek), ADR-0039 (Mehrfachauswahl) und ADR-0044 (Gruppen).

## Kontext
Strg+C legt heute immer das ganze Bild in die Zwischenablage. Wer einen fertig eingestellten Pfeil, eine Sprechblase
oder eine Gruppe aus Nummer und Kasten in einer anderen Aufnahme wiederverwenden will, muss sie neu zeichnen. Ein Bild,
das ein anderes Programm kopiert hat (Browser, Bildbetrachter, Spectacle), lässt sich nicht als Aufnahme öffnen, ohne
es erst als Datei zu speichern.

Unter Wayland liest ein Fenster die Zwischenablage nur mit Tastaturfokus (`wl_data_device`); Gio liest dabei nur Text.
Den Halter (ADR-0024) gibt es schon über data-control (`ext_data_control_v1`, sonst `zwlr_data_control_v1`), das auch
lesen kann, ohne Fokus und ohne Gio.

## Entscheidung
1. **Strg+C kopiert die Auswahl, sonst das Bild.** Ist mindestens ein Objekt gewählt (eins, mehrere, eine Gruppe),
   kopiert Strg+C die gewählten Objekte. Ohne Auswahl wie bisher das gerenderte Bild. **Strg+Umschalt+C** („Copy
   image“, Menü Datei) kopiert immer das Bild. **Strg+X** („Cut“) kopiert die Auswahl und löscht sie in einem
   Rückgängig-Schritt; ohne Auswahl tut es nichts. In der Texteingabe gilt keines davon (wie bisher).
2. **Inhalt einer Objekt-Kopie:** zwei Typen.
   - `application/x-schnapp-objects`: SVG wie der Block snAP (`fileformat.AngabenSVG`, docs/format.md) eines Dokuments
     **ohne Pixel** mit Größe, Zuschnitt, Streifen und Kantenform der Quelle und genau den kopierten Objekten in
     Stapelreihenfolge. Es trägt keine Metadaten und keine Pixel; Schwärzungen sind Pixel und kommen nie mit.
   - `image/png`: die kopierten Objekte allein auf durchsichtigem Grund (`render.Objektebene` eines Abzugs nur mit
     ihnen), auf ihre sichtbaren Pixel zugeschnitten — für fremde Programme. Eine Lupe zeigt darin ihren Inhalt.
   - Dazu wie bei Bildern die leere Klipper-Marke (ADR-0024 Nachtrag 2).
   Der Halter bekommt dafür die Art `ArtObjekte` (Argument `--internal-keep-clipboard-objects`); beide Teile gehen
   in einem Rahmen (8 Byte Länge des SVG, SVG, PNG) über die Standardeingabe. Geht kein Halter, bietet das Fenster
   selbst nur das PNG an (Gio kennt einen Typ je Schreiben) und merkt die Objekte im Prozess; Strg+V im selben
   Fenster nimmt sie von dort, solange das Fenster die Zwischenablage hält.
3. **Strg+V fügt Objekte ein** (Menü Bearbeiten → „Paste“, Kontextmenü des Bilds). Gelesen wird über data-control in
   einer eigenen kurzen Verbindung (`wlclip.Lies`), in einer Goroutine, nur auf ausdrückliche Handlung, nie im
   Hintergrund. Liegt `application/x-schnapp-objects` vor:
   - Es wird mit denselben Grenzen gelesen wie eine Arbeitsdatei (`fileformat.LeseAngaben`, höchstens `maxSVG`), und
     das Dokument mit den eingefügten Objekten muss das Budget des Öffnens bestehen (`render.Pruefe`); sonst eine
     Meldung, nichts eingefügt.
   - **Lage:** Die Objekte behalten ihre Lage im sichtbaren Bild: Die linke obere Ecke ihrer gemeinsamen Grenzen wird
     in der Quelle in die Ansicht abgebildet und im Ziel aus der Ansicht ins Original zurück; alle Objekte werden um
     denselben Betrag verschoben (Formen bleiben gleich). Läge das Ganze dann nicht vollständig im sichtbaren Bild,
     kommt es in die Mitte des sichtbaren Fensterausschnitts (ins Bild begrenzt). Stehen an der Stelle schon gleiche
     Objekte (erneutes Einfügen in dieselbe Aufnahme), rückt es um den Versatz des Doppelns (10 Fensterpixel) weiter,
     bis die Stelle frei ist.
   - **Größen** bleiben in Pixeln, wie sie sind (kein Umrechnen über den Kantenfaktor): Kopieren zwischen Aufnahmen
     desselben Bildschirms ist der Regelfall.
   - Eingefügt wird zuoberst, alle in **einem** Rückgängig-Schritt, und die eingefügten sind danach gewählt. Gruppen
     bekommen neue Kennungen (`neueGruppenFuer`, ADR-0044 Punkt 4), Ausgeblendet und Gesperrt fallen weg (sonst wäre
     das Eingefügte unsichtbar oder nicht wählbar); Name, Deckkraft und Schatten bleiben.
   - Liegt nur ein Bild vor, sagt die Fußzeile, dass Datei → „New from clipboard“ (Strg+Umschalt+V) daraus eine
     Aufnahme macht. Liegt nur Text vor oder nichts, sagt sie, dass es nichts einzufügen gibt. Text als Textobjekt
     einzufügen ist eine spätere Karte.
4. **Datei → „New from clipboard“, Strg+Umschalt+V** (wie „Als neues Bild einfügen“ in GIMP): liest `image/png`, sonst
   `image/jpeg`, mit denselben Grenzen wie das Öffnen einer Datei (`fileformat.OeffneBild`: Größe, JPEG-Bedarf,
   EXIF-Ausrichtung), legt das Bild als neue Aufnahme in der Bibliothek ab (Kantenform aus `[capture] seam`, wie eine
   Aufnahme) und wechselt dorthin wie ein Klick in der Leiste (die offene wird vorher gespeichert). Höchstens
   `fileformat.MaxDatei` Byte, Frist 10 s ohne Fortschritt.
5. **Ohne data-control** (oder mit `WAYLAND_SOCKET`) gibt es kein Lesen: Strg+V nimmt nur die im Prozess gemerkten
   Objekte (Punkt 2), „New from clipboard“ meldet den Grund in der Fußzeile.
6. **Menü:** Datei: Open…, New from clipboard │ Save │ Export…, Copy image │ Settings… │ Library │ Close.
   Bearbeiten: Undo, Redo │ Cut, Copy, Paste │ Edit text, Delete, Select all │ Swap colors │ Copy text from image.
   Kontextmenü eines Objekts: Cut und Copy vor Doppeln; Kontextmenü des Bilds: Paste nach Copy.

## Erwogene Alternativen
- **Gio-Patch für weitere Lesetypen:** liest nur mit Fokus, braucht einen größeren Patch (ADR-0015) und hilft dem
  „New from clipboard“ aus der vollen Bibliothek nicht. data-control gibt es schon, im eigenen Halter erprobt.
- **Eigenes Binärformat statt snAP-SVG:** doppelte Pflege; das snAP-SVG ist spezifiziert, geprüft und hat Grenzen.
- **PNG mit dem Bild darunter:** wirkt in fremden Programmen wie ein Ausschnitt der Aufnahme, nicht wie die Objekte;
  wer den Ausschnitt will, schneidet zu und kopiert das Bild.
- **Einfügen an der Mausposition:** in Inkscape üblich, in Snagit nicht; über das Tastenkürzel ist die Lage des
  Zeigers oft zufällig. Spätere Karte, falls gewünscht (Kontextmenü am Klickpunkt).
- **Größen über den Kantenfaktor umrechnen:** änderte Lage und Form zugleich und überrascht beim Regelfall.

## Konsequenzen
`internal/wlclip` (`ArtObjekte`, Rahmen, `Lies`), `internal/document` (`OhnePixel`, `NurObjekte`),
`internal/fileformat` (`OeffneBild`), `internal/canvas` (`zwischenablage.go`: Auswahl kopieren, Lage beim Einfügen),
`internal/ui` (`einfuegen.go`, Menü, Kürzel, Kontextmenü, Übersicht F1), `cmd/schnapp` (Ablegen eines Bilds aus der
Zwischenablage über `ui.Anbindung.Ablegen`, Halter-Argument). Format: docs/format.md bekommt einen informativen
Abschnitt zum Typ der Zwischenablage; FormatVersion unverändert. Sichtprobe im virtuellen KWin (Kopieren und Einfügen
zwischen zwei Aufnahmen über die Leiste); Klipper und fremde Programme prüft Andi.

## Nachtrag 1 (28.11.2026, Kritik Fable zu 6dbfaea, F-984 bis F-993, Karte #393)
1. **Gemerkte Objekte zuerst (F-984):** Hält das Fenster selbst eine Objekt-Kopie (kein Halter, aus welchem Grund
   auch immer), nimmt Strg+V die gemerkten Objekte, ohne zu lesen — gelesen würde sonst nur ihr Bild.
2. **Streifen im Ziel (F-985):** Die alte Lage gilt nur, wenn das Eingefügte dort keinen Stauch-Streifen des Ziels
   überdeckt (die Ecken liegen in der Ansicht so weit auseinander wie im Original); sonst die Mitte, und überdeckt
   auch die einen Streifen, die nächste Lage direkt vor oder hinter einer Naht.
3. **Eingabefelder (F-986):** Mit Fokus in einem Eingabefeld der Leiste wirken Strg+C, Strg+X und Strg+V nicht auf die
   Leinwand.
4. **Gesamtfrist und Abbruch (F-987):** `wlclip.Lies` liest höchstens `LeseFrist` = 60 s und nimmt einen Kontext; Esc
   bricht ein laufendes Lesen ab, das Schließen ebenso.
5. **Schließen (F-988):** Ein laufendes „New from clipboard“ wird beim Schließen abgebrochen; ein schon begonnenes
   Ablegen wird höchstens `wechselFrist` abgewartet, sein Ort steht auf der Fehlerausgabe.
6. **Ausschneiden erst nach Erfolg (F-989):** Strg+X löscht, wenn die Antwort des Kopierens ohne Fehler kommt und das
   Dokument seitdem unverändert ist; sonst bleiben die Objekte, und die Fußzeile sagt es.
7. **Budget (F-990):** Die Prüfung läuft in der Lese-Goroutine gegen die Objekte beim Tastendruck; hat sich das
   Dokument bis zur Antwort geändert, prüft das Einfügen neu. Mit `--ignore-limit` geöffnet gilt keine Grenze (wie beim
   Export, ADR-0033 Nachtrag 4).
8. **Wechsel (F-991):** Solange gelesen wird, nimmt die Leiste keinen Wechsel an; bei offener Bibliothek fügt eine
   Antwort nicht ins verdeckte Bild ein.
