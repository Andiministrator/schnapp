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

## Nachtrag 1 (19.11.2026): Blöcke festgelegt (E-03)

**Status:** Akzeptiert (Andi, 19.11.2026: „ja, passt“ zum Vorschlag zwei Blöcke).

- **Zwei private PNG-Blöcke:**
  - `snAP` — das SVG mit allen Ebenen, zlib-komprimiert (wie `zTXt`), mit Formatversion.
  - `snAO` — das Original als PNG-Daten (binär, unverändert komprimiert); das SVG verweist auf diesen Block statt das
    Original als Base64 einzubetten. Spart rund ein Drittel der Originalgröße (Base64 vergrößert bereits komprimierte
    Daten um 4/3, zlib holt das nicht zurück). Fehlt `snAO` (unbeschnittenes, ungestauchtes Bild), ist das sichtbare
    Bild ohne Beschriftungen das Original — ob das trägt, klärt die Umsetzung (ADR-0008 Konsequenzen).
- **Groß-/Kleinschreibung nach PNG-Spezifikation:** 1. Buchstabe klein = zusätzlicher Block (Betrachter ignorieren
  ihn, das Bild erscheint überall), 2. klein = privat, 3. groß = vorgeschrieben, 4. **groß = nicht kopiersicher**:
  Ein fremdes Programm, das die Pixel ändert, muss die Blöcke verwerfen — sonst lägen beim Öffnen in Schnapp alte
  Ebenen über einem fremd bearbeiteten Bild.
- Export als reines SVG bettet das Original als Daten-URI ein. Schwärzen überschreibt die Pixel im `snAO`-Original
  wie zuvor im eingebetteten (Kernprinzip 2); der [S]-Test deckt beide Blöcke ab.
