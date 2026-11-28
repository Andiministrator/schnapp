# ADR-0049: Freihand-Stift (geglätteter Linienzug) und freier Textmarker

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #333 in Sprint 14; Vorschlag Claude, von Andi am 25.11.2026 übernommen;
Andi 28.11.: „arbeite selbstständig“). Ergänzt ADR-0021 (Zeichenwerkzeuge), ADR-0048 (Textmarker), ADR-0008 (Format),
ADR-0037 (Schnellstile), ADR-0041 (Stil je Werkzeug).

## Kontext
Zum Umkreisen, Unterstreichen und Anstreichen fehlt ein freier Strich. Die Karte verlangt einen Freihand-Strich, der
beim Loslassen geglättet und vereinfacht wird (Punktzahl begrenzt), Umschalt = gerade Strecke, gezeichnet über den
überlappungsfreien Linienstrich des Renderers. ADR-0048 Punkt 4 legt den freien Textmarker in diese Karte: dieselbe
Mischung „Multiplizieren“ auf einem freien Strich.

Kein vorhandener Objekttyp trägt einen Linienzug: Der Pfeil ist eine quadratische Kurve aus drei Punkten. Objekte sind
vergleichbare Werte (`==`, `document.Gleich`); ein Feld mit einer Slice ginge nicht.

## Entscheidung
1. **Neuer Objekttyp `document.Freihand`** mit `Ebene`, `ID`, `Strich` und `Zug *[]Punkt` (Original). Wie die Spitze der
   Sprechblase wird der Zeiger nie durchgeschrieben, sondern neu gesetzt; `Gleich` vergleicht die Punkte dem Wert nach.
   Mindestens 2, höchstens `MaxFreihandPunkte` = 4096 Punkte, alle endlich und im Größenrahmen des Dokuments.
   `Grenzen` ist die Hülle der Punkte um die halbe Strichbreite.
2. **Zeichnen:** offener Linienzug über `render.ZerlegeStrich` (überlappungsfreie Teile, runde Enden, Strichart wie der
   Pfeilschaft), je sichtbarem Bereich beschnitten wie der Pfeil (`document.StreckenStuecke`); Editor und Vorschau füllen
   dieselben Teile. Der Trockenlauf zählt Punkte und Teile wie bei der Ellipse.
3. **Aufnehmen und Glätten:** Während des Zugs kommt ein Punkt nur dazu, wenn er mindestens 2 Fensterpixel vom letzten
   entfernt ist. Beim Loslassen wird der Zug vereinfacht (Ramer–Douglas–Peucker, Toleranz 0,5 Fensterpixel, im Original
   durch den Zoom geteilt) und dann zweimal nach Chaikin geglättet (Ecken abgerundet, Anfangs- und Endpunkt bleiben);
   über `MaxFreihandPunkte` wird gleichmäßig ausgedünnt. **Umschalt** = gerade Strecke vom Anfang zum Zeiger. Ein Zug
   kürzer als `MinObjekt` bleibt nicht liegen.
4. **Werkzeug Freihand**, Taste **F** (allein frei), Symbol `icons.Freihand`, in der Werkzeugleiste nach dem Textmarker.
   Eigener Stil `[freehand]` in `tools.ini` (Voreinstellung wie der Pfeil), Schnellstil-Art `freehand` mit den
   eingebauten Stilen des Pfeils. Keine Griffe an den Punkten: Ein gewählter Freihand-Strich wird verschoben; Größe und
   Form ändert man durch neu Zeichnen.
5. **Freier Textmarker:** Der Textmarker bekommt in der rechten Leiste das Kästchen „Freehand“ (Werkzeugstil, Schlüssel
   `freehand` in `[highlighter]`). Ist es an, legt das Werkzeug Textmarker einen `Freihand` mit Mischung
   „Multiplizieren“ an; Umschalt zieht dann die gerade waagerechte Strecke wie beim geraden Textmarker. Ein `Freihand`
   mit dieser Mischung gilt für Leiste, Ebenen und Schnellstile als Textmarker (`objektWerkzeug`).
6. **Format:** Objekttyp `s:typ="freihand"` mit `s:punkte` (Punkte nach 5.2, durch Leerzeichen getrennt), `s:strich` und
   `s:strichart`; Element `<path>` mit `M`/`L`, runden Enden und Ecken. Ein neuer Objekttyp verlangt nach §11 Regel 2 eine
   neue Version: **FormatVersion 3**, nur mit einem Freihand-Strich geschrieben (`versionFreihand`), sonst wie bisher.
   Ein Leser der Version 2 meldet eine Datei mit Freihand als „neuer“ und bietet das sichtbare Bild an.
7. **Bedienung ohne Fenster:** `--annotate`/MCP Typ `freehand` mit `points` (2 bis 4096 Punkte), `color`, `width`,
   `line_style`, `shadow`; `highlight` nimmt statt `from`/`to` auch `points` (freier Textmarker). Die Punkte werden
   übernommen, nicht geglättet.

## Erwogene Alternativen
- **Pfeil mit vielen Stützpunkten:** Der Pfeil ist eine Bézier-Kurve mit Griff; Köpfe, Griffe und Einrasten setzen
  drei Punkte voraus. Ein eigener Typ hält beides einfach.
- **Glätten als Kurve (Catmull-Rom, Bézier im Format):** schöner bei wenigen Punkten, aber Renderer, Editor, SVG und
  Treffer bräuchten eine zweite Kurvenrechnung; der geglättete Linienzug mit wenigen Punkten reicht für Handschrift.
- **Freihand in Version 2:** ein Leser der Version 2 hielte die Datei für beschädigt statt für neuer; §11 Regel 2 lässt
  in einer Version nur deren Objekttypen zu.

## Konsequenzen
- Schnapp liest und schreibt Version 3; Dateien ohne Freihand-Strich bleiben bei 1 oder 2.
- Punktgriffe (einzelne Punkte ziehen) wären eine eigene Karte.

## Nachtrag 1 (28.11.2026, Kritik #333, F-1036, F-1037)

Punkt 3 ändert sich:

- **Beim Loslassen:** vereinfachen (Toleranz 0,5 Fensterpixel, im Original durch den Zoom geteilt), zweimal nach
  Chaikin glätten, **danach noch einmal vereinfachen mit der halben Toleranz** und **auf 1/100 px runden**
  (`document.FreihandAusZug`, `document.Gerundet`; gleich gewordene Nachbarn fallen weg). Chaikin vervierfacht die
  Punkte, auf flachen Stücken liegen sie fast auf einer Geraden; das zweite Vereinfachen nimmt sie wieder heraus, die
  Linie weicht um höchstens 0,25 Fensterpixel vom geglätteten Zug ab. Gespeichert werden **höchstens so viele Punkte
  wie roh**; ergäbe das Glätten mehr (kurze Züge mit spitzen Ecken), bleibt der vereinfachte, ungeglättete Zug.
  `MaxFreihandPunkte` begrenzt zuletzt wie bisher.
- **Verschieben** rundet ebenfalls auf 1/100 px (`Freihand.Verschoben`), sonst trüge ein Zoom wie 0,73 wieder Zahlen
  mit 17 Stellen in die Datei.
- **Während eines sehr langen Zugs** werden die Rohpunkte bei `MaxFreihandPunkte` nicht mehr gleichmäßig ausgedünnt,
  sondern auf höchstens die Hälfte vereinfacht (`canvas.rohVerdichtet`: Toleranz 1/8 Fensterpixel, verdoppelt, bis es
  reicht). Gleichmäßiges Ausdünnen nahm dem Anfang bei jeder Runde wieder die Hälfte seiner Punkte.
- **Darstellung im SVG** (informativ, docs/format.md §8.9): `M` mit dem ersten Punkt, dann `l` mit den übrigen relativ
  zum Vorgänger, alles in Hundertsteln aus den gerundeten Lagen (`fileformat.freihandPfad`).

Gemessen (langer Zug, 4096 Rohpunkte, Zoom 1 und 0,73, mit und ohne Zittern): vorher 4096 Punkte, 32 bzw. 75 Byte je
Punkt, 130 bzw. 306 KB je Strich; nachher 1142 bis 2189 Punkte, 25 bis 27 Byte je Punkt, höchstens 58 KB je Strich.
Unter 20 Byte je Punkt ginge nur mit einer einzigen Kopie der Punkte (heute `s:punkte` und der Pfad); das wäre eine
Änderung des Formats und ist nicht Teil dieses Nachtrags.
- Nach dem Verify (28.11.2026): Punkt 1 — ein Strich aus lauter gleichen Punkten ist ungültig (Datei beschädigt,
  `--annotate` lehnt ab, F-1040); Verschieben rundet nur den Versatz, die Punkte wandern starr. Punkt 2 — der Editor
  merkt Zerlegung und Pfad je Strich, getrennt für Leinwand und Vorschau der Ebenenliste (`internal/ui/freihandmerker.go`,
  F-1038).

## Nachtrag 2 (28.11.2026, F-1045, F-1049)

Punkt 6 (Format) ändert sich, solange FormatVersion 3 nicht ausgeliefert ist (kein `v*`-Tag seit 2c48ba8):

- **`s:punkte` relativ:** der erste Punkt absolut, jeder weitere als Abstand zu seinem Vorgänger, in ganzen
  Hundertsteln (`fileformat.punkteRelativ`); die Darstellung `M`/`l` trägt dieselben Zahlen. Der Leser summiert die
  Abstände ungerundet; jede Lage wird auf 1/100 px gerundet.
- **Freihand-Punkte liegen immer auf 1/100 px:** `Freihand.MitPunkten` (und damit `NeuerFreihand`) rundet jede Lage,
  gleich gewordene Nachbarn bleiben stehen. So sind Schreiben und Lesen verlustfrei und deckungsgleich; ein Strich unter
  1/100 px Länge (auch aus `--annotate`/MCP) ist ein Strich ohne Länge und wird abgelehnt (F-1040).
- **Gesamtgrenze** `MaxFreihandGesamt` = 1 048 576 Punkte je Datei (F-1049, docs/format.md §10), im Leser vor dem
  Teilen gezählt; `PruefeInhalt` prüft dieselbe Grenze. (Eine eigene Prüfung in `SchreibeArbeitsdatei` aus
  aef7250 ist mit dem nächsten Punkt entfallen: kein Dokument kommt mehr über die Grenze.)
- **Beim Hinzufügen** (F-1051): `Hinzufuegen`, `FuegeEin`, `HinzufuegenMehrere`, `FuegeEinMehrere` und `SetzeAngaben`
  lehnen Objekte ab, deren Freihand-Punkte nicht mehr passen (`ErrFreihandVoll`); ein neuer Freihand-Zug
  (`BeginneNeuenZug`) braucht Platz für einen ganzen Strich (`MaxFreihandPunkte`), weil er bis dahin wächst. Freihand
  und freier Textmarker melden das vor dem Druck (`canvas.AblehnungFreihand`), Einfügen und `--annotate` scheitern mit
  Meldung. So kann kein Bild entstehen, das Speichern oder Rettungskopie nicht mehr tragen.

Gemessen (derselbe lange Zug wie in Nachtrag 1): im SVG 21 bis 23 statt 25 bis 27 Byte je Punkt, im komprimierten
`snAP` 3,5 bis 4,7 statt 8,7 bis 10,8 Byte je Punkt. Beide Kopien lauten gleich, zlib verweist die zweite auf die erste.

Erwogen: nur eine Kopie (die Punkte allein im Pfad oder kein Pfad). Daten stehen im Format sonst immer in
`s:`-Attributen, die Darstellung ist informativ und für Betrachter da; komprimiert kostet die zweite Kopie unter
1 Byte je Punkt. Verworfen.

