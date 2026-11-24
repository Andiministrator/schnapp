# ADR-0026: Menü im Editor — jede Tastenfunktion auch per Maus, Kürzel sichtbar

## Status
Akzeptiert (Claude Opus 5.5, 20.11.2026, Karte #104 in Sprint 7). Grundsatz von Andi (E-21, 20.11.2026): „alles was mit
tastenkürzeln geht, muss auch über die werkzeugleiste oder das menü gehen.“ Die Wahl zwischen Menü und Knöpfen hat
Andi offen gelassen; Claude entscheidet, Andi ist informiert.

## Kontext
Mehrere Funktionen des Editors gehen nur per Tastatur oder per Doppelklick mit gewähltem Werkzeug:
- **Kopieren** (Strg+C), **Löschen** des ausgewählten Objekts (Entf, Rücktaste), **Schließen** (Strg+W),
- **Text bearbeiten** (Enter am ausgewählten Text, sonst Doppelklick),
- **Naht entfernen** (Doppelklick mit X auf die Naht, ADR-0023).

Die Leiste bricht schon jetzt um, wenn das Fenster schmal ist (F-148). Fünf weitere Textknöpfe würden das verschärfen.
Gio hat kein eingebautes Menü und keine Tooltips; `gioui.org/x` brächte beides, ist aber ein weiteres Fremdpaket für
wenig Code (ADR-0011).

## Entscheidung

1. **Ein Knopf „Menü“** in der ersten Zeile der Leiste, links neben „Speichern“; „Speichern“ bleibt rechts außen. Ein Klick
   klappt eine Liste darunter auf. Ein Klick auf einen Eintrag führt ihn aus und schließt die Liste. Ein Klick daneben,
   Esc oder ein zweiter Klick auf „Menü“ schließen sie, ohne etwas auszuführen. Die Liste liegt über der Leinwand und
   verdeckt sie nur, solange sie offen ist.
2. **Einträge** in Gruppen, jeweils mit dem Kürzel rechtsbündig, nicht bedienbare Einträge grau:
   - Kopieren (Strg+C), Exportieren … (Strg+Umschalt+E), Speichern (Strg+S)
   - Rückgängig (Strg+Z), Wiederholen (Strg+Umschalt+Z, auch Strg+Y)
   - Text bearbeiten (Enter), Löschen (Entf)
   - Ganzes Bild, Gestauchtes zurückholen (englisch „Restore cut-out areas“)
   - Einpassen (Strg+0), 1:1 (Strg+1)
   - Schließen (Strg+W)

   Das Menü führt also auch Funktionen, die schon als Knopf da sind. So zeigt es alle Kürzel an einer Stelle und
   ersetzt die Tooltips, die Gio nicht hat.
3. **Werkzeugknöpfe zeigen ihr Kürzel beim Darüberfahren** als kleine Beschriftung unter dem Knopf („Arrow (A)“). Das
   ist ohne Verzögerung und ohne Fremdpaket gebaut (`Clickable.Hovered`).
4. **Gestauchtes zurückholen** ist neu: Diese Aktion hebt alle Stauch-Streifen in einem Rückgängig-Schritt auf,
   der Zuschnitt bleibt. Sie ist die Menü-Entsprechung zum Doppelklick mit X. Der Doppelklick entfernt weiter eine
   einzelne Naht. Eine einzelne Naht ohne Zeiger auszuwählen, lohnt den Aufwand nicht: Wer nur eine zurückwill, holt
   alle zurück und staucht neu oder nutzt den Doppelklick.
5. **Ausnahmen ohne Menüeintrag**, weil die Maus schon einen gleichwertigen Weg hat oder die Taste nur in der
   Texteingabe gilt:
   - Esc in Stufen: Text verwerfen (danach Rückgängig), Ziehen abbrechen (Maus loslassen außerhalb oder Rückgängig),
     Abwählen (Klick auf leere Fläche), zurück zu Auswählen (Knopf „Auswählen“).
   - Enter beim Tippen (Text bestätigen: Klick daneben), Umschalt+Enter (Zeilenumbruch) und Rücktaste beim Tippen.
   - Enter auf einem Knopf mit Tastaturfokus.
   - Die Buchstaben der Werkzeuge (V, A, R, T, N, B, P, C, X): Sie haben ihre Knöpfe in der Leiste.
6. **Ein Test hält die Regel:** Alle Kürzel stehen in einer Tabelle ohne Gio (`ui/kuerzel.go`). Aus ihr entstehen die
   Tastenfilter, die Menüeinträge und die Hinweise. Der Test verlangt für jedes Kürzel einen Knopf oder Menüeintrag oder
   eine Ausnahme aus Punkt 5 mit Begründung. Eine neue Taste ohne Eintrag macht ihn rot.

## Konsequenzen
- Neue Oberflächentexte (Englisch, Deutsch in `po/de.po`).
- Die Leiste wird um einen Knopf breiter. Drehbücher mit festen Koordinaten werden neu gefahren
  (`knowledge/topics/lehren.md`).
- Neue Funktionen bekommen künftig im selben Zug einen Menüeintrag (E-21).
- Tooltips über einem Text oder einer Nummer auf der Leinwand gibt es weiter nicht.

## Nachtrag 1 (20.11.2026): Nacharbeit aus der Kritik zu Sprint 7 (Fable)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #104, Nacharbeit Kritik Sprint 7).

- **Punkt 4 genauer (F-260):** „Gestauchtes zurückholen“ hebt die Stauch-Streifen auf, die den Zuschnitt schneiden.
  Streifen ganz außerhalb des Zuschnitts sind ohnehin nicht zu sehen und bleiben; „Ganzes Bild“ zeigt sie danach
  wieder mit Naht, und der Eintrag wird wieder bedienbar.
- **Punkt 5 ergänzt (F-254):** Umschalt beim Ziehen (Pfeil in 45°-Schritten, Kasten als Quadrat) bleibt eine Taste ohne
  Menüeintrag. Ohne Umschalt entsteht dieselbe Form frei gezogen; die Einrastung ist eine Hilfe, keine eigene Funktion.
- **Offenes Menü (F-257):** Solange die Liste offen ist, wirken nur Esc (schließt) und Enter (löst den Knopf oder
  Eintrag mit Tastaturfokus aus). Andere Tasten tun nichts, damit nicht unbemerkt hinter der Liste gelöscht oder das
  Werkzeug gewechselt wird. *Erweitert: Pfeile, Pos1, Ende (Nachtrag 2), Leertaste und Tab (Nachtrag 3).*
- **Maus und Taste in der Texteingabe (F-257):** Ein Menüeintrag bestätigt einen laufenden Text und führt dann aus
  (wie die Knöpfe der Leiste). Per Taste wirken Kopieren und Export während der Texteingabe nicht, weil Strg+C dort
  nicht als Kopieren des Bilds gemeint sein muss. Einträge, die eine abgeschlossene Eingabe brauchen (Löschen,
  Text bearbeiten, Ganzes Bild, Gestauchtes zurückholen), sind während der Eingabe grau.
- **Niedrige Fenster (F-252):** Passt die Liste nicht unter den Knopf, rückt sie nach oben; ist das Fenster niedriger
  als die Liste, entfallen die Gruppenabstände und die Zeilen werden niedriger (mindestens 20 dp, höchstens 32 dp).
- **Offen (Karte):** Bedienung der Liste allein per Tastatur (Fokus in die Liste, Pfeiltasten, Fokusring), F-255 — erledigt in Nachtrag 2.

## Nachtrag 2 (20.11.2026): Menü allein per Tastatur bedienbar (F-255)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #110).

- **Öffnen per Tastatur** (Tab bis „Menü“, Enter oder Leertaste) setzt den Fokus auf den ersten bedienbaren Eintrag.
  Per Maus geöffnet bleibt der Fokus unberührt, damit kein Fokusring ohne Tastatur erscheint.
- **Bei offenem Menü wirken zusätzlich zu Esc und Enter** die Pfeile hoch und runter, Pos1 und Ende: Sie wählen den
  Eintrag, ohne etwas auszuführen. Graue Einträge werden übersprungen. An den Enden **läuft die Liste um** (wie die
  Menüs in KDE und Qt); bei zwölf Einträgen ist das schneller als Anhalten, und der Umlauf ist umkehrbar. Alle anderen
  Tasten und getippter Text wirken weiter nicht (Nachtrag 1).
- **Der Fokus kehrt immer zurück:** Esc schließt und gibt ihn an den Menüknopf *(eingeschränkt durch Nachtrag 3, V-1: nur nach Öffnen per Tastatur und ohne laufende Texteingabe, sonst an die Leinwand)*; nach einem ausgeführten Eintrag oder
  einem Klick daneben liegt er auf der Leinwand. Ohne das fiele er auf nichts, und nach „Text bearbeiten“ ginge
  getippter Text verloren.
- **Fokusring:** 2 px orange (Corporate Design §6) an Knöpfen, am Menüknopf und am fokussierten Menüeintrag, nicht
  nur die Hover-Fläche. *Abstand und Ausnahme am Menüeintrag: Nachtrag 3, V-4.*
- **Werkzeug-Hinweis** (Name und Buchstabe) erscheint auch bei Tastaturfokus auf dem Werkzeugknopf, nicht nur beim
  Zeiger; ist ein Knopf fokussiert, zeigt nur er den Hinweis.

## Nachtrag 3 (20.11.2026): Nacharbeit aus dem Verify-Lauf zu Nachtrag 2 (Fable)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #110, Nacharbeit Verify Fable).

- **Leertaste** führt am fokussierten Menüknopf wie Enter aus (öffnet mit Fokus auf dem ersten Eintrag); am
  fokussierten Eintrag löst sie ihn aus (Gio-Knopf). Die Art des Öffnens kommt aus der Taste, nicht aus dem Zeigerort:
  Ein ruhender Zeiger über dem Knopf machte die Leertaste sonst zur Maus. Enter und Leertaste öffnen sofort, nicht
  erst im nächsten Bild.
- **Tab** fährt bei offenem Menü weiter wie sonst (Einträge, Leinwand, Leistenknöpfe); das Menü bleibt offen und
  bedienbar. Wer von einem Leistenknopf aus Pfeil runter drückt, landet wieder in der Liste.
- **V-1, Fokus nach Esc:** Esc gibt den Fokus nur dann an den Menüknopf, wenn die Liste per Tastatur geöffnet wurde und
  keine Texteingabe läuft; sonst geht er an die Leinwand. Sonst gingen nach einem per Maus geöffneten Menü getippte
  Zeichen verloren, die Leertaste öffnete das Menü (Tab ist während der Texteingabe gesperrt, F-153), und Enter öffnete
  es statt „Text bearbeiten“.
- **V-3, Pfeile sofort:** Die Filter für Pfeile, Pos1 und Ende stehen immer, nicht erst im Bild nach dem Öffnen; kommen
  Enter und Runter ohne Bild dazwischen, geht Runter nicht mehr verloren. Bei geschlossenem Menü bleiben sie ohne
  Wirkung. Begründung, dass nichts verschluckt wird: Gio fährt den Fokus nur per Tab (`MoveFocus`), Pfeiltasten
  verarbeitet dort nur die mobile Fensterschicht; im Editor hatten sie keine Bedeutung.
- **V-4, Fokusring mit Abstand:** Leistenknöpfe: Ring 2 px breit, 2 px Abstand zur Randlinie (im Probelauf gemessen).
  Menüeintrag, **begründete Ausnahme**: Die Zeilen stoßen aneinander und an den Rand der Liste, außen ist kein Platz;
  der Ring liegt innen, 2 px breit mit 2 px Abstand zum Zeilenrand.

## Nachtrag 4 (20.11.2026): Menü-Navigation als benannte Ausnahme von der Tastentabelle (F-286)

**Status:** Akzeptiert (Claude Sonnet 5.5, Karte #116).

- **Pfeil hoch, Pfeil runter, Pos1, Ende und die Leertaste am Menüknopf** (Nachträge 2 und 3) stehen nicht in
  `tastenTabelle`. Sie sind keine Kürzel mit eigener Handlung, sondern die Bedienung der offenen Liste; ein Knopf oder
  Menüeintrag, den der Test aus Punkt 6 verlangt, wäre für sie sinnlos (der Menüeintrag *ist* ihr Ziel). Sie werden in
  `ui/kuerzel.go` (`menueNavigationsFilter`) als einzige Tasten außerhalb der Tabelle angemeldet. Das ist die sechste
  begründete Ausnahme neben den Punkt-5-Ausnahmen.
- **Der Test hält die Ausnahme eng:** `TestTastenfilterNurTabelleUndMenueNavigation` verlangt, dass die Tastenschleife
  (`editor.alleTastenFilter`) genau die Filter der Tabelle, den Fokus der Leinwand und diese fünf Navigationsfilter
  anmeldet, und dass kein Navigationsfilter zugleich in der Tabelle steht. Eine weitere Taste außerhalb der Tabelle
  macht ihn rot; sie gehört dann in die Tabelle (mit Knopf, Menüeintrag oder begründeter Ausnahme) oder bekommt einen
  eigenen Nachtrag.
- **Verworfen:** die fünf Tasten über die Tabelle führen. Dafür bräuchte die Tabelle ein Feld „Navigation“, und die
  Leertaste mit Tag (`Focus`) passt nicht zur Form der Einträge; die Tabelle zählte dann Tasten ohne Handlung mit.

## Nachtrag 5 (23.11.2026): Menüzeile statt Knopf „Menü“ (ADR-0031)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #187).

Punkt 1 (Knopf „Menü“ neben „Speichern“) und Punkt 3 (Hinweis unter dem Werkzeugknopf) sind durch ADR-0031 ersetzt:
eine Menüzeile mit sechs Menüs oben, Werkzeugleiste links mit Hinweis rechts neben dem Knopf. Die Regeln der Punkte 2
(Einträge mit Kürzel, graue Einträge), 4 bis 6 und der Nachträge 1 bis 4 gelten weiter; die Menü-Navigation (Nachtrag
4) kommt um F10 und Pfeil links/rechts dazu.


## Nachtrag 6 (23.11.2026): Eintrag „Kantenform/Seam style“ (F-425, Karte #182)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #182).

Das Menü „Bild“ trägt den Eintrag **Kantenform** (englisch „Seam style“; `aktKantenform` in `internal/ui/kuerzel.go`) und
folgt damit ADR-0023 Nachträgen 6 und 7:

- **Reihum:** Jeder Aufruf wechselt die Form der Stauch-Kante weiter (Abriss, Welle, Zacken, wieder Abriss) als ein
  Rückgängig-Schritt (`WechsleKante`); es gibt kein eigenes Kürzel, das Kürzel-Feld in `kuerzel.go` bleibt leer.
- **Grau ohne Naht:** Ohne Naht (`KannKanteWechseln` falsch) ist der Eintrag grau und tut nichts, das Bild bleibt ungeändert.
- **Zeigt die aktuelle Form:** Der Menütext ist die Aussage „Seam style: torn edge / wave / zigzag“ (`kantenMeldung`), also
  auch ohne Wechsel sichtbar; dieselbe Zeile erscheint nach einem Wechsel in der Fußzeile.

Getestet von `TestMenueKantenform` in `internal/ui/menue_test.go` (Eintrag vorhanden, Text gleich der aktuellen Form, grau
ohne Naht, Reihenfolge, Rückgängig).

## Nachtrag 7 (24.11.2026): Eintrag „Einstellungen…/Settings…“ mit Strg+, (ADR-0034, Karte #175)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #175).

Das Menü „Datei“ trägt hinter „Kopieren“ in einer eigenen Gruppe den Eintrag **Einstellungen…** (englisch „Settings…“,
`aktEinstellungen`), Kürzel **Strg+,** (Komma) in der Tastentabelle `internal/ui/kuerzel.go`; die Hilfe (F1) nennt es
damit von selbst. Er öffnet die Einstellungstafel nach ADR-0034 (`internal/ui/einstellungen.go`). Solange die Tafel offen
ist, wirkt keine Taste der Tabelle: Die Tafel holt ihre Tasten (Tab, Umschalt+Tab, Strg+Bild↑/↓, Leertaste, Enter, Esc,
Pfeile, −/+, Rücktaste) vor `tasten` ab, `tasten` verwirft den Rest; ihre Auswahllisten laufen als Liste des Menüs
(`listeEinstellung`) und werden wie jede Liste in `tasten` bedient. Strg+Bild↑/↓ wechselt in der Tafel den Reiter statt
der Ebene. Getestet von `TestMenueZeigtKuerzel` und `TestTafelSperrtEditor`.

## Nachtrag 8 (24.11.2026): Eintrag „Text aus dem Bild kopieren“ mit Strg+Umschalt+T; graue Einträge mit Erklärung (ADR-0035, Karte #195)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #195).

- **Eintrag:** Das Menü „Bearbeiten“ trägt am Ende in einer eigenen Gruppe **Text aus dem Bild kopieren** (englisch
  „Copy text from image“, `aktTextKopieren`), Kürzel **Strg+Umschalt+T** in der Tastentabelle. Das Kürzel war frei: T
  allein wählt das Werkzeug Text, und die Tabelle trennt Umschalter genau (`passt`). Der Eintrag schaltet den
  Bereichsmodus nach ADR-0035 Punkt 5 ein.
- **Grau mit Erklärung (neu):** Ein grauer Eintrag kann eine Erklärung tragen (`menueEintrag.erklaerung`). Solange er
  überfahren oder per Tastatur angesteuert ist, zeigt die Fußzeile sie statt Meldung oder Hinweis (`menueErklaerung`). Er
  meldet sich dafür als Klickfläche an (ohne Hand-Zeiger) und ist, anders als Nachtrag 2 für graue Einträge sagt
  („werden übersprungen“), mit Pfeilen, Pos1 und Ende erreichbar (Fokusring ohne Hintergrund); auslösen lässt er sich weder per Klick noch
  per Enter. Graue Einträge ohne Erklärung bleiben, wie sie waren. Bisher trägt nur „Text aus dem Bild kopieren“ eine:
  „Looking for tesseract…“ (Suche läuft), „Install tesseract to recognize text.“, „Install language data for tesseract to
  recognize text.“, „Text recognition is not available: …“ und „Text recognition is already running.“. Das Kürzel zeigt
  dieselbe Erklärung als Meldung, statt still nichts zu tun.

Getestet von `TestMenueZeigtKuerzel`, `TestMenuePfeiltasten`, `TestMenueFokussierterEintragWirdGrau` (Ende und Umlauf
erreichen jetzt den grauen Eintrag mit Erklärung), `TestOcrEintragGrauMitErklaerung`, `TestOcrKuerzelOhneTesseract` und
`TestOcrUeberRouter`; Mutanten U19 bis U24 in `scripts/mutanten/ocr-ui.json`.

## Nachtrag 9 (24.11.2026): Eintrag „Letzte Aufnahmen/Recent captures“ mit Strg+Umschalt+R (ADR-0036, Karte #174)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #174 Teil 1).

Das Menü „Ansicht“ trägt am Ende in einer eigenen Gruppe den Schalter **Letzte Aufnahmen** (englisch „Recent captures“,
`aktLeiste`), Kürzel **Strg+Umschalt+R** in der Tastentabelle; R allein bleibt das Werkzeug Rechteck. Er blendet die
Leiste der letzten Aufnahmen ein und aus (ADR-0036 Punkt 2) und merkt den Zustand in `[library] bar` (ADR-0018
Nachtrag 6). Kennt das Fenster keine Bibliothek, ist der Eintrag grau mit Erklärung nach Nachtrag 8 („Not available in
this window: it has no library.“; im Programm kommt das nicht vor, nur in Tests ohne Anbindung). Die Vorschauen der
Leiste sind Knöpfe der Fokusfahrt: Tab erreicht sie, Pfeile links und rechts, Pos1 und Ende wandern zwischen ihnen
(nicht in der Tastentabelle, wie die Kacheln der Schnellstile), Enter und Leertaste öffnen. Getestet von
`TestMenueZeigtKuerzel`, `TestLeisteSchalter`, `TestLeisteTastatur` und `TestLeisteLeertaste`.
