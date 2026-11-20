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

- **Punkt 4 genauer (F-260):** „Gestauchtes zurückholen“ hebt die Stauch-Streifen auf, die den Zuschnitt schneiden.
  Streifen ganz außerhalb des Zuschnitts sind ohnehin nicht zu sehen und bleiben; „Ganzes Bild“ zeigt sie danach
  wieder mit Naht, und der Eintrag wird wieder bedienbar.
- **Punkt 5 ergänzt (F-254):** Umschalt beim Ziehen (Pfeil in 45°-Schritten, Kasten als Quadrat) bleibt eine Taste ohne
  Menüeintrag. Ohne Umschalt entsteht dieselbe Form frei gezogen; die Einrastung ist eine Hilfe, keine eigene Funktion.
- **Offenes Menü (F-257):** Solange die Liste offen ist, wirken nur Esc (schließt) und Enter (löst den Knopf oder
  Eintrag mit Tastaturfokus aus). Andere Tasten tun nichts, damit nicht unbemerkt hinter der Liste gelöscht oder das
  Werkzeug gewechselt wird.
- **Maus und Taste in der Texteingabe (F-257):** Ein Menüeintrag bestätigt einen laufenden Text und führt dann aus
  (wie die Knöpfe der Leiste). Per Taste wirken Kopieren und Export während der Texteingabe nicht, weil Strg+C dort
  nicht als Kopieren des Bilds gemeint sein muss. Einträge, die eine abgeschlossene Eingabe brauchen (Löschen,
  Text bearbeiten, Ganzes Bild, Gestauchtes zurückholen), sind während der Eingabe grau.
- **Niedrige Fenster (F-252):** Passt die Liste nicht unter den Knopf, rückt sie nach oben; ist das Fenster niedriger
  als die Liste, entfallen die Gruppenabstände und die Zeilen werden niedriger (mindestens 20 dp, höchstens 32 dp).
- **Offen (Karte):** Bedienung der Liste allein per Tastatur (Fokus in die Liste, Pfeiltasten, Fokusring), F-255 — erledigt in Nachtrag 2.

## Nachtrag 2 (20.11.2026): Menü allein per Tastatur bedienbar (F-255)

- **Öffnen per Tastatur** (Tab bis „Menü“, Enter oder Leertaste) setzt den Fokus auf den ersten bedienbaren Eintrag.
  Per Maus geöffnet bleibt der Fokus unberührt, damit kein Fokusring ohne Tastatur erscheint.
- **Bei offenem Menü wirken zusätzlich zu Esc und Enter** die Pfeile hoch und runter, Pos1 und Ende: Sie wählen den
  Eintrag, ohne etwas auszuführen. Graue Einträge werden übersprungen. An den Enden **läuft die Liste um** (wie die
  Menüs in KDE und Qt); bei zwölf Einträgen ist das schneller als Anhalten, und der Umlauf ist umkehrbar. Alle anderen
  Tasten und getippter Text wirken weiter nicht (Nachtrag 1).
- **Der Fokus kehrt immer zurück:** Esc schließt und gibt ihn an den Menüknopf; nach einem ausgeführten Eintrag oder
  einem Klick daneben liegt er auf der Leinwand. Ohne das fiele er auf nichts, und nach „Text bearbeiten“ ginge
  getippter Text verloren.
- **Fokusring:** 2 px orange (Corporate Design §6) an Knöpfen, am Menüknopf und am fokussierten Menüeintrag, nicht
  nur die Hover-Fläche.
- **Werkzeug-Hinweis** (Name und Buchstabe) erscheint auch bei Tastaturfokus auf dem Werkzeugknopf, nicht nur beim
  Zeiger; ist ein Knopf fokussiert, zeigt nur er den Hinweis.
