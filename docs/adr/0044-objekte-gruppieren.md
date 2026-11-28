# ADR-0044: Objekte gruppieren

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #354 in Sprint 14, im Auftrag Andis: „arbeite selbstständig und ziehe
auch selbstständig karten“). Grundlage: Andi, 26.11.2026: „sollte man objekte zusammengruppieren können, was meinste?“;
Einschätzung Claude auf der Karte: sinnvoll, etwa Pfeil + Text + Nummer als eine Einheit verschieben und ausblenden.
Setzt die Mehrfachauswahl voraus (ADR-0039 mit Nachträgen 1 bis 4). Ergänzt ADR-0032 (Ebenen) und ADR-0008 (Format).

## Kontext
Eine Beschriftung besteht oft aus mehreren Objekten: Pfeil und Text, Nummer und Kasten. Seit ADR-0039 lassen sie sich
gemeinsam wählen, verschieben und ausrichten, aber die Zusammengehörigkeit geht mit dem nächsten Klick verloren. Vorbilder:
Snagit, Google Slides und PowerPoint gruppieren flach (eine Gruppe wird als Ganzes gewählt, Strg+G / Strg+Umschalt+G);
Inkscape, Photoshop und Krita verschachteln Gruppen als eigene Knoten mit Kindern.

## Entscheidung
1. **Flache Gruppen:** Eine Gruppe ist eine Kennung an der Ebene jedes Mitglieds (`document.Ebene.Gruppe`, 0 = keine).
   Es gibt kein Gruppenobjekt und keine Verschachtelung: Die Stapelreihenfolge, Deckkraft, Schatten und Sichtbarkeit
   bleiben je Objekt, Renderer, SVG-Export und Editor zeichnen wie bisher. Eine Gruppe hat mindestens zwei Mitglieder;
   bleibt nach Löschen oder Rückgängig nur eines übrig, wirkt sie nicht mehr (ein einzelnes Mitglied verhält sich wie
   ein ungruppiertes Objekt).
2. **Gruppieren, Gruppierung aufheben:** **Strg+G** (Menü *Ebene* → „Group“) macht alle gewählten zu einer neuen Gruppe
   (ab zwei gewählten; gehörten sie schon zu Gruppen, werden diese ersetzt). **Strg+Umschalt+G** („Ungroup“) nimmt alle
   gewählten aus ihren Gruppen. Je ein Rückgängig-Schritt.
3. **Wählen:** Ein Klick auf ein Mitglied (Werkzeug Auswählen, auch Rechtsklick) wählt die ganze Gruppe; das
   angeklickte wird Hauptauswahl (Griffe, rechte Leiste wie ADR-0039). Umschalt+Klick nimmt die ganze Gruppe dazu
   bzw. heraus. Der Rahmen nimmt eine Gruppe nur, wenn alle ihre wählbaren Mitglieder darin liegen. Strg+A wählt alle.
   **Einzelnes Mitglied:** Ein Klick in der Ebenenliste wählt genau diese Zeile (wie bisher), damit sich ein Mitglied
   allein ändern lässt; Strg-/Umschalt-Klick in der Liste bleiben zeilenweise (ADR-0039 Nachtrag 4 Punkt 5).
4. **Gemeinsam:** Was ADR-0039 für mehrere gewählte vorsieht (Verschieben, Löschen, Doppeln, Ausrichten, Stil),
   gilt für die gewählte Gruppe von selbst. Zusätzlich wirken **Ausblenden (Strg+H) und Sperren (Strg+L)** an einem
   Mitglied auf die ganze Gruppe, wenn die Gruppe gewählt ist; Auge und Schloss in der Ebenenliste wirken weiter je
   Zeile. **Doppeln** gibt den Kopien eine neue Gruppe (dieselbe Aufteilung, andere Kennungen).
5. **Ebenenliste:** Mitglieder einer Gruppe tragen links an der Zeile einen farbigen Streifen (eine Farbe je Gruppe aus
   einer festen Reihe); der Name bleibt. Keine Aufklapp-Knoten (keine Verschachtelung).
6. **Format:** `s:gruppe="N"` (ganze Zahl 1 bis 2³⁰) als Ebenen-Attribut am Element (§8.5). Die Zahl bedeutet nur
   innerhalb der Datei etwas; Schnapp übernimmt sie beim Laden, eine neue Gruppe bekommt die kleinste freie Zahl. Eine Datei mit
   Gruppe schreibt **FormatVersion 2** (wie Lupe, Spitze, Schatten); ein Leser der Version 1 ignoriert das Attribut,
   die Objekte bleiben ungruppiert und sonst unverändert. Beispiel `testdata/format/groups`.
7. **Nicht Teil dieser Entscheidung:** verschachtelte Gruppen, Gruppendeckkraft, Skalieren der Gruppe als Ganzes
   (Griffe bleiben an der Hauptauswahl), `--annotate`/MCP.

## Erwogene Alternativen
- **Gruppenobjekt mit Kindern (Inkscape):** erlaubt Verschachtelung und Gruppendeckkraft, verlangt aber in jedem
  Durchgang über die Objekte (Renderer, Treffer, Einrasten, Nähte, Lupe, SVG, Rettungskopie, rund 80 Stellen)
  einen Baum statt einer Liste. Für „Pfeil + Text als Einheit“ reicht die flache Gruppe; ein späterer Umbau bleibt
  möglich, weil das Format nur ein Attribut trägt.
- **Ebenengruppe nur in der Liste (Photoshop-Ordner):** ändert das Wählen auf der Leinwand nicht, verfehlt den Wunsch
  „als Einheit verschieben“.
- **Mitglieder beim Gruppieren aneinanderrücken (Stapel):** verändert die Darstellung, wenn Fremdes dazwischen liegt;
  flache Gruppen brauchen es nicht.

## Konsequenzen
- `internal/document`: Feld `Ebene.Gruppe`, Gleichheit, Prüfung (≥ 0); `internal/fileformat`: Lesen/Schreiben von
  `s:gruppe`, `schreibVersion`, Spezifikation §8.5 und Beispiel im selben Commit (ADR-0008 Nachtrag 3).
- `internal/canvas`: `Gruppieren`, `GruppierungAufheben`, Wählen ganzer Gruppen (Klick, Umschalt+Klick, Rahmen),
  Ausblenden/Sperren der Gruppe, Doppeln mit neuen Kennungen; `internal/ui`: Menüeinträge und Kürzel
  (Tastentabelle, Übersicht F1), Streifen in der Ebenenliste.
- Tests, Mutanten, Sichtprobe; README en/de.

## Nachtrag 1 (28.11.2026, Kritik #354, F-958 bis F-963)
1. **Wann eine Gruppe als gewählt gilt (F-958, F-959):** wenn mindestens zwei Mitglieder und alle wählbaren gewählt
   sind, oder wenn Strg+H/Strg+L sie eben geschaltet hat und die Auswahl seither nicht anders geändert wurde. Ein
   einzeln über die Ebenenliste gewähltes Mitglied ist keine gewählte Gruppe, auch wenn die übrigen gesperrt oder
   ausgeblendet sind. Einblenden bzw. Entsperren der Gruppe wählt sie wieder ganz; so wechselt jeder Druck den
   Zustand aller Mitglieder. Menü und Kontextmenü heißen dann „Hide group“/„Show group“, „Lock group“/„Unlock group“.
2. **Gruppieren** vergleicht die gewählten mit den wählbaren Mitgliedern: Eine ganz gewählte Gruppe mit gesperrtem
   Mitglied lässt sich nicht erneut gruppieren (F-960).
3. **Keine verwaiste Kennung (F-962):** Gruppieren und Aufheben nehmen einem Objekt, das danach als einziges seine
   Kennung trüge, die Kennung im selben Schritt; „Ungroup“ wirkt nur an wirksamen Gruppen; die Datei schreibt
   `s:gruppe` nur für Gruppen mit mindestens zwei Mitgliedern (sonst bliebe sie ohne Grund Version 2).
4. **Aufwand (F-961):** Rahmen und Menüzustand gehen den Stapel je Frage höchstens zweimal durch.
5. **Grenzen (F-963):** Ein einzelnes Mitglied lässt sich auf der Leinwand nicht allein verschieben (ein Druck wählt die
   Gruppe); dafür die Gruppierung aufheben. Ein Doppelklick auf einen Text der Gruppe öffnet die Texteingabe und wählt
   nur diesen Text.

## Nachtrag 2 (28.11.2026, Verify #390, F-964 bis F-969)
Ersetzt Nachtrag 1 Punkt 1 für Strg+H und Strg+L:
1. **Was ein Druck schaltet:** Ist die Gruppe der Hauptauswahl gewählt, ihre Mitglieder, die noch nicht im Ziel sind
   (ein vorher einzeln ausgeblendetes oder gesperrtes Mitglied bleibt, wie es ist, F-966). Ist die Hauptauswahl selbst
   ausgeblendet bzw. gesperrt und alle Mitglieder ihrer Gruppe sind es auch (etwa nach Rückgängig und Wiederholen),
   alle Mitglieder zurück (F-965). Sonst nur die Hauptauswahl.
2. **Zurückschalten:** Der nächste Druck desselben Schalters schaltet genau die eben geschaltete Menge zurück und stellt
   die Auswahl von davor her, solange die Auswahl seither nicht anders geändert wurde — auch wenn die Hauptauswahl beim
   Ausblenden auf ein mitgewähltes Fremdobjekt rückte (F-964). Das Fremdobjekt selbst schaltet Strg+H nicht mit
   (Ebenen-Handlungen bleiben an der Hauptauswahl, ADR-0039 Nachtrag 4 Punkt 6).
3. Das Format lässt `s:gruppe` an einem einzelnen Mitglied weg (§8.5, SHOULD; F-968).

## Nachtrag 3 (28.11.2026, Sitzungs-QA, F-978, F-981, F-983)
1. **Verhältnis der Nachträge (F-981):** Von Nachtrag 1 Punkt 1 bleiben die Bedingung „mindestens zwei und alle
   wählbaren Mitglieder gewählt“ (F-959) und die Menütexte „Hide/Show/Lock/Unlock group“; überholt durch Nachtrag 2
   sind nur „oder wenn Strg+H/Strg+L sie eben geschaltet hat“ und „Einblenden wählt sie wieder ganz“ (jetzt: genau die
   geschaltete Menge zurück, mit der Auswahl von davor; bei einer ganz ausgeschalteten Gruppe ohne Gedächtnis wird sie
   wieder ganz gewählt).
2. **Ergänzungen zu Punkt 2 und 4:** „Group“/„Ungroup“ stehen auch im Kontextmenü eines Objekts, dort nur, wo sie
   wirken. Die Kopie eines einzelnen Mitglieds (Doppeln über die Ebenenliste) steht allein, ohne Gruppe.
3. **Vorbilder (F-983):** Die Angaben zu Snagit, Google Slides, PowerPoint, Inkscape, Photoshop und Krita im Kontext
   stammen aus allgemeiner Kenntnis der Programme, nicht aus einer nachgeschlagenen Quelle; sie begründen nur die Wahl
   flacher Gruppen mit Strg+G und sind keine Spezifikation.
4. **Verweise:** ADR-0032 (Strg+H/L, Ebenenliste), ADR-0037 (Schnellstil an mehreren), ADR-0038 (Kontextmenü) und
   ADR-0021 Nachtrag 26 (Feld Schatten) tragen je einen Verweis auf diese ADR bzw. ADR-0039 Nachtrag 4.
5. **Sitzungs-Kritik (F-970 bis F-973):** Das Feld Schatten gehört bei mehreren gewählten einem eigenen Ziel „mehrere
   mit dieser Hauptauswahl“, das nie mit dem eines einzelnen Objekts zusammenfällt; nach Texteingabe oder
   Einzelauswahl kehrt es nicht ohne Klick zurück. Der Auswahl-Zähler zählt nur echte Auswahländerungen, so überstehen
   Anker der Ebenenliste und Gedächtnis von Strg+H/L Deckkraft, Umbenennen und das Bestätigen einer Texteingabe;
   Rückgängig/Wiederholen verwerfen das Gedächtnis ausdrücklich. Strg+H/L bestätigen eine Texteingabe, bevor sie
   rechnen. Das Gedächtnis gilt je Schalter: Ein Druck des anderen Schalters löscht es nicht.
6. **Grenze (F-973):** Nach Strg+H und Rückgängig ist nur die Hauptauswahl gewählt (Rückgängig stellt keine Auswahl
   her, ADR-0039 Nachtrag 2); ein weiteres Strg+H schaltet dann nur sie. Wer die Gruppe meint, klickt sie erneut an.
