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
   Abbildung; Teile im Streifen liegen auf der Naht). *(Seit Nachtrag 15: abgeschnitten wie das Bild, in allen Ausgängen.)* Eine Beschriftung, die **ganz** im Verborgenen liegt
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

## Nachtrag 8 (22.11.2026): Mindestrest am Rand (F-445) — umgesetzt am 23.11.2026 (Karte #177)

**Status:** Akzeptiert (Entscheidung Claude im Auftrag Andis, 22.11.2026; umgesetzt auf der Karte #177 „Kanten-Reste“).

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
- **Umsetzung (23.11.2026):** `document.KantenSichtRest` = 8 und `canvas.stauchBand` (`Faktor·(ZackenBandHalb + KantenSichtRest)`),
  Tests `TestStauchenMindestrestJeSeite` (Rest 23 wird Zuschnitt, 24 bleibt Naht), `TestStauchenMindestrestFaktor2` (47 und 48),
  `TestStauchenVorschauGleichErgebnis`; Mutanten `K177-1` und `K177-2`. Trefferweite von „Naht zurückholen“ unverändert
  (halbe Bandhöhe). README (englisch und deutsch) nennen die rund 14 verdeckten Pixel je Seite und den Mindestrest 24.
  `docs/format.md` bleibt unverändert, weil die Datei nichts Neues speichert.

## Nachtrag 9 (23.11.2026): Kreuzende Nähte, Form ohne Naht, Rechenzeit vieler Nähte (Karte #177, F-438, F-439, F-430, F-444, F-408, F-429)

**Status:** Akzeptiert (Entscheidung Claude im Auftrag Andis; umgesetzt).

- **Kreuzende und eng liegende Nähte (F-438):** Der Fasersaum einer Naht liegt nicht im durchsichtigen Spalt einer
  anderen. `document.NaehteBeruehren(naehte, faktor)` ist wahr, wenn es Nähte beider Achsen gibt oder zwei Nähte
  derselben Achse näher als die ganze Bandhöhe (2·`ZackenBandHalb`·Faktor) beieinanderliegen. Nur dann zeichnen alle
  drei Ausgänge in dieser Reihenfolge: Spalte leeren, Fasersaum und Abdunklung (`Schicht.ImSpalt()` falsch), Spalte
  **noch einmal** leeren, Schatten (`ImSpalt()` wahr; er liegt im Spalt der eigenen Naht und bleibt dort, auch im Spalt
  der kreuzenden). Renderer: `zeichner.alles`; SVG-Export: Gruppen `naht-…` (Saum, Abdunklung) unter derselben Maske wie
  die Kacheln und `schatten-…` darüber; Editor: Schachbrett zweimal. Ohne Berührung bleibt jede Ausgabe wie vorher
  (bitgleich, Prüfsummen vor und nach); die zweite Leerung kostet eine Füllung je Naht mehr und wird mitgezählt. Tests:
  `TestKreuzendeNaehteSaumNichtImSpalt`, `TestNaehteBeruehren`, `TestExportSVGNaehteBeruehrenSchatten`; Mutanten
  `K177-3` bis `K177-5`, `K177-8`, `K177-9`. Der Editor-Zweig ist nicht automatisch geprüft (Zeichnen ohne Fenster nicht
  testbar); Sichtprobe über `scripts/probe.sh` mit dem Drehbuch `stauchkante-formen-96.txt` (Foto 03).
- **Form ohne Naht (F-439) — so lassen:** Die Kantenform gehört zum Dokument, nicht zu den Nähten. Sind alle Nähte
  zurückgeholt, bleibt die gewählte Form; die nächste Naht erscheint in ihr, nicht in der aus `[capture] seam` (die gilt
  für neu entstehende Dokumente). Das ist, was die Person zuletzt gewählt hat, und es lässt Rückgängig und Wiederholen
  der Form unberührt; der Menüeintrag „Seam style“ bleibt ohne Naht grau (F-426). Test `TestKanteBleibtOhneNaehte`.
- **Rechenzeit (F-430, F-429):** Schichten lassen sich nicht bitgleich bündeln (verschiedene Farben und Deckkräfte
  übereinander); auf die sichtbare Ansicht beschneiden ist nicht nötig (`Bild` rastert nur die Ansicht). Bitgleich
  schneller sind: `Bild` zählt die Kosten der Füllungen nicht mit (nur `Aufwand` und der Trockenlauf von `Pruefe`
  brauchen sie), die Rasterschleifen nehmen `min`/`max` statt `math.Min`/`math.Max`, ein Pixel (0,0,0,0) ohne Deckung
  wird übersprungen. 120 Abriss-Nähte auf 32768×2100: 11,8 s auf 6,5 s (Benchmark `BenchmarkNaehteAbriss120`, ein Kern,
  Entwicklungsrechner, 23.11.2026). Das Ziel „unter 4 s“ ist damit **nicht** erreicht; es bleibt bei der
  Rasterarbeit je Füllung (rund 7 ns je Pixel, 13 Füllungen je Naht über die ganze Länge). Das Zeitversprechen des
  Rechenbudgets (Nachtrag 3 von ADR-0020) lautet deshalb **rund 14 s** statt 10 s: ein Dokument knapp unter allen drei
  Grenzen zugleich (90 Nähte, 59 × „W“ Grad 4096, bildgroße Rahmen bis zum Budget) rastert in rund 14 s
  (`BenchmarkNaehteAlleGrenzen`). *Seit Karte #224 (23.11.2026, F-589, ADR-0020 Nachtrag 3 mit Ergänzungen) lautet das
  Zeitversprechen **rund 21 s**: `GlyphBudget` 3e9 statt 1e9 (177 × „W“ statt 59), und gemessen wird mit dem teuersten
  Glyphenmuster („@“ im Grad 74, `BenchmarkNaehteAlleGrenzenKleineGlyphen/Klammeraffe74`); mit den „W“ sind es rund
  12 s. Mit Systemschriften gilt es erst seit ADR-0020 Nachtrag 13 (Karte #238, F-601: Glyphen aus Systemschriften
  zählen mit dem Faktor 2,5), gemessen mit eingebauten und Systemschriften.*
- **Grenze der Nahtzahl (F-444, F-408):** `KantenBudget` bleibt 1e10. `Pruefe` nimmt bei Abriss-Nähten auf 8192 px Länge
  rund 364, auf 32768 px rund 91 an (seit F-438 mit der zweiten Leerung; vorher 392 und 98); mehr lehnt der Editor beim
  Öffnen ab, Speichern bricht nie ab. Das Heben der Grenze auf 600 Nähte bei 8192 px hieße rund 1,6e10 und (hochgerechnet) rund 9 s
  Rasterzeit für Kanten allein; das wäre die Wartezeit einer präparierten Datei ohne echten Nutzen (reale Dateien
  haben wenige Nähte). Die Grenze ist hier dokumentiert und in `TestNahtZahlGrenze` festgehalten.
- **Editor und Renderer mischen unterschiedlich (F-407, Messung):** Mit `scripts/probe.sh` (Zoom 100 %, Skala 1)
  weicht der Fasersaum im Editor vom gerenderten PNG um im Mittel 22 von 255 je Kanal ab (bis 43), der Editor ist
  heller. Die Abweichung deckt sich mit linearer statt sRGB-Mischung der Deckkraft durch Gio (gerechnet mit linearer
  Mischung nur 5 im Mittel). Gelöst ist das nicht; es bleibt als eigenes Finding (siehe Register).

## Nachtrag 10 (23.11.2026): SVG-Export der Kanten schlank (F-422, Karte #182)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #182). Betrifft nur den SVG-Export (`fileformat.ExportiereSVG`) und
die Hilfen in `document`; Renderer, Editor und Arbeitsdatei (`snAP`, `docs/format.md`, `FormatVersion`) bleiben wie sie sind.

**Anlass:** Der Export schrieb je Achse mit Naht 13 Pfade mit je einem Punktpaar je Pixel (Abriss): 1920×1080 mit einer
Naht kostete +300 kB (Spalten-Naht) bis +557 kB (Zeilen-Naht), bei Zackenform +28 bis +54 kB. Die Voreinstellung (Abriss)
trifft jeden gestauchten Export. Ziel laut F-422: höchstens 100 kB je Naht.

**Entscheidung — drei Maßnahmen, die zusammen wirken:**
1. **Gemeinsame Linie, weniger Stützpunkte.** Alle Schichten einer Achse sind quer verschobene Kopien derselben
   Mittellinie. `Kantenlinie.Vereinfacht(Toleranz)` (Douglas-Peucker über die Querlage) streicht Stützpunkte, die höchstens
   `fileformat.KantenToleranz` = 0,4 px quer neben der Verbindung ihrer Nachbarn liegen (Abriss 1920 px: 792 von 1921
   Punkten; Welle rund 38 %; der Zacken hat nur seine Ecken und bleibt). Spalt und Schichten benutzen dieselbe vereinfachte
   Linie und bleiben deshalb **genau** deckungsgleich; gegenüber dem Renderer weicht jede Kante um höchstens 0,4 px ab,
   also unter einem halben Pixel. Seit der Sitzungs-QA vom 23.11.2026 (B-10) wächst die Toleranz mit dem Faktor (`KantenToleranzFuer(faktor)` = 0,4·Faktor; die Ansicht verkleinert um den Faktor, es bleibt unter einem halben Bildschirmpixel): 3840×2160, Faktor 2, eine Abriss-Naht +40,5 kB (Spalten) bzw. +85,2 kB (Zeilen) statt +64,6/+137,4 kB (`TestSVGNahtZuwachs4K`, Grenze 100 kB).
2. **Stücke statt überlappender Schichten.** `document.KantenFlach()` zerlegt die zwölf Bänder je Farbe an allen
   ihren Grenzen in überlappungsfreie Stücke; jedes trägt die Deckkraft, die alle Bänder darüber ergäben
   (1 − Π(1 − aᵢ), auf 1/255 gerundet). Gleichfarbige Bänder mischen so dasselbe Bild (Test mit Untergründen schwarz, grau,
   weiß: höchstens eine Stufe Abweichung). Es gibt nur fünf verschiedene Breiten (0,8 / 1 / 2 / 3 / 16 px).
3. **Jede Breite einmal.** In `<defs>` steht je Achse und Breite ein Pfad ohne Füllung `band-<Achse>-<Breite in
   Hundertsteln>` (das Band von 0 bis zur Breite); jedes Stück ist ein `<use>` darauf mit Versatz (`x` bzw. `y`, Hundertstel
   mal Faktor), Farbe und Deckkraft. Die Gruppen `naht-…` und `schatten-…` (F-438), der Spalt-Pfad `spalt-…`, die Maske
   und die `<use>` je Naht bleiben, wie sie waren; Stücke im Spalt (Schatten) liegen in `schatten-…`.

**Ergebnis** (Test `TestSVGNahtZuwachs`, einfarbiges Bild 1920×1080, Zuwachs gegenüber demselben Bild ohne Naht, einmal die
Spalten-Naht über 1080 px, einmal die Zeilen-Naht über 1920 px): Abriss 300 kB / 557 kB → 44 kB / 85 kB; Welle 139 kB /
260 kB → 19 kB / 36 kB; Zacken 28 kB / 54 kB → 10 kB / 20 kB. Grenze im Test: 100 kB.

**Geprüft:** `TestExportSVGKante` (Pfade, Versatz, Farbe, Deckkraft je Stück; Abweichung gegen die volle Linie des Renderers
höchstens `KantenToleranz`), `TestKantenlinieVereinfacht`, `TestKantenFlachGleichKantenSchichten`, `TestSVGNahtFaktor2`;
Mutanten `K182-1` bis `K182-4` in `scripts/mutanten/kante.json`.

**Verworfen:** Relative Pfadbefehle (rund −25 % je Punkt, aber schlechter lesbar und nicht nötig), Masken und
Clip-Pfade statt Bändern (kleinste Daten, aber je Schicht und Naht eine Maske in jedem Betrachter; Darstellung
schwerer zu prüfen).

## Nachtrag 11 (23.11.2026): Fasersaum im Editor wie im Renderer, Rechenzeit vieler Nähte (F-407, F-430, F-499, Karte #212)

**Status:** Akzeptiert (Claude Sonnet 5.5 mit Claude Opus 5.5, Karte #212).

- **Fasersaum im Editor (F-407):** Gio mischt halbdurchsichtige Flächen linear, `render` in sRGB; im Fasersaum zeigte der
  Editor im Mittel 22 (höchstens 43) von 255 je Kanal zu hell (Messung bei Zoom 100 %, Skala 1, Versatz genau
  (226,200): keine Skalierung im Spiel). Jetzt mischt `render.KantenOhneSchatten` Fasersaum und Abdunklung (auch die zweite
  Leerung bei berührenden Nähten) ins Bild der Leinwand (`editor.bildNeu`); Gio legt nur noch das Schachbrett des Spalts
  und darüber den Schatten. Nach der Änderung weicht das Foto im Mittel um 0,02 ab, höchstens 2 je Kanal, kein Pixel über 2.
  Der Stand (`kantenBild`: Nähte, Form, Faktor, Werkzeug) löst ein Neubauen aus, auch beim Wechsel der Kantenform.
  Beim Werkzeug Zuschneiden (Vollansicht) bleibt es beim alten Weg in Gio. Folge: Objekte liegen im Editor über dem
  Fasersaum, nicht darunter wie in Export und Renderer (nur in den wenigen Pixeln am Rand einer Naht).
- **Rechenzeit (F-430):** Der Renderer merkt sich die Deckung großer Füllungen (`masken.go`) unter ihren genauen
  Pfadkoordinaten; Nähte einer Achse sind querverschobene Kopien, die Füllungen der übrigen Nähte spielen die Maske ab.
  Das Abspielen teilt sich in Spaltenstreifen auf höchstens 8 Kerne (jedes Pixel hängt nur von sich ab). Dazu schnelleres
  Einmischen auf deckendem Grund. 120 Abriss-Nähte auf 32768×2100, Bild bitgleich (`TestNahtPruefsumme`): 6,46 s auf
  4,4 s mit einem Kern, 3,0 s mit zwei, 2,5 s mit allen Kernen des Entwicklungsrechners. `BenchmarkNaehteAlleGrenzen`
  (ein Kern): rund 11 s statt rund 14 s. *Seit Karte #224 (dreifaches `GlyphBudget`) rund 12 s mit den „W“, mit dem
  teuersten Glyphenmuster rund 21 s — das Zeitversprechen, siehe Nachtrag 9 und ADR-0020 Nachtrag 3; mit
  Systemschriften seit ADR-0020 Nachtrag 13.*
- **Test (F-499):** `TestNahtAblaufZweiteLeerung` und `TestKantenOhneSchatten` (Sollpixel im Spalt bei kreuzenden Nähten).
- **Grenze, bewusst hingenommen (Sitzungs-QA 23.11.2026, Kritiker A-4/B-11):** `KantenOhneSchatten` läuft in `bildNeu`
  synchron in der Ereignisschleife (jede Schwärzung, Rückgängig/Wiederholen, Kantenform, Wechsel zu oder von
  Zuschneiden), mit eigenem Maskenspeicher je Aufruf. Gemessen auf 16 Kernen: 1920×1080 mit einer Naht 3 ms,
  3840×2160 mit zehn Nähten 26 ms, 8000×3000 mit 20 Nähten 74 ms, an der Nahtgrenze (32768×2100, 91 Nähte) rund 1,2 s
  zusätzlich zu `Ansichtsbild`. Im Alltag unkritisch; nur präparierte Grenzdokumente lassen das Fenster je Schwärzung
  gut eine Sekunde stocken. Abhilfe, falls nötig: den Maskenspeicher am Editor halten, solange Größe, Form und Faktor
  gleich bleiben, oder nur das geänderte Rechteck neu mischen (F-525).

- **Überholt durch ADR-0020 Nachtrag 14 (F-520):** Gio mischt seither ebenfalls in sRGB; das Vorrastern des Fasersaums bleibt, ist aber nicht mehr nötig.

## Nachtrag 12 (24.11.2026): Text an der Naht — Grenze bis zur zeilenweisen Lösung (F-739, F-740)

**Status:** Überholt durch Nachtrag 13 (Karte #281). Zuvor: Akzeptiert (Claude Opus 5.5 beim Sprint-Abschluss 12, im Auftrag Andis: „arbeite selbstständig“). Anlass:
dritter Verify-Lauf V3-1 und V3-2. Ausnahmepfad [S] P3 (docs/qa/sprint-abschluss.md), Lösung mit Karte #281. Andi hat
die Entscheidung am 24.11.2026 an Claude gegeben („entscheide du“): Die Grenze gilt bis zur Umsetzung von #281.

1. **Punkt 7 stimmt für Text nicht wörtlich:** Text wird nicht gestaucht. Seine Zeilen stehen starr ab dem
   abgebildeten Anker; liegt der Anker in einem Streifen, fällt er auf die Naht und die Zeile steht dahinter. Solange
   ein Teil von Tinte oder Schild sichtbar bleibt (`ObjektSichtbar`, `TextSichtbar`), zeichnen Editor, Renderer und
   SVG-Export den **ganzen** Text so; ein Text, der nur wenige Pixel aus einem Streifen ragt, steht also lesbar hinter
   der Naht. Ganz verborgener Text fällt weg (Punkt 7), eine Sprechblase mit nur sichtbarer Spitze zeigt nur das
   Dreieck (ADR-0021 Nachtrag 14 Punkt 4, F-735).
2. **Warum vorerst so:** Alle Ausgänge zeigen dasselbe, was der Editor zeigt — kein verdecktes Leck. Stauchen ist
   kein Schwärzen (Kernprinzip 2 gilt nur für Schwärzen und Verpixeln); wer einen Text loswerden will, löscht oder
   schwärzt ihn. Die saubere Lösung (Zeilen einzeln prüfen oder Text an der Naht beschneiden, in allen drei
   Ausgängen gleich) ist eigene Arbeit für einen Sprint, Karte #281.
3. **Texteingabe (F-740):** Ein Klick mit dem Text- oder Sprechblasen-Werkzeug auf das sichtbare Dreieck einer
   Blase mit verborgenem Schild öffnet deren Eingabe; der Text erscheint erst, wenn das Schild aus dem Streifen
   wächst. Gehört zur Lösung unter Karte #281.

## Nachtrag 13 (24.11.2026): Text an der Naht abgeschnitten wie das Bild (F-739, F-740)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #281). Löst Nachtrag 12 ab; dessen Grenze gilt nicht mehr.

1. **Eine Rechnung:** `document.Stuecke` zerlegt die Ansicht in die sichtbaren Bereiche zwischen Zuschnittkanten und
   Nähten; in jedem gilt Ansicht = Original + Versatz. `Dokument.TextStuecke(t)` sind die Bereiche, die Tinte oder Schild
   von t berühren (dieselbe Reichweite wie `TextSichtbar`).
2. **Renderer und Editor** zeichnen die Glyphenzeilen eines Texts je Stück, um dessen Versatz verschoben; reicht der
   Text über eine Naht (mehr als ein Stück), beschneiden sie jedes Stück auf seinen Bereich (Renderer: `zeichner.schnitt`
   je Glyphe, Editor: `clip.Rect`). Text verhält sich damit an der Naht wie das Bild darunter: Was im Streifen liegt,
   fehlt, der Rest steht an seinem Platz. Der SVG-Export schnitt Objekte schon immer kachelweise so ab und bleibt
   unverändert; alle drei Ausgänge zeigen jetzt dasselbe. Liegt der Text in genau einem Bereich, steht er wie bisher.
3. **Schild, Unter- und Durchstrich** bleiben über ihre Ecken abgebildet; für achsenparallele Rechtecke ist das genau
   dasselbe wie Abschneiden. Das Dreieck einer Sprechblase bleibt über seine Ecken abgebildet (F-695, Karte #275).
4. **Texteingabe (F-740):** Liegen Glyphen und Schild eines Texts ganz in einem Streifen (nur das Dreieck einer
   Sprechblase ist zu sehen), beginnt weder Klick (Text- oder Sprechblasen-Werkzeug), Doppelklick, Enter noch der
   Menüeintrag „Text bearbeiten“ eine Eingabe (`Sitzung.TextBearbeiten`, `KannTextBearbeiten`); der Text wird gewählt,
   die Statusleiste sagt „This text lies in a compressed area. Restore cut-out areas (Image menu) to edit it.“
   (`AblehnungTextVerborgen`). Blindes Tippen entfällt.
5. **[S]:** Unverändert gilt, dass Stauchen kein Schwärzen ist; Text im Streifen bleibt in der Arbeitsdatei. In
   Renderer-Bild, Editor und SVG-Export steht vom Text jetzt aber nichts mehr, was im Streifen liegt (bisher konnte
   ein knapp herausragender Text ganz hinter der Naht stehen).

Tests: `internal/document/stuecke_test.go`, `internal/render/text_naht_test.go`,
`internal/ui/text_naht_headless_test.go`, `internal/canvas/text_verborgen_test.go`; Mutanten
`scripts/mutanten/text-naht.json` (N1–N10); Sichtprobe `scripts/drehbuecher/text-naht-281.txt`.

## Nachtrag 14 (24.11.2026): Schräge Kanten über einer Naht wie im SVG-Export (F-695)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #275, Abschluss-Kritik Sprint 12 A-4).

1. Das Dreieck einer Sprechblase (gefüllt mit Schild oder als Schenkel) und die Verbindungslinie einer Lupe zeichnen
   Renderer (`zeichner.jeStueck`) und Editor (`editor.jeStueck`) je sichtbarem Bereich (`document.Stuecke`, Nachtrag
   13), um dessen Versatz verschoben und auf ihn beschnitten. Eine schräge Kante, die eine Naht kreuzt, ist dort
   abgeschnitten wie im SVG-Export (Kacheln) und im Treffer (`Text.Enthaelt` im Original); vorher zog der Renderer sie
   zwischen den abgebildeten Ecken neu (841 von 896 Pixeln abweichend). ADR-0021 Nachtrag 13 Punkt 2 und ADR-0033
   Punkt 4 („eine Rechnung wie die Kacheln des SVG-Exports“) stimmen damit auch über einer Naht.
2. Die Spitze eines Dreiecks, die in einem Streifen liegt, fällt so weg (bisher lag sie auf der Naht). Die
   Kantenglättung an Bereichskanten rundet um höchstens eine Stufe anders (anderer Ausschnittursprung im Rasterkern).
3. Pfeile bleiben vorerst über ihre 48 Stützpunkte abgebildet (Punkt 7: mit dem Bild gestaucht), der SVG-Export schneidet
   sie kachelweise ab; der Unterschied ist als F-745 erfasst (gelöst mit Nachtrag 15). Rechtecke (Kasten, Schild, Rahmen der Lupe, Striche)
   sind von beiden Rechnungen gleich abgebildet.

Tests: `internal/render/schraege_naht_test.go`, `TestEditorSchneidetSchraegeKantenAnDerNaht`
(`internal/ui/text_naht_headless_test.go`); Mutanten N11–N15 in `scripts/mutanten/text-naht.json`, S15/S34 nachgezogen.

## Nachtrag 15 (24.11.2026): Beschriftungen über einer Naht überall abgeschnitten (F-745)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #284). Ersetzt in Punkt 7 den Satz „Eine Beschriftung, die über eine
Naht läuft, wird mit dem Bild gestaucht abgebildet“.

1. **Eine Regel für alle Ausgänge:** Eine Beschriftung, die über eine Naht reicht, ist dort abgeschnitten wie das Bild
   darunter; was im Streifen liegt, fehlt, der Rest steht um den Versatz seines Bereichs verschoben. So zeigte es der
   SVG-Export (Kacheln) schon immer, so rechnet der Treffer (im Original). Renderer und Editor zeichnen dafür Pfeil und
   Schritt-Nummer jetzt ebenfalls je sichtbarem Bereich (`jeStueck`, wie Text, Dreieck und Lupen-Linie seit den
   Nachträgen 13 und 14). Eine Nummer, deren Mitte im Streifen liegt, steht nicht mehr ganz auf der Naht, sondern zeigt
   nur ihren herausragenden Teil.
2. **Kasten, Schild, Striche und Rahmen** bleiben über ihre Ecken abgebildet: Für achsparallele Rechtecke ist das
   dasselbe wie Abschneiden. Bei einem Pfeil unterscheiden sich die beiden Rechnungen nur auf der Strecke über dem
   Streifen, also im Kantenband der Naht (Mutanten N16/N18 äquivalent).
3. **Auswahlrahmen, Griffe und Hilfslinien** des Editors bilden weiter die Ecken ab; sie sind Bedienung, keine Ausgabe.

Tests `TestPfeilUndNummerUeberNaht` (`internal/render/schraege_naht_test.go`) und
`TestEditorSchneidetSchraegeKantenAnDerNaht`; Mutanten N16–N19 in `scripts/mutanten/text-naht.json`.

