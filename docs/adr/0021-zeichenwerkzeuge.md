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
