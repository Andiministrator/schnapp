# ADR-0008: Dateiformat PNG mit eingebettetem SVG-Block

## Status
Akzeptiert (Andi, 18.11.2026; übernommen am 19.11.2026)

## Kontext
Snagit speichert in einem eigenen Format, das andere Programme nicht anzeigen. Fireworks speicherte dagegen PNGs, die
überall als fertiges Bild erscheinen und die Ebenen zusätzlich mitführen. Andi will dieses Prinzip: eine Datei, die
man überall verschicken kann und die in Schnapp voll bearbeitbar bleibt.

## Entscheidung
- Die Arbeitsdatei ist ein **PNG**: sichtbar das fertige Bild.
- In einem **eigenen PNG-Block** (privater Chunk, Name in E-03 festzulegen) liegt ein **vollständiges SVG** mit allen
  Ebenen: Screenshot als Pixelebene, Beschriftungen als Vektorebenen, das **unbeschnittene Original** und die
  **Angaben zum Stauchen** und Zuschneiden.
- Exporte: reines **SVG**, **PDF**, flaches **PNG/JPG**.
- **Schwärzen und Verpixeln überschreiben die Pixel im mitgespeicherten Original.** Das ist die einzige Stelle, an der
  Schnapp absichtlich zerstörend arbeitet. Ein geschwärzter Inhalt darf in keinem Teil der Datei rekonstruierbar sein.
- **Weitergeben ohne Bearbeitungsdaten** ist wählbar (flaches PNG ohne Block).
- **Keine Metadaten** mit lokalen Pfaden, Programm- oder Rechnernamen; der Autor ist einstellbar.
- Der Block trägt eine eigene Formatversion (ADR-0006).

## Konsequenzen
- Manche Dienste rechnen Bilder neu und werfen den Block weg; die Datei bleibt dann ein normales Bild. Das ist
  hinzunehmen und in der Oberfläche/Doku zu erwähnen.
- Das eingebettete Original kann die Datei groß machen; Kompression und ggf. Verzicht auf Original bei unbeschnittenen
  Bildern prüfen.
- Tests: Rundlauf (speichern → laden → identische Ebenen), Schwärzen-Test über alle Exportwege (ADR-0004),
  Metadaten-Test (keine Pfade, keine Rechnernamen).
