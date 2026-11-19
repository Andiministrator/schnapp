# Marke Schnapp

Logo, Symbol und Farben. Stand 19.11.2026: **Runde 6, Entwürfe** — noch nichts entschieden.

## Vorgaben (Andi, 19.11.2026)

- Cool und schlicht.
- Monoline-Stil: dicke Linie in einer Farbe, runde Enden, ein Buchstabe oder Zeichen trägt das Bild.
- Eigene Farben und eigene Schrift (nicht die eines anderen Projekts des Maintainers).
- Ideen von Andi: Filmklappe (wirkt nach Videoschnitt — verworfen), Maskottchen „Schnappi“ (Krokodil).
- Runde 2 (Andi): dunkelgrauer Grund, dezentes Grün und Orange, auch für die Oberfläche; das S als Krokodilkopf
  oder ein langes Krokodil mit offenem Maul unter dem Schriftzug.
- Runde 3 (Andi): S-Krokodil mit dem unteren Bogen als Maul (offen nach links), oberer Bogen als Kopf; das lange
  Krokodil ausgemalt und runder; Grün dunkler, Orange heller.
- Runde 4 (Andi): Richtung ausgemaltes Krokodil; Maul innen nicht orange; Orange an Buckeln, Füßen oder gar nicht
  am Krokodil; das „s“ im Schriftzug orange mit dem Krokodilauge.
- Runde 5 (Andi): Krokodil weg, nur das orange „s“ mit Auge als Besonderheit.
- **Festgelegt (Andi, 19.11.2026): Runde 5 b** — oranges „s“ mit Auge, Pupille blickt nach rechts auf „chnapp“.
- Runde 6 (Andi): „h“ auf die Höhe der „pp“ herunter oder eine kurze Unterzeile („Linux Screenshots“) unter „schna“.
- Unterzeile gewählt (Andi, 19.11.2026): **f3 „screenshots for linux“**. Probe e2: ganzes „h“ tiefer; e3: e2 mit „SCREENSHOTS“ unter „sc“ und „FOR LINUX“ unter „na“ (Andi: wenn Unterzeile, dann volle Breite); e4: ohne Unterzeile, h tiefer und pp höher, gleich weit
  über und unter dem Band (Andi: gefällt); e5: e4 mit „CAPTURE FOR LINUX“.
- Unterzeile als Pfade aus Open Sans SemiBold (SIL OFL 1.1), damit das SVG keine Schriftdatei braucht.

## Festlegungen der Entwürfe

- **Schriftzug** selbst gezeichnet (`werkzeug/runde1.py`), keine Schriftdatei: Bögen als abgerundete Rechtecke,
  passend zu Rahmen und Bildschirmen. x-Höhe 100, Strich 22.
- **Farben:** ab Runde 2 die Palette in [`farben.md`](farben.md) (Grau, Grün, Orange). Runde 1 nutzte Nachtblau,
  Petrol und Markergelb.
- **Name des Maskottchens:** nicht „Schnappi“ — so heißt eine bekannte Kinderlied-Figur (vermutlich geschützt).

## Entwürfe

`entwuerfe/runde-6/vergleich.html` (aktuell), ältere Runden daneben. Neu erzeugen: `python3 marke/werkzeug/runde6.py`
(entsprechend `runde1.py` bis `runde5.py`; `runde6.py` braucht Python-fontTools und Open Sans).
