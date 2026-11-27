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
   übrige nach.
2. **Wählen (Werkzeug Auswählen):**
   - **Umschalt+Klick** auf ein Objekt nimmt es zur Auswahl hinzu (es wird Hauptauswahl) oder, ist es schon gewählt,
     heraus. Ein Klick ohne Umschalt wählt nur das getroffene (wie bisher), ein Klick auf freie Fläche hebt alles auf.
   - **Rahmen:** Ziehen ab einer freien Stelle **im Bild** zieht einen Rahmen; beim Loslassen ist jedes sichtbare, nicht
     gesperrte Objekt gewählt, dessen Grenzen ganz im Rahmen liegen (wie Inkscape; ein Streifen über ein großes Bild
     wählt nicht alles). Mit Umschalt kommen sie zur bisherigen Auswahl hinzu. Hauptauswahl wird das oberste.
   - **Strg+A** (Menü Bearbeiten → „Select all“) wählt alle sichtbaren, nicht gesperrten Objekte. **Esc** hebt die
     Auswahl auf (Stufe „abwählen“ wie bisher).
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
     dazwischen gleich große Lücken (nach Lage der linken bzw. oberen Kante sortiert); ab drei gewählten. Ein Objekt,
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
- `internal/document`: Zug und Befehle über mehrere Objekte; `internal/canvas`: `Mehr`, `Gewaehlt`, Rahmen-Zug,
  Ausrichten und Verteilen (`ausrichten.go`); `internal/ui`: Umschalt-Klick, Rahmen zeichnen, Menüeinträge (Bearbeiten →
  „Select all“ Strg+A, Ebene → Ausrichten/Verteilen), Darstellung der weiteren Auswahl.
- Tests, Mutanten, Sichtprobe; README en/de (Ziehen im Bild wählt jetzt, statt die Ansicht zu verschieben).
