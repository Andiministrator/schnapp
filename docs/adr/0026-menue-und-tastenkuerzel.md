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
