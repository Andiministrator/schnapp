# ADR-0048: Werkzeug Textmarker (Mischung „Multiplizieren“ an der Ebene)

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #329 in Sprint 14; Vorschlag Claude, von Andi am 25.11.2026 übernommen;
Andi 28.11.: „arbeite selbstständig“). Ergänzt ADR-0021 (Zeichenwerkzeuge), ADR-0032 (Ebenen), ADR-0008 (Format),
ADR-0015 (Gio-Patch), ADR-0037 (Schnellstile), ADR-0041 (Stil je Werkzeug).

## Kontext
Zum Hervorheben von Text in einer Aufnahme fehlt ein Textmarker: ein breiter Strich, der das Darunterliegende einfärbt,
ohne es zu verdecken. Ein halbdurchsichtiger Strich in normaler Mischung macht schwarze Schrift grau und weißen Grund
blass gelb; ein echter Textmarker färbt nur den hellen Grund und lässt dunkle Schrift schwarz. Das leistet die Mischung
„Multiplizieren“: jeder Farbkanal des Untergrunds wird mit dem Kanal der Markerfarbe malgenommen (Snagit, Photoshop,
Krita). Die Karte fragt, ob die gerade Variante ein Kasten mit Mischmodus oder ein eigener Typ wird; Freihand gehört zu
Karte #333.

Renderer, SVG-Export und Editor müssen gleich aussehen (ADR-0020 Nachtrag 14). Der Renderer mischt selbst
(`einmischer`), SVG kennt `mix-blend-mode: multiply`. Gio mischt nur „über“ (src-over); sein GPU-Treiber kennt aber den
Faktor `DstColor` (für Schnittmasken), und Deckkraft-Ebenen (`paint.PushOpacity`) zeichnet Gio erst in ein eigenes
Bild und mischt es dann ein — genau die Stelle, an der eine andere Mischung ansetzt.

## Entscheidung
1. **Daten: Mischung an der Ebene.** `document.Ebene` bekommt `Mischung` (`Mischart`: `MischNormal` = Nullwert,
   `MischMultiplizieren`). Sie wirkt wie die Deckkraft auf das Objekt als Ganzes samt Schatten (Gruppe, ADR-0032 Punkte 2
   und 10): erst entsteht das Objekt in einem Zwischenbild, dann wird es mit der Deckkraft multiplizierend eingemischt.
   Formel je Kanal (W3C Compositing, sRGB-Werte wie jede Mischung in Schnapp, ADR-0020 Nachtrag 14), Quelle `Cs` mit
   Deckung `as` (Pixel × Deckkraft), Untergrund `Cb` mit `ab`: `co = as·(1−ab)·Cs + as·ab·Cs·Cb + (1−as)·ab·Cb`,
   `ao = as + ab·(1−as)`. Auf deckendem Grund ist das `Cb·(1 − as·(1 − Cs))`; auf durchsichtigem Grund wirkt es wie
   „über“.
2. **Der Textmarker ist ein Pfeil mit dieser Mischung.** Ein `document.Pfeil` mit `Mischung = MischMultiplizieren` gilt
   für Leiste, Ebenenliste, Stil je Werkzeug und Schnellstile als Art Textmarker (`objektWerkzeug`, wie der Text mit
   Spitze als Sprechblase). Neue Textmarker haben keine Köpfe (Anfang und Ende „keine“, also runde Enden wie die Linie,
   ADR-0021 Nachtrag 20). Alles, was den Pfeil behandelt — Griffe (Anfang, Ende, Biegung), Verschieben, Einrasten,
   Treffer, Strichart, Naht, Zuschnitt, Gruppen, Kopieren, Mehrfachstil —, gilt unverändert. Die Leiste zeigt beim
   Textmarker keinen Abschnitt „Arrow style“; der Setter der Köpfe übergeht Textmarker auch in einer Mehrfachauswahl.
3. **Editor: Gio-Patch `paint.PushMultiply`.** Wie `PushOpacity`, aber die Ebene wird mit der Mischung
   `src·DstColor + dst·(1 − srcA)` eingemischt; mit vormultiplizierter Quelle ist das `dst·(1 − a·(1 − c))`, also Punkt 1
   auf deckendem Grund. Der Patch berührt `op/paint` (neue Funktion), `internal/ops` (ein Byte am Deckkraft-Op) und
   `gpu` (Pipeline mit dieser Mischung für die Ebenen-Textur). `zeichneObjekt` nimmt `PushMultiply` statt
   `PushOpacity`, wenn die Ebene multipliziert (auch bei Deckkraft 100 %). **Grenzen:** Auf durchsichtigen Pixeln des
   Bilds liegt im Editor die Schachbrett- bzw. Fensterfarbe darunter, der Renderer mischt „über“ — dort weicht der
   Editor ab. Über `bereichGrenze` zeigt der Editor die Objektebene des Renderers (ADR-0023 Nachtrag 20); dort steht der
   Textmarker auf durchsichtigem Grund und erscheint wie „über“ gemischt. Beides betrifft nur Randfälle.
4. **Werkzeug Textmarker**, Taste **H** (allein frei; Strg+H bleibt Ausblenden), Symbol `icons.Textmarker`, in der
   Werkzeugleiste nach dem Pfeil. Aufziehen vom Anfang zum Ende wie der Pfeil; **Umschalt = waagerecht** (das Ende liegt
   in der Ansicht auf der Höhe des Anfangs). Freihand bringt Karte #333 (Linienzug); sie soll dieselbe Mischung nutzen.
5. **Voreinstellung:** eigener Stil (ADR-0041, Abschnitt `[highlighter]` in `tools.ini`): Strich 20 px mal Faktor,
   Gelb `#FFE628` deckend, Strichart voll, kein Schatten. Die Farbe ist deckend; durchsichtig wirkt der Marker durch das
   Multiplizieren. Eine Deckkraft unter 100 % schwächt ihn ab.
6. **Schnellstile:** eigene Art `highlighter` in `styles.ini`, eingebaut Gelb `#FFE628`, Grün `#7CFC6B`, Rosa `#FF8AD8`
   (je 20 px); die Kacheln zeigen einen Strich in der Farbe.
7. **Format:** Ebenen-Attribut `s:mischung="multiply"` (§8.5), nur bei Abweichung vom Nullwert geschrieben.
   **FormatVersion 2** mit einer Mischung (`versionMischung`); in Version 1 ist das Attribut unbekannt und wird ignoriert
   (Regel 3, §11). Arbeitsdatei und SVG-Export schreiben am Element bzw. an der Gruppe `style="mix-blend-mode:multiply"`.
8. **Bedienung ohne Fenster:** `--annotate`/MCP kennen den Typ `highlight` mit `from`, `to`, `color`, `width` (wie
   `arrow` ohne Köpfe).
9. **Ebenenliste und Leiste** nennen das Objekt „Highlighter“ (de „Textmarker“). Eine Mischung für andere Objekte ist
   im Datenmodell möglich, aber nicht bedienbar; ein Schalter im Ebenen-Abschnitt wäre eine eigene Karte.

## Erwogene Alternativen
- **Kasten mit Mischmodus:** markiert nur Rechtecke; ein Textmarker folgt einer Zeile mit einstellbarer Breite, und die
  Biegung des Pfeils deckt schräge oder gebogene Zeilen ab. Die Mischung an der Ebene lässt einen Kasten später trotzdem
  zu.
- **Eigener Objekttyp:** jede Stelle, die den Pfeil kennt, bräuchte einen zweiten Zweig (vgl. ADR-0047).
- **Mischung am Pfeil statt an der Ebene:** Gemischt wird beim Einmischen der Gruppe (Zwischenbild bzw. Gio-Ebene) —
  dort, wo auch Deckkraft und Schatten wirken; die Ebene ist der richtige Ort.
- **Editor ohne Gio-Patch, Leinwandbild über den Renderer:** Alles bis zum obersten Textmarker müsste der Renderer in
  jedem Bild eines Zugs neu rechnen; bei 4K zu langsam. Der Patch ist klein und bleibt im Mischen der Ebenen.
- **Halbdurchsichtige Farbe ohne Mischung:** graut dunkle Schrift aus; genau das soll ein Textmarker nicht.

## Konsequenzen
- Ältere Schnapp-Stände (Version 1) lehnen Dateien mit Textmarker ab, wie bei Lupe und Ellipse.
- Der Gio-Patch wächst um eine Funktion; sie ist kein Teil der Gio-PRs #204/#205.
