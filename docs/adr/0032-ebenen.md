# ADR-0032: Ebenen — jede Beschriftung eine Ebene, Leiste mit Liste, Sichtbarkeit, Sperre, Deckkraft und Namen

## Status
Akzeptiert (Claude Opus 5.5, 23.11.2026, Karte #191 in Sprint 12). Grundlage: Wunsch Andi vom 22.11.2026 („die ebenen
sollten sich etwas mehr an normalen ebenen-programmen orientieren, gern aber einfacher bedienbar sein (… keine
unter-/masken-ebenen …, aber vielleicht sichtbar/nicht-sichtbar?)“) und Muster v2
(`tmp/sprint12/muster/bilder/187-oberflaeche-v2.png`, Block „Ebenen“, abgenommen mit ADR-0031). Ergänzt ADR-0031
Punkt 3 (rechte Leiste) und Punkt 1 (Menüzeile), ADR-0021 (Objekte) und ADR-0008 (Arbeitsdatei).

## Kontext
Die Beschriftungen liegen im Dokument schon als geordnete Liste über dem Bild (`document.Dokument.objekte`, unten
zuerst). Es gibt aber keinen Weg, die Reihenfolge zu ändern, ein Objekt zu verbergen, vor versehentlichem Verschieben zu
schützen oder halbdurchsichtig zu machen; ein Objekt unter einem anderen ist auf der Leinwand kaum zu greifen. Echte
Ebenenprogramme (GIMP, Krita, Photoshop) kennen dazu Gruppen, Masken und Mischmodi — das will Andi ausdrücklich nicht.

Offene Fragen aus der Karte: Werden unsichtbare Ebenen exportiert? Gibt es eine Deckkraft je Ebene neben der Deckkraft
der Farbe? Folgen für Dokumentmodell, Rückgängig, Arbeitsdatei und Rettungskopie.

## Entscheidung

1. **Jede Beschriftung ist eine Ebene**, das Bild ist die unterste. Die Reihenfolge der Ebenen ist die Zeichenreihenfolge
   der Objekte. Das Bild ist immer gesperrt und hat keine Aktionen; **Schwärzungen gehören zum Bild** (endgültig,
   ADR-0022) und erscheinen nicht als eigene Ebene. Gruppen, Masken, Mischmodi und Ebenen mit eigenen Pixeln gibt es
   nicht.
2. **Angaben je Ebene** (`document.Ebene`, in jedes Objekt eingebettet):
   - `Name` — leer heißt: die Oberfläche zeigt einen Namen aus Art und Inhalt („Text „Hier klicken““, „Nummer 1“,
     „Pfeil“, „Rechteck“). Höchstens 100 Zeichen, gültiges UTF-8, keine Steuerzeichen.
   - `Ausgeblendet` — siehe Punkt 4.
   - `Gesperrt` — siehe Punkt 5.
   - **Deckkraft** in ganzen Prozent 0 bis 100, gespeichert als `Transparenz` = 100 − Deckkraft, damit der Nullwert
     eines Objekts deckend ist und kein bestehender Aufrufer sich ändert. Sie wirkt **zusätzlich** zur Deckkraft der
     Farbe und **auf das Objekt als Ganzes** (Gruppendeckkraft): Schaft und Spitze eines Pfeils, Schild und Schrift eines
     Texts überlagern sich nicht doppelt.
3. **Rückgängig:** jede Aktion der Leiste ist genau ein Schritt — Umbenennen, Aus-/Einblenden, Sperren, Deckkraft
   (Ersetzen), Umordnen (neuer Befehl), Doppeln (Einfügen), Löschen (Entfernen).
4. **Ausgeblendete Ebenen** zeichnet weder der Editor noch der Renderer; Treffer auf der Leinwand übergehen sie. Sie
   fehlen in Export (PNG, JPG, SVG), Zwischenablage, sichtbarem Bild der Arbeitsdatei und Vorschau der Bibliothek.
   **Arbeitsdatei und Rettungskopie behalten sie** (sonst wäre Ausblenden Löschen). Ausblenden ist kein Schwärzen:
   Der Inhalt steht weiter im Block `snAP`; die Leiste sagt das in ihrem Hinweis.
5. **Gesperrte Ebenen** sind auf der Leinwand nicht zu treffen, zeigen keine Griffe und lassen sich dort nicht verschieben.
   In der Liste bleiben sie wählbar; was man dort bewusst tut (Eigenschaften in der Leiste, Umbenennen, Umordnen, Doppeln,
   Löschen, Entsperren), wirkt weiter.
6. **Abschnitt „Ebenen“ der rechten Leiste** (unter Farbe, Muster v2):
   - Feld **Deckkraft** der gewählten Ebene (Prozent, wie das Feld beim Farbwähler; ohne Auswahl grau).
   - **Liste** von oben (vorderste Ebene) nach unten, zuletzt die Zeile **Bild** mit Schloss. Je Zeile: Auge
     (Klick blendet aus/ein), **Vorschau** (das Objekt verkleinert in ein Feld eingepasst, mit seinen eigenen
     Zeichenwegen), Name, Schloss (Klick sperrt/entsperrt; bei offenen Ebenen nur beim Überfahren sichtbar).
     Ausgeblendete Zeilen stehen im Nebentext. Ein Klick auf die Zeile wählt das Objekt (und das Werkzeug Auswählen).
   - **Ziehen** einer Zeile ordnet um; eine Linie zeigt die Einfügestelle. **Doppelklick** auf den Namen oder F2 benennt
     um (Eingabefeld in der Zeile, Enter übernimmt, Esc verwirft, Klick daneben übernimmt).
   - Darunter die Knöpfe **↑ ↓ Doppeln Löschen** und der Hinweis „Schwärzen ist Teil des Bilds (endgültig).
     Ausgeblendetes bleibt in der Arbeitsdatei.“
   - Der Abschnitt hängt am **unteren Rand** der Leiste (höchstens acht Zeilen, dann blättert die Liste): Ein Klick auf
     eine Zeile wählt das Objekt und ändert den Inhalt darüber (Schrift, Strichstärke, Farbe); hinge die Liste daran,
     spränge sie unter dem Zeiger weg, und der zweite Klick eines Doppelklicks oder ein Ziehen träfe eine andere Zeile.
     Fehlt Platz, rückt er nach oben und die Liste schrumpft.
   - Die Liste blättert, wenn sie nicht passt. Die Überschriften **FARBE** und **EBENEN** klappen ihren Abschnitt per
     Klick ein und aus (für niedrige Fenster; der Zustand gilt für das Fenster und wird nicht gespeichert).
7. **Doppeln** legt eine Kopie direkt über das Original, um 10 Fensterpixel nach rechts unten versetzt, und wählt sie.
   Die Kopie übernimmt alle Angaben der Ebene (Name, Sperre, Deckkraft), wie in Ebenenprogrammen üblich.
8. **Menü „Ebene“** zwischen Werkzeuge und Hilfe (ändert ADR-0031 Punkt 1: sieben Menüs; die Titel davor behalten
   ihre Lage, Drehbücher mit Menü-Koordinaten bleiben gültig), mit Kürzeln in der
   Tastentabelle: Doppeln (Strg+D), Umbenennen (F2), Ausblenden/Einblenden (Strg+H), Sperren/Entsperren (Strg+L), Nach
   vorn (Strg+Bild↑), Nach hinten (Strg+Bild↓), Ebene darüber wählen (Alt+Bild↑), Ebene darunter wählen (Alt+Bild↓).
   Damit ist die Liste allein per Tastatur bedienbar (ADR-0026). Ohne Auswahl sind die Einträge grau.
9. **Arbeitsdatei** (`docs/format.md` §8.5): vier freiwillige Attribute an jedem Objekt — `s:name`, `s:ausgeblendet`
   (`true`), `s:gesperrt` (`true`), `s:deckkraft` (ganze Zahl 0 bis 100); geschrieben nur, wo sie von der Voreinstellung
   abweichen. Fehlend oder ungültig heißt: kein Name, sichtbar, offen, 100 % — kein Schaden. Für Betrachter trägt
   ein ausgeblendetes Objekt zusätzlich `display="none"`, eine Deckkraft unter 100 % `opacity`. Wie `s:schrift`
   (ADR-0021 Nachtrag 10) ändert das die `FormatVersion` nicht; ein älterer Leser zeigt alle Objekte sichtbar, offen und
   deckend. Der SVG-Export lässt ausgeblendete Objekte weg und schreibt die Deckkraft als `opacity`.
10. **Mischen:** Der Renderer zeichnet ein Objekt mit Deckkraft unter 100 % erst in ein durchsichtiges Zwischenbild in
    der Größe seines Umrisses und mischt dieses dann mit der Deckkraft ein (gleiche sRGB-Rechnung wie jede Füllung).
    Der Editor nimmt Gios Deckkraft-Ebene (`paint.PushOpacity`); Gio mischt linear, die bekannte Abweichung F-520 gilt
    auch hier.

## Begründung
- Die Angaben am Objekt statt in einer eigenen Ebenenliste halten Kopien (`Abzug`), Rettungskopie und Format ohne
  zweite Buchführung stimmig: Ein Objekt ist eine Ebene, Rückgängig geht über die bestehenden Befehle.
- Gruppendeckkraft ist, was man von „Deckkraft einer Ebene“ erwartet; das Multiplizieren der Farbdeckkraft zeigte
  Überlappungen dunkler (Pfeilspitze, Schrift auf dem Schild).
- Ausgeblendetes nicht exportieren entspricht jedem Ebenenprogramm und ist der sichere Fall (nichts Verborgenes geht
  hinaus); es in der Arbeitsdatei zu behalten ist der Sinn des Ausblendens.

## Folgen
- `document`: `Ebene` in Pfeil, Kasten, Text, Nummer; `Objekt` bekommt `Ebenenwerte()`/`mitEbene`; neue Befehle
  `umordnen`; `MitEbene`, `FuegeEin`, `Umordnen`, `Stelle`.
- `render`, `fileformat` (SVG schreiben/lesen, Export), Editor-Zeichnen, `canvas` (Treffer, Griffe, Leiste, Menü),
  `ui` (Leiste, Menü „Ebene“, Tasten), `docs/format.md` §8.5 mit Beispiel `testdata/format/layers`.
- Die Leiste wird höher; die einklappbaren Abschnitte fangen niedrige Fenster ab. Die Überschrift „Farbe“ ist jetzt ein
  Knopf und steht in der Tab-Reihenfolge vor den Farbfeldern; danach folgen Überschrift „Ebenen“, Deckkraft und die vier
  Knöpfe, dann die Fußzeile.
- Die Vorschau zeichnet Striche mindestens 1,5 Pixel breit, sonst verschwänden Pfeil und Rechteck in der Verkleinerung.
- Sichtprobe: `scripts/drehbuecher/ebenen-191.txt`. Drehbücher mit festen
  Leisten-Koordinaten (`farbwaehler-189.txt`, `schrift-190.txt`) verschieben sich nicht, weil Ebenen unter der Farbe
  stehen.
