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
   Dunkelgrau `#1E2124` (grau-900). Strichstärke Pfeil und Kasten 4 px (Spitze damit 16 px; *seit Nachtrag 16:* 6 px), Kasten ungefüllt mit
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
   (`canvas.StilFuer`): Full HD und kleiner 4 / 20 / 14 px, 4K 8 / 40 / 28 px, 8K 16 / 80 / 56 px (*seit Nachtrag 16:* Strich 6 / 12 / 24 px). Ganzzahlig, damit
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
  Voreinstellung folgt dem Bild, die Stufen der Bedienung. *Seit Nachtrag 16 (Strich 6 px)* gelten andere Faktoren,
  siehe dort.
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
   (`canvas.ZuletztGrenze`; *seit Nachtrag 17:* vier); dieselbe Farbe rückt nach vorn statt doppelt zu stehen. Die Liste gilt je Fenster und wird
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
   256 MiB zugleich im Speicher (Nachtrag 15: darüber verdrängt die am längsten unbenutzte Datei, F-697; passt eine Datei auch dann nicht, Go; die Grenze gilt nicht als Fehler der Datei), die Suche bricht nach 10 s ab und
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
- g. **Umsetzung und Tests:** `einrasten_grenzfaelle_test.go` (canvas und ui), `hilfslinienFlaechen` (Lage der Linie im
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
  `Huelle` und `Reichweite` auch dann, wenn das Dreieck entfällt (für den SVG-Export eingeschränkt durch Nachtrag 14).
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

## Nachtrag 14 (24.11.2026): Sprechblase im SVG-Export nur mit sichtbarer Schrift (F-692, F-694, Karte #265)

**Status:** Akzeptiert (Claude Opus 5.5 im Auftrag Andis: „arbeite selbstständig“, 24.11.2026). Anlass: Abschluss-Kritik
Sprint 12, A-1 und A-3. Schränkt den Satz aus Nachtrag 13 („Die Spitze zählt in `Huelle` und `Reichweite` auch dann,
wenn das Dreieck entfällt“) für den SVG-Export ein; ADR-0023 Punkt 7 und ADR-0008 Nachtrag 2 (F-66, F-181) gehen vor.

1. **Reichweite bleibt:** `Reichweite` und `ObjektSichtbar` rechnen weiter mit der Hülle samt Spitze. Sie entscheiden,
   ob Editor, Renderer und Treffer ein Objekt überhaupt ansehen; der Griff an der Spitze bleibt so auch erreichbar,
   wenn das Schild weggeschnitten ist. *(Berichtigt, F-735:)* Der frühere Satz „Gezeichnet wird ohnehin nur, was in der
   Ansicht liegt“ stimmte für Stauch-Streifen nicht: Text wird starr ab seinem Anker gesetzt, ein Anker im Streifen
   fällt auf die Naht, und die Zeilen standen dahinter lesbar im Bild. Punkt 4 behebt das, wenn Schrift und Schild ganz verborgen sind; ragt ein Teil heraus, steht der Text weiter ganz hinter der Naht (Grenze, ADR-0023 Nachtrag 12, F-739).
2. **Export getrennt nach Teilen:** Für den SVG-Export zählt eine Sprechblase in zwei Teilen
   (`Dokument.TextSichtbar`): **Schrift** = Reichweite ohne Spitze (Tinte der Glyphen, mit Füllung das Schild) und
   **Dreieck** = Rechteck um `Spitzendreieck` samt halber Schenkelbreite und 1 Pixel (kein Dreieck, wenn es entfällt).
   Ist die Schrift sichtbar, schreibt der Export die Blase wie bisher ganz. Ist nur das Dreieck sichtbar, schreibt er
   nur das Dreieck (in der Füllfarbe, **ohne** den Umriss des Schilds, dessen Breite und Höhe sonst die ungefähre Länge und Zeilenzahl des Texts verrieten, F-728) bzw. die Schenkel, **ohne** `<text>` und ohne Unter- und Durchstreichung.
   Ist keins von beiden sichtbar, fehlt das Objekt im Export. Ein Text, dessen Glyphen und Schild nirgends zu sehen
   sind, steht so nie im Klartext in einer exportierten Datei. Die Arbeitsdatei (snAP) trägt weiter alles
   (nicht zerstörend, Kernprinzip 1).
3. **Wertvergleich der Objekte (F-694):** Weil `Text.Spitze` ein Zeiger ist und jede Änderung einen neuen setzt,
   vergleicht `document.Gleich` Objekte dem Wert nach (Spitze dereferenziert). `BeendeZug` und die Prüfungen „nichts
   geändert“ in `canvas` nutzen es; ein Zug ohne Wirkung an einer Sprechblase hinterlässt so keinen Rückgängig-Schritt.
4. **Renderer und Editor wie der Export (F-735, Verify-Lauf V2-1):** `Dokument.NurSpitzeSichtbar(t)` (Spitze gesetzt,
   Schrift laut `TextSichtbar` unsichtbar) gilt für alle Zeichenwege. Renderer (`render.Bild`, also PNG, JPG,
   Zwischenablage, Vorschau, Texterkennung) und Editor zeichnen dann nur das Dreieck (gefüllt) bzw. die Schenkel,
   keine Zeile, keinen Strich und kein Schild. Die Vorschau der Ebenenliste zeigt das Objekt weiter ganz (sie bildet
   das Original ohne Ansicht ab).
5. **Basis des Dreiecks im SVG (F-736, bewusst):** Das Dreieck steht im SVG in Originalkoordinaten, auch wenn seine
   Basis außerhalb der Ansicht liegt. Daraus lassen sich die Lage einer Schildkante und die Basisbreite (meist der
   Schriftgrad) ablesen, nicht aber Breite, Höhe oder Inhalt des Schilds. Das gleicht dem Bestand (teils sichtbare
   Pfeile und Kästen stehen ganz im SVG, ADR-0023 Punkt 7) und bleibt so; ein Beschnitt auf die sichtbaren Kacheln
   wäre eigener Code nur für diesen Fall. Ausnahmepfad [S] P3 (docs/qa/sprint-abschluss.md).

## Nachtrag 15 (24.11.2026): Schriftgrenze ist ein Zwischenspeicher mit Verdrängung (F-697, Karte #276)

Ändert die Grenze aus Nachtrag 10 Punkt 3 („höchstens 32 geladene Schriftdateien bzw. 256 MiB“) in ihrer Wirkung, nicht
in ihrer Höhe. Anlass: Seit dem Wechsel der Aufnahme im selben Fenster (ADR-0036) lebt der Prozess über viele Dokumente;
die lebenslange Sperre ließ später geöffnete Dokumente ihre Schrift als fehlend (Ersatz Go) zeigen.

1. **Verdrängen statt Sperren:** Ist die Grenze erreicht, wirft `fonts` die am längsten unbenutzten Schriftdateien aus
   dem Speicher (`verdraenge`, Takt je Benutzung), bis die neue passt, samt ihren geladenen Schnitten. Die Obergrenze
   32 Dateien/256 MiB gilt weiter für die zugleich gehaltenen Dateien. Das gilt bei jedem Laden, auch aus der
   Hintergrundsuche und aus Renderer und Export (alles unter `mu`).
2. **Gehaltene Schriften bleiben gültig:** Wer eine `*Schrift` schon hat (laufender Renderer, Texteditor), behält sie
   benutzbar; sie fällt nur aus der Tabelle, der Garbage Collector gibt sie frei, wenn niemand mehr sie hält. Die
   Merker je `*sfnt.Font` (`document.schriftenMass`, Faces und Umrisse in `internal/ui/textsatz.go`) halten verdrängte
   Fassungen fest; sie fangen deshalb über 2 × 32 Schriften leer neu an (Sitzungs-Kritik B-1).
3. **Restfall:** Eine einzelne Datei über 256 MiB oder eine Datei, die auch ohne alle anderen nicht passt, bleibt
   `errGrenze` (Ersatz Go, nicht als Dateifehler gemerkt). **Ruhefrist (Sitzungs-Kritik B-1):** Eine Datei, die vor
   weniger als 2 s benutzt wurde, wird nicht verdrängt (`ruhefrist`). Braucht ein Bild oder ein Rendererlauf mehr als
   32 Dateien zugleich, bekommt die überzählige Ersatz Go wie vor F-697; sonst lud jedes Bild alle reihum neu und legte
   je Bild neue Fassungen an. Über längere Zeit (Wechsel der Aufnahme) wird weiter verdrängt.
4. Tests: `internal/fonts/verdraengung_test.go` (33 und mehr Dateien in Folge, älteste zuerst, nebenläufig unter `-race`,
   Ruhefrist `TestRuhefristVerdraengtNichtImGebrauch`).

## Nachtrag 16 (25.11.2026): Voreinstellung der Strichstärke 6 statt 4 (Karte #316)

**Status:** Akzeptiert (Claude Opus 5.5). Ändert die Strichstärke aus Punkt 4 und Nachtrag 7 sowie die Stufen-Zahlen aus Nachtrag 8.

Andi (25.11.2026): „die defaulteinstellungen für die strichstärke von pfeilen, umrandungen usw könnte einen tick stärker
sein.“ Die Voreinstellung (`canvas.StandardStil`) steigt um eine Stufe von `StrichStufen` auf **6 px** bei Faktor 1, also
Full HD 6, 4K 12, 8K 24 (Nachtrag 7, mal `KantenFaktor`). Sie gilt für Pfeil, Kasten, Lupe und die Schenkel der
Sprechblase ohne Füllung. Die eingebauten Schnellstile (ADR-0037) ziehen mit: normal 6, „dick“ 12 statt 8, damit der
Stand eines frischen Werkzeugs weiter zur ersten Kachel passt; fehlt in `styles.ini` die Breite, gilt ebenfalls 6
(`settings.stilBreite`). Schriftgrad 20 und Nummernradius 14 bleiben. Gespeicherte Dateien und eigene Schnellstile behalten
ihre Breite. Die Voreinstellung trifft die Stufen von `StrichStufen` damit nur noch bei Faktor 1, 2 und 4 (6, 12,
24 px); Faktor 3 (18) und 5 (30) liegen zwischen zwei Stufen, ab Faktor 6 liegt sie über 32 px (ersetzt die Zahlen in
Nachtrag 8, Punkt „Stufen“).

## Nachtrag 17 (25.11.2026): Hex ohne Deckkraft, Pipette im Farbpaar (Karte #318)

**Status:** Akzeptiert (Claude Opus 5.5). Ändert aus Nachtrag 9 und ADR-0031 Nachtrag 5 die Anzeige des Hex-Felds und den
Platz der Pipette.

Andi (25.11.2026): „wenn ich die transparenz ändere, ändert sich der hex-farbwert – das verwirrt mich … das
pipettensymbol sieht nicht nach einer pipette aus und ist auch deplatziert.“

1. **Hex zeigt immer `#RRGGBB`** (`canvas.HexText`), auch unter 100 % Deckkraft; die Deckkraft steht nur im Feld daneben
   und in der Fußzeile („#D93A33 · 74 %“). Sechs eingegebene Ziffern lassen die Deckkraft stehen, **acht Ziffern**
   (`#RRGGBBAA`) werden weiter angenommen und setzen beide (`LeseHex` unverändert).
2. **Pipette** mit eigenem Symbol `icons.Pipette` („eyedropper“: Druckkappe, Kragen, Röhrchen, Spitze mit Tropfen in
   Akzentfarbe, schräg wie das Werkzeug in der Hand). Sie steht **rechts in der Zeile des Farbpaars** — sie holt eine Farbe
   für das aktive Feld und gehört dorthin —, nicht mehr als letztes Feld im Raster. Tab-Reihenfolge: Farbpaar, „Keine“,
   Pipette, Farbfelder, Hex, Deckkraft.
3. Das frei gewordene Feld im Raster nimmt eine **vierte zuletzt benutzte Farbe** (`canvas.ZuletztGrenze` 4, Raster 2 × 4).

Tests `internal/ui/farbwaehler_hex6_test.go`, Mutanten `scripts/mutanten/farbwaehler-318.json`, Sichtprobe
`scripts/drehbuecher/farbwaehler-318.txt`.

## Nachtrag 18 (25.11.2026): Mauszeiger je Lage (Karte #323)

**Status:** Akzeptiert (Claude Opus 5.5).

Andi (25.11.2026): „wenn ich ein objekt mit dem auswahlwerkzeug markiere, sollte sich der mauszeiger sinnvoll anpassen,
wenn ich auf einen der punkte (ecke) oder so komme … das sollte überall sinnvoll eingebaut werden.“

Die Form entscheidet `canvas.Sitzung.Zeiger(p)` (`internal/canvas/zeiger.go`) nach denselben Regeln wie `Druecken`, die
Oberfläche bildet sie über `ui.systemZeiger` auf die Zeiger des Systems ab:

| Lage | Form | System |
|---|---|---|
| Ecke links oben/rechts unten (Kasten, Quelle und Ziel der Lupe, Zuschnittrahmen) | `ZeigerNWSO` | ↖↘ |
| Ecke rechts oben/links unten | `ZeigerNOSW` | ↗↙ |
| Kante oben/unten bzw. links/rechts | `ZeigerNS` / `ZeigerOW` | ↕ / ↔ |
| Pfeilende, Bogengriff, Spitze der Sprechblase | `ZeigerGreifen` | Hand |
| über einem greifbaren Objekt (Auswählen) und beim Verschieben | `ZeigerVerschieben` | vier Pfeile |
| Ansicht ziehen (nach der Ziehschwelle) | `ZeigerGreifend` | greifende Hand |
| Zeichenwerkzeuge, Pipette, Bereich „Text aus dem Bild“ | `ZeigerFadenkreuz` | Fadenkreuz |
| Text, Sprechblase | `ZeigerText` | Textcursor |

Griffe gelten wie beim Drücken nur an einer bearbeitbaren Auswahl und nicht beim Zuschneiden/Stauchen; ein laufender Zug
behält seine Form, auch wenn der Zeiger den Griff verlässt. Ändert sich die Form beim Bewegen, folgt ein Bild, auch wenn
das Pixel unter dem Zeiger gleich bleibt (starker Zoom). Der Drehgriff kommt mit Karte #324. Tests
`internal/canvas/zeiger_test.go`, `internal/ui/zeiger_form_test.go`, Mutanten `scripts/mutanten/zeiger-form.json`.
Im Echtlauf vom 25.11.2026 wechselte der Zeiger in der Plasma-Sitzung nicht (F-818, Karte #342). Ursachen (Kritiker
Fable 5.2): (1) Knöpfe setzten `pointer.CursorPointer` nach `Clickable.Layout`, also außerhalb ihres Ausschnitts —
die Hand galt für den umgebenden Bereich und lag über der Leinwand; jetzt `ui.handZeiger` mit eigenem Ausschnitt,
belegt über den Gio-Router (`TestZeigerFormErreichtDenRouter`, `TestZeigerHandUeberWerkzeugknopf`). (2) Breeze kennt
einige X11-Namen nicht: Ausweichliste im Gio-Patch (ADR-0015 Nachtrag 5). (3) Während Speichern oder Laden zeigt der
Zeiger keine Griffe (`gesperrt`, Kritik K-5); die Form wird je Bewegung einmal gerechnet (`editor.zeigerForm`).
Ob er jetzt in der echten Sitzung wechselt, zeigt Andis nächster Echtlauf.

## Nachtrag 19 (26.11.2026): Pfeilspitze neu, Spitzenarten und Enden (Karte #315)

*Punkt 1 (Kopf/Enden), Punkt 4 (Format, FormatVersion 1), Punkt 5 (Bedienung), Punkt 7 (`head`/`ends`) und das „Verworfen“ zu Spitzen je Ende sind durch Nachtrag 20 abgelöst; ebenso das „Verworfen“ zu FormatVersion 2 (jetzt gewählt, Nachtrag 20 Punkt 4). Die Sichtprobe `pfeilform-315.txt` ist entfernt (d256d16), es gilt `pfeilform-351.txt`.*

**Status:** Akzeptiert (Claude Opus 5.5).

Andi (25.11.2026, Foto an der Karte): Die Spitze wirkt schmal, der Strich läuft sichtbar in sie hinein. Gewünscht:
breiter und kürzer im Verhältnis zur Strichstärke, der Strich endet unter der Spitze; Spitzenarten wählbar (gefülltes
Dreieck, offene Spitze, Punkt, Strich quer); Enden keine (= Linie), Ende, Anfang, beide.

1. **Zwei neue Felder am Pfeil** (`document.Pfeil`): `Kopf` (`KopfDreieck`, `KopfWinkel`, `KopfPunkt`, `KopfQuer`) und
   `Enden` (`EndenEnde` = Spitze bei B, `EndenAnfang` bei A, `EndenBeide`, `EndenKeine` = Linie). Beide Nullwerte sind
   das bisherige Verhalten (Dreieck bei B), alte Dateien und Objekte bleiben Pfeile. Eine Spitzenart gilt für beide
   Enden. Eine Linie bleibt das Objekt Pfeil (Griffe, Bogen, Einrasten wie bisher), kein neues Werkzeug.
2. **Maße** (w = Strichstärke, alles in Pixeln des Originals; eine Rechnung `document.Pfeil.Form` für Renderer, Editor,
   SVG-Export, Treffer und Reichweite):
   - Dreieck: Länge `3w + 5`, Breite `4,5w + 6` (vorher Länge = Breite = `max(8, 4w)`): breiter und etwas kürzer,
     Öffnungswinkel rund 70°.
   - Winkel (offene Spitze): dasselbe Dreieck als Umriss, Schenkel w dick, hinten gerade abgeschnitten.
   - Punkt: Scheibe vom Radius `1,5w + 3` um das Ende.
   - Quer: Balken w dick, `5w + 6` lang, mittig auf dem Ende, senkrecht zur Richtung.
   Die Richtung ist die Sehne vom Ende des Schafts zum Ende des Pfeils; der Schaftpunkt liegt auf der Kurve. Bei einem
   gebogenen Pfeil sitzt die Spitze so auf der Kurve, nicht auf der Tangente am Ende. Der Umriss des Kopfs schließt an
   die Normale der letzten Schaftstrecke an (`Kopfform.SchaftNormale`); bei starker Krümmung und großem Kopf weicht sie
   von der Sehne ab, die Fuge bleibt trotzdem dicht. Ohne Schaft bekommt jeder Kopf ein gedachtes Schaftende auf der
   Sehne und so seine volle Form. Bleibt eine Kurve überall näher am Ende als der Kopf lang ist, ist aber länger
   (Haarnadel, Schleife), endet der Schaft nach Bogenlänge (Sitzungs-Kritik 26.11. K-4); der Kopfumriss kann dann eine
   Kerbe zum Schaft hin haben (bewusste Grenze). Der Rahmen einer Ebene mit Deckkraft unter 100 % fasst die ganze
   Gestalt (`render.ebenenRahmen` über `Form().Huelle`, K-5).
3. **Der Schaft endet unter der Spitze** (`PfeilForm.Schaft`, gekürzte Kurve). Der Kopf schließt den Schaft genau an
   seinem Ende ab: Der Renderer ersetzt die runde Kappe des Linienstrichs durch den Umriss des Kopfs (gemeinsame Kante aus
   denselben Punkten wie die Kappe, ADR-0023 Nachtrag 22). So überlappen Schaft und Kopf nicht, auch eine halbdurchsichtige
   Farbe deckt gleichmäßig. Der Editor füllt dieselben Teile (`render.ZerlegePfeil`, `fuelleTeile` in `internal/ui/editor.go`); Gios Strich hätte nur runde Enden.
   Ist der Pfeil kürzer als seine Köpfe, entfällt der Schaft, die Köpfe bleiben.
4. **SVG-Export:** Linie ohne Kopf wie bisher als Strich mit runden Enden; mit Kopf der Schaft als Strich mit geraden
   Enden, je Kopf ein gefülltes Vieleck, an einem offenen Ende eine halbe Scheibe. Format: freiwillige Attribute
   `s:kopf` (`open`, `dot`, `bar`; fehlt bei `triangle`) und `s:enden` (`start`, `both`, `none`; fehlt bei `end`) am
   `<g s:typ="pfeil">`, Werte englisch wie bei `s:kante` und `s:stil`, dieselben Namen wie bei `--annotate`. **FormatVersion bleibt 1:** Ein älterer Leser zeigt einen gewöhnlichen Pfeil, das ist kein
   Schaden (wie `s:kante`). Unbekannte Werte gelten als Voreinstellung. Beispiel `testdata/format/arrow-heads`.
5. **Bedienung:** Die rechte Leiste zeigt beim Werkzeug Pfeil und bei einem gewählten Pfeil zwei Zeilen „Head“ (vier
   Symbolknöpfe) und „Ends“ (vier Symbolknöpfe), Name im Hinweis der Statuszeile. Ohne Auswahl gilt die Wahl für
   folgende Pfeile (`canvas.Stil.Kopf`/`Enden`), mit Auswahl ändert sie den Pfeil als einen Rückgängig-Schritt
   (`SetzeKopf`, `SetzeEnden`) und merkt sie fürs Werkzeug, wie die Strichstärke. Jeder Knopf zeigt einen kleinen Pfeil
   aus derselben Zerlegung wie die Leinwand (`internal/ui/pfeilform.go`).
6. **Schnellstile** tragen weiter nur Farbe und Strichstärke: Spitzenart und Enden sind Form, nicht Stil. Ein Schnellstil
   macht aus einer Linie keinen Pfeil.
7. **`--annotate` und MCP:** beim Typ `arrow` die freiwilligen Schlüssel `head` (`triangle`, `open`, `dot`, `bar`) und
   `ends` (`end`, `start`, `both`, `none`).
8. **Treffer** folgen der Gestalt: Schaft mit halber Strichstärke, Köpfe als Fläche (`canvas.trifft`); vorher galt ein
   Kreis um B, der auch vor der Spitze traf. Tests `internal/document/pfeilform_test.go`, `internal/render/pfeil_test.go`
   (halbdurchsichtig: kein Pixel doppelt gedeckt), `internal/canvas/pfeilform_test.go`, `internal/ui/pfeilform_test.go`,
   `internal/fileformat/pfeilform_test.go`; Mutanten `scripts/mutanten/pfeilform.json`; Sichtprobe
   `scripts/drehbuecher/pfeilform-315.txt`.

Verworfen: je Ende eine eigene Spitzenart (mehr Knöpfe, selten gebraucht); ein eigenes Werkzeug „Linie“ (die Wahl
„keine“ ist kürzer und behält alles andere); FormatVersion 2 (würde alte Leser ohne Not aussperren).

## Nachtrag 20 (26.11.2026): Pfeil-Stil je Ende, mehr Spitzenarten, Klappfelder (Karte #351, F-830)

**Status:** Akzeptiert (Andi, 26.11.2026 nach dem Echtlauf: „pfeilspitzen würde ich eher als Pfeil-Stil bezeichnen -
und können wir da noch ein paar cool aussehende arten einfügen? wäre Pfeil-Stil und Pfeil-Ende als Klappmenu
besser/flexibler (würde das überhaupt gehen mit dem Vorschaubild)?“; Ausgestaltung Claude Opus 5.5, Andi: „arbeite
selbstständig“).

1. **Je Ende eine Spitzenart** (ersetzt `Kopf` und `Enden`): `document.Pfeil.Ende` (Spitze bei B, `Kopfart`, Nullwert
   Dreieck) und `Anfang` (bei A, Typ `Anfangskopf`: die Art um eins verschoben, damit der Nullwert „keine“ ist; lesen mit
   `KopfAnfang()`, setzen mit `MitKoepfen(anfang, ende)`). Ein Pfeil ohne Angabe sieht aus wie vorher: Dreieck bei B.
   Keine Spitze an beiden Enden ist die Linie (bleibt das Objekt Pfeil).
2. **Arten** (`document.Kopfarten`, Reihenfolge der Bedienung; Name in Datei, `--annotate` und MCP): keine (`none`),
   gefülltes Dreieck (`triangle`), schlank (`narrow`, `4w+7` lang, `3w+4` breit), eingekerbt (`notched`, Stealth:
   `3,5w+7` lang, `4,5w+7` breit, Kerbe 35 % der Länge), offen (`open`), halbe Spitze (`half`, nur der Widerhaken links
   der Richtung, im Code „links“: auf der Seite `(-ry, rx)`, im Bild mit y nach unten rechts in Blickrichtung), Punkt (`dot`), Raute (`diamond`, `4w+8` lang, `3w+6` breit, vordere Ecke am Ende), Quadrat (`square`,
   Seite `2,6w+5`, um das Ende), Querstrich (`bar`). Bei Kerbe und Raute endet der Schaft dort, wo seine Kanten die
   hinteren Kanten der Spitze treffen (`KopfRueckstand`): Der Umriss schließt ohne Lücke und ohne Zipfel an die
   Schaftpunkte an. Ein Ring (offener Kreis) ist verworfen: Er hätte ein Loch, der Umriss eines Kopfs ist ein Vieleck.
3. **Bedienung:** Der Abschnitt heißt „Arrow style“ (deutsch „Pfeil-Stil“) mit zwei Klappfeldern „Start“ und „End“. Jedes
   zeigt einen kleinen Pfeil mit der gewählten Spitze, ihren Namen und den Aufklapp-Pfeil der Fußzeilenfelder. Klick,
   Enter oder Leertaste öffnen eine Liste aller zehn Arten über die Menü-Mechanik (`listeKopfAnfang`, `listeKopfEnde`;
   Tastatur, Klick daneben, Sperre beim Speichern wie bei den Listen der Fußzeile), unter dem Feld; jede Zeile zeigt
   vor dem Namen eine Vorschau aus derselben Zerlegung wie die Leinwand (`zeichneKopfMuster`, `musterPfeil`; Punkt,
   Quadrat und Querstrich rücken um ihren Überstand ein, `KopfUeberstand`). Per Tastatur steht der Fokus auf der
   gewählten Art. Wie bisher gilt die Wahl ohne Auswahl für folgende Pfeile (`canvas.Stil.KopfEnde`/`KopfAnfang`), mit
   gewähltem Pfeil ändert sie ihn als einen Rückgängig-Schritt (`SetzeKopf(amAnfang, art)`). Die Statuszeile nennt
   „Arrow style at the start/end: …“. Die Leiste wird dadurch etwas höher; `farbwaehler-189.txt` ist nachgemessen.
4. **Format (F-830): FormatVersion 2.** Attribute `s:ende` (fehlt bei `triangle`) und `s:anfang` (fehlt bei `none`) am
   `<g s:typ="pfeil">`; `s:kopf`/`s:enden` aus Nachtrag 19 entfallen (nie in einem Release geschrieben, v0.7.0 kannte
   sie nicht). Weicht ein Pfeil von der Voreinstellung ab, schreibt der Schreiber `s:format="2"` (`schreibVersion`,
   `versionPfeilKoepfe`), wie bei Lupe und Sprechblasen-Spitze: Ein älterer Leser meldet die Datei als neuer, statt
   einen anderen Pfeil zu zeigen und die Attribute beim Speichern zu verlieren (Kritik K-12). In Version 1 sind die
   Attribute unbekannter Inhalt und gelten nicht; unbekannte Werte gelten als Voreinstellung. Beispiel
   `testdata/format/arrow-heads` (jetzt Version 2), `docs/format.md` §8.1. Dateien, die ein Zwischenstand vom 26.11.2026
   mit `s:kopf`/`s:enden` gespeichert hat, zeigen einen gewöhnlichen Pfeil (Dreieck bei B); bewusst, betroffen ist nur die
   Bibliothek des Maintainers.
5. **`--annotate` und MCP:** beim Typ `arrow` die freiwilligen Schlüssel `start_head` (Voreinstellung `none`) und
   `end_head` (Voreinstellung `triangle`) mit den Namen aus Punkt 2; `head`/`ends` entfallen.
6. Schnellstile tragen die Spitzen weiter nicht (Nachtrag 19 Punkt 6), obwohl der Abschnitt jetzt „Pfeil-Stil“ heißt: Der
   Name folgt Andis Wortwahl, in der Sache bleibt die Spitze Form.
7. Ein Pfeil kürzer als seine Spitzen hat keinen Schaft; Renderer und Editor füllen die Köpfe dann in einer Füllung,
   damit überlappende Köpfe halbdurchsichtig einmal decken (Sitzungs-Kritik 26.11. K-6, `TestKurzerPfeilKoepfeDeckenEinmal`,
   Mutant PF29).

Tests: `internal/document/pfeilform_test.go` (Nullwerte, Maße, Kerbe und Raute schließen an den Schaft),
`internal/render/pfeil_test.go` (jede Art an jedem Ende deckt einmal), `internal/canvas/pfeilform_test.go`,
`internal/ui/pfeilform_test.go` (Klappfelder, Liste, Tastatur, Vorschau im Feld), `internal/fileformat/pfeilform_test.go`
(Version 2, Version 1 ignoriert), `internal/annotate`; Mutanten `scripts/mutanten/pfeilform.json` (PF1–PF28); Sichtprobe
`scripts/drehbuecher/pfeilform-351.txt` (ersetzt `pfeilform-315.txt`).

Verworfen: FormatVersion 1 behalten (Nachtrag 19; ein älterer Leser zeigte einen anderen Pfeil und verlöre die Attribute
beim Speichern, F-830); `s:kopf`/`s:enden` weiter lesen (kein Release hat sie geschrieben); zweimal zehn Symbolknöpfe
statt der Klappfelder (zu hoch für die Leiste, kein Platz für Namen); ein Ring (Loch im Umriss).

## Nachtrag 21 (26.11.2026): Gestalt des Pfeils, SVG ohne Fuge (Karte #349, F-826, F-829)

**Status:** Akzeptiert (Claude Opus 5.5 aus der Sitzungs-Kritik 26.11., K-6 und K-11; Andi: „arbeite selbstständig“).

1. **Gestalt statt Grenzen, wo es um das Sichtbare geht (F-829).** `document.Pfeil.Gestalt()` ist das Rechteck um
   Schaft (mit halber Strichstärke) und Köpfe (`Form().Huelle`), ohne den Kontrollpunkt K. Damit rechnen
   `canvas.sichtbarIn` (Verschieben und Griffe halten mindestens `MinSichtbar` im Bild, Nachtrag 2) und das Feld
   „Auswahl“ der Fußzeile (Größe samt Kopf). Ein Punkt-, Quadrat- oder Balkenkopf, der über das Ende ragt, zählt also
   als sichtbar; ein Pfeil, von dem nur der Kontrollpunkt im Bild läge, nicht.
2. **`Pfeil.Grenzen` bleibt** (Endpunkte, K und Strich): Treffer und Einrasten (Ziele sind die Enden, an denen Pfeile
   andocken) rechnen weiter damit; die Griffe sitzen an A, B und K. Grenzen auf die Gestalt umzustellen bräche Einrasten und Verschieben
   (erster Versuch vom 26.11., zurückgesetzt).
3. **SVG ohne Fuge (F-826).** Schaft (Strich), Kopf und halbe Scheibe bleiben getrennte Formen; Kopf und halbe Scheibe
   reichen um `pfeilFuge` = 1 Pixel (höchstens die Schaftlänge) in den Schaft zurück, sodass jedes Pixel an der
   gemeinsamen Kante von einer Form ganz gedeckt ist. Bei durchscheinender Farbe stehen die Formen deckend in einer
   Gruppe `<g opacity="…">`, damit die Überlappung nicht doppelt deckt — wie der Renderer, der den Pfeil in einer
   Füllung zeichnet. Gemessen mit rsvg 12-fach am Beispiel `arrow-heads`: 129 auffällige Pixel vorher, 38 nachher; die
   übrigen sind Ecken nach innen an Winkel und Kerbe, keine Fuge. Das Beispiel wächst von 3,3 auf 3,7 KB.

Tests: `internal/canvas/pfeil_gestalt_test.go`, `internal/fileformat/svg_pfeil_fuge_test.go` (rsvg, Mittellinie gleich
der Schaftmitte, deckend und durchscheinend); Mutanten PF30–PF33 in `scripts/mutanten/pfeilform.json`.

Verworfen: der ganze Pfeil als ein Pfad aus `render.ZerlegePfeil` (Datei elffach größer, Importzyklus fileformat →
render); Schaft als Umriss mit Versatzkurve (eine quadratische Kurve hat keine exakte Versatzkurve als Bézier).

## Nachtrag 22 (26.11.2026): Regler der Strichstärke (Karte #352)

**Status:** Akzeptiert (Andi, 26.11.2026: „macht bei der strichstärke eher (oder zusätzlich) ein slider sinn oder
nicht?“; Ausgestaltung Claude Opus 5.5, Andi: „arbeite selbstständig“).

1. **Zusätzlich**, nicht statt − / +: Unter der Zeile „Line width“ steht eine Schiene über die volle Breite der Leiste
   (16 dp hoch, Marke wie im Farbwähler; `internal/ui/strichregler.go`). − / + und die Stufen (`StrichStufen`) bleiben für
   die Tastatur und genaue Schritte; der Regler nimmt keinen Fokus.
2. **Skala** logarithmisch von 1 bis 32 px (kleinste und größte Stufe, `canvas.ReglerMin`/`ReglerMax`), auf ganze Pixel
   gerundet (`BreiteAusRegler`, `ReglerLage`): Die feinen Stärken bekommen so viel Weg wie die groben. Stärken über 32 px
   (über `KantenFaktor` oder aus einer Datei) zeigt die Marke am rechten Rand.
3. **Ziehen ist ein Rückgängig-Schritt** wie im Farbwähler (Nachtrag 9): `canvas.Sitzung.BeginneBreiteZug`,
   `BreiteZugSetze`, `BeendeBreiteZug`; das gewählte Objekt ändert sich laufend über den Zug des Dokuments. Esc und
   Zeiger-Cancel stellen das Objekt her (`BrecheBreiteZugAb`), die Stärke des Werkzeugs bleibt die gezogene. Während der
   Texteingabe und während des Speicherns beginnt kein Zug; ein laufender Zug zählt als `Zieht` (Schließen und Signal
   brechen ihn ab).
4. Die Leiste wird beim Strich 22 dp höher; `pfeilform-351.txt`, `farbwaehler-189.txt` und `farbwaehler-318.txt` sind
   nachgezogen, `ebenen-191.txt` und `mischen-215.txt` unverändert geprüft.

Tests `internal/canvas/breitezug_test.go`, `internal/ui/strichregler_test.go`; Mutanten `scripts/mutanten/strichregler.json`
(SR1–SR14); Sichtprobe `scripts/drehbuecher/strichregler-352.txt`.

## Nachtrag 23 (26.11.2026): Farbpaar vereinfacht (Karte #359)

**Status:** Akzeptiert (Andi, 26.11.2026, Foto beim Pfeil: „wenn man keine hintergrundfarbe einstellen kann, wie z.B.
bei Pfeile – dann verwirrt das symbol mit der hg-farbe und auch das farbe tauschen symbol … was bedeutet das „keine“ …
überleg mal, ob man den farb-bereich nicht prinzipiell einfacher/userfreundlicher machen könnte“; Ausgestaltung Claude
Opus 5.5, Andi: „arbeite selbstständig“).

Ändert Nachtrag 11 Punkte 4 und 9 für die Oberfläche; Logik (`canvas.Fuellbar`, `AktivesFeld`, `KeineFuellung`) bleibt.

1. **Ohne Füllung nur ein Feld.** Wo `Fuellbar` nicht gilt (Pfeil, Lupe, Nummer, Werkzeuge ohne Farbe), zeigen die Zeile
   über dem Farbwähler und das Paar unter der Werkzeugleiste nur das Farbfeld. Füllfeld, ⇄ und „keine“ werden weder
   gezeichnet noch angemeldet (kein graues Feld, kein grauer Pfeil, nicht per Tab erreichbar). Der Hinweis am Feld
   unter der Werkzeugleiste heißt dann „Color“ statt „Foreground“. Menü und Strg+Umschalt+X bleiben grau bzw. wirkungslos
   wie bisher.
2. **„Keine“ ist ein Farbfeld,** kein Textknopf: weiß mit rotem Schrägstrich (dasselbe Zeichen wie auf einer
   durchsichtigen Füllung, `zeichneKeine`), nach ⇄. Ein Klick macht die Füllung zum aktiven Feld **und** durchsichtig
   (vorher wirkte der Knopf nur bei schon aktiver Füllung und war sonst grau). Grün umrandet, wenn die aktive Füllung
   durchsichtig ist.
3. **Name des aktiven Felds** neben dem Paar (`farbfeldName`, Kontext `color field`): „Color“, „Fill“ oder „Fill: none“.
   So ist ohne Hinweis zu sehen, worauf Farbfläche, Hex und Deckkraft wirken. Die Zeile wird nicht höher, die Lage der
   Elemente darunter bleibt; Drehbücher mit Leistenkoordinaten gelten weiter.
4. Offen für #317 (Farbschemata) und #344 (Farbwähler als Aufklapper am Paar): Die Beschriftung je Art („Line“/„Fill“,
   „Text“/„Label“) wäre genauer, braucht aber Platz; erst mit dem neuen Muster von #317 entscheiden.

Tests `internal/ui/farbpaar_einfach_test.go`; Mutanten `scripts/mutanten/farbpaar-einfach.json`, U219-3 nachgezogen.

## Nachtrag 24 (26.11.2026): Eigene Farbfelder (Karte #364)

**Status:** Akzeptiert (Andi, 26.11.2026: „bei den favoriten farben rechts sollten die vorhandenen aber gelöscht und
auch neue hinzugefügt werden können, oder?“ und „entscheide du das“; Ausgestaltung Claude Opus 5.5).

Ändert Nachtrag 9 (Raster neben der Farbfläche) und Nachtrag 17 (vier zuletzt benutzte).

1. **Zwei Arten, klar getrennt.** *Farbfelder* sind die eigene, gepflegte Palette; *zuletzt benutzt* füllt sich von
   selbst. Vorher standen die vier eingebauten Beschriftungsfarben fest im Raster und ließen sich weder entfernen noch
   ergänzen.
2. **Farbfelder** stehen unter Hex und Deckkraft in Reihen (20 dp, so viele je Reihe wie die Leiste breit ist), „+“ am
   Ende nimmt die aktuelle Farbe auf (grau, wenn sie durchsichtig ist, schon ein Feld ist oder schon 24 Felder da sind;
   der Hinweis sagt, warum). Rechtsklick oder Entf an einem Feld entfernt es, wie bei eigenen Schnellstilen (ADR-0037).
   Ein Klick setzt die Farbe wie bisher. Ohne Datei gelten die eingebauten Felder `canvas.StandardFarbfelder` (die vier
   Beschriftungsfarben, dazu Weiß, Gelb und Orange); wer alle entfernt, hat keine.
3. **Datei** `colors.ini` neben `styles.ini` (`internal/settings/farben.go`): Abschnitt `[swatches]`, je Feld
   `color = #RRGGBBAA`. Jede Änderung liest die Datei unter der Sperre `colors.ini.lock` neu und schreibt sie atomar
   (0600/0700), so verliert ein zweites Fenster nichts; Unlesbares wird vorher als `colors.ini.bak` gesichert, gibt es die
   schon, bleibt alles (`ErrFarbenSicherung`). Höchstens 24 Felder, 64 KiB, Warnungen statt Abbruch.
4. **Zuletzt benutzt:** Das Raster neben der Farbfläche zeigt nur noch die acht zuletzt benutzten Farben
   (`canvas.ZuletztGrenze` 8), auch Farben aus den Farbfeldern; freie Plätze sind leere Rahmen. Durchsichtig zählt
   weiter nicht.
5. Tab-Reihenfolge: Farbpaar, Pipette, zuletzt benutzte, Hex, Deckkraft, Farbfelder, „+“. Ein Klick auf ein Farbfeld
   verwirft eine schwebende Hex-Eingabe wie die übrigen Farbziele (`inFarbwahl`).

Tests `internal/settings/farben_test.go`, `internal/ui/farbfelder_test.go`; Mutanten `scripts/mutanten/farbfelder.json`.
