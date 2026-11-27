# ADR-0039: Mehrfachauswahl auf der Leinwand, Ausrichten und Verteilen

## Status
Akzeptiert (Claude Opus 5.5, 27.11.2026, Karte #334 in Sprint 14, im Auftrag Andis: „arbeite selbstständig und ziehe
auch selbstständig karten“). Grundlage: Vorschlag Claude, von Andi am 25.11.2026 übernommen („Mehrere Objekte wählen
(Rahmen aufziehen im leeren Bereich, Umschalt-Klick, Strg+A), gemeinsam verschieben, löschen, Stil setzen; Ausrichten
(links, Mitte, rechts, oben, unten) und gleichmäßig Verteilen. Ein Rückgängig-Schritt je Aktion.“). Ergänzt ADR-0021
(Werkzeuge), ADR-0026 (Menü und Kürzel), ADR-0032 (Ebenen); Gruppieren folgt mit Karte #354.

## Kontext
Der Editor kennt genau ein ausgewähltes Objekt (`canvas.Sitzung.Auswahl`). Wer drei Pfeile gemeinsam verschieben oder
untereinander ausrichten will, muss jeden einzeln ziehen. Snagit, Inkscape und Krita wählen mehrere per Umschalt-Klick
und Rahmen. Das Dokument kennt nur Züge und Befehle an einem Objekt.

## Entscheidung
1. **Modell:** `Sitzung.Auswahl` bleibt die **Hauptauswahl** — das zuletzt gewählte Objekt, an dem Griffe, rechte
   Leiste, Farbe, Kontextmenü und Texteingabe wie bisher wirken. Neu ist `Sitzung.Mehr`: die weiteren gewählten
   Objekte (Kennungen, ohne die Hauptauswahl). `Gewaehlt()` liefert alle in Stapelreihenfolge. Verschwindet ein Objekt
   (Rückgängig, Löschen, Ausblenden, Sperren), fällt es aus der Auswahl; fällt die Hauptauswahl weg, rückt das oberste
   übrige nach. Das gilt auch für die Hauptauswahl: Wird sie neben weiteren gesperrt oder ausgeblendet, fällt sie heraus,
   und nichts Gesperrtes wird mitverschoben, ausgerichtet, gedoppelt oder gelöscht. Allein gewählt (etwa per
   Ebenenliste) bleibt eine gesperrte oder ausgeblendete Hauptauswahl, damit Strg+L und Strg+H sie wieder freigeben.
   Was einmal aus der Auswahl gefallen ist, kommt nicht zurück — auch nicht, wenn Rückgängig es wieder sichtbar macht
   oder die frühere Hauptauswahl wieder gewählt wird (F-895, F-896).
2. **Wählen (Werkzeug Auswählen):**
   - **Umschalt+Klick** auf ein Objekt nimmt es zur Auswahl hinzu (es wird Hauptauswahl) oder, ist es schon gewählt,
     heraus. Ein Klick ohne Umschalt wählt nur das getroffene (wie bisher), ein Klick auf freie Fläche hebt alles auf.
   - **Rahmen:** Ziehen ab einer freien Stelle **im Bild** zieht einen Rahmen; beim Loslassen ist jedes sichtbare, nicht
     gesperrte Objekt gewählt, dessen Grenzen ganz im Rahmen liegen (wie Inkscape; ein Streifen über ein großes Bild
     wählt nicht alles). Verglichen werden die aufs sichtbare Bild begrenzten Grenzen: Ein Objekt, das über den
     Bildrand oder in den weggeschnittenen Teil ragt, ist mit einem Rahmen über seinen sichtbaren Teil gewählt. Mit
     Umschalt kommen sie zur bisherigen Auswahl hinzu. Hauptauswahl wird das oberste. Esc während des Rahmens bricht
     ihn ab und stellt die Auswahl von vorher her.
   - **Strg+A** (Menü Bearbeiten → „Select all“) wählt alle sichtbaren, nicht gesperrten Objekte. Mit einem
     Zeichenwerkzeug wechselt es dabei zu Auswählen, wie ein Klick in die Ebenenliste; in Zuschneiden und Stauchen
     und während der Texteingabe wirkt es nicht (der Menüeintrag ist grau). **Esc** hebt die Auswahl auf (Stufe
     „abwählen“ wie bisher).
   - **Ansicht verschieben:** Ziehen ab einer freien Stelle **außerhalb des Bilds** (Rand der Leinwand) verschiebt
     weiter die Ansicht, die mittlere Taste überall, der Navigator bei Vergrößerung. Vorher verschob jedes Ziehen auf
     freier Fläche die Ansicht; im Bild wird daraus der Rahmen. Abweichung vom Kartenwunsch keine; die Änderung ist im
     README beschrieben.
3. **Gemeinsam (ein Rückgängig-Schritt je Handlung):**
   - **Verschieben:** Ziehen an einem gewählten Objekt verschiebt alle gewählten um denselben Weg. Eingerastet wird
     der Rahmen um alle (Vereinigung der Grenzen) an den übrigen Objekten und am Bild; begrenzt wird so, dass jedes
     gewählte Objekt im sichtbaren Bild bleibt (je Achse der kleinste erlaubte Anteil, wie `begrenzeVerschieben`).
     Ein Ziehen an einem nicht gewählten Objekt wählt nur dieses (wie bisher).
   - **Löschen** (Entf, Menü, Kontextmenü) entfernt alle gewählten. **Doppeln** (Strg+D) doppelt alle; die Kopien
     werden die Auswahl.
   - **Ausrichten** (Menü „Ebene“): links, waagerecht mittig, rechts, oben, senkrecht mittig, unten — an den Grenzen
     des Rahmens um alle gewählten; ab zwei gewählten. **Verteilen** waagerecht bzw. senkrecht: die äußeren bleiben,
     dazwischen gleich große Lücken (nach Lage der linken bzw. oberen Kante sortiert); ab drei gewählten. Äußere sind
     das Objekt mit der kleinsten linken und das übrige mit der größten rechten Kante (senkrecht: oberen, unteren);
     die anderen kommen nach ihrer linken Kante geordnet dazwischen, und gleich sind die Lücken zwischen der rechten
     Kante des einen und der linken des nächsten — so überlappt nach dem Verteilen nichts, was zwischen den äußeren
     Platz hat; reicht der Platz nicht, werden es gleich große Überlappungen (F-897).
   - **Rechtsklick** auf ein gewähltes Objekt macht es zur Hauptauswahl, die übrigen bleiben gewählt (das
     Kontextmenü wirkt dann auf alle, wo Punkt 3 das vorsieht); auf ein nicht gewähltes wählt er nur dieses. Ein Objekt,
     das dabei aus dem sichtbaren Bild fiele, macht die Handlung nicht (Meldung in der Statuszeile).
   - Das Dokument bekommt dafür einen Zug über mehrere Objekte (`BeginneZugMehrere`, `ZugSetzeMehrere`) und Befehle
     über mehrere (`ErsetzeMehrere`, `EntferneMehrere`), die als **ein** Befehl im Verlauf landen.
4. **Was an der Hauptauswahl bleibt (Teil 1):** Griffe, rechte Leiste (Farbe, Strich, Strichart, Schatten, Schrift),
   Schnellstile, Ebenen-Handlungen außer Doppeln (Ausblenden, Sperren, Umbenennen, vor/zurück), Texteingabe und Farben
   tauschen wirken nur auf die Hauptauswahl. **Stil für alle gewählten** und die Mehrfachauswahl in der Ebenenliste
   folgen mit einer eigenen Karte; das ist die Abweichung von „Stil setzen“ im Kartentext.
5. **Darstellung:** Die Hauptauswahl trägt ihre Griffe wie bisher, jedes weitere gewählte Objekt einen Auswahlrahmen
   ohne Griffe. Der aufgezogene Rahmen ist eine dünne Linie in der Akzentfarbe mit leicht getönter Fläche. Die Fußzeile
   bzw. Statuszeile nennt die Zahl („3 objects selected“).
6. **Andere Werkzeuge:** Mit einem Zeichenwerkzeug bleibt ein Klick wie bisher (neues Objekt); ein Werkzeugwechsel hebt
   die weiteren gewählten auf, die Hauptauswahl bleibt wie bisher. Zuschneiden und Stauchen wählen nie.

## Erwogene Alternativen
- **Liste statt Hauptauswahl:** jede Stelle, die heute `Auswahl` liest (rechte Leiste, Griffe, Farbe, Texteingabe,
  rund 60 Stellen), müsste entscheiden, was bei mehreren gilt. Die Hauptauswahl hält Teil 1 klein und verhält sich wie
  Inkscape, das Griffe und Werkzeugleiste ebenfalls auf ein Objekt bezieht.
- **Rahmen nur mit Umschalt:** behält das Verschieben der Ansicht im Bild, ist aber kaum zu entdecken; der Kartentext
  verlangt den Rahmen im leeren Bereich.
- **Rahmen trifft berührte Objekte:** wählt bei großen Beschriftungen zu viel; Inkscape und Krita nehmen ganz
  enthaltene.

## Konsequenzen
- `internal/document`: Zug und Befehle über mehrere Objekte; `internal/canvas`: `mehrfach` (Feld `Sitzung.mf`), `Gewaehlt`, Rahmen-Zug,
  Ausrichten und Verteilen (`mehrfach.go`); `internal/ui`: Umschalt-Klick, Rahmen zeichnen, Menüeinträge (Bearbeiten →
  „Select all“ Strg+A, Ebene → Ausrichten/Verteilen), Darstellung der weiteren Auswahl.
- Tests, Mutanten, Sichtprobe; README en/de (Ziehen im Bild wählt jetzt, statt die Ansicht zu verschieben).

## Nachtrag 1 (27.11.2026, Andi: „ja, finde ich gut“)
Auf Andis Frage nach dem Vorbild der anderen (Inkscape, Krita, GIMP, Photoshop, Figma: Rahmen ohne Umschalt, Ansicht
mit Leertaste+Ziehen): **Leertaste halten + Ziehen verschiebt die Ansicht**, in jedem Werkzeug; der Zeiger zeigt
solange die offene Hand, beim Ziehen die greifende (Korrektur F-908). Während einer Texteingabe ist die Leertaste Text, dort bleibt die mittlere Taste. Rahmen
ohne Umschalt (Punkt 2) bleibt; mittlere Taste, Ziehen neben dem Bild und Navigator bleiben. Damit verschiebt man die
Ansicht auch am Touchpad ohne mittlere Taste. „Mausrad rollt, Strg+Rad zoomt“ ist eine eigene Karte, nicht Teil von #334.

## Nachtrag 2 (27.11.2026, Claude Opus 5.5 im Auftrag Andis: „arbeite selbstständig“; Umsetzung Karte #334)
1. **Rückgängig und Auswahl (F-900):** Rückgängig und Wiederholen stellen die Auswahl nicht wieder her — weder nach
   dem Löschen oder Doppeln mehrerer noch nach dem eines einzelnen Objekts (dort wählt Rückgängig schon heute nichts
   neu). Die Regel aus Punkt 1 („was herausgefallen ist, kommt nicht zurück“) gilt damit ohne Ausnahme; ein Verlauf
   der Auswahl je Schritt wäre ein eigenes Konzept für alle Handlungen, nicht nur die mehrerer. Strg+A oder ein
   Rahmen wählt die zurückgeholten Objekte in einem Schritt wieder.
2. **Darstellung (zu Punkt 5):** Alle Auswahlrahmen (Hauptauswahl, weitere, aufgezogener Rahmen) tragen außen eine
   dunkle Kontur von 1 dp; die Sichtprobe zeigte den grünen Rahmen auf grünem Bildgrund sonst nicht.
3. **Treffer:** Das leere Innere eines ungefüllten Kastens trifft wie bisher nichts; Umschalt+Klick dorthin beginnt
   einen Rahmen (mit Umschalt: zur Auswahl dazu) und ändert ohne Ziehen nichts.
4. **Leertaste, Ergänzungen aus der Kritik (F-905 bis F-907):** Das Loslassen der Leertaste gilt mit jedem Umschalter
   (Gio prüft Umschalter auch beim Loslassen); ein Druck mit Strg, Alt oder Super hält sie nicht. Bei gehaltener
   Leertaste ist ein zweiter Druck kein Doppelklick (kein Aufheben von Zuschnitt oder Naht, keine Texteingabe), und ein
   Verschiebe-Druck zählt nicht für einen folgenden Doppelklick; ein Doppelklick aus einer Texteingabe heraus
   bestätigt sie und wirkt danach wie ohne Leertaste (F-914). Die Leinwand holt sich beim Öffnen und nach jedem
   Wechsel der Aufnahme den Tastaturfokus, damit Leertaste+Ziehen ohne vorherigen Klick ins Bild wirkt. Liegen Menü,
   Übersicht, Einstellungstafel oder volle Ansicht der Bibliothek davor, wartet das, bis sie geschlossen sind; kam der
   Wechsel per Taste (Enter oder Entf) an der Vorschau der Leiste der letzten Aufnahmen, die den Fokus hatte, bleibt
   er dort — Pfeile und Enter bedienen die Leiste weiter (ADR-0036 Punkt 1, F-912). Ein Mausklick auf eine Vorschau
   oder ein Wechsel aus dem Menü gibt ihn der Leinwand, auch wenn ein Knopf der Leiste ihn vorher hielt (F-918). Menü, Übersicht, Tafel, volle Ansicht,
   Texteingabe, Fokusverlust der Leinwand und des Fensters lassen eine gehaltene Leertaste los.

## Nachtrag 3 (27.11.2026, Andi; Umsetzung Claude Opus 5.5, Karte #382)
Andi, 27.11.2026: „wenn ich links ein neues werkzeug auswähle, sollte die markierung des alten objekts verschwinden und
rechts die details zum neuen werkzeug angezeigt werden“. Ändert Punkt 6:
1. **Wechsel zu einem anderen Werkzeug als Auswählen** (Werkzeugleiste, Taste, Menü Werkzeuge) hebt jede Auswahl auf,
   Haupt- wie weitere; die rechte Leiste zeigt danach das Werkzeug mit seinen Einstellungen, Änderungen dort gelten nur
   für die nächsten Objekte. Vorher blieb die Hauptauswahl und die Leiste zeigte weiter das alte Objekt, sodass eine
   Änderung für das neue Werkzeug das alte Objekt traf.
2. **Wechsel zu Auswählen** (V, Esc, Ebenenliste, Strg+A) behält die Hauptauswahl wie bisher; die weiteren fallen weg.
3. **Dasselbe Werkzeug noch einmal** ändert die Auswahl nicht (etwa ein per Rechtsklick gewähltes Objekt).
4. Zuschneiden und Stauchen heben die Auswahl weiter immer auf (F-174).

Test `internal/canvas/werkzeugwechsel_test.go`, Mutanten `scripts/mutanten/werkzeugwechsel.json` (WW1–WW3; die weiteren beim Wechsel zu Auswählen hält MA18 in `mehrfachauswahl.json`).

## Nachtrag 4 (27.11.2026, Claude Opus 5.5 im Auftrag Andis: „arbeite selbstständig und ziehe auch selbstständig karten“; Karte #377)
Ersetzt Punkt 4 für den Stil und die Ebenenliste (Teil 2 der Mehrfachauswahl).
1. **Stil für alle gewählten:** Sind mehrere Objekte gewählt (und läuft keine Texteingabe), wirkt jede Änderung der
   rechten Leiste auf **alle gewählten, die die Eigenschaft tragen** — Farbe, Füllung und Schattenfarbe, Strichstärke
   (Pfeil, Kasten, Lupe), Strichart, Spitzen (Pfeil), Schatten, Schrift, Größe, Schnitt und Ausrichtung (Text,
   Sprechblase), Farben tauschen (Kasten, Text), Schnellstil (nur die gewählten derselben Art wie der Stil). Objekte
   ohne die Eigenschaft bleiben unverändert. Jede Handlung ist **ein** Rückgängig-Schritt, das Ziehen an einem Regler
   oder in der Farbfläche ebenso (Zug über mehrere, `BeginneZugMehrere`).
2. **Gemerkt (ADR-0041 Punkt 2):** Jedes geänderte Objekt gibt die Änderung an sein Werkzeug weiter — wer einen Pfeil
   und einen Kasten gemeinsam auf 12 px stellt, zeichnet danach Pfeile und Kästen mit 12 px; ein mitgewählter Text
   ändert den Text-Stil dabei nicht. Ein abgebrochener Zug stellt alle beteiligten Werkzeug-Stile wieder her.
3. **Schatten:** Geändert wird, was sich an der Leiste geändert hat, nicht der ganze Schatten: Wer an der Scheibe die
   Richtung dreht, dreht sie an allen gewählten mit Schatten und lässt deren Länge, Weichheit und Farbe stehen;
   Objekte ohne Schatten bleiben ohne. Das Kästchen „Shadow“ schaltet alle ein, wenn mindestens einer aus ist, sonst
   alle aus; ein Objekt ohne eigene Werte bekommt beim Einschalten die seines Werkzeugs.
4. **Was die Leiste zeigt:** Sie zeigt die Abschnitte aller gewählten Arten (ein Pfeil und ein Text: Strich, Spitzen
   und Schrift). Der Kopf nennt die Zahl („Selection: 3 objects“). Spitze und Vergrößerung (Eigenschaften eines
   einzelnen Objekts, keine Stilwerte) und die Schnellstile gelten weiter der Hauptauswahl. Die Werte kommen von der
   Hauptauswahl, trägt sie eine Eigenschaft nicht, vom obersten gewählten, das sie trägt. **Gemischte Werte** zeigt die
   Leiste erkennbar: Strichstärke, Strichart, Spitzen und Schrift nennen statt des Werts „mixed“ (Vorschau der
   Hauptauswahl bleibt), ein Farbfeld mit verschiedenen Farben ist diagonal geteilt (Hauptauswahl oben links, die
   nächste andere unten rechts), das Kästchen „Shadow“ zeigt einen Strich statt des Hakens, wenn nur ein Teil einen
   Schatten hat. Schnitt (fett, kursiv …) und Ausrichtung zeigen den Stand der Hauptauswahl.
5. **Ebenenliste:** Jede gewählte Zeile ist hervorgehoben. **Strg+Klick** nimmt eine Zeile zur Auswahl hinzu (sie wird
   Hauptauswahl) oder heraus, wie Umschalt+Klick auf der Leinwand; **Umschalt+Klick** wählt alle wählbaren Zeilen
   zwischen der Hauptauswahl und der geklickten (die geklickte wird Hauptauswahl). Gesperrte und ausgeblendete Zeilen
   lassen sich nicht dazunehmen (Punkt 1); ein Strg- oder Umschalt-Klick auf sie wählt sie allein wie ein einfacher
   Klick, damit Strg+L und Strg+H sie freigeben. Ziehen mit Strg oder Umschalt ordnet nicht um, und es gibt keinen
   Doppelklick zum Umbenennen.
6. **Bleibt an der Hauptauswahl:** Griffe, Texteingabe, Ebenen-Handlungen außer Doppeln und Löschen (Ausblenden,
   Sperren, Umbenennen, vor/zurück, Deckkraft) und „Save style as quick style“.
