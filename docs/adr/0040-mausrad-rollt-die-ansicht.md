# ADR-0040: Mausrad rollt die Ansicht, Strg+Rad zoomt

## Status
Akzeptiert (Claude Opus 5.5, 27.11.2026, Karte #375 in Sprint 14). Grundlage: Andi, 27.11.2026: „ja, finde ich gut“ auf
den Vorschlag, das Rad wie in den meisten Programmen rollen zu lassen. Ändert ADR-0020 (Leinwand: „Mausrad zoomt zum
Zeiger“) und ergänzt ADR-0039 Nachtrag 1 (Leertaste+Ziehen) sowie ADR-0031 Nachtrag 9 (Navigator).

## Kontext
Im Editor zoomt das Mausrad bisher zum Zeiger, in jedem Werkzeug. Inkscape, Figma, Krita (Voreinstellung),
GIMP, Photoshop und Browser rollen mit dem Rad, zoomen mit Strg+Rad. Am Touchpad gilt das umso mehr: Zwei Finger
schieben dort in allen Programmen den Inhalt, in Schnapp zoomten sie — schwer zu bedienen.

Was Gio unter Wayland liefert (gepatchtes Gio unter `third_party/gio`, `app/os_wayland.go`): Mausrad und Touchpad
kommen beide als `pointer.Scroll` mit `Source: pointer.Mouse`; `wl_pointer.axis_source` wird ignoriert, die Quelle ist
also nicht unterscheidbar. Ein Rastschritt des Rads kommt als rund 150 px (Wayland-Wert × 10), das Touchpad fein und
mit Nachlauf (Gio-`fling`). Umschalt allein macht aus dem senkrechten Rad ein waagerechtes. Die Umschalter
(`Modifiers`) liegen dem Ereignis bei. Gesten zum Zoomen mit zwei Fingern (pointer-gestures, Pinch) meldet Gio nicht.

## Entscheidung
1. **Rad ohne Strg rollt die Ansicht:** senkrecht hoch/runter, waagerecht (Neigerad, Touchpad seitwärts, Umschalt+Rad)
   links/rechts, um die gemeldete Strecke in Fensterpixeln. Das gilt in jedem Werkzeug, auch während einer
   Texteingabe; während eines Zugs (Ziehen, Rahmen, Leertaste+Ziehen) geschieht nichts, wie bisher beim Zoomen.
2. **Strg+Rad zoomt zum Zeiger** wie bisher das Rad allein (10 % je Rastschritt, gleiche Grenzen). Strg+Umschalt+Rad
   zoomt ebenso (Gio meldet es senkrecht).
3. **Touchpad:** Zwei Finger rollen (Punkt 1), Strg + zwei Finger zoomen. Pinch kommt nicht an und bleibt außen vor,
   bis Gio pointer-gestures meldet.
4. **Bleibt:** Das Rad über dem Navigator zoomt um die Fenstermitte (ADR-0031 Nachtrag 9) — dort gibt es nichts zu
   rollen. Rechte Leiste, Tafeln, Listen, Bibliothek und Leiste der letzten Aufnahmen rollen wie bisher selbst.
5. **Rollen begrenzt nicht** — wie Leertaste+Ziehen und die mittlere Taste darf die Ansicht über den Bildrand hinaus
   verschoben werden; Strg+0 passt wieder ein.

## Erwogene Alternativen
- **Rad zoomt weiter, Strg+Rad rollt:** hält die Gewohnheit seit Sprint 4, widerspricht aber allen Vorbildern und
  dem Touchpad.
- **Quelle unterscheiden (Rad zoomt, Touchpad rollt):** wäre ohne Gio-Patch nicht machbar (axis_source verworfen) und
  verhielte sich je Gerät anders — schwer zu erklären.
- **Einstellung in `settings.ini` (zurück zum zoomenden Rad):** Andi, 27.11.2026: „nein, erstmal nicht“. Erst die
  übliche Voreinstellung, ein Schalter nur bei Bedarf.

## Konsequenzen
- `internal/canvas`: `Sitzung.Rollen(dx, dy)` (verschiebt die Abbildung), `Scrollen` bleibt das Zoomen; `internal/ui`:
  Zeigerziel der Leinwand meldet `ScrollX` und `ScrollY`, Strg entscheidet zwischen Zoomen und Rollen.
- README en/de, Übersicht F1 („Zoom: Ctrl+wheel“, „Scroll the view: wheel“), CLAUDE.md, Tests, Mutanten, Sichtprobe
  im virtuellen KWin (die Fernbedienung kann Rad-Schritte senden).
