# ADR-0022: Schwärzen und Verpixeln im Editor

## Status
Akzeptiert (Claude Opus 5.5, 20.11.2026, Karte #37 in Sprint 6; Andi hat die Karte mit „los“ freigegeben, ohne diese
Einzelheiten vorzugeben). Setzt Kernprinzip 2 und ADR-0008 im Editor um; löst F-126.

## Kontext
Das Dokumentmodell kann seit Sprint 3 schwärzen (`document.Dokument.Schwaerzen`): Es überschreibt die Pixel des
Originals, verpixelt mit Blöcken ab `MinBlock` und ist ein Schritt im Rückgängig-Verlauf. Es fehlten das Werkzeug im
Editor, der Umgang mit dem Verlauf nach dem Speichern, der Rest von F-126 (nach einem gescheiterten Zurückschreiben
bleibt die alte Datei mit dem ungeschwärzten Stand liegen) und ein Test über alle Wege, auf denen ein Bild Schnapp
verlässt. Verpixeln mit dem reinen Blockmittel ist angreifbar: Werkzeuge wie Depix rendern Kandidatentext, verpixeln
ihn mit demselben Raster und vergleichen die Blockfarben.

## Entscheidung

1. **Zwei Werkzeuge**, kein Objekt: **Schwärzen** (Kürzel B, Füllung schwarz, deckend) und **Verpixeln** (Kürzel P).
   Man zieht einen Rahmen; beim Loslassen werden die Pixel des Originals sofort überschrieben. Unter der Mindestgröße
   (`canvas.MinObjekt`) passiert nichts. Der Rahmen wird in der Ansicht gezogen und nach außen auf ganze Pixel des
   Originals gerundet, damit kein angeschnittenes Pixel stehen bleibt; liegt ein Stauch-Streifen darin, wird der
   verborgene Teil mitgeschwärzt.
2. **Verpixeln mit Rauschen.** Jeder Block bekommt sein Mittel plus ein zufälliges Rauschen je Farbkanal (±24, aus
   `crypto/rand`, auf 0–255 begrenzt); die Blockgröße ist 12 Pixel, bei kleinen Bereichen mindestens `MinBlock`.
   Damit stimmen die Blockfarben nicht mehr mit einem nachgerechneten Kandidaten überein. Ist der Bereich schmaler als
   `MinBlock`, wird geschwärzt statt verpixelt, mit Hinweis in der Statusleiste.
3. **Rückgängig bis zum Speichern.** Vor dem Speichern lässt sich eine Schwärzung zurücknehmen (die alten Pixel liegen
   nur im Speicher). Nach einem gelungenen Speichern, dessen Verlauf eine Schwärzung enthält, leert der Editor den
   Verlauf (`VergissVerlauf`): Ein versehentliches Rückgängig mit anschließendem Speichern kann den ungeschwärzten
   Stand nicht zurück auf die Platte bringen.
4. **F-126:** Scheitert das Zurückschreiben in die bisherige Datei und weicht Schnapp auf eine neue Aufnahme aus,
   obwohl seit dem Öffnen geschwärzt wurde, entfernt Schnapp die bisherige Datei samt Vorschau und Index-Eintrag
   (endgültig, nicht in den Papierkorb — dort läge der ungeschwärzte Stand weiter). Die neue Aufnahme enthält alles
   außer den geschwärzten Pixeln. Scheitert auch das Entfernen, nennt die Warnung im Fenster die alte Datei als
   „enthält noch den ungeschwärzten Stand“. Hat ein anderes Programm die Datei inzwischen geändert (Konflikt), wird
   sie nicht entfernt, sondern ebenso gewarnt.
5. **Prüfung über alle Ausgänge:** Ein Test legt ein Muster ins Bild, schwärzt bzw. verpixelt es und sucht es in der
   Arbeitsdatei (sichtbares Bild und `snAO`), im SVG-Export, im flachen PNG und JPG, im Vorschaubild der Bibliothek und
   im PNG der Zwischenablage.

## Konsequenzen
- Die Werkzeugleiste wird um zwei Knöpfe breiter; F-148 (Leiste in schmalen Fenstern) wird dringlicher.
- Verpixelte Bereiche sehen leicht unruhig aus; das ist gewollt.
- Schnapp ersetzt Dateien per Umbenennen; alte Datenblöcke auf dem Datenträger überschreibt es nicht. Wer das braucht,
  verschlüsselt den Datenträger.
- Eine Arbeitsdatei, die vor dem Schwärzen schon weitergegeben oder anderswo kopiert wurde, bleibt ungeschwärzt;
  Schnapp kann nur die eigenen Dateien schützen.
