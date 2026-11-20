# ADR-0021: Zeichenwerkzeuge im Editor — Pfeil, Kasten, Text, Schritt-Nummer

## Status
Akzeptiert (Claudes Entscheidung bei der Umsetzung von Karte #36, 20.11.2026; Andi wird informiert)

## Kontext
Der Editor (ADR-0020, Karte #35) kann öffnen, zoomen, auswählen, verschieben, Griffe ziehen und speichern, hat aber
nur das Werkzeug „Auswählen“. Das Dokumentmodell kennt die vier Beschriftungen schon (`document/objekte.go`: Pfeil als
quadratische Bézier-Kurve mit Griff, Kasten, Text mit Hintergrund, Nummer), Renderer und SVG-Export zeichnen sie. Karte
#36 verlangt die erste Fassung von A-20/A-21: Pfeil (gerade, über einen Griff gebogen), Kasten, Text mit Hintergrund,
Schritt-Nummern, die selbst hochzählen — im Editor und in allen Exporten gleich. Vorbild ist Snagit (Kernprinzip 4:
Bedienung vor Funktionsfülle).

## Entscheidung

1. **Werkzeuge und Bedienung** (Logik in `canvas.Sitzung`, ohne Gio; `ui` übersetzt nur Ereignisse):
   - **Auswählen** (`V`): wie bisher (ADR-0020).
   - **Pfeil** (`A`): Ziehen vom Anfang zur Spitze. Mit **Umschalt** rastet die Richtung in 45°-Schritten (Länge
     bleibt). Nach dem Loslassen ist der Pfeil ausgewählt; der mittlere Griff **biegt** ihn (`Pfeil.MitGriff`), die
     Griffe an den Enden verschieben Anfang und Spitze und behalten die Biegung.
   - **Kasten** (`R`, wie „Rechteck“): Ziehen von Ecke zu Ecke; mit **Umschalt** ein Quadrat.
   - **Text** (`T`): Ein Klick setzt ein Textfeld (linke obere Ecke am Klickpunkt), Tippen schreibt, **Umschalt+Enter**
     beginnt eine neue Zeile, die **Rücktaste** löscht das letzte Zeichen. **Enter** oder ein Klick daneben bestätigt,
     **Esc** verwirft die Eingabe. Ein leerer Text (auch nur Leerraum) wird nicht angelegt; wird ein vorhandener Text
     leer bestätigt, wird er entfernt. Vorhandenen Text bearbeitet ein **Doppelklick** mit dem Werkzeug Auswählen
     (mit einem Zeichenwerkzeug hätte der erste Klick schon etwa eine Nummer gesetzt), **Enter** bei ausgewähltem
     Text oder ein Klick mit dem Text-Werkzeug auf den Text. Die Einfügemarke steht immer am Ende
     (kein Cursor in der Zeile in der ersten Fassung).
   - **Schritt-Nummer** (`N`): Ein Klick setzt eine Nummer; Ziehen vor dem Loslassen verschiebt sie noch. Der Wert ist
     **größter vorhandener Wert + 1** (`Dokument.NaechsteNummer`), sonst 1. Nach Rückgängig oder Löschen der letzten
     Nummer kommt deren Wert wieder; eine Lücke in der Mitte bleibt (kein Umnummerieren).
   - **Mindestgröße:** Pfeil und Kasten entstehen erst ab der Ziehschwelle (3 Fensterpixel) und bleiben nur, wenn sie
     beim Loslassen mindestens 6 Fensterpixel lang bzw. je Seite groß sind — ein versehentlicher Klick legt nichts an.
   - **Griff vor Neuanlage:** Liegt der Druck auf einem Griff des ausgewählten Objekts, zieht er den Griff — auch mit
     einem Zeichenwerkzeug. So lässt sich ein gerade gezogener Pfeil sofort biegen. Überall sonst legt ein
     Zeichenwerkzeug ein neues Objekt an, auch über einem vorhandenen (wie Snagit).
   - Die mittlere Maustaste verschiebt in jedem Werkzeug die Ansicht; das Mausrad zoomt, auch während der Texteingabe.

2. **Tastenkürzel** `V`, `A`, `R`, `T`, `N` ohne Modifikator (an Snagit und gängige Grafikprogramme angelehnt). Sie
   kollidieren nicht mit den vorhandenen Kürzeln des Editors (Strg+Z/Umschalt+Strg+Z/Strg+Y, Strg+S, Strg+0, Strg+1,
   Strg+W, Esc, Entf, Rücktaste). Während der Texteingabe sind Buchstaben Text, keine Kürzel; Strg-Kürzel wirken
   weiter. **Esc** wirkt in Stufen: Texteingabe verwerfen → laufendes Ziehen abbrechen → Auswahl aufheben → zurück zum
   Werkzeug „Auswählen“.

3. **Das Werkzeug bleibt nach dem Anlegen aktiv** (wie Snagit). Begründung: Der Hauptfall von A-21 ist, mehrere
   Schritt-Nummern oder Pfeile nacheinander zu setzen; ein Rücksprung zur Auswahl kostete dort je Objekt einen Klick
   oder Tastendruck. Das neue Objekt ist ausgewählt, damit sein Griff (Pfeil biegen) sofort greift. Zurück zur Auswahl
   mit `V`, dem Knopf oder Esc.

4. **Voreinstellungen:** Die Beschriftungsfarben sind laut Corporate Design (§3, letzte Regel) eine eigene Auswahl für
   Bilder, nicht die Oberflächenpalette. Erste Fassung: vier kräftige Farben, auf denen weiße Schrift lesbar bleibt —
   **Rot `#D93A33`** (Voreinstellung, auf Bildschirmfotos am auffälligsten), Blau `#2F6FDB`, Grün `#2E8B57`,
   Dunkelgrau `#1E2124` (grau-900). Strichstärke Pfeil und Kasten 4 px (Spitze damit 16 px), Kasten ungefüllt mit
   Eckradius 4 px (CD §5: runde Ecken), Text 20 px weiß auf einem Schild in der gewählten Farbe, Nummer Radius 14 px
   mit weißer, fetter Ziffer (wie Renderer und SVG-Export). Größen in Pixeln des Originals, unabhängig vom Zoom.
   - **Schrift:** Das Corporate Design nennt Open Sans; ausgeliefert wird sie noch nicht (CD §4: „soll … ausgeliefert
     werden“, dann mit OFL-Text unter `licenses/`). Editor und Renderer setzen bis dahin beide die Go-Schriften
     (ADR-0020 Nachtrag 1), der SVG-Export nennt „Open Sans, sans-serif“. Die Umstellung betrifft Renderer, Editor,
     das Kostenmodell des Rechenbudgets und die Lizenzdateien zugleich und gehört zu F-108 (eigene Karte), nicht in
     diese Karte.

5. **Eigenschaften:** nur die Farbe. Der Knopf „Farbe“ schaltet reihum durch die vier Farben; die Farbe gilt für alle
   folgenden Objekte und färbt das ausgewählte Objekt um (ein Rückgängig-Schritt). Beim Text färbt sie das Schild.

6. **Rückgängig:** je angelegtem Objekt genau ein Schritt — Anlegen läuft als Zug am neuen Objekt
   (`BeginneNeuenZug`/`BeendeZug`), Text kommt erst beim Bestätigen in den Verlauf. Biegen, Verschieben und Umfärben
   sind je ein Schritt. Abgebrochenes (Esc, zu klein, leerer Text) hinterlässt nichts. Während der Texteingabe sind
   Rückgängig und Wiederholen gesperrt (sie läuft als Zug, F-18); **Strg+S** und Schließen bestätigen den Text zuerst,
   damit er nicht verloren geht.

7. **Gleiche Darstellung:** keine neue Zeichenlogik. Die Vorschau beim Ziehen ist das Objekt im laufenden Zug; der
   Editor zeichnet es über den vorhandenen Weg (`ui.zeichneObjekte`), Datei und PNG/JPG über `render`, SVG über
   `fileformat`. Bekannte Abweichungen der Maße zwischen den drei Wegen (Text-Grundlinie, Text-Hintergrund aus der
   Schätzung `Text.Grenzen`, Kastenecken, Schriftname) bleiben F-108.

## Konsequenzen
- `canvas` bekommt Werkzeug, Stil und Texteingabe samt Tests mit Sollwerten; `ui/editor.go` bekommt Knöpfe, Kürzel,
  Textereignisse (`key.EditEvent` an den fokussierten Editor) und eine Einfügemarke.
- Neue Oberflächentexte (Werkzeugnamen, Hinweise in der Statusleiste) laufen über `i18n` mit Einträgen in `po/de.po`.
- Ein Objekt verschieben heißt bei aktivem Zeichenwerkzeug: erst `V`. Das ist der Preis von Punkt 3.
- **Später** (eigene Karten): Ellipse, Linie, Freihand, Textmarker, Sprechblasen (Rest von A-20/A-21); Wahl von
  Strichstärke, Schriftgrad, Füllung und „Text ohne Schild“; Schnellstile (A-22); Einfügemarke in der Zeile
  (Pfeiltasten, Auswahl im Text), Einfügen aus der Zwischenablage; Umnummerieren; Open Sans (F-108).
