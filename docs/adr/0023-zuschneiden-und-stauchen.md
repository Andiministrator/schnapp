# ADR-0023: Zuschneiden und Stauchen im Editor

## Status
Akzeptiert (Claude Opus 5.5, 20.11.2026, Karte #38 in Sprint 6; Andi hat die Karte freigegeben, ohne diese
Einzelheiten vorzugeben). Setzt A-23 und A-40 (Zuschneiden) im Editor um; beantwortet die offene Frage aus ADR-0022
Nachtrag 1 (Schwärzen und Zuschnitt).

## Kontext
Das Dokumentmodell kennt Zuschnitt und Stauch-Streifen seit Sprint 3 als reine Angaben (ADR-0008): `SetzeZuschnitt`,
`SetzeStreifen`, die Abbildung Ansicht↔Original, die Grenzen `MaxStreifen`/`MaxKacheln`, den Rundlauf über die
Arbeitsdatei und den SVG-Export, der Weggeschnittenes nicht enthält. Es fehlten die Werkzeuge im Editor, die
Zackenkante an den Stauch-Nähten (A-23), eine Regel für Beschriftungen, die ganz im Verborgenen liegen, und die
Entscheidung, ob Schwärzen auch jenseits der Zuschnittkante wirkt.

Bisher zeichneten Renderer und Editor eine Beschriftung ganz in einem weggestauchten Streifen auf der Naht
zusammengedrückt, der SVG-Export ließ sie weg (F-66) — drei Ausgänge, zwei Regeln.

## Entscheidung

1. **Zuschneiden: Werkzeug mit Kürzel C** (ohne Strg; Strg+C bleibt Kopieren). Man zieht in der Ansicht einen
   Rahmen; **beim Loslassen** gilt er (wie Schwärzen, ADR-0022 — kein Enter nötig), **Esc** während des Ziehens bricht
   ab. Unter der Mindestgröße (`canvas.MinObjekt`) passiert nichts. Der Rahmen wird auf ganze Ansichtspixel gerundet
   und ins Original umgerechnet (letztes enthaltenes Pixel + 1 als rechte/untere Kante, so endet ein Zuschnitt nie
   mitten in einem Streifen). Zuschneiden verkleinert immer innerhalb der aktuellen Ansicht. Ein Rahmen über die
   ganze Ansicht tut nichts — geprüft an der Ansicht, nicht am umgerechneten Zuschnitt, der bei einem Streifen am Rand
   abweichen kann (F-182).
2. **Zuschnitt aufheben: Doppelklick mit dem Werkzeug Zuschneiden** zeigt wieder das ganze Original (ein
   Rückgängig-Schritt). Die Statusleiste nennt das. Rückgängig nimmt jeden einzelnen Zuschnitt zurück.
3. **Stauchen: Werkzeug mit Kürzel X** („Cut out“). Man zieht quer über den Teil, der herausfallen soll; die
   **Zugrichtung entscheidet**: mehr senkrecht gezogen nimmt Zeilen heraus (das Bild wird niedriger, der Fall „lange
   Website“), mehr waagrecht Spalten. Während des Ziehens zeigt der Editor das Band über die ganze Bildbreite bzw.
   -höhe. Keine Umschalttaste nötig. Loslassen übernimmt, Esc bricht ab, unter der Mindestgröße nichts.
   Liegt das Band an einer Naht oder über ihr, verschmilzt der neue Streifen mit dem alten (überlappende und
   anstoßende Streifen werden einer) — es entsteht nie eine doppelte Naht. Ein Band, das die ganze Ansicht nähme,
   wird abgelehnt (Meldung), ebenso eines über `MaxStreifen`/`MaxKacheln`.
4. **Streifen entfernen:** Rückgängig, oder **Doppelklick auf eine Naht** mit dem Werkzeug Stauchen: Alle Streifen
   dieser Naht kommen zurück (ein Schritt). Ein Zuschnitt-Aufheben lässt die Streifen stehen.
5. **Jede Änderung ist genau ein Rückgängig-Schritt** (`SetzeZuschnitt` bzw. `SetzeStreifen` mit der ganzen neuen
   Liste). Ändert ein Schritt (auch Rückgängig/Wiederholen) die Ansicht, passt der Editor sie neu ins Fenster ein.
6. **Zackenkante nur an Stauch-Nähten**, nicht an Zuschnittkanten: Ein Zuschnitt ist ein gewollter Rahmen, eine
   Stauch-Naht verbirgt etwas mitten im Bild — nur dort muss der Betrachter sehen, dass etwas fehlt. Eine Naht am Rand
   der Ansicht (Streifen schließt an den Zuschnitt an) bekommt keine Kante. Die Kante ist ein Zackenband in
   Ansichtspixeln: Die Zackenlinie pendelt ±4 px quer zur Naht, ein Zahn ist entlang der Naht 16 px weit. Darum liegt
   ein weißes Band, darin eine dunkelgraue Linie; ihre Dicken (6 px bzw. 2 px) sind **quer zur Naht** gemessen (die
   Zackenlinie wird senkrecht zur Naht um ±3 px bzw. ±1 px verschoben, nicht senkrecht zur schrägen Zackenlinie — an
   den Flanken wirkt das Band deshalb etwas schmaler). Das Band liegt **über** Bild und Beschriftungen, im Editor, im
   gerenderten Bild (Arbeitsdatei, PNG/JPG, Zwischenablage) und im SVG-Export gleich. Die Geometrie steht einmal in
   `document` (`Naehte`, `Zackenband`). Der SVG-Export schreibt das Band je Achse einmal in `<defs>` und jede Naht als
   `<use>`, der Editor zeichnet die Pfade einmal auf und baut sie erst nach einer Änderung der Nähte neu (F-183). Das
   Band verdeckt an der Naht einige Pixel; das ist gewollt. Der Renderer zählt das Band im Kostenmodell
   (`render.Pruefe`) wie jede Füllung.
7. **Beschriftungen und Nähte/Zuschnitt:** Zuschneiden und Stauchen ändern keine Beschriftung (nicht zerstörend).
   Eine Beschriftung, die über eine Naht läuft, wird mit dem Bild gestaucht abgebildet (ihre Punkte folgen der
   Abbildung; Teile im Streifen liegen auf der Naht). Eine Beschriftung, die **ganz** im Verborgenen liegt
   (außerhalb des Zuschnitts oder ganz in einem Streifen), ist **überall unsichtbar**: Editor, Renderer, SVG-Export,
   Trefferprüfung und der Zähler der verborgenen Beschriftungen folgen derselben Regel `Dokument.ObjektSichtbar`
   (die bisherige Regel des SVG-Exports, F-66): `Sichtbar` für die `Reichweite` des Objekts. Bei Text und
   Schritt-Nummer ist die Reichweite die **echte Tinte der Glyphen** (Go Regular bzw. Go Bold aus denselben
   Schriftdateien wie Renderer und Editor, plus 1 px Kantenglättung), mit Hintergrund zusätzlich dessen Fläche, bei der
   Nummer zusätzlich der Kreis — keine Schätzung je Zeichen, sonst stünde ein unsichtbarer Text im SVG (F-181). Die
   Messung liegt in `document`, damit die Regel nicht davon abhängt, welches Paket gebunden ist. Ein verborgenes Objekt
   bleibt im Dokument und kehrt zurück, wenn der Zuschnitt aufgehoben oder der Streifen entfernt wird. Damit das nicht
   unbemerkt geschieht, meldet die Statusleiste, wie viele Beschriftungen jetzt verborgen sind, sobald es nach
   Zuschneiden, Stauchen, Rückgängig oder Wiederholen **mehr** sind als vorher (F-184); eine verborgene Auswahl wird
   aufgehoben.
8. **Schwärzen, Zuschnitt und Stauchen:** Der Schwärzungsrahmen bleibt auf die Ansicht begrenzt — man schwärzt, was
   man sieht. Was jenseits der Zuschnittkante oder in einem Stauch-Streifen liegt, bleibt im Original der Arbeitsdatei
   (`snAO`); weder Zuschnitt noch Stauchen ist Schwärzen. Ausgänge, die das Bild weitergeben (SVG/PNG/JPG-Export,
   Zwischenablage, sichtbares Bild der Arbeitsdatei), enthalten Weggeschnittenes und Herausgenommenes nicht (ADR-0008
   Nachtrag 2). Die Statusleiste der Werkzeuge Zuschneiden **und** Stauchen sagt: Das Entfernte bleibt in der
   Arbeitsdatei, Heikles schwärzen (F-185); die README sagt: Wer etwas sicher entfernen will, schwärzt es, bevor er
   zuschneidet. Ein Test legt ein Muster jenseits der Kante und belegt beides (bleibt im `snAO`, fehlt in jedem
   Ausgang).

## Konsequenzen
- Die Werkzeugleiste wird um zwei Knöpfe breiter; F-148 (Leiste in schmalen Fenstern, Karte #80) wird dringlicher.
- Eine weitergegebene Arbeitsdatei enthält das ganze Original, auch Weggeschnittenes; das war schon so (ADR-0008) und
  steht jetzt in der Statusleiste.
- Ein Streifen, der an den Rand der Ansicht stößt, wirkt wie ein Zuschnitt, ohne Naht; er lässt sich über
  Rückgängig entfernen oder, nach dem Aufheben des Zuschnitts, über seine dann sichtbare Naht.
- Die Zackenkante verdeckt einige Pixel an der Naht. Wer genau dort etwas zeigen will, staucht woanders.
- Beschriftungen ganz in einem Streifen werden nicht mehr auf der Naht zusammengedrückt gezeichnet; Renderer und
  SVG-Export stimmen darin jetzt überein.
