# ADR-0021: Zeichenwerkzeuge im Editor — Pfeil, Kasten, Text, Schritt-Nummer

## Status
Akzeptiert (Claudes Entscheidung bei der Umsetzung von Karte #36, 20.11.2026; Andi im Echtlauf am 20.11.2026: alle
Werkzeuge gehen; Farbwahl von Andi bestätigt, Nachtrag 1; Nachtrag 2 Claudes Entscheidung, Andi informiert)

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
   *Seit Nachtrag 11:* zwei Farben je Werkzeug; beim Text ist die gewählte Vordergrundfarbe die Schrift, das Schild die
   Füllung (Voreinstellung weiß auf Rot).
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
  (Pfeiltasten, Auswahl im Text), Einfügen aus der Zwischenablage; Umnummerieren; Open Sans ausliefern (eigenes Thema,
  seit Nachtrag 3 nicht mehr F-108).

## Nachtrag 1 (20.11.2026): Nacharbeit nach dem Kritiker-Lauf (F-150, F-151, F-153, F-154)

- **Nur im Bild anlegen (F-150):** Ein Zeichenwerkzeug beginnt ein Objekt nur, wenn der Druckpunkt im sichtbaren Bild
  liegt (Ansicht nach Zuschnitt und Stauchen, Bildkante eingeschlossen). Im grauen Rand passiert nichts: kein Objekt,
  kein Verlaufsschritt, keine Änderung, bei Text kein unsichtbares Textfeld. Beim Ziehen über den Rand hinaus wird der
  Endpunkt auf das Bild begrenzt (Pfeil-Spitze, Kasten-Ecke, Nummer; auch nach dem Einrasten mit Umschalt). Die
  Griffe des ausgewählten Objekts gehen weiter vor und bleiben im Rand greifbar (Griff vor Neuanlage, Punkt 1).
- **Meldungen vor Hinweis (F-151):** Eine frische Meldung der Statusleiste (Speichern, „Keine Änderungen“, Fehler,
  Grenze erreicht) hat Vorrang vor dem Werkzeughinweis, bis das Dokument sich ändert (anderer Verlaufsstand) oder das
  Werkzeug wechselt; danach zeigt die Leiste wieder den Hinweis. Ein **Fehler** überdeckt der Werkzeugwechsel nicht, er
  bleibt bis zur nächsten Änderung oder bis eine neue Meldung (etwa ein gelungenes Speichern) ihn ablöst. Die
  Entscheidung ist die reine Funktion `canvas.StatusZeile`.
- **Tab während der Texteingabe (F-153):** Tab wird abgefangen und hat **keine Wirkung** (kein Leerzeichen, kein
  Tabulator im Text, kein Fokuswechsel) — in der ersten Fassung ist ein Zeichen, das nur Wirkung auf den Fokus hat,
  mehr Schaden als Nutzen. Ohne Texteingabe bleibt Tab die Fokusfahrt der Leiste. **Enter** löst ohne Texteingabe
  einen per Tastatur fokussierten Knopf aus (statt von der Leinwand verbraucht zu werden).
- **Grenzen sichtbar (F-154, Punkt 1):** Ist `document.MaxObjekte` erreicht oder die nächste Nummer größer als
  `document.MaxNummer`, legt das Werkzeug nichts an und die Statusleiste sagt es (rote Meldung); der Nummer-Hinweis
  verspricht dann keine Nummer mehr. Ein Klick auf einen vorhandenen Text bearbeitet ihn auch bei vollem Dokument.
- **Mittlere Taste im Text (F-154, Punkt 2):** Sie verschiebt nur die Ansicht, die Texteingabe läuft weiter (Punkt 1
  dieser ADR galt schon, nur die Umsetzung bestätigte den Text).
- **Verwerfen-Marke (F-154, Punkt 3):** Jede neue Bearbeitung nach einem gescheiterten Speichern (Zeichnen,
  Verschieben, Löschen, Umfärben, Text bestätigen) hebt die Marke auf, auch wenn Rückgängig danach denselben Stand
  wieder herstellt; Rückgängig und Wiederholen allein heben sie nicht auf.
- **Entschieden (F-154, Punkt 4; Andi, 20.11.2026):** Die Farbwahl färbt das ausgewählte, eben gezeichnete Objekt um
  (Punkt 5 dieser ADR) und gilt zugleich für die nächsten Objekte — wie in Snagit. Das neue Objekt bleibt ausgewählt,
  damit sein Griff sofort greift.

## Nachtrag 2 (20.11.2026): Umschalt am Rand, Rasten mit Streifen, Objekte im Bild, Statusleiste (F-156, F-157, F-160)

*Entscheidungen von Claude, Andi informiert (Karten #84, #85).*

- **Umschalt am Bildrand (F-156):** Mit Umschalt wird nicht je Achse geklemmt (das verbog Pfeil und Kasten), sondern
  entlang der Richtung gekürzt: Der Pfeil bleibt auf 45° und endet am Bildrand, der Kasten bleibt ein Quadrat mit der
  Seite, die der Platz in beiden Richtungen zulässt. Nachtrag 1 („auch nach dem Einrasten mit Umschalt“) gilt weiter,
  nur ohne Formverlust.
- **Rasten in der Ansicht (F-157):** Einrasten und Quadrat rechnen in Ansichtskoordinaten (nach Zuschnitt und
  Stauch-Streifen) und gehen danach zurück ins Original. Die Spitze liegt so nie in einem herausgeschnittenen Streifen;
  in der Ansicht bleibt der Pfeil 45°. Im Original ist der Winkel über einen Streifen hinweg dann nicht mehr 45° — was
  zählt, ist, was der Nutzer sieht.
- **Objekte bleiben im Bild (F-160, Punkt 1):** Verschieben und Griffe dürfen ein Objekt nicht ganz aus dem sichtbaren
  Bild bringen. Die Grenzen des Objekts müssen das Bild (Ansicht) je Achse um mindestens `canvas.MinSichtbar` = 8
  Fensterpixel schneiden (ein kleineres Objekt: um seine ganze Ausdehnung). Beim Ziehen wird die Verschiebung je Achse
  entsprechend gekürzt, sodass das Objekt am Rand entlanggleitet; bei Griffen wird die Zeigerposition je Achse zum
  Ausgangsort des Griffs zurückgenommen. Ein Objekt, das schon beim Öffnen anders liegt, wird nicht verändert, ein
  Zug in Richtung Bild bleibt erlaubt. Damit bleibt jedes Objekt greifbar und ein Rückgängig nicht der einzige Ausweg.
- **Statusleiste (F-160, Punkte 2 bis 4):** Eine Meldung, die einmal abgelöst war (Standwechsel, bei gewöhnlichen
  Meldungen auch Werkzeugwechsel), kehrt nicht zurück — weder nach Ändern und Rückgängig noch nach Werkzeugwechsel
  hin und zurück (`Meldung` merkt es sich, die Oberfläche fragt nach jedem Ereignis). Während einer Texteingabe geht
  der Werkzeughinweis vor „Gespeichert“; ein Fehler bleibt sichtbar. `canvas.StatusZeile` nimmt dafür `*Meldung` und
  die Angabe, ob eine Texteingabe läuft.

## Nachtrag 3 (21.11.2026): Gleiche Maße in Renderer, SVG und Editor (F-108, F-149, F-172)

**Status:** Akzeptiert (Claude, Andi informiert). Umgesetzt von Claude Sonnet 5.5 (Karte #60, Commits 6729624 und
a9f36c1), Sichtprobe im virtuellen KWin.

- **Schrift im SVG:** Der Export nennt die Go-Schrift, wie Renderer und Editor sie setzen
  (`font-family="Go, Go Regular, sans-serif"`); der Satz zu „Open Sans, sans-serif“ unter „Schrift“ gilt nicht mehr.
  Open Sans auszuliefern bleibt ein eigenes Thema (CD §4), nicht mehr F-108.
- **Text-Schild:** gemessene Tinte der Glyphen plus 0,25 Schriftgrad Innenabstand rundum (`document.Text.Grenzen`,
  `TextInnenabstand`). Renderer, SVG und Editor füllen den Hintergrund genau dort; Treffer, Auswahlrahmen und
  Verschieben nutzen dieselbe Fläche. Ohne ladbare Schrift gilt die alte Schätzung mit demselben Rand. Ein Umbruch bei
  `BreiteGrenze` ist nicht umgesetzt; die Fläche kappt deshalb nicht.
- **Nummer:** `Nummer.Grenzen` umfasst Kreis und Ziffern (bei drei und mehr Stellen ragen sie hinaus); die Ziffern
  sitzen auf derselben Grundlinie (halbe Versalhöhe unter der Mitte) in Renderer, SVG und Editor.
- **Kasten:** Ecken rund, auch im SVG (`stroke-linejoin="round"`); der Eckradius ist überall gleich begrenzt
  (`document.WirksamerEckradius`: höchstens die halbe kürzere Seite, nie elliptisch, unter 0,5 px keiner).
- **Editor:** setzt Text und Ziffern auf die Grundlinie. Gios `layout.Dimensions.Baseline` zählt von der **Unterkante**
  des Felds; die erste Fassung zog sie von der Oberkante ab und setzte den Text zu tief (in der Sichtprobe gefunden,
  `TestEditorTextAufGrundlinie` fängt es jetzt).
- *Ergänzt (F-342, F-343, Karten #152, #153):* Die Editor-Grundlinie steht in `textGrundlinieAnsicht` und
  `nummerGrundlinieAnsicht`, geprüft gegen `document`. Der SVG-Text trägt `xml:space="preserve"` und zeigt
  `document.Zeichenbar` (Tabulator als vier Leerzeichen); `textLength` bewusst nicht gesetzt (verzerrte die
  Ersatzschrift, `fileformat` kennt keine Glyphenbreiten).

## Nachtrag 4 (23.11.2026): Voreinstellung eckig, Eckradius 0 (Karte #186)

**Status:** Akzeptiert (Claude im Auftrag Andis, 22.11.2026: „keine runden Ecken“; ADR-0031 Punkt 8).

- **Voreinstellung:** `canvas.StandardStil` legt Kästen mit Eckradius 0 an (vorher 4). Das Text-Schild ist schon eckig,
  eine Sprechblase gibt es nicht. Der Radius bleibt im Datenmodell (`document.Kasten.Eckradius`), in Renderer, SVG-Export
  und Editor einstellbar und gleich begrenzt (Nachtrag 3, `WirksamerEckradius`).
- **Gespeicherte Dateien bleiben:** ein gespeicherter Eckradius wird beim Öffnen nicht verändert, auch nicht auf 0 gesetzt.
- **Oberfläche:** die Overlay-Schilder (Maß, Position) sind eckig (`schildUmriss`); Knöpfe, Menü, Leisten und Felder der neuen
  Oberfläche (ADR-0031) und der Bestätigungsdialog waren es schon. Kreise als Inhalt (Schritt-Nummer, „s“) bleiben rund.
- **Nicht geändert:** die Strichecken des Kastens bleiben rund (`stroke-linejoin="round"`, Nachtrag 3); bei Radius 0 sind die
  Außenecken dadurch um die halbe Strichbreite gerundet. Eckige Strichecken wären eine eigene Entscheidung (Renderer, SVG, Editor zugleich).
- **Begründung:** Gegen den üblichen KI-Look (Pillen, weiche Karten); Regeln in `brand/corporate-design.md` §1 und §5.

## Nachtrag 5 (23.11.2026): Spitze Strichecken beim Kasten ohne Rundung (F-498, Karte #211)

**Status:** Akzeptiert (Claude Sonnet 5.5 im Auftrag der Karte; ersetzt den Punkt „Nicht geändert“ aus Nachtrag 4 und die
runden Strichecken aus Nachtrag 3 für Kästen ohne Rundung).

- **Regel:** Ist `WirksamerEckradius` 0 (Eckradius 0 oder unter 0,5 px), hat der Strich spitze Außenecken (Gehrung); bei
  Radius > 0 bleibt alles wie bisher (runde Strichecken, SVG `stroke-linejoin="round"`).
- **Gleich in drei Ausgängen:** Renderer und Editor füllen `document.EckigerRahmen` (vier überlappungsfreie Rechtecke,
  bei verschwundenem Innenraum eins; Gios Strich kennt nur runde Ecken, deshalb Füllung statt Strich); das SVG sagt
  `stroke-linejoin="miter"` (Gehrung 90° = 1,41 unter der Standardgrenze 4). Die Ausdehnung bleibt `Kasten.Grenzen`
  (Bereich ± halbe Strichbreite): Trefferfläche, Griffe und Auswahlrahmen ändern sich nicht.
- **Format:** `docs/format.md` und `FormatVersion` unverändert (`stroke-linejoin` ist Darstellungsattribut, kein
  `s:`-Attribut); nur die Beispiele `view.png` und `view.snap.svg` tragen jetzt `miter`.
- **Nicht geändert:** Pfeil (runde Enden, Spitze ein Dreieck), Text-Schild (Rechteck ohne Strich) und Nummer (Kreis).

## Nachtrag 6 (23.11.2026): Farbwahl über Leiste und Listen statt reihum (Karte #187)

**Status:** Akzeptiert (Claude Opus 5.5). Ändert Punkt 5: Der Knopf „Farbe“ schaltet nicht mehr reihum. Die vier
Farben wählt man in der rechten Leiste oder in der Farbliste des Farbfelds links bzw. der Fußzeile (ADR-0031
Nachtrag 3); Umfärben des gewählten Objekts als ein Rückgängig-Schritt und „gleiche Farbe ist kein Schritt“ bleiben
(`canvas.Sitzung.SetzeFarbe`). `NaechsteFarbe` ist entfallen.

## Nachtrag 7 (23.11.2026): Strichstärke wählbar, Voreinstellungen wachsen mit der Bildgröße (Karte #216)

**Status:** Akzeptiert (Claude Opus 5.5). Anlass: Andi, 23.11.2026: „die strichstärke der pfeile usw [sollte sich]
relativ zur bildgröße(-auflösung) anpassen … bei pfeil usw fehlt noch die strichstärke“. Ändert Punkt 4 (feste
Größen) und ergänzt Punkt 5 (nur die Farbe).

1. **Voreinstellung nach Bildgröße:** Beim Öffnen gelten Strich, Schriftgrad und Nummernradius aus Punkt 4 mal dem
   Faktor der Stauch-Kante, `document.KantenFaktor(Originalgröße)` = max(1, kurze Seite / 1080), ganzzahlig
   (`canvas.StilFuer`): Full HD und kleiner 4 / 20 / 14 px, 4K 8 / 40 / 28 px, 8K 16 / 80 / 56 px. Ganzzahlig, damit
   die Werte die Stufen unten treffen und gleiche Bilder gleiche Größen bekommen. Maßgeblich ist das Original (wie bei
   der Kante, ADR-0023 Nachtrag 2), nicht der Zuschnitt. Gespeicherte Objekte ändern sich nie; die Voreinstellung gilt
   je Fenster und wird nicht gespeichert (Einstellungen mit #175).
2. **Strichstärke in der rechten Leiste** für Pfeil und Kasten (Werkzeug oder gewähltes solches Objekt): − und + gehen
   durch die Stufen 1, 2, 3, 4, 6, 8, 12, 16, 24, 32 px (`canvas.StrichStufen`); eine Breite zwischen zwei Stufen (aus
   einer älteren Datei) geht zur nächsten in der Richtung, an den Enden werden die Knöpfe grau. Die Anzeige zeigt die
   Breite des gewählten Objekts, sonst die des Werkzeugs, gerundet in ganzen Pixeln. Wie bei der Farbe ändert die Wahl
   das gewählte Objekt als einen Rückgängig-Schritt (`canvas.Sitzung.SetzeBreite`), dieselbe Breite ist kein Schritt.
   Die Pfeilspitze wächst wie bisher mit dem Strich.
3. Schriftgröße (#190) und Nummerngröße bekommen ihre Bedienung mit ihren Karten; die Fußzeile zeigt die Strichstärke
   nicht (die Leiste genügt).

## Nachtrag 8 (23.11.2026): Faktor wie die Kante gedeckelt, Stufen und Faktor (Sitzungs-QA, A-7, B-6)

**Status:** Akzeptiert (Claude Opus 5.5). Berichtigt Nachtrag 7 Punkt 1.

- **Deckel:** Die Sitzung nimmt den Faktor des Dokuments (`document.Dokument.KantenFaktor`), der wie bei der Kante
  (ADR-0023 Nachtrag 4) durch die kurze Seite der aktuellen Ansicht gedeckelt ist (`canvas.StilMitFaktor`); ein
  4K-Original, auf 300×80 zugeschnitten, beginnt mit Faktor 1 statt mit 40-px-Text. `canvas.StilFuer(Größe)` bleibt
  für den ungedeckelten Faktor einer Bildgröße.
- **Stufen:** Der Satz „Ganzzahlig, damit die Werte die Stufen unten treffen“ gilt nur für die Faktoren 1, 2, 3, 4, 6
  und 8. Faktor 5 (20 px) und 7 (28 px) liegen zwischen zwei Stufen, ab Faktor 9 liegt die Voreinstellung über 32 px;
  − und + gehen dann zur nächsten Stufe in der Richtung (+ ist grau, − springt auf 32). Das ist gewollt: Die
  Voreinstellung folgt dem Bild, die Stufen der Bedienung.
- Nachtrag 7 löst den Satz in ADR-0031 Nachtrag 3 „… und Strichstärke fehlen noch“ ab; siehe ADR-0031 Nachtrag 4.

## Nachtrag 9 (23.11.2026): Freie Farbe mit Deckkraft, zuletzt benutzte Farben, Pipette (Karte #189)

**Status:** Akzeptiert (Claude Opus 5.5). Anlass: Andi, 22.11.2026: „einen farbregler und ein (hex?) eingabefeld für
die farbe und zusätzlich einen regler für die transparenzstufe“; Anforderung A-41 (Farbpipette). Ergänzt Punkt 4 und
Nachtrag 6: Neben den vier Beschriftungsfarben ist jede Farbe wählbar, mit Deckkraft.

1. **Deckkraft gehört zur Farbe.** Die Farbe eines Werkzeugs bzw. Objekts ist ein `color.NRGBA` mit Alpha (nicht
   vormultipliziert). Dokumentmodell, Arbeitsdatei (`#rrggbbaa`, `docs/format.md` §5.2), SVG-Export (`fill-opacity`/
   `stroke-opacity`) und Renderer tragen Alpha schon; das Format ändert sich nicht, `FormatVersion` bleibt. Wirkung je
   Objekt wie beim Umfärben: Strich von Pfeil und Kasten, Schild des Texts (die Schrift bleibt weiß und deckend),
   Kreis der Nummer (die Ziffer bleibt weiß). Eine Deckkraft von 0 % ist erlaubt (unsichtbar, aber wählbar und mit
   Griffen). **Schwärzen und Verpixeln** haben keine Farbwahl und bleiben deckend (Kernprinzip 2).
2. **Ein Rückgängig-Schritt je Wahl.** Ein Klick auf ein Farbfeld, die Pipette, eine bestätigte Eingabe (Hex,
   Deckkraft) ist ein Schritt am gewählten Objekt, wie bisher (`SetzeFarbe`). Ziehen in der Farbfläche oder an einem
   Regler färbt das gewählte Objekt sichtbar mit, bleibt aber **ein** Schritt beim Loslassen (`BeginneFarbZug`,
   `FarbZugHSV`, `BeendeFarbZug` über den Zug des Dokuments); während der Texteingabe gehört es zur Eingabe. Solange
   gezogen wird, ruhen Speichern und Leinwand (`Zieht`). Esc, Schließen und ein Signal brechen das Ziehen ab, das
   Objekt steht wie vorher — in der Texteingabe der Text der Eingabe (Sitzungs-QA A-1, A-4, Verify V-3); Schließen
   und Signal speichern danach wie sonst.
3. **Farbton bleibt beim Ziehen erhalten.** Die Sitzung merkt sich den Farbton (HSV), damit Grau, Schwarz oder Weiß beim
   Ziehen in der Fläche den Ton nicht verlieren; HSV ist nur Bedienung, gespeichert wird RGB.
4. **Zuletzt benutzte Farben:** Jede Farbe, die nicht über ein Farbfeld gewählt wurde (Fläche, Regler beim Loslassen,
   Hex, Deckkraft, Pipette) und nicht unter den Beschriftungsfarben steht, kommt vorn in eine Liste von höchstens drei
   (`canvas.ZuletztGrenze`); dieselbe Farbe rückt nach vorn statt doppelt zu stehen. Die Liste gilt je Fenster und wird
   nicht gespeichert (Einstellungen mit #175).
5. **Pipette:** Ein Klick auf die Pipette schaltet sie ein; der nächste Linksklick ins Bild übernimmt die Farbe des
   Bildpunkts darunter aus dem Original **mit** Schwärzungen und **ohne** Beschriftungen, deckend (Alpha 255) — so
   färbt man eine Beschriftung passend zum Bild, ohne eine andere Beschriftung aufzugreifen. Ein Klick neben das Bild,
   Esc oder ein Werkzeugwechsel beendet sie ohne Wirkung; ein Klick trifft dann kein Objekt und beginnt keinen Zug.
   Den Bildpunkt bestimmt dieselbe Umrechnung wie der Zeigerort der Fußzeile (beim Zuschneiden die Vollansicht,
   Sitzungs-QA A-6). Ein ganz durchsichtiger Bildpunkt (Alpha 0) wird nicht aufgenommen, die Pipette endet ohne
   Wirkung: Seine Farbe sieht niemand (A-9, entschieden von Claude im Auftrag Andis, 23.11.2026).
6. **Eingaben:** Hex nimmt 3, 6 oder 8 Hex-Ziffern, mit oder ohne `#`, Groß- oder Kleinschreibung; 3 und 6 Ziffern
   behalten die Deckkraft (bei Deckkraft 0 seit Nachtrag 11 Punkt 7: deckend), 8 setzen sie mit. Deckkraft nimmt eine ganze Zahl 0 bis 100, mit oder ohne `%`. Ungültige
   Eingaben ändern nichts. Umrechnung Prozent ↔ Alpha: Alpha = round(p · 255 / 100), angezeigt round(Alpha · 100 / 255)
   (`canvas.Deckkraft`).
7. **Editor-Mischung:** Gio mischt halbdurchsichtige Flächen linear, Renderer und Export in sRGB (F-520, Karte #215);
   halbdurchsichtige Beschriftungen sehen im Editor daher etwas heller aus als in PNG/JPG. Das bleibt bis #215.
8. **Nicht hier:** Eine zweite Farbe „Füllung“ (Fläche von Kasten und Text-Schild getrennt vom Strich, mit Tausch-Knopf;
   Entscheidung im Kommentar an #189 vom 22.11.) ändert die Bedeutung der Farbe beim Text und bekommt eine eigene Karte
   mit eigenem Nachtrag.

## Nachtrag 10 (23.11.2026): Schriftart, Größe, Stil und Ausrichtung im Text-Werkzeug (Karte #190)

**Status:** Akzeptiert (Claude Opus 5.5). Anlass: Andi, 22.11.2026: „bei der schrift brauchen wir auch eine
schrift-auswahl, größen-auswahl, stil-auswahl und so“; Muster v2 (Block „Werkzeug: Text“). Ändert Punkt 4 (nur Go
Regular) und präzisiert Nachtrag 3 (gleiche Maße überall).

1. **Modell:** Ein Text trägt zusätzlich `Schrift` (Familienname, leer = Go), `Fett`, `Kursiv`, `Unterstrichen`,
   `Durchgestrichen` und `Ausrichtung` (links, Mitte, rechts). Die Voreinstellung (Go, normal, links) ist genau der
   bisherige Text; vorhandene Dateien ändern sich nicht.
2. **Arbeitsdatei:** drei freiwillige Attribute am Objekt `text` (`docs/format.md` §8.3): `s:schrift` (Familienname,
   höchstens 128 Zeichen), `s:stil` (Wörter aus `bold italic underline strike`, durch Leerzeichen getrennt; unbekannte
   Wörter werden übergangen) und `s:ausrichtung` (`left`, `center`, `right`). Fehlend oder unbekannt heißt Go, normal,
   links; das ist kein Schaden. Wie `s:kante` (ADR-0023 Nachtrag 7) ändert das die `FormatVersion` nicht: Ein älterer
   Leser übergeht die Attribute und zeigt den Text in Go Regular (Regel 3 der Spezifikation). Kein Pfad, nur der Name
   (Kernprinzip 3).
3. **Schriftquellen** (`internal/fonts`): eingebaut die Go-Schriften „Go“ und „Go Mono“ mit je vier Schnitten; dazu die
   Systemschriften aus den XDG-Schriftordnern (`$XDG_DATA_HOME/fonts`, `~/.fonts`, `<XDG_DATA_DIRS>/fonts`, Voreinstellung
   `/usr/local/share/fonts` und `/usr/share/fonts`), gelesen mit `golang.org/x/image/font/sfnt` — kein cgo, kein
   fontconfig, nichts wird auf die Platte geschrieben (Gio selbst würde einen Index im Cache anlegen). Gelesen werden
   TrueType und OpenType (`.ttf`, `.otf`) sowie Sammlungen (`.ttc`, `.otc`); höchstens 20 000 Dateien und 64 MiB je
   Datei, Verweise auf Ordner werden nicht verfolgt. Nach der Sitzungs-QA: höchstens 32 geladene Schriftdateien bzw.
   256 MiB je Prozess (darüber Go; die Grenze gilt nicht als Fehler der Datei), die Suche bricht nach 10 s ab und
   `Laden` wartet höchstens 11 s; `schnapp mcp` sucht nur im Heim der Benutzerdatenbank und den Systemordnern;
   Farb-Bitmap-Schriften fehlen in der Liste. Aus jeder Schrift zählen der Familienname (typografische Familie,
   sonst Familie) und die vier Grundschnitte (normal, fett, kursiv, fett kursiv, erkannt am Namen des Schnitts: „Bold“,
   „Italic“/„Oblique“; „Regular“, „Book“, „Normal“, „Roman“ oder leer = normal). Andere Schnitte (Light, Medium,
   Condensed …) übergeht die Liste. Die Suche läuft einmal je Prozess im Hintergrund beim Öffnen des Editors; bis sie
   fertig ist, zeigt die Liste die eingebauten Schriften.
4. **Wahl des Schnitts:** Fehlt ein verlangter Schnitt in der Familie, gilt der nächste vorhandene (fett kursiv → fett
   → normal, kursiv → normal); künstliches Fett oder Schrägstellen gibt es nicht. Fehlt die Familie (Datei von einem
   anderen Rechner), setzen Messung, Renderer und Editor in Go; der Name bleibt im Dokument und kommt beim Speichern
   wieder mit, die Leiste zeigt ihn mit „(missing)“.
5. **Ein Satz für alles:** Messung des Schilds, Renderer, Editor und die Lage im SVG-Export setzen Text aus derselben
   Schriftdatei mit Vorschub und Paarunterschneidung (`GPOS`, sonst `kern`, im Schriftgrad skaliert:
   `document.KernPaar`), ohne Hinting. Es gibt kein Shaping (keine Ligaturen, keine Schriften, die Shaping brauchen);
   dafür stimmen die Maße überall. Berichtigt nach der Sitzungs-QA (B-4): Anfangs hieß es „keine GPOS-Unterschneidung“,
   und die Unterschneidung kam über `opentype.Face.Kern` unskaliert (gleich viele Pixel bei jeder Größe).
   Deshalb zeichnet der Editor Text nicht mehr über den Shaper von Gio, sondern aus den Glyphenumrissen derselben
   Datei; die Schreibmarke misst mit demselben Satz.
6. **Ausrichtung:** Jede Zeile wird innerhalb der Blockbreite (Vorschub der längsten Zeile) links, mittig oder rechts
   gesetzt; der Ort bleibt die linke obere Ecke des Blocks.
7. **Unter- und Durchstrich** sind Rechtecke über den Vorschub der Zeile: Unterstrich mit der Oberkante 0,1 Schriftgrad
   unter der Grundlinie, Durchstrich mit der Mitte 0,3 Schriftgrad über der Grundlinie, Dicke max(1 px, 0,06 Schriftgrad).
   Sie gehören zur Tinte und damit zum Schild. Im SVG-Export stehen sie als `<rect>`, nicht als `text-decoration`.
8. **SVG-Export:** `font-family` nennt die Familie und dahinter Go und `sans-serif`, `font-weight="bold"` und
   `font-style="italic"` je nach Stil, jede Zeile mit eigenem `x` nach der Ausrichtung. Die Schrift wird nicht
   eingebettet (Lizenzen); ein Betrachter ohne die Schrift nimmt die nächste.
9. **Bedienung** (rechte Leiste, Text-Werkzeug oder gewählter Text, ADR-0031 Nachtrag 6): Schrift (Liste), Größe (− / +
   in den Stufen `canvas.SchriftStufen` und Anzeige in px), Stil (vier Umschalter F, K, U, S) und Ausrichtung (drei
   Knöpfe). Jede Wahl gilt für neue Texte und ändert den gewählten Text als einen Rückgängig-Schritt; während der
   Texteingabe gehört sie zur Eingabe. Gleiches erneut gewählt ist kein Schritt.

**Ergänzung (23.11.2026, Karte #221, F-575, F-576):** Die Sichtbarkeitsgrenzen für Zeilen und Glyphen am Bildrand
folgen der Schrift (`fonts.Schrift.Huelle`): head-Rahmen aller Glyphen plus 0,05 em, mindestens 1,5 em über, 1 em unter
der Grundlinie und 1,25 em seitlich (höchstens 8 em); Go und Go Mono bleiben genau bei den Mindestwerten. Renderer,
Zeichenfläche des Editors und Trockenlauf nutzen dieselbe Hülle. Eine offene Schriftliste liest nach dem Ende der
Hintergrundsuche neu. Die Unterschneidung ist mit der synthetischen Testschrift `testdata/fonts/schnapp-testkern.ttf`
(eigenes Werk, CC0 1.0) geprüft.

## Nachtrag 11 (23.11.2026): Zweite Farbe „Füllung“, Farbpaar je Werkzeug, Tauschen (Karte #219)

**Status:** Akzeptiert (Claude Opus 5.5). Grundlage: Entscheidung vom 22.11.2026 an Karte #189 (Andi fragte nach einer
Hintergrundfarbe: „zwei Farben wie in Bildprogrammen“), Muster v2 (zwei Felder unter der Werkzeugleiste). Löst
Nachtrag 9 Punkt 8 ein; ändert Punkt 4 (eine Farbe für alles) und die Bedeutung der Farbe beim Text.

1. **Zwei Farben:** **Vordergrund** — Linie von Pfeil und Kasten, Schrift des Texts, Kreis der Nummer — und **Füllung** —
   Fläche des Kastens, Schild des Texts (später die Sprechblase). Füllung kann **keine** sein (Deckkraft 0). Pfeil und
   Nummer haben keine Füllung; dort ist das Feld grau. Modell und Format ändern sich nicht: `Kasten.Fuellung`,
   `Text.Farbe` und `Text.Hintergrund` gibt es schon.
2. **Farbpaar je Werkzeug:** Pfeil, Kasten, Text und Nummer merken sich ihr eigenes Paar, solange das Fenster offen ist
   (nicht in den Einstellungen; Schnellstile #192 bauen darauf auf). Voreinstellung: Pfeil Rot, Kasten Rot ohne Füllung,
   **Text weiße Schrift auf rotem Schild** (ein neuer Text sieht aus wie bisher), Nummer Rot. Folge: Eine Farbe, die bei
   einem Werkzeug gewählt wird, gilt nicht mehr für alle Werkzeuge (bisher Punkt 4) — so arbeiten Bildprogramme und
   Snagit, und nur so passt die Voreinstellung des Texts zu der der anderen Werkzeuge. Die Zuletzt-Farben bleiben
   gemeinsam.
3. **Aktives Feld:** Farbwähler, Farbfelder, zuletzt benutzte Farben, Pipette, Hex, Deckkraft und die Farbliste der
   Fußzeile wirken auf das **aktive** Feld (Vordergrund oder Füllung). Gewählt wird es per Klick auf eines der beiden
   Felder — unter der Werkzeugleiste (vorn links Vordergrund, hinten rechts Füllung) oder in der Zeile über dem
   Farbwähler der rechten Leiste. Hat das Werkzeug bzw. Objekt keine Füllung, ist Vordergrund aktiv. Ist ein Objekt
   gewählt, zeigen und ändern die Felder seine Farben (ein Rückgängig-Schritt je Änderung, wie bisher).
4. **Keine Füllung:** Knopf „Keine“ neben den beiden Feldern der Leiste (nur bei aktiver Füllung an); gleichbedeutend mit
   Deckkraft 0 %. Ein Text ohne Schild zeigt nur die Schrift.
5. **Tauschen:** Knopf ⇄ neben den Feldern (Werkzeugleiste und rechte Leiste), Menü Bearbeiten → „Farben tauschen“,
   Kürzel **Strg+Umschalt+X** (X allein ist Stauchen; Umschalt+X allein träfe beim Tippen eines großen X im Text). Mit
   gewähltem Objekt werden seine beiden Farben getauscht (ein Rückgängig-Schritt), sonst das Paar des Werkzeugs; ohne
   Füllung (Pfeil, Nummer) ist Tauschen grau.
6. **Fußzeile** zeigt die Farbe des aktiven Felds; die Statuszeile nennt das Feld nicht eigens.
7. **Ganz durchsichtige Farbe wird beim Wählen deckend** (ergänzt bei der Umsetzung, 23.11.2026): Ist die Farbe des
   aktiven Felds ganz durchsichtig (Füllung „keine“), übernimmt eine Hex-Eingabe mit 3 oder 6 Ziffern Deckkraft 100 %
   statt 0 %, und ein Zug in Farbfläche oder Farbton-Regler beginnt deckend. Sonst bliebe die gewählte Farbe
   unsichtbar. Das ändert Nachtrag 9 Punkt 6 nur für Deckkraft 0; 8 Ziffern und der Deckkraft-Regler setzen sie wie
   bisher.
8. **Umsetzung** (Karte #219): Logik `internal/canvas/farbpaar.go` (`Farbfeld`, `AktivesFeld`, `SetzeFeld`, `Farbe`,
   `KeineFuellung`, `TauscheFarben`; `Stil.Fuellung`; das Paar des zuletzt gewählten Werkzeugs mit Farbe steht in
   `Stil`, die übrigen in der Sitzung). Eine Farbe am gewählten Objekt geht in das Paar des Werkzeugs, das diese Art
   zeichnet (ein umgefärbter Kasten färbt auch die nächsten Kästen); Tauschen am Objekt ändert das Paar nicht.
   Werkzeugleiste: das Quadrat unter den Werkzeugen ist geviertelt, Vordergrund oben links, Füllung unten rechts,
   ⇄ oben rechts, ohne Überlappung (`farbpaarRahmen`). Rechte Leiste: Zeile Vordergrund, Füllung, ⇄, „Keine“ über der
   Farbfläche (`zeichneFarbpaarZeile`).
9. **Umsetzung nach Kritik** (23.11.2026, Kritik zu #219): Ohne gewähltes Objekt gilt die Füllung nur beim gewählten
   Werkzeug mit Füllung (Kasten, Text). Bei einem Werkzeug ohne Farbe — Auswählen ohne Objekt, Schwärzen, Verpixeln,
   Zuschneiden, Stauchen — sind Feld Füllung, „Keine“ und Tauschen grau, obwohl `Stil` das Paar des zuletzt benutzten
   Farbwerkzeugs trägt; sonst tauschte Strg+Umschalt+X unsichtbar dessen Paar. Einzige Quelle dafür ist
   `Sitzung.Fuellbar` (das Feld `SeitenInhalt.Fuellung` entfällt). Der Farbton-Regler auf einer ganz durchsichtigen
   Farbe setzt Sättigung und Helligkeit voll, damit Punkt 7 einen sichtbaren Ton ergibt statt Schwarz (Füllung „keine“
   ist `{0,0,0,0}`). Eine ganz durchsichtige Farbe kommt wie „Keine“ nicht in die zuletzt benutzten Farben.

## Nachtrag 12 (23.11.2026): Hilfslinien und Einrasten (Karte #196)

**Status:** Akzeptiert (Claude Sonnet 5.5). Beschriftungen sollen sich ohne Augenmaß ausrichten lassen: an anderen
Objekten und am Bild.

1. **Woran gerastet wird.** Kanten und Mitte (je Achse drei Linien) der Grenzen (`Objekt.Grenzen`, wie der
   Auswahlrahmen) aller anderen Objekte, die nicht ausgeblendet sind und in die Ansicht reichen
   (`ObjektSichtbar`; gesperrte zählen mit, sie sind zu sehen), dazu die zwei Ränder und die Mitte des sichtbaren Bilds
   (Ansicht nach Zuschnitt und Stauchen, ins Original zurückgerechnet). Ausgeblendete Ebenen und Objekte ganz im
   Verborgenen sind keine Ziele. Das Objekt, das gezogen wird, ist selbst kein Ziel.
2. **Was einrastet.** Das Verschieben eines Objekts: seine drei Linien je Achse gegen die Ziele, eingerastet wird die
   kleinste Verschiebung je Achse. Das Ziehen eines Griffs (Pfeilende, Pfeilanfang, Kastenkanten und -ecken): der
   Griffpunkt je Achse; der Bogengriff des Pfeils rastet mit, er ist ein Punkt wie die anderen. Das Aufziehen eines
   neuen Pfeils oder Kastens: der Endpunkt; eine neue Nummer wie ein Verschieben. Nicht einrasten: der Anfangspunkt eines
   neuen Objekts (der Druck liegt dort, wo der Nutzer ihn setzt), Schwärzen, Verpixeln, Zuschneiden, Stauchen (Rahmen,
   keine Objekte; im Zuschneiden zeigt die Leinwand die Vollansicht) und Text (ein Klick setzt nur den Ort). Mit
   gedrückter Umschalttaste beim Aufziehen gilt 45° bzw. Quadrat (Nachtrag 1) und rastet nichts ein: Beides zugleich
   widerspräche sich.
3. **Fangweite** `canvas.FangWeite` = 6 Fensterpixel, unabhängig vom Zoom (im Original 6 / Zoom). Gerechnet wird im
   Original; bei gestauchten Bildern weicht die Weite an einer Naht geringfügig ab, das ist hinzunehmen. Die Fangweite
   selbst zählt noch dazu; bei Gleichstand gewinnt das zuerst genannte Ziel (Bild vor Objekten).
4. **Alt schaltet ab.** Solange Alt gedrückt ist, rastet nichts ein. Die Oberfläche setzt es je Zeigerereignis
   (`Sitzung.SetzeAlt`, wie `Umschalt`); eine Alt-Taste ohne Zeigerbewegung wirkt sofort (Ergänzung unten, Punkt c). Das
   ist eine Ausnahme nach ADR-0026 Punkt 5, wie Umschalt beim Ziehen: ein Zustand während des Ziehens, kein Kürzel;
   der Knopf dazu ist der Menüeintrag in Punkt 6. Die Übersicht (F1) nennt „Alt: Einrasten beim Ziehen aus“.
5. **Hilfslinie.** Dünn (1 dp), in der Akzentfarbe Orange (`farbeAkzent`, Corporate Design: Orange nur als Akzent),
   über die ganze Breite oder Höhe des sichtbaren Bilds, nur während des Zugs und nur für die Achsen, an denen
   wirklich eingerastet ist (beim Kantengriff nur die Achse, die der Griff bewegt; wird die Verschiebung danach durch
   die Sichtbarkeitsgrenze gekürzt, fällt die Linie weg). Sie erscheint nie im Bild, in der Datei oder im Export; sie
   liegt nur in der Leinwand (`Sitzung.Hilfslinien`, `internal/ui/hilfslinien.go`).
6. **Schalter.** Menü Ansicht, Eintrag „Snap to objects“ („An Objekten einrasten“), Voreinstellung an, nicht
   gespeichert (kein Eintrag in `settings.ini`; ein Zustand der Sitzung wie die Seitenleiste). Haken wie bei „Sidebar“.
   Kein Kürzel.
7. **Rückgängig.** Das Einrasten ändert nur, wohin der Zug zeigt: Ein Zug bleibt ein Schritt im Verlauf, Esc bricht
   ihn ab, Rückgängig stellt den Stand davor her. Die Sichtbarkeitsgrenze (ADR-0021 Nachtrag 2) gilt weiter und geht dem
   Einrasten vor.
8. **Umsetzung.** Logik `internal/canvas/einrasten.go` (`rastZiele`, `rasteAchse`, `rasteRechteck`, `rastePunkt`,
   `Hilfslinie`), Einhängen in `Sitzung.Ziehen` und `zieheNeu`; Tests `einrasten_test.go` (canvas und ui), Mutanten
   `scripts/mutanten/einrasten.json`, Sichtprobe `scripts/drehbuecher/einrasten-196.txt`.

**Ergänzung aus der Kritik (23.11.2026, Claude Fable 5.2, Karte #196).** Sie stellt die Punkte 1, 4 und 5 genauer:

- a. **Nur sichtbare Linien sind Ziele (zu Punkt 1).** Von einem Objekt, das nur zum Teil zu sehen ist, zählen die
  Linien nicht, die außerhalb des Zuschnitts oder in einem herausgeschnittenen Streifen liegen (`linieSichtbar`; die
  Kanten eines Streifens selbst sind zu sehen und zählen). Sonst stünde die Hilfslinie an der Naht oder neben dem
  Bild, und das Ende eines neuen Pfeils oder Kastens läge im Verborgenen (gegen F-150 und F-157).
- b. **Mitte des Bilds (zu Punkt 1).** Gemeint ist die Mitte der Ansicht, ins Original zurückgerechnet, nicht die
  Mitte zwischen den Rändern des Originals: Bei gestauchten Bildern fallen beide auseinander, die zweite kann in
  einem Streifen liegen.
- c. **Alt ohne Zeigerbewegung (zu Punkt 4).** Die Alt-Taste kommt zusätzlich als Tastenereignis an (gedrückt und
  losgelassen, `Sitzung.AltTaste`): Ein laufender Zug wird sofort am letzten Zeigerort neu gerechnet, das Einrasten
  schaltet also ohne die nächste Zeigerbewegung ab und wieder an; es bleibt ein Schritt im Verlauf. Die Taste steht
  dafür mit ihrer Begründung als Ausnahme in `tastenTabelle` (ADR-0026 Punkte 5 und 6), ohne eigene Handlung. Liefert
  die Tastaturbelegung für die Taste ein anderes Zeichen als Alt (etwa Umschalt+Alt als Meta), kommt kein
  Tastenereignis; dann gilt weiter der Stand des nächsten Zeigerereignisses.
- d. **Annahme zu KDE (zu Punkt 4).** Unter Plasma 5 verschob Alt+Ziehen in der Voreinstellung das Fenster; unter
  Plasma 6 ist die Voreinstellung der Modifikatortaste für Fensteraktionen Meta. Angenommen wird: Mit der
  Voreinstellung von Plasma 6 erreicht Alt beim Ziehen Schnapp. Wer die KWin-Einstellung auf Alt stellt (oder eine
  Umgebung nutzt, die Alt+Ziehen selbst verbraucht), bekommt den Zug nicht in Schnapp und schaltet das Einrasten
  über den Menüeintrag „Snap to objects“ (Punkt 6) ab. Gegen eine echte Sitzung ist das noch nicht geprüft; eine
  zweite Taste wird erst erwogen, wenn sich die Annahme als falsch erweist.
- e. **Kasten über die Gegenseite gezogen (zu Punkt 5).** Beim Kasten zählt für die Hilfslinie je bewegter Achse
  (`griffAchsen`) jede der beiden Kanten des Ergebnisses, nicht der Ort des Griffs: Klappt der Kasten um, liegt die
  gezogene Kante weiter auf der Linie, der Griffname gehört aber zur anderen Seite.
- f. **Nicht erreichbar:** Die Abfrage der Vollansicht des Zuschneidens in `einrastenAktiv` ist eine Absicherung; in
  diesem Werkzeug gibt es weder Objektzug noch Objektgriffe noch neue Objekte.
- g. **Umsetzung und Tests:** `einrasten_kritik_test.go` (canvas und ui), `hilfslinienFlaechen` (Lage der Linie im
  Fenster, ohne Gio prüfbar), Mutanten E19 bis E35.

## Nachtrag 13 (24.11.2026): Sprechblase — Text-Schild mit Spitze (Karte #194)

**Status:** Akzeptiert (Claude Opus 5.5 im Auftrag Andis: „arbeite selbstständig“, 24.11.2026). Grundlage: Wunsch
Andis vom 22.11.2026 (Ideenliste, Idee 5), Anforderung A-21 („Sprechblasen“). Baut auf Nachtrag 3 (Schild-Maße),
Nachtrag 4 (eckig), Nachtrag 10 (Schrift) und Nachtrag 11 (Farbpaar) auf.

1. **Kein eigenes Objekt:** Eine Sprechblase ist ein `document.Text` mit Spitze. `Text` bekommt das Feld
   `Spitze *Punkt` (Originalkoordinaten; `nil` = keine Spitze, also der bisherige Text). Schrift, Stil, Ausrichtung,
   Farbpaar, Ebene, Rückgängig, Treffer und Texteingabe gelten unverändert. So gibt es keine zweite Textmessung und
   keine zweite Satzregel.
2. **Form (eine Rechnung für alle):** `document.Sprechblase(t Text) []Punkt` liefert den Umriss als ein Vieleck: das
   eckige Schild (`Text.Grenzen`, Nachtrag 3) mit einem eingesetzten Dreieck zur Spitze. Die Basis des Dreiecks liegt
   auf der Schildseite, die der Strahl von der Schildmitte zur Spitze schneidet; ihre Mitte ist dieser Schnittpunkt,
   so verschoben, dass die Basis ganz auf der Seite liegt und von jeder Ecke mindestens 0,25 Schriftgrad Abstand hält.
   Basisbreite: 1 Schriftgrad, höchstens die halbe Länge dieser Seite abzüglich der Eckabstände. Liegt die Spitze im
   Schild oder näher als 0,25 Schriftgrad an ihm, entfällt das Dreieck (Umriss = Schild). Renderer, SVG-Export
   (Pfad mit `stroke-linejoin="miter"`, kein Strich), Editor, Treffer und Einrasten nutzen nur diese Funktion.
3. **Farben:** Die Spitze trägt die **Füllung** (Schild-Farbe, Nachtrag 11). Ist die Füllung „keine“, zeichnen alle
   Ausgänge die beiden Schenkel des Dreiecks als Linie in der Schriftfarbe, Breite `max(1, 0,06 × Schriftgrad)` (wie
   Unterstreichen, Nachtrag 10), damit die Blase auch ohne Schild auf ihre Stelle zeigt.
4. **Grenzen:** `Text.Grenzen` bleibt das Schild (Textmessung, Texteingabe, Ausrichtung). Neu `Text.Huelle()` =
   Schild plus Spitze; sie gilt für Auswahlrahmen, Einrasten (Ziele: Kanten und Mitten des Schilds, die Spitze als
   Punkt), Budget (`render.Pruefe`) und Ebenen-Zwischenbild (`ebenenRahmen`). Treffer: Punkt im Vieleck aus Punkt 2.
5. **Werkzeug Sprechblase:** eigener Knopf in der Werkzeugleiste unter Text, Kürzel **K** (frei; vorher prüfen, dass
   kein Eintrag der Tastentabelle und kein Menü-Kürzel K allein belegt), Symbol im eckigen Satz (`internal/icons`).
   Bedienung: **Aufziehen** — der Druckpunkt ist die Spitze (die Stelle, auf die gezeigt wird), das Loslassen setzt den
   Text dort (Ort = linke obere Ecke wie beim Text-Werkzeug), dann Texteingabe. **Klick** ohne Zug setzt den Text an die
   Klickstelle mit Spitze 3 Schriftgrade links unter der linken unteren Ecke des Schilds. Leerer Text am Ende der
   Eingabe verwirft das Objekt wie beim Text-Werkzeug. Das Farbpaar der Sprechblase ist das des Text-Werkzeugs
   (ein gemeinsames Paar „Text“, Nachtrag 11 Punkt 2).
6. **Bearbeiten:** Verschieben bewegt Schild und Spitze gemeinsam. Ein Griff an der Spitze verschiebt nur die Spitze;
   er rastet ein wie andere Griffe (Nachtrag 12). Die rechte Leiste zeigt bei gewähltem Text einen Schalter „Spitze“
   (an: Spitze an der Stelle aus Punkt 5 „Klick“; aus: `Spitze = nil`), je ein Rückgängig-Schritt. Doppelklick in die
   Blase bearbeitet den Text wie bisher.
7. **Format:** Attribut `s:spitze` (Punkt, 5.2) am Objekt `"text"`, OPTIONAL. Die Spitze gehört zu **Format-Version
   2** (wie die Lupe, ADR-0033): Version 2 ist noch nicht ausgeliefert (v0.5.0 schreibt 1), deshalb keine neue
   Nummer. Ein Schreiber schreibt Version 2, sobald eine Lupe **oder** eine Spitze vorkommt, sonst 1. In einer Datei
   der Version 1 ist `s:spitze` unbekannter Inhalt (§11 Regel 3, ignoriert, kein Schaden); so verliert ein Leser der
   Version 1 die Spitze nie still beim Speichern, weil er Version 2 nur als Bild öffnet. Die Darstellung im SVG-Block
   zeigt die Blase als Pfad; Beispiel `testdata/format/callout`, `docs/format.md` §8.3 und die Versionsliste im selben
   Commit wie `FormatVersion` (ADR-0008 Nachtrag 3).
8. **Agenten:** `--annotate` und das MCP-Werkzeug `annotate` bekommen beim Typ `text` den optionalen Schlüssel
   `tip: [x, y]` (Koordinaten im sichtbaren Bild wie `at`); `docs/agents.md` nennt ihn.
9. **Nicht in diesem Schritt:** runde oder ovale Blasen, gebogene Spitzen, mehrere Spitzen, Gedankenblasen. Eckig
   bleibt Vorgabe (Nachtrag 4, `brand/corporate-design.md` §1).

**Ergänzt bei der Umsetzung (Teil 1, 24.11.2026, ohne Oberfläche):**

- **Basisbreite (Punkt 2):** „höchstens die halbe Länge dieser Seite abzüglich der Eckabstände“ ist als
  `min(1 Schriftgrad, (Seitenlänge − 2 × 0,25 Schriftgrad) / 2)` umgesetzt; ist das nicht positiv (Seite höchstens zwei
  Eckabstände lang), entfällt das Dreieck wie bei zu naher Spitze. Der Umriss läuft im Uhrzeigersinn (y nach unten),
  das Dreieck steht an der Seite des Strahls (`document.Sprechblase`, `Text.Spitzendreieck`).
- **Spitze und Grenzen (Punkt 1):** `Text.pruefe` verlangt für die Spitze nur endliche Koordinaten im Bereich
  ±`MaxKoordinate` wie für `Ort`; außerhalb des Bilds ist erlaubt (wie ein Pfeil über den Rand). Die Spitze zählt in
  `Huelle` und `Reichweite` auch dann, wenn das Dreieck entfällt.
- **Schenkel (Punkt 3):** Der Renderer zeichnet sie mit dem Strichzeichner des Pfeils (runde Enden und Ecken), der
  SVG-Pfad dazu mit `stroke-linecap`/`stroke-linejoin="round"`; der gefüllte Pfad trägt `stroke-linejoin="miter"` ohne
  Strich.
- **Format (Punkt 7):** `s:spitze` wird nur in Dateien ab Version 2 gelesen (`leser.version`); ein nicht lesbarer Wert
  macht eine Version-2-Datei beschädigt wie jeder andere Punkt, in Version 1 bleibt er unbeachtet.
- **Rettungskopie:** keine Änderung nötig, ihre Angaben sind das snAP-SVG (`AngabenSVG`) und tragen die Spitze mit
  (`internal/rescue/spitze_test.go`).

**Ergänzt bei der Umsetzung (Teil 2, 24.11.2026, Oberfläche):**

- **Farbpaar und Art (Punkte 5 und 6):** Die Sprechblase hat ein eigenes Werkzeug (`WerkzeugSprechblase`), aber kein eigenes
  Farbpaar: `paarSchluessel` bildet sie auf das Paar „Text“ ab. Ein Text mit Spitze gilt für Leiste, Ebenenliste und
  Schnellstile als Art Sprechblase (`objektWerkzeug`); die eingebauten Schnellstile der Sprechblase sind die des Texts.
- **Aufziehen (Punkt 5):** Schon während des Ziehens steht der Text (leer) im Zug des Dokuments, damit der Editor die Blase
  mit der wandernden Spitze-und-Schild-Lage zeigt; beim Loslassen geht derselbe Zug in die Texteingabe über. Spitze und
  Schild rasten dabei ein (Nachtrag 12). Ein Klick ohne Zug legt den Text erst beim Loslassen an. Die Standard-Spitze
  (Klick und Schalter) wird ins sichtbare Bild begrenzt (`StandardSpitze`), ebenso die Spitze beim Ziehen am Griff;
  eine Spitze außerhalb des Bilds bleibt für geladene Dateien erlaubt (Teil 1).
- **Griff (Punkt 6):** Ein Text mit Spitze hat genau einen Griff, `GriffSpitze` an der Spitze; die Auswahl zeigt den Rahmen der
  Hülle (`Text.Huelle`). Treffer: Schild mit Toleranz, Dreieck (`Text.Enthaelt`) und die Spitze selbst mit Toleranz.
- **Schalter (Punkt 6):** „Spitze“/„Tail“ in der rechten Leiste unter der Schrift, nur bei einem gewählten oder gerade
  eingegebenen Text; in der Eingabe gehört er zur Eingabe, sonst ein Rückgängig-Schritt.
- **Werkzeugleiste:** Mit elf Werkzeugen und dem Farbfeld passt die Leiste in 480 × 320 nur, wenn die Knöpfe bis auf 18 dp
  schrumpfen (vorher 20 dp, ADR-0031 Nachtrag 3, `werkzeugKnopfMinDp`).

**Ergänzt (F-634, 24.11.2026):** Die Standard-Spitze (Klick und Schalter „Spitze“) versucht nach dem Begrenzen auf das Bild die Ecken der Reihe nach — links unten, rechts unten, links oben, rechts oben, jeweils 3 Schriftgrade vom Schild weg —, bis `Text.Spitzendreieck` ein Dreieck liefert. Sitzt das Schild in einer Bildecke, fällt die Spitze sonst ins Schild oder an dessen Rand und die Blase bliebe ohne Dreieck. Geht keine Ecke, bleibt die erste.
