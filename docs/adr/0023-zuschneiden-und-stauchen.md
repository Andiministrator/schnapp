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

## Nachtrag 1 (20.11.2026): Zackenkante skaliert mit der Bildgröße (F-173)
Die Maße der Zackenkante (Punkt 6: Höhe 4, Zahnbreite 16, Rand ±3, Linie ±1) gelten nur bis zu einer kurzen Seite der
Ansicht von 1080 Pixeln. Darüber werden alle vier mit dem ganzzahligen Faktor `max(1, kurze Seite / 1080)`
multipliziert (Ganzzahldivision; 1920×1080 → 1, 3840×2160 → 2, 1000×5000 → 1, 7680×4320 → 4), damit die Kante bei
HiDPI-Aufnahmen und kleinem Zoom erkennbar bleibt. Die kurze Seite ist die der Ansicht nach Zuschnitt und Stauchen;
beides kann den Faktor ändern. *(Überholt durch Nachtrag 2: maßgeblich ist die Größe des Originals.)* Renderer, SVG-Export (`<defs>`-Band) und Editor holen ihn aus `Dokument.ZackenFaktor`;
der Zwischenspeicher des Editors ist über die Ansichtsgröße schon richtig geschlüsselt. Die Trefferweite von
`EntferneNaht` wächst mit.

## Nachtrag 2 (20.11.2026): Zackenfaktor nach dem Original, nicht nach der Ansicht (F-219)
Der Faktor aus Nachtrag 1 gilt für die Größe des **Originals**, nicht der Ansicht nach Zuschnitt und Stauchen. Anlass:
Eine 4K-Aufnahme hat genau 2160 Pixel kurze Seite; jedes Stauchen in dieser Richtung — also gerade der Fall, in dem
Zacken entstehen — drückte den Faktor nach der Ansicht auf 1 (Probelauf Karte #92: 13 statt 26 Pixel Bandhöhe). Der
Faktor misst die Pixeldichte der Aufnahme, und die ändert sich durch Schneiden nicht. Er bleibt damit für ein Dokument
fest; Renderer, SVG-Export, Editor und Trefferweite holen ihn weiter aus `Dokument.ZackenFaktor`. Entschieden von Claude
im Rahmen von F-173 (leicht umkehrbar, Andi informiert).

## Nachtrag 3 (20.11.2026): Band am Bildrand wird Zuschnitt, Knopf „Ganzes Bild“, Schwärzen an Naht und Kante (F-222, F-223)

**Status:** Akzeptiert (Entscheidung Andi E-21; umgesetzt von Claude Opus 5.5, Karte #99).

- **Punkt 3 ergänzt (F-222):** Ein Stauch-Band, das bis an den Rand der Ansicht reicht, wird **kein Streifen**,
  sondern ein Zuschnitt der betroffenen Achse (`canvas` `randZuschnitt`); die andere Achse behält ihren Zuschnitt. Am
  Rand gäbe es keine Naht, über die sich der Streifen zurückholen ließe — nach dem Leeren des Verlaufs (Speichern mit
  Schwärzung, Wiederöffnen) war er verloren, entgegen Kernprinzip 1. Die Statusleiste sagt es einmal („Der Teil reichte
  bis an den Rand …; ‚Ganzes Bild‘ holt ihn zurück“), sofern nicht schon die Meldung über verborgene Beschriftungen gilt.
- **Punkt 2 ergänzt:** Neuer Knopf **„Ganzes Bild“** in der Leiste (eigene Gruppe vor „Einpassen“, grau, wenn schon
  alles zu sehen ist), gleichwertig mit dem Doppelklick des Werkzeugs Zuschneiden (E-21: jede Tastenfunktion auch über
  die Leiste). „Ganzes Bild“ setzt den Zuschnitt auf das ganze Original und entfernt dabei Streifen, die dann am Rand
  des Originals anliegen (ohne Naht; so entstanden sie vor diesem Nachtrag, ältere Dateien können sie tragen), in einem
  Rückgängig-Schritt (`Dokument.SetzeAnsicht`). Streifen mit Naht bleiben.
- **Punkt 8 genauer (F-223):** Der Schwärzungsrahmen wird nach außen auf ganze Ansichtspixel gerundet, auf die Ansicht
  begrenzt und je Achse wie beim Zuschneiden ins Original umgerechnet (erstes Pixel bis hinter das letzte). Endet oder
  beginnt er genau an einer Naht oder an der Kante der Ansicht, bleibt der verborgene Streifen dahinter unberührt; ein
  Streifen mitten im Rahmen wird weiter mitgeschwärzt.
- **Probe:** `scripts/drehbuecher/rueckfall-randband-f222.txt` (virtueller KWin, Fotos 02 bis 05).
- **Folge:** Die Leiste bricht im Probelauf bei 1600 px Fensterbreite in zwei Zeilen um (F-148 regelt das
  Umbrechen); Karte #104 prüft die übrigen Tastenfunktionen und ob Menüs die Leiste entlasten.

## Nachtrag 4 (20.11.2026): Zackenband gedeckelt durch die Ansicht, Mindestrest je Seite (F-226)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #100).

Anlass: Der Faktor nach dem Original (Nachtrag 2) ließ bei einer 7680×4320-Aufnahme, die auf 300×80 zugeschnitten
wird, ein Band von ±28 Pixeln auf 80 Pixeln Höhe stehen; der Mindestrest aus F-215 (6 Pixel) lag unter der halben
Bandhöhe, es konnten zwei Splitter von je 3 Pixeln neben der Naht übrig bleiben.

- **Punkt 6 genauer:** `Dokument.ZackenFaktor` ist das Minimum aus dem Faktor nach dem **Original** (Nachtrag 2) und
  `max(1, kurze Seite der aktuellen Ansicht / 56)`. Die Bandhöhe (`2·ZackenBandHalb·Faktor`, `ZackenBandHalb` = 7 =
  Zackenhöhe + halber Rand) bleibt damit höchstens ein Viertel der kurzen Ansichtsseite; bei Faktor 1 ist das die
  Untergrenze. Eine ganze 4K- oder 8K-Aufnahme behält ihren Faktor (Nachtrag 2); nur eine sehr kleine Ansicht
  verkleinert das Band. Der Faktor kann sich dadurch beim Zuschneiden und Stauchen ändern; Renderer, SVG-Export, Editor
  und Trefferweite holen ihn weiter aus derselben Methode und zeichnen gleich.
- **Punkt 3 ergänzt:** Beim Stauchen bleibt je Seite der Naht mindestens die halbe Bandhöhe (`Faktor·ZackenBandHalb`
  Ansichtspixel, bei Faktor 1 also 7). Bliebe auf einer Seite weniger, reicht das Band bis an den Rand dieser Seite und
  wird dort wie jedes Randband zum Zuschnitt (Nachtrag 3, F-222). Die Regel gilt vor der Prüfung auf den Gesamtrest
  (F-215, Mindestgröße eines Rahmens).
- Technische Folge: `Dokument.OriginalGroesse` und `Dokument.OriginalBild` liefern Größe bzw. Original ohne Kopie
  (F-227); `Original` bleibt die Kopie für alles, was ändert oder aufbewahrt.
- Mit F-225 mitentschieden (ADR-0025 Punkt 4 gilt unverändert): Der SVG-Export schreibt keine `s:`-Attribute und keinen
  Programm-Namensraum; die Kacheln tragen `viewBox="0 0 B H"` und verschieben ihren Inhalt per `transform`. Die Lage
  des Zuschnitts im Original steht damit weiter in der Datei (im `transform`, im x/y des Bilds und in den Objekt-
  Koordinaten, die Original-Koordinaten sind) — berichtigt nach dem Verify-Lauf (V-5, F-242): Sie ist keine Spur im Sinn
  von Kernprinzip 3 (keine Pfade, Programm- oder Rechnernamen); sie ganz zu entfernen hieße alle Objekte umzurechnen.
  Die Arbeitsdatei (snAP) behält die Attribute für den Rundlauf.

## Nachtrag 5 (22.11.2026): Zuschneiden zeigt das ganze Original, Rahmen mit Griffen (F-386)

**Status:** Akzeptiert (Entscheidung Andi 22.11.2026, Variante a; umgesetzt von Claude Opus 5.5, Karte #169).

Anlass: Wer das Werkzeug Zuschneiden wählte, sah nichts und musste erst einen neuen Rahmen ziehen; einen Zuschnitt
nachzubessern hieß, ihn aufzuheben und neu zu ziehen. Erwartet (wie in Snagit): Mit der Wahl liegt ein Rahmen über
dem Bild, mit Griffen, die man nach innen zieht.

- **Punkt 1 ersetzt:** Solange das Werkzeug Zuschneiden aktiv ist, zeigt die Leinwand die **Vollansicht**: das ganze
  Original mit denselben Stauch-Streifen, nur ohne Zuschnitt. Was außerhalb des Zuschnitts liegt, ist abgedunkelt; auf
  dem Zuschnitt liegt ein Rahmen mit **acht Griffen** (Ecken und Kantenmitten). Greifbar ist die ganze Kante, nicht nur
  der Griff (Toleranz wie bei Objektgriffen, `canvas.Toleranz` Fensterpixel); an einer Ecke gewinnt die Ecke. Ein
  Griff-Zug verschiebt eine Kante (Ecke: zwei), nach innen wie nach außen bis an den Rand der Vollansicht, und **gilt
  beim Loslassen** als ein Rückgängig-Schritt — zurück per Strg+Z oder durch Zurückschieben des Griffs. Ein Druck
  **abseits der Kanten** (im Zuschnitt oder im abgedunkelten Teil) zieht wie bisher einen neuen Rahmen, jetzt überall
  in der Vollansicht. Mindestgröße, Esc, Runden auf ganze Pixel und „Rahmen gleich dem Zuschnitt tut nichts“ gelten
  weiter; die Umrechnung ins Original läuft über die Vollansicht (letztes enthaltenes Pixel + 1).
- **Koordinaten:** Die Ansicht nach dem Zuschnitt ist ein verschobener Ausschnitt der Vollansicht (die Streifen vor
  dem Zuschnittsanfang verschieben nur), `Dokument.VollVersatz` ist die Lage ihres Ursprungs darin. Die Leinwand rechnet
  deshalb weiter in Ansichtskoordinaten; im Modus Zuschneiden reicht die Bildfläche (`Sitzung.Bildflaeche`) von
  `-VollVersatz` bis `VollGroesse - VollVersatz`. Einpassen passt diese Fläche ein, die Vollansicht bleibt also beim
  Zuschneiden stehen. Ein Werkzeugwechsel passt nur neu ein, wenn der Nutzer nicht selbst gezoomt hat.
  Ändert sich beim Zuschneiden nur der Zuschnitt (Griff, neuer Rahmen, Rückgängig, Wiederholen), bleibt die Abbildung:
  eingepasst bleibt eingepasst, eine selbst gezoomte Ansicht behält Zoom und Lage der Vollansicht (der Versatz rückt um
  die Änderung von `VollVersatz` mal Zoom nach; Kritik KA-5, F-398). Esc verlässt das Werkzeug über denselben Weg wie
  jeder Werkzeugwechsel (KA-1, F-394).
- **Was die Vollansicht nicht zeigt:** Beschriftungen und Zackenkanten bleiben auf den Zuschnitt begrenzt (wie im
  gespeicherten Bild); was außerhalb liegt, kommt nach dem Loslassen. Streifen am Rand des Originals (ältere Dateien,
  Nachtrag 3) bleiben auch in der Vollansicht verborgen; „Ganzes Bild“ holt sie wie bisher zurück.
- Punkt 2 (Doppelklick hebt auf), Punkt 8 (Zuschneiden ist kein Schwärzen) und der Knopf „Ganzes Bild“ bleiben.
- **Widerlegungsbedingung aus F-386 geprüft:** Bei einem kleinen Zuschnitt (Kanten näher als zwei Toleranzen) bliebe
  im Zuschnitt kein Platz, einen neuen Rahmen zu beginnen; er lässt sich aber im abgedunkelten Teil beginnen, den es
  vorher nicht gab. Rahmen-ziehen wird also nicht schlechter.
- **Probe:** `scripts/drehbuecher/zuschneiden-griffe-f386.txt` (virtueller KWin).

## Nachtrag 6 (22.11.2026): Kantenform wählbar, Abriss mit Spalt und Schatten als Voreinstellung (Karte #96)

**Status:** Akzeptiert (Andi 22.11.2026: „Form 4 find ich am besten — einstellbar machen und eine als default nehmen“;
Referenzbild Muster 4: [`bilder/0023-kante-muster-4.png`](bilder/0023-kante-muster-4.png), synthetisch; Einzelheiten von
Claude Opus 5.5).

Anlass: Die Zackenkante war erkennbar, aber nicht schön (Andi, Echtlauf #92).

- **Punkt 6 ersetzt — drei Formen** der Kante, alle Maße in Ansichtspixeln bei Faktor 1, die Mittellinie f(t) quer zur
  Naht entlang der Länge t:
  - `tear` (Abriss, **Voreinstellung**): f(t) = 0,8 · (2,6·sin(2πt/71 + 0,3) + 1,6·sin(2πt/29 + 1,9) +
    1,0·sin(2πt/13 + 4,1) + 0,6·sin(2πt/7,3 + 2,2) + 0,35·sin(2πt/3,1 + 5,7)), abgetastet je 1 px. Fest, ohne Zufall:
    Editor, Renderer und SVG zeichnen dasselbe.
  - `wave` (Welle): f(t) = 4·sin(2πt/32), abgetastet je 2 px.
  - `zigzag` (Zacken): wie bisher ±4, Zahnbreite 16.
- **Gestaltung für alle Formen:** Zwischen den beiden Bildteilen liegt ein Spalt von 10 px (je 5 px beiderseits der
  Mittellinie, beide Kanten mit derselben Form) in hellem Grau; beide Teile werfen einen weichen Schatten in den Spalt,
  der obere bzw. linke kräftiger (Licht von oben links), und ihre Kante ist um 1 px leicht abgedunkelt. Der Schatten
  besteht aus wenigen gestuften, halbdurchsichtigen Bändern derselben Form — keine Weichzeichner-Filter, damit Editor,
  Renderer und SVG-Export gleich zeichnen. Die Schichten (Abstände, Farben, Deckkraft) stehen einmal als Tabelle in
  `document`; diese Tabelle entsteht erst mit der Umsetzung (Karte #96), die ihre Werte am Referenzbild Muster 4
  abstimmt.
- **Bandhöhe:** halbe Bandhöhe einheitlich 11 px (Auslenkung bis 5, Spalt 5, Kante 1) statt 7; Zackenfaktor-Deckel
  (Nachtrag 4), Mindestrest je Seite und Trefferweite von „Naht zurückholen“ folgen der neuen Höhe.
- **Je Dokument gespeichert:** Attribut `s:kante` (`tear` | `wave` | `zigzag`) an `s:ansicht` in der Arbeitsdatei.
  Fehlt es (ältere Dateien) oder ist der Wert unbekannt, gilt `zigzag` — ältere Dateien behalten ihre Form, bekommen
  aber Spalt und Schatten. `FormatVersion` bleibt 1 (ein älterer Leser übergeht das Attribut, `docs/format.md` §11
  Regel 3); Spezifikation §7 und Beispiele ändern sich im selben Commit.
- **Wahl:** Neue Dokumente (Aufnahme, Öffnen eines einfachen PNG/JPEG) bekommen die Form aus der Einstellung
  `[capture] seam` = `tear` (Voreinstellung) | `wave` | `zigzag` (ADR-0018). Im Editor wechselt ein Menüeintrag
  „Seam style“ die Form des Dokuments reihum, als ein Rückgängig-Schritt.
- Das Rechenbudget (`render.Pruefe`) zählt alle Schichten.
- **Schichttabelle** (umgesetzt in `document.KantenSchichten`, Karte #96, d30d26f; abgestimmt an Muster 4; Maße quer zur
  Mittellinie bei Faktor 1, negativ oben/links; in Zeichenreihenfolge):

  | # | von | bis | Farbe | Deckkraft (A) | Zweck |
  |---|---|---|---|---|---|
  | 1 | −5 | 5 | #D3D7DC | 255 | Spalt |
  | 2 | −5 | 2 | #202428 | 0x21 | Schatten oben/links |
  | 3 | −5 | −1 | #202428 | 0x16 | Schatten oben/links |
  | 4 | −5 | −3 | #202428 | 0x2A | Schatten oben/links |
  | 5 | −5 | −4 | #202428 | 0x33 | Schatten oben/links |
  | 6 | 1 | 5 | #202428 | 0x0F | Schatten unten/rechts |
  | 7 | 3 | 5 | #202428 | 0x10 | Schatten unten/rechts |
  | 8 | 4 | 5 | #202428 | 0x17 | Schatten unten/rechts |
  | 9 | −6 | −5 | #000000 | 0x1C | Kantenabdunklung |
  | 10 | 5 | 6 | #000000 | 0x1C | Kantenabdunklung |

  Übereinander ergibt das vom oberen Teil je Pixeltiefe 0–1, 1–2, 2–4 und 4–7 die Deckung 0,47, 0,34, 0,21 und 0,13,
  vom unteren je Tiefe 0–1, 1–2 und 2–4 die Deckung 0,20, 0,12 und 0,06 (Muster: 0,55·e^(−s/3) bzw. 0,25·e^(−s/2)
  über das Pixel gemittelt). Das Muster rechnete mit Faktor 1,4, Spalt 12 und Wellenperiode 36; umgesetzt sind die
  Werte dieses Nachtrags (0,8, 10, 32), die Kante wirkt dadurch etwas ruhiger.
- **Ergänzungen bei der Umsetzung:** Querlagen werden auf 1/100 px gerundet (gleiche Zahlen in Editor, Renderer und
  SVG, kurzes SVG). Die Rettungskopie (ADR-0029) trägt die Form mit (`fileformat.Angaben.Kante`), sonst ginge ein
  Formwechsel bei der Wiederherstellung verloren. `document.Neu` legt `tear` an (wie ein Nutzer ohne
  Einstellungsdatei); Aufnahme, Öffnen und Laden setzen die Form ausdrücklich ohne Rückgängig-Schritt
  (`VorgabeKante`).
- **Ergänzung 22.11.2026 (F-418):** Die Mittellinie wird je Form, Länge und Faktor einmal gerechnet
  (`document.Kantenlinie`), die Schichten verschieben sie nur quer (`Band`). Die Zahlen, die Renderer, Editor und
  SVG-Export bekommen, sind unverändert (ganze Hundertstel, Schichttabelle wie oben; `testdata/format/` unverändert).

## Nachtrag 7 (22.11.2026): Naht durchsichtig, Kanten nur als Papier-Abriss angedeutet (Karte #183)

**Status:** Akzeptiert (Andi 22.11.2026: „die naht sollte transparent sein und nur die kanten angedeutet wie
abgerissen“; Wahl „2, aber etwas mehr (vertikaler) abstand zwischen den beiden hälften“). Referenzbild
`docs/adr/bilder/0023-naht-durchsichtig-muster.png` (mittlere Zeile, 16 px). Ersetzt aus Nachtrag 6 den grauen Spalt
und die Schichttabelle; Formen (Abriss, Welle, Zacken), Speicherung (`s:kante`), Einstellung `[capture] seam` und
Menüeintrag bleiben.

- **Spalt:** 16 px bei Faktor 1 (je 8 px beiderseits der Mittellinie), **durchsichtig** — keine Füllfarbe mehr. Beide
  Kanten haben dieselbe Form. Auslenkung des Abrisses 0,9 statt 0,8 (wie im gewählten Muster).
- **Kanten (Papier-Abriss):** am Rand jedes Bildteils ein heller Fasersaum (rund 1,6 px, Deckung bis 0,75 zum Weiß
  hin) und eine leichte Abdunklung der Kante (0,18·e^(−d/0,8), am unteren bzw. rechten Teil × 0,7); dazu ein feiner
  Schatten der oberen bzw. linken Kante in den Spalt (0,35·e^(−d/2,2), dunkel, halbdurchsichtig).
- **Wie bisher ohne Filter:** Saum, Abdunklung und Schatten sind wenige gestufte Bänder derselben Form, damit Editor,
  Renderer und SVG-Export gleich zeichnen. Die Schwankung des Saums im Muster entfällt, wenn sie sich nicht in allen
  drei Ausgängen gleich darstellen lässt (sie entfällt, siehe Schichttabelle unten).
- **Ausgaben:** PNG und Zwischenablage tragen den Spalt als Alpha (durchsichtig, der Schatten halbdurchsichtig); SVG
  lässt den Spalt leer. JPEG kennt keine Durchsicht: dort wird der Spalt **weiß** gefüllt (wie Papier), der Schatten
  darauf gemischt. Der Editor zeigt Durchsicht als Schachbrett (übliche Darstellung in Bildprogrammen).
- **Kantenform ohne Naht:** Der Menüeintrag „Kantenform“ ist grau, solange das Dokument keine Naht hat (Entscheidung
  Claude, F-426) — sonst gälte ein nur angesehenes Bild als geändert.
- **Folgen:** Bandhöhe, Zackenfaktor-Deckel, Mindestrest und Trefferweite folgen der neuen Breite; das Rechenbudget
  zählt die neuen Schichten; F-407 (Mischung im Editor), F-421 (Schichttabelle als Literal-Soll), F-422 (SVG-Größe)
  und F-430 (Rastern) beziehen sich danach auf die neuen Schichten. `docs/format.md` ändert sich nicht im Format
  (`s:kante` bleibt), nur in der informativen Beschreibung der Darstellung.
- **Schichttabelle (umgesetzt, Karte #183, Commits 86c0b50 und 8a3e0af):** Maße bei Faktor 1 quer zur Mittellinie,
  negativ oben/links, in Zeichenreihenfolge. Der Spalt −8..8 (`document.Spaltband`) ist keine Schicht: Renderer
  radiert ihn (Alpha 0, erst die Spalte aller Nähte, dann alle Schichten), der SVG-Export legt Bild und Objekte unter
  eine Maske, der Editor zeigt ein Schachbrett (`internal/ui/schachbrett.go`, 8 px, #9A9EA3/#7A7E83).

  | # | von | bis | Farbe | Deckkraft (A) | Zweck |
  |---|---|---|---|---|---|
  | 1 | −9,6 | −8 | #FFFFFF | 0x30 | Fasersaum oben/links |
  | 2 | −8,8 | −8 | #FFFFFF | 0x76 | Fasersaum oben/links |
  | 3 | 8 | 9,6 | #FFFFFF | 0x30 | Fasersaum unten/rechts |
  | 4 | 8 | 8,8 | #FFFFFF | 0x76 | Fasersaum unten/rechts |
  | 5 | −10 | −8 | #000000 | 0x08 | Abdunklung oben/links |
  | 6 | −9 | −8 | #000000 | 0x13 | Abdunklung oben/links |
  | 7 | 8 | 10 | #000000 | 0x05 | Abdunklung unten/rechts (× 0,7) |
  | 8 | 8 | 9 | #000000 | 0x0D | Abdunklung unten/rechts (× 0,7) |
  | 9 | −8 | −1 | #101214 | 0x08 | Schatten der oberen/linken Kante in den Spalt |
  | 10 | −8 | −4 | #101214 | 0x10 | Schatten |
  | 11 | −8 | −6 | #101214 | 0x18 | Schatten |
  | 12 | −8 | −7 | #101214 | 0x20 | Schatten |

  Übereinander je Tiefe s ab der Spaltkante: Saum 0–0,8 → 0,56, 0,8–1,6 → 0,19; Abdunklung 0–1 → 0,10, 1–2 → 0,03
  (unten 0,07/0,02); Schatten 0–1, 1–2, 2–4, 4–7 → 0,28, 0,18, 0,09, 0,03. Saum ohne Schwankung (erlaubt oben).
  Auslenkung Abriss 0,9 · 6,15 = 5,54 ≤ 6; `ZackenBandHalb` = 6 + 8 + 2 = 16 (Teiler im Faktor-Deckel 128). Je Naht
  13 Füllungen (12 Schichten und der Spalt); JPEG legt das Bild wie bisher in `fileformat.ExportiereJPG` auf Weiß.
  Offen: F-438 (Saum einer Naht im Spalt der kreuzenden), F-439.

## Nachtrag 8 (22.11.2026): Mindestrest am Rand (F-445) — entschieden, noch nicht umgesetzt

**Status:** Akzeptiert (Entscheidung Claude im Auftrag Andis, 22.11.2026; Umsetzung auf der Karte „Kanten-Reste“).

Anlass: Der durchsichtige Spalt (Nachtrag 7) reicht bis ±8 px um die Naht, mit Saum und Abdunklung bis ±9,6 bzw. ±10 px
und der Auslenkung der Kante bis ±13,5 px · Faktor. Eine Naht nahe am Rand ließ nach dem Mindestrest aus Nachtrag 4
(halbe Bandhöhe, bei Faktor 1 sieben Pixel) kaum sichtbares Bild übrig: ältere Dateien mit 11 bis 15 px Rest behalten
rund 24 % sichtbare Pixel, auch bei 16 px bleiben 2,5 bis 8 px.

- **Entscheidung:** Der Mindestrest je Seite ist die halbe Bandhöhe **plus 8 sichtbare Pixel**, beides mal Zackenfaktor
  (`Faktor·(ZackenBandHalb + 8)`, bei Faktor 1 also 24). Bliebe auf einer Seite weniger, wird das Band wie bisher zum
  Zuschnitt (Nachtrag 3, F-222; Nachtrag 4, Punkt 3).
- **Ältere Dateien bleiben unverändert:** Eine gespeicherte Naht mit kleinerem Rest wird beim Öffnen nicht verschoben
  und nicht in einen Zuschnitt verwandelt (Kernprinzip 1); die Regel gilt für neu gesetzte und verschobene Nähte.
- **Zu wissen:** Jede Naht verdeckt bis zu ±13,5 px · Faktor Bild um sich herum (Spalt, Saum und Auslenkung der Kante
  zusammen). Das ist Folge der Papier-Abriss-Darstellung (Nachtrag 7), kein Fehler, und gehört in die Doku der
  Kantenformen.
- **Umsetzung:** nicht in diesem Nachtrag, sondern auf der Karte „Kanten-Reste“ (Code, `document`-Konstanten,
  Tests, `docs/format.md` unverändert, weil die Datei nichts Neues speichert).
