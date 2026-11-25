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
   Text über eine Naht, beschneiden sie jedes Stück auf seinen Bereich (Renderer: `zeichner.schnitt` je Glyphe, Editor:
   `clip.Rect`) — seit der Sitzungs-Kritik (A-1) auch bei nur einem Stück, sonst stand ein Text, der im Streifen endet,
   mit diesem Teil hinter der Naht. Text verhält sich damit an der Naht wie das Bild darunter: Was im Streifen liegt,
   fehlt, der Rest steht an seinem Platz. Der SVG-Export schnitt Objekte schon immer kachelweise so ab und bleibt
   unverändert; alle drei Ausgänge zeigen jetzt dasselbe.
3. **Schild, Unter- und Durchstrich** bleiben über ihre Ecken abgebildet; für achsenparallele Rechtecke ist das genau
   dasselbe wie Abschneiden. Das Dreieck einer Sprechblase zeichnen Renderer und Editor seit Nachtrag 14 je Bereich.
4. **Texteingabe (F-740):** Liegen Glyphen und Schild eines Texts ganz in einem Streifen (nur das Dreieck einer
   Sprechblase ist zu sehen), beginnt weder Klick (Text- oder Sprechblasen-Werkzeug), Doppelklick, Enter noch der
   Menüeintrag „Text bearbeiten“ eine Eingabe (`Sitzung.TextBearbeiten`, `KannTextBearbeiten`); der Text wird gewählt,
   die Statusleiste sagt „This text lies in a compressed area. Restore cut-out areas (Image menu) to edit it.“
   (`AblehnungTextVerborgen`). Blindes Tippen entfällt.
5. **[S]:** Unverändert gilt, dass Stauchen kein Schwärzen ist; Text im Streifen bleibt in der Arbeitsdatei. In
   der Darstellung von Renderer-Bild, Editor und SVG-Export steht vom Text jetzt nichts mehr, was im Streifen liegt
   (bisher konnte ein knapp herausragender Text ganz hinter der Naht stehen). Die SVG-Datei trägt einen teilweise
   sichtbaren Text aber weiter ganz im Klartext (`<text>`, je Kachel per `<use>` beschnitten); das ist bewusst so
   (Ausnahmepfad [S] P3, Sitzungs-Kritik A-7, F-760): Stauchen ist kein Schwärzen, Text in Pfade je Kachel zu
   zerlegen wäre ein großer Umbau ohne Schutzgewinn gegenüber der Arbeitsdatei. Ganz verborgener Text fehlt weiter
   (F-66, F-181).

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

1. **Eine Regel als Ziel:** Eine Beschriftung, die über eine Naht reicht, ist dort abgeschnitten wie das Bild
   darunter; was im Streifen liegt, fehlt, der Rest steht um den Versatz seines Bereichs verschoben. So zeigte es der
   SVG-Export (Kacheln) schon immer, so rechnet der Treffer (im Original). Renderer und Editor zeichnen dafür die
   Schritt-Nummer jetzt ebenfalls je sichtbarem Bereich (`jeStueck`, wie Text, Dreieck und Lupen-Linie seit den
   Nachträgen 13 und 14). Eine Nummer, deren Mitte im Streifen liegt, steht nicht mehr ganz auf der Naht, sondern zeigt
   nur ihren herausragenden Teil.
2. **Pfeil, Kasten, Schild, Striche und Rahmen** bleiben über Ecken bzw. Stützpunkte abgebildet: Für achsparallele
   Rechtecke ohne Rundung ist das dasselbe wie Abschneiden. Beim Pfeil unterscheiden sich die Rechnungen auf der Strecke
   über dem Streifen; liegen die 48 Stützpunkte weiter auseinander als das Kantenband breit ist (lange Pfeile), bei
   breiten Strichen (ab 32 px) und an einer Spitze im Streifen auch daneben, ebenso ein Kasten mit Eckradius, dessen
   Kante im Streifen liegt (Sitzungs-Kritik A-4, A-5, C-4; F-748 und F-749, gelöst mit Nachtrag 17). Ein Pfeil je Bereich hätte über
   vielen Nähten (bis `MaxStreifen`) jedes Mal den ganzen Pfeil aufgebaut; der Prüflauf beim Öffnen (F-136) wurde in
   der CI dadurch zu langsam. Glyphkosten eines Texts zählt der Trockenlauf je sichtbarer Zeile einmal, nicht je
   Bereich.
3. **Auswahlrahmen, Griffe und Hilfslinien** des Editors bilden weiter die Ecken ab; sie sind Bedienung, keine Ausgabe.

Tests `TestPfeilUndNummerUeberNaht` (`internal/render/schraege_naht_test.go`) und
`TestEditorSchneidetSchraegeKantenAnDerNaht`; Mutanten N17 und N19 in `scripts/mutanten/text-naht.json`.

## Nachtrag 16 (25.11.2026): Text und Schritt-Nummer über vielen Nähten zeilenweise, Grundkosten je Füllung (F-747, F-750)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #285). Ergänzt die Nachträge 13 und 15; das Bild bleibt dasselbe.

1. **Zeilenweise statt je Bereich:** Die sichtbaren Bereiche sind das Kreuzprodukt waagerechter und senkrechter
   Abschnitte, zwischen zwei Nähten ist die Ansicht eine Verschiebung um ganze Pixel (`document.Abschnitte`,
   `TextAbschnitte`, `Beschneide`, `AbschnitteIn`). Renderer und Editor laufen deshalb einmal über die Zeilen eines
   Texts; jede Zeile sucht die senkrechten Abschnitte ihrer Höhe per Binärsuche. Vorher lief jeder Bereich über alle
   Zeilen: 20 Texte à 10 000 Zeilen über 1024 Zeilenstreifen prüfte `Pruefe` in 5,2 s statt 22 ms, der Editor brauchte
   5,3 s je Bild (Sitzungs-Kritik A-2, F-747). Jetzt rund 30 ms bzw. 4 ms.
2. **Jede Glyphe einmal:** Der Renderer rastert jede Glyphe einmal und mischt sie je Bereich um dessen Versatz
   verschoben und auf ihn beschnitten ein (`render.textBaender`); der Editor zeichnet jede Zeile einmal im Original auf
   und spielt sie je Bereich verschoben ab, ebenso alles, was er über `jeStueck` zeichnet (Nummer, Dreieck der
   Sprechblase, Lupen-Linie). Über 1023 Spaltenstreifen rasterte der Renderer vorher 76 650 statt 1 740 Glyphen (Bild
   4,6 s statt 33 ms), gezählt war jede nur einmal (A-3, F-750). Die Ziffern der Schritt-Nummer zählen damit wie Text
   einmal statt je Bereich; der Kreis bleibt je Bereich eine Füllung.
3. **Grundkosten je Füllung:** Jede Füllung kostet unabhängig von Fläche und Kanten rund 5,5 µs (Rasterizer, Pfad,
   Ausschnitt). Über vielen Nähten zeichnet jedes schräge Objekt je Bereich eine eigene Füllung: 1000 Sprechblasen über
   4096 Kacheln rasterten in 26 s bei 63 % von `KantenBudget`. Das Kostenmodell zählt deshalb je Füllung
   `aufrufGewicht` = 12 000 Einheiten bei den Kanten. Im Alltag sind es wenige Tausend Füllungen (unter 1 % von
   `KantenBudget`); Sprechblasen über 4096 Kacheln passen bis rund 183 unter die Grenze und rastern dort in rund 4,6 s
   (`BenchmarkSprechblasenKantenGrenze`, ein Lauf, 25.11.2026). Das Zeitversprechen von rund 21 s (ADR-0020
   Nachtrag 12) gilt damit auch für dieses Muster.

Tests `internal/render/naht_bereiche_test.go`, `TestEditorBildUeberVielenNaehten` (`internal/ui/naht_bereiche_test.go`),
Sollwerte in `TestKostenmodellSollwerte`; Mutanten N1–N5, N17, N20 nachgezogen, N3b und NA1–NA5 neu
(`scripts/mutanten/text-naht.json`), QA3-B12b und S238-4 nachgezogen (`scripts/mutanten/schrift.json`).

## Nachtrag 17 (25.11.2026): Pfeil und runder Kasten abgeschnitten, Eingabe am verborgenen Textende (F-748, F-749, F-751, F-752)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #286). Ersetzt Nachtrag 15 Punkt 2 für Pfeil und Kasten mit Eckradius.

1. **Pfeil:** Renderer und Editor zeichnen den Strich je sichtbarem Bereich, aber nur mit den Strecken in dessen Nähe
   (`document.StreckenStuecke`: jede der 48 Strecken samt halber Strichbreite den Bereichen zugeordnet, die sie berührt;
   Aufwand Strecken × berührte Bereiche). Der Renderer füllt je Bereich diese Strecken und ihre runden Gelenke, um den
   Versatz verschoben und beschnitten (`render.strichJeStueck`); der Editor zeichnet den Pfeil einmal auf und spielt
   ihn in diesen Bereichen ab (`editor.jeStueckIn`). Die Spitze läuft über `jeStueck`. Damit ist der Pfeil über einer
   Naht abgeschnitten wie im SVG-Export, auch bei langen Pfeilen, breiten Strichen und einer Spitze im Streifen
   (Sitzungs-Kritik A-4, C-4). Über 1023 Spaltenstreifen trägt ein bildbreiter Pfeil rund 1024 Bereiche mit je ein bis
   zwei Strecken, nicht 1024 ganze Pfeile (F-136; `TestPfeilUeberVielenNaehtenNurBeruehrteBereiche`).
2. **Kasten mit Eckradius:** je Bereich abgeschnitten (Füllung über `jeStueck`, Strich über `strichJeStueck` bzw. die
   Aufzeichnung im Editor), statt über die abgebildeten Ecken neu gerundet (A-5). Ein Kasten ohne Rundung bleibt über
   die Ecken abgebildet; dort ist das dasselbe.
3. **Eingabe am verborgenen Textende:** Während der Texteingabe zeichnet der Editor den bearbeiteten Text ungeschnitten
   ab dem abgebildeten Anfang jeder Zeile, wie die Einfügemarke ihn schon immer setzte. Liegt das Textende im Streifen,
   sieht man so, was man tippt (A-9). Nach der Eingabe ist der Text wieder beschnitten; Renderer, Export und Datei
   ändern sich nicht. Gesperrt wird die Eingabe nur, wenn Schrift und Schild ganz verborgen sind (Nachtrag 13).
4. **Editor-Tests:** `TestEditorKreuzendeNaehteMitZuschnitt` vergleicht über Spalten- und Zeilennaht mit Zuschnitt,
   auch mit Deckkraft unter 100 %, für Text, Nummer, Sprechblase, Lupe, Pfeil und runden Kasten (A-6).
5. Beim Vergleich fiel auf, dass der Rasterkern im Festkomma-Weg je nach Ausschnitt entlang langer Kanten driftet; er
   rechnet seitdem immer im Gleitkomma-Weg (ADR-0028 Nachtrag 1, F-764).

Tests `internal/render/pfeil_kasten_naht_test.go`, `TestEditorKreuzendeNaehteMitZuschnitt`
(`internal/ui/naht_kreuz_headless_test.go`), `TestEditorTextEndetImStreifen` (Eingabe); Mutanten P1–P8 in
`scripts/mutanten/text-naht.json`, N12 nachgezogen; `render.text` läuft über `textBaender` (ein Bereich), das Feld
`zeichner.schnitt` entfällt.

## Nachtrag 18 (25.11.2026): Editor und Renderer über vielen Nähten ohne ungezählte Wiederholung (F-765, F-766)

**Status:** Akzeptiert (Claude Opus 5.5, Karten #289, #290). Ergänzt Nachträge 16 und 17.

1. **Runder Kasten im Editor (F-765):** Die Füllung bleibt je Bereich des Rechtecks (`jeStueck`). Den Strich zeichnet der
   Editor in denselben Bereichen wie der Renderer, nämlich nur in denen, die der Umriss berührt
   (`document.StreckenStuecke` über `document.RundesRechteck`, dieselbe Rechnung wie `render.rundesRechteck`), und dort
   nur die Strecken in der Nähe:
   jede Strecke ein Trapez zwischen den Gehrungslinien ihrer Ecken (der Umriss knickt höchstens um 7,5°), selbst auf den
   Bereich beschnitten (Sutherland-Hodgman) und verschoben, alle zusammen **ein** Pfad mit **einer** Füllung
   (`editor.strichJeStueck`). Nachbarn teilen sich die Gehrungslinie und überlappen nicht, sonst addierte Gio die
   Kantenglättung doppelt. *Berichtigt durch Nachtrag 19:* Die Form deckt sich mit dem Renderer nur, solange der halbe
   Strich nicht breiter ist als der Eckradius; darüber kreuzten sich die inneren Ecken, die Trapeze wurden Schleifen und
   der Editor zeigte ein Loch (F-783). Eine Füllung mit eigenem Schnitt je Bereich kostete mit llvmpipe rund 40 µs je Bereich
   (200 Kästen über 4096 Kacheln: 23,7 s je Bild vorher, 3,3 s mit Schnitt je Bereich, 0,5 bis 0,7 s als ein Pfad).
   Ohne Naht sieht der Strich aus wie Gios Strich um das abgerundete Rechteck (`TestEditorRunderKastenStrichWieGio`).
2. **Text im Editor (F-765):** Jede Zeile wird in Läufen von Glyphen aufgezeichnet, deren Ursprünge höchstens vier
   seitliche Hüllen auseinanderliegen (`textsatz.zeileInLaeufen`); jeder Lauf spielt nur in den Spalten, die seine Tinte
   berührt (erste Glyphe minus, letzte plus seitliche Hülle). Rechts endet eine Zeile so mit ihrer letzten Glyphe, und
   eine lange Zeile spielt je Spalte nur ihre Glyphen in der Nähe. Über 1023 Spaltenstreifen: 4 Texte mit einer langen
   und 399 kurzen Zeilen 4,6 s → 0,2 s, 5 Texte 6×60 „@“ Grad 74 2,8 s → 0,6 s je Bild mit llvmpipe.
3. **Messung:** `TestEditorGanzesBildUeberVielenNaehten` misst ein ganzes Bild über die Zeichenfläche ohne Fenster
   (Grenze 2 s, großzügig für llvmpipe), nicht nur den Befehlsaufbau.
4. **Glyphen im Renderer (F-766):** `textBaender` mischt jede Glyphe in **einem** Aufruf in alle Bereiche ein, die sie
   berührt (`glyphBaender`): je Spalte und Zeile der Glyphe die Stelle in der Ansicht, dann Zeile für Zeile mit derselben
   Formel wie `glyphMischen` — dasselbe Bild Bit für Bit (`TestGlyphenGebuendeltBitgleich`), ein Aufruf je Glyphe statt
   einer je Bereich (`TestGlyphEinAufrufUeberVielenNaehten`; 270 „@“ über 1023 Spaltenstreifen vorher 7880 Aufrufe).
   Das Kostenmodell bleibt: Die Pixel einer Glyphe mischt der Renderer ohnehin, nur die Grundkosten je Aufruf fallen
   weg. Die Messung der Kritik (4,5 s statt 1,1 s, hochgerechnet 46 s) enthielt rund 2,5 s für die Kanten der 1023 Nähte
   selbst, die bei jeder Textmenge gleich bleiben und unter `KantenBudget` zählen; hochgerechnet wurden sie mit.
   An der Grenze (`BenchmarkGlyphenSpaltenGrenze`, 2019 Texte à 270 „@“ Grad 74, Glyphkosten = `GlyphBudget`, über
   1023 Spaltenstreifen; ein Lauf, 25.11.2026): vorher 11,6 s, jetzt 9,6 s (davon rund 2,5 s Kanten), ohne Streifen
   8,8 s. Das Zeitversprechen von rund 21 s (ADR-0020 Nachtrag 12) hält.

Renderer: Tests `internal/render/glyphen_mischen_test.go`, Zählnahtstelle `aufwand.einmischungen`; Mutanten G1–G3 neu,
N1–N3 und N3b auf `glyphBaender` nachgezogen.

Editor: Tests `internal/ui/naht_bereiche_test.go` (`TestEditorGanzesBildUeberVielenNaehten`, `TestEditorRunderKastenStrichWieGio`,
`TestEditorTextUeberSchmalenSpalten`), Fall „KastenRundOhneFuellung“ in `TestEditorSchneidetSchraegeKantenAnDerNaht`;
Mutanten B1–B5 neu, P6 nachgezogen (`scripts/mutanten/text-naht.json`).

## Nachtrag 19 (25.11.2026): Strich des runden Kastens im Editor bei breitem Strich (F-782, F-783)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #297; Befunde aus dem Verify-Lauf Sprint 13, Claude Fable 5.2).
Berichtigt Nachtrag 18 Punkt 1.

1. **Kein Loch bei Strich breiter als die Rundung (F-783):** Ist der halbe Strich hw größer als der Eckradius, liegen die
   inneren Gehrungsecken einer Rundung jenseits ihres Mittelpunkts in umgekehrter Reihenfolge; jedes Trapez war eine
   Schleife, deren Hälften sich in der Füllung (non-zero) aufhoben. Ein Kasten 40×40, Radius 20, Strich 60 zeigte im
   Editor ein Loch (Mitte Deckung 2), der Renderer füllt (255); Gios eigener Strich hatte dasselbe Loch. Jetzt liegen die
   inneren Ecken höchstens auf dem Rand des **Lochs**, des um hw verkleinerten umschließenden Rechtecks (bei zu kleinem
   Rechteck dessen Mittellinie bzw. Mitte): genau die Punkte weiter als hw vom Umriss, die auch der Renderer frei lässt
   (`editor.strichVierecke`). Die Strecken einer Rundung fallen dann auf eine gemeinsame innere Spitze; sie werden ein
   konvexer Fächer (äußere Ecken, dann die Spitze), nicht einzelne Dreiecke — liefen viele schmale Dreiecke in einem
   Punkt zusammen, verlöre Gio dort durch die gegeneinander aufgerechneten Kanten bis zu 3 Stufen Deckung. Liegen alle
   inneren Ecken auf einer Stelle, ist der Strich der äußere Umriss. Alle Teile laufen gleich herum und überlappen nicht.
   *Berichtigt durch Nachtrag 21:* Das galt nicht für hw zwischen Radius·cos(3,75°) und Radius (F-792).
2. **Volle Bereiche als Rechteck (F-782):** Ein Bereich, den die beschnittenen Teile ganz decken (Summe ihrer Flächen
   gleich der Fläche des Bereichs; sie überlappen nicht), wird ein Rechteck; benachbarte volle Bereiche einer Zeile
   werden ein Rechteck. Gio rasterte vorher je Bereich alle Kanten der Vierecke; der Rand der Teile gegen den Bereich
   liegt auf ganzen Pixeln, das Bild bleibt gleich. Das umschließende Rechteck je Teil spart den Schnitt mit Bereichen,
   die es nicht berührt. 200 runde Kästen über 4096 Kacheln, llvmpipe, ganzes Bild (Messungen schwanken auf dem
   geteilten Rechner um bis zu 50 %): Strich 1 0,45–0,5 s (unverändert), Strich 40 1,4–1,8 s → 0,55–1,0 s, Strich 120
   2,7–3,2 s → 0,8–1,1 s. Strich 41 war bei 200 Kästen der breiteste, den `render.Pruefe` annahm (42 lehnte es ab; berichtigt nach Verify 2, F-795; seit Nachtrag 21 nimmt es bis Strich 132 an und lehnt 133 ab, weil der Strich ohne Kreise gefüllt wird; gemessen nach Verify 3, F-801); der
   Fall steht in `TestEditorGanzesBildUeberVielenNaehten` (Grenze 2 s), der Test verlangt, dass `Pruefe` ihn annimmt.
   Das Budget bleibt unverändert.
3. **Abgleich mit dem Renderer:** Abseits der Kanten (das Pixel und alle bis zwei Schritte weit deckt der Renderer gleich)
   zeigt der Editor dasselbe wie der Renderer (`rgbaNah`), kein Pixel steht voll gegen leer. An den Kanten weicht die
   Glättung ab: Der Renderer füllt Rechtecke je Strecke und Kreise an den Gelenken in einem Durchgang und addiert dort
   die Kantenglättung der überlappenden Teile (bis 255 statt rund 128 auf dem äußeren Rand), der Editor glättet die
   Gehrungsteile einfach. Über Nähten zeigt jedes Pixel der Ansicht, was der Editor ohne Naht an seiner Stelle im
   Original zeigt.
   *Berichtigt durch Nachtrag 21:* „Kein Pixel steht voll gegen leer“ galt nur für die geprüften Fälle; bei kleinem
   Radius (0,5) und breitem Strich standen bis 26 Pixel voll gegen leer, weil der Renderer die Glättung addierte (F-793).

Tests: `TestEditorDickerStrichKleinerKastenWieRenderer` (40/20/60, 30/10/50, 20/10/24, 42/20/60, 200×130/8/24,
60×50/12/20; je ohne Naht gegen den Renderer und über Nähten gegen das Bild ohne Naht), Fall „Strich 40“ in
`TestEditorGanzesBildUeberVielenNaehten`. Mutanten BK6 (F-783), B7 (F-782) und B8 neu, B2 nachgezogen; B6 ist der Rand-Mutant aus F-784 (`scripts/mutanten/text-naht.json`; Namen berichtigt nach Verify 2, F-794).

## Nachtrag 20 (25.11.2026): Editor über vielen Nähten mit Bereichsgrenze und Objektebene (F-787)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #299). Ergänzt Nachträge 18 und 19.

`render.Pruefe` schützt den Export, nicht das Editorbild: An seiner Grenze zeichnete der Editor über 4096 Kacheln
2000 runde Kästen ohne Füllung in 7,6 bis 9 s je Bild (llvmpipe, 16 Kerne), 120 gefüllte in 1,7 s; über dem Budget
(`--ignore-limit`) beliebig länger. Abgewogen wurden (a) eine Grenze je Bild für Objekte × Bereiche und (b) eine
zwischengespeicherte Objektebene. Gewählt ist beides zusammen:

1. **Leeres Abspielen fällt weg:** Ein runder Kasten ohne Füllung spielte seine (leere) Füllung trotzdem in jedem
   Bereich seines Rechtecks ab, 4096-mal je Kasten. Das war fast die ganze Zeit: 2000 Kästen 7,6–9 s → 0,63 s, 200
   Kästen 0,5 s → 0,07 s je Bild. Der Strich selbst kostet wenig (Nachtrag 19).
2. **Bereichsgrenze:** Der Editor zählt beim Zeichnen die abgespielten Bereiche (`jeStueckIn`, Läufe einer Textzeile,
   Bereiche und Läufe voller Bereiche des Kastenstrichs). Gio braucht mit llvmpipe je Bereich rund 1,3 µs (Strich) bis
   3,3 µs (Füllung eines runden Kastens). Über `bereichGrenze` = 200 000 (höchstens rund 0,7 s je Bild) verwirft er
   seine Aufzeichnung und zeigt stattdessen
3. **die Objektebene des Renderers** (`render.Objektebene`): dieselben Objekte, auf durchsichtigem Grund in der Größe
   der Ansicht gerastert, ohne Bild und ohne Kanten der Nähte (die zeichnet der Editor wie immer darüber), in
   waagerechten Bändern auf bis zu acht Kernen. In Bändern weicht die Kantenglättung um höchstens 1 von einem Band ab
   (float32 relativ zum Ausschnitt); über das Ansichtsbild gemischt ist sie das Bild des Renderers bis auf 2 je Kanal.
   Sie gilt, solange Dokumentstand, Pixelstand, Ansichtsgröße und alle Objekte (`document.Gleich`; während eines Zugs
   ändern sie sich ohne Schritt) gleich bleiben; jedes weitere Bild ist dann eine Textur (rund 5 ms). Ändert sich
   etwas, zählt der Editor neu und baut sie wieder: an der Grenze von `Pruefe` 0,7 s (120 gefüllte Kästen) bis 1,8 s
   (2000 Kästen) auf 16 Kernen; auf einem Rechner mit weniger Kernen entsprechend länger, höchstens so lange wie der
   Renderer selbst (Zeitversprechen rund 21 s, ADR-0020 Nachtrag 12).

Das Ziel „Bild unter 2 s an der Grenze“ hält damit jedes Bild ohne Änderung (5 bis 7 ms) und jedes eigene Zeichnen unter
der Grenze (gemessen 0,4 s knapp darunter); ein Bild nach einer Änderung über der Grenze dauert so lange wie der Aufbau
der Objektebene (gemessen 0,7 bis 1,8 s, im Test bis 10 s zugelassen für Runner mit wenigen Kernen). Bewusst nicht
gewählt: eine eigene Grenze in `Pruefe` für das Editorbild (sie lehnte Dokumente ab, die der Renderer schafft) und das
Rastern im Hintergrund (eine veraltete Ebene nach jeder Änderung wäre verwirrender als ein langsames Bild in einem
Dokument, das ohnehin niemand von Hand so baut).

Grenzen der Objektebene: Sie zeigt die Objekte so, wie der Renderer sie rastert (Kantenglättung des Renderers, beim
Vergrößern die Pixel der Ansicht statt Vektoren), und der gerade bearbeitete Text steht wie nach der Eingabe auf seinen
Bereich beschnitten (Nachtrag 17 Punkt 3 gilt dort nicht). Das betrifft nur Dokumente über der Bereichsgrenze.

Zusatz nach Verify 3 (F-799, F-800): Die Ebene gilt auch nur für ihr Dokument (`objektEbene.dok`); zwei frisch
geöffnete Dokumente mit gleichen Objekten tragen denselben Stand und Pixelstand, ohne den Vergleich zeigte ein Wechsel
der Aufnahme die Ebene der alten (unter einer Lupe deren ungeschwärzte Pixel). Wird die Ebene nicht mehr gezeigt
(unter der Grenze, Wechsel der Aufnahme in `uebernimm`, Fehler beim Bauen), gibt `frei` Bild und Objekte frei; vorher
blieb das Bild in Ansichtsgröße (bis 275 MiB) bis zum nächsten Aufbau liegen. Jede Bedingung von `passt` prüft
`TestObjektebenePasstJeBedingung` einzeln (Mutanten OE8 bis OE10). Aufbau gemessen 0,7 bis 1,9 s auf 16 Kernen,
2,3 s auf 2 Kernen (F-801).

Tests: `TestObjektebeneUeberAnsichtWieBild`, `TestObjektebeneBaenderWieEinBand` (`internal/render/objektebene_test.go`),
`TestObjektebeneGiltNurFuerIhrenStand`, `TestEditorZeigtObjektebeneWieRenderer` (`internal/ui/objektebene_test.go`),
Fälle „2000 runde Kästen ohne Füllung (Budget)“, „120 gefüllte runde Kästen (Budget)“ und „45 gefüllte runde Kästen
(knapp unter bereichGrenze)“ in `TestEditorGanzesBildUeberVielenNaehten`. Unter `-race` loggen die Zeittests nur
(`internal/ui/zeitgrenze_race_test.go`, F-791).

## Nachtrag 21 (25.11.2026): Ein Strich des runden Kastens für Renderer, Editor und Vorschau (F-788, F-789, F-792, F-793)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #299; F-792, F-793 aus dem Verify-Lauf Sprint 13). Berichtigt Nachtrag 19
Punkte 1 und 3.

1. **Eine Zerlegung für alle (F-788, F-789):** Die konvexen, überlappungsfreien, gleich herum laufenden Teile des
   Strichs (Gehrungsvierecke, Fächer je Rundung; Nachtrag 19 Punkt 1) rechnet jetzt `document.NeuerKastenStrich`, nicht
   mehr der Editor allein. Der Renderer füllt je sichtbarem Bereich dieselben Teile in einer Füllung
   (`render.kastenStrich`); ihre gemeinsamen Kanten heben sich im Rasterizer genau auf, jedes Pixel ist so gedeckt wie
   die Fläche des Strichs. Vorher füllte er je Strecke ein Rechteck und je Gelenk einen Kreis in einer Füllung; die
   Kantenglättung der überlappenden Teile addierte sich, die äußeren Randpixel einer Rundung standen bis 255 statt rund
   128 (Abweichung von der analytischen Deckung bis 0,45 bis 0,82, nachher höchstens 0,06 bei Strich 60, sonst unter
   0,035). Die Vorschau der Ebenenliste füllt dieselben Teile als einen Pfad (`kastenStrichVorschau`) statt mit Gios
   Strich, der bei Strich > 2·Radius dasselbe Loch zeigte wie F-783.
2. **Innere Ecke höchstens bis zum Mittelpunkt der Rundung (F-792):** Die Gehrung reicht hw/cos(Knick/2) weit, an einer
   Rundung 1,00214·hw. Für hw zwischen Radius·cos(3,75°) und Radius lagen die inneren Ecken einer Rundung jenseits ihres
   Mittelpunkts, aber noch im Loch; sie wurden nicht begrenzt, die Vierecke liefen gegeneinander (Kasten 120×80,
   Radius 20, Strich 40: Editor 249–252 statt 255 auf einer Linie vom Mittelpunkt nach außen). Reicht die Gehrung einer
   Ecke bis zum Mittelpunkt ihrer Rundung (g ≥ Radius), liegt die innere Ecke jetzt auf dem Mittelpunkt; die wahre
   innere Kante ist dort ein Kreis vom Radius Radius − hw < 0,0022·Radius. Die Begrenzung auf das Loch bleibt.
   Zusatz F-798: Die innere Ecke springt schon auf den Mittelpunkt, wenn die Gehrung weniger als `mittelpunktFang`
   = 0,25 px davor endet. Sonst entstehen dort Teile von Bruchteilen eines Pixels Breite, die llvmpipe je nach
   Mesa-Fassung verschieden rastert (Strich 39,9 bei Radius 20: in der CI 26 Pixel Abweichung zwischen Editor und
   Renderer, lokal keine). Hinzu kommt ein Keil von höchstens 0,25 px Tiefe innerhalb des Strichs. Test: Strich 39,6
   und 39,9 in `TestKastenStrichFaecherJeRundung`, Mutant KS6.
3. **Bild:** Die Gehrung der Teile (Knick 7,5°) reicht höchstens 0,22 % der halben Breite über den runden Strich hinaus
   (Strich 60: 0,064 Pixel); das SVG sagt weiter `stroke-linejoin="round"`. Das Kostenmodell zählt die Teile wie jede
   Füllung (weniger Ecken als Rechtecke und Kreise). Pfeil, Lupen-Linie und Schenkel der Sprechblase füllen weiter
   Rechtecke und Kreise; dort addiert sich die Glättung an den Gelenken nach wie vor (offen).
4. **Abgleich Editor–Renderer:** Weil beide dieselben Teile füllen, steht in den geprüften Fällen kein Pixel voll gegen
   leer, auch bei Radius 0,5 und Strich 30 bis 139 (vorher 12 bis 26 Pixel, F-793). An den Kanten darf Gios Glättung
   weiter abweichen.

Tests: `TestKastenStrichDecktWieAnalytisch` (`internal/render/strichdeckung_test.go`, 32×32 Stichproben je Pixel gegen
die Fläche aller Punkte höchstens hw vom Umriss), `TestKastenStrichFaecherJeRundung` (`internal/document`),
`TestVorschauRundKastenWieRenderer`, Fälle für F-792 (120×80/20/40, 39,9, 41, 40,2; 40×40/8/16; 100×40/20/40;
120×80/50/80; 120×80/10/20) und F-793 (40×40/0,5/30, 61, 90, 139) in `TestEditorDickerStrichKleinerKastenWieRenderer`.
Mutanten B1, B2, B6, B8, B9, BK6 nachgezogen (B6, B8, BK6 jetzt in `internal/document/kastenstrich.go`), KS1–KS5 neu
(`scripts/mutanten/text-naht.json`).

## Nachtrag 22 (25.11.2026): Strich von Pfeil, Lupen-Linie und Schenkeln ohne doppelte Kantenglättung (F-796)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #301). Schließt den offenen Rest aus Nachtrag 21 Punkt 3.

Pfeil (auch gebogen), Lupen-Linie und Schenkel der Sprechblase ohne Füllung füllte der Renderer als Rechteck je Strecke
und Kreis je Punkt in einer Füllung. Der Rasterizer summiert die Deckung; wo zwei Teile dieselbe Kante in einem
Randpixel teilen, zählte sie doppelt. Gemessen gegen die analytische Deckung (32×32 Stichproben je Pixel, Fläche aller
Punkte höchstens hw vom Linienzug): Pfeil gerade Strich 24 bis 0,46, gebogen Strich 3/24/60 bis 0,45/0,58/0,63,
Lupen-Linie Strich 2/8 bis 0,18/0,31, Schenkel Strich 6/12 bis 0,39/0,34, eine Scheibe (ein Punkt) 0,15.

Abgewogen:

- **(a) Überlappungsfreie Teile wie beim Kasten** (Nachtrag 21) — gewählt.
- **(b) Deckung je Pixel als Maximum statt Summe:** Jeder Teil bräuchte eine eigene Maske (beim Pfeil rund 100 je
  Objekt), und das Maximum ist nicht genau: An der Innenseite eines spitzen Gelenks (eine einspringende Ecke) läge die
  Deckung bis rund 0,25 zu niedrig, und Teile, die sich nicht überlappen, hinterließen eine Fuge.
- **(c) Umriss als ein Pfad** (wie Gios Strich): Auf der Innenseite jedes Gelenks schneidet sich der Umriss selbst; die
  Schleifen laufen entgegen und höben in der summierenden Füllung Fläche auf (ein Loch wie F-783), oder man rechnet die
  innere Kante aus — das tut (a) ohnehin, ohne die Schleifen.

1. **Zerlegung** (`render.neuerLinienStrich`, `internal/render/linienstrich.go`): je Strecke ein Sechseck, das je Seite
   vom Ende am Anfang zum Ende am Schluss und über die beiden Punkte zurück läuft; je Gelenk ein Fächer auf der
   Außenseite (Bogen von der Normalen der vorigen zu der der nächsten Strecke); je offenem Ende eine halbe Scheibe. Ein
   Ende liegt außen und an einem offenen Ende senkrecht hw neben dem Punkt, innen auf dem Schnittpunkt der beiden
   inneren Kanten (Gehrung hw/cos(Knick/2)), den beide Nachbarn teilen. Gemeinsame Kanten entstehen aus denselben
   Punkten und heben sich in der Füllung genau auf. Ein einzelner Punkt ist eine Scheibe. `strich` (Lupen-Linie,
   Schenkel, in der Ansicht) füllt alle Teile; `strichJeStueck` (Pfeil, je sichtbarem Bereich) füllt die Sechsecke der
   Strecken in der Nähe samt Fächern und Kappen an ihren Enden, jeden einmal je Bereich. Jeder Teil liegt höchstens hw
   von seiner Strecke, die Suche über `document.StreckenStuecke` mit hw+1 bleibt.
2. **Faltung:** Reichen die Gehrungen beider Enden einer Seite zusammen weiter, als die Strecke lang ist
   (hw·tan(|Knick|/2) je Ende; bei einer Kurve heißt das Krümmungsradius unter hw, bei einem V Enden näher als 2·hw,
   dazu jede Kehrtwende), liefen die Sechsecke gegeneinander und höben sich auf. An einem solchen Gelenk enden beide
   Nachbarn innen senkrecht wie außen; sie überlappen dann auf der Innenseite. Kein Loch, nichts daneben; die Glättung
   addiert sich nur noch an diesen Gelenken und nur, wo die Überlappung eine Kante berührt: Haarnadel Strich 24
   (6 gefaltete Gelenke) 0,018, Kehrtwende 0,017, sehr spitzes kurzes V Strich 12 0,30 an der Spitze. Gelenke werden
   nur gefaltet, nie zurückgenommen; die Prüfung endet.
3. **Bögen:** Fächer und Kappen bekommen so viele Ecken, dass die Sehne höchstens 0,02 Pixel unter dem Bogen liegt
   (`bogenSehne`; bis hw 4250, darüber höchstens 1024 Ecken je Vollkreis). Die bisherigen Kreise mit max(16, 2r) Ecken
   lagen bis 0,1 Pixel darunter (Strich 24). Die Kreise der Schritt-Nummer bleiben.
4. **Ergebnis:** alle Fälle oben höchstens 0,022 (gebogen Strich 60), sonst 0,014 bis 0,021; zulässig im Test
   1/32 + `bogenSehne` + 0,01 = 0,061. Ganz innen und außen weicht kein Pixel ab.
5. **Kosten:** Das Kostenmodell zählt die Teile wie jede Füllung, eine eigene Rechnung braucht es nicht. Es sind weniger
   Ecken als vorher (gebogener Pfeil Strich 24: 487 statt 1368), und Kanten zählen die Polygone selbst statt der
   oberen Schranke der Kreise (`kreisSchritte`). Messung (25.11.2026, ein Lauf, geteilter Rechner):
   - Alltagsdokument (50 Pfeile Strich 6) unverändert 4,07e10, 2,3 → 2,5 s; diagonaler Pfeil auf 8000×8000 0,25 →
     0,28 s; gebogene Pfeile Strich 24 auf 2000×2000 bis `Budget` 1404 → 1406 Stück, 7,3 → 7,7 s.
   - An `KantenBudget` nimmt `Pruefe` mehr Pfeile an, weil die alten Kreise viel zu teuer zählten: breite Pfeile
     (Strich 1e5) weit außerhalb eines 10×10-Bilds 171 → 429 Stück, 0,32 → 4,3 s (0,43 ns je Einheit; vorher 0,03);
     gebogene Pfeile Strich 24 über 4096 Kacheln 275 → 417 Stück, 4,5 → 4,9 s (0,48 ns je Einheit). Beides liegt bei
     der Annahme rund 0,5 ns je Kanten-Einheit (ADR-0020, Ergänzung zu F-418) und unter den rund 6,5 s der Abriss-Nähte
     an derselben Grenze; das Zeitversprechen rund 21 s (ADR-0020 Nachtrag 12) bleibt. Budgets und Gewichte bleiben.
6. **Editor und SVG:** Der Editor zeichnet Pfeil, Lupen-Linie und Schenkel weiter mit Gios Strich: ein Umriss mit runden
   Gelenken und Kappen, dessen Glättung Gio einfach rechnet; dort gab es den Befund nicht, angeglichen wird nichts. An
   den Kanten darf Gios Glättung wie bisher abweichen. Der SVG-Export schreibt weiter `stroke-linecap="round"` bzw.
   `stroke-linejoin="round"`; der Pfeil bleibt dort die Kurve, im Renderer und Editor der Linienzug aus 48 Strecken.

Tests `TestPfeilStrichDecktWieAnalytisch`, `TestStrichDecktWieAnalytisch`, `TestStrichGefaltetOhneLoch`
(`internal/render/strichgelenke_test.go`). Mutanten SG1–SG11 (`scripts/mutanten/strichgelenke.json`); SG11 übernimmt
die Absicht von P1 (`text-naht.json`), dessen Muster im neuen Code nicht mehr steht.
