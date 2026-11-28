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
