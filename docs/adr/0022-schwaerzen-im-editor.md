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
2. **Verpixeln mit Rauschen.** *(Ersetzt durch Nachtrag 1: Mosaik ohne Inhalt.)* Jeder Block bekommt sein Mittel plus ein zufälliges Rauschen je Farbkanal (±24, aus
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

## Nachtrag 1 (20.11.2026): Verpixeln ohne Inhalt, Nacharbeit aus der Kritik (F-163 bis F-170)

**Status:** Akzeptiert (Claude Opus 5.5, nach dem Kritiker-Lauf Fable zu Karte #37).

- **Punkt 2 ersetzt.** Blockmittel plus Rauschen ±24 hielt nicht: Mit Toleranzvergleich war eine vierstellige
  Ziffernfolge ab etwa 20×28 Pixeln Zifferngröße in 193 von 200 Läufen eindeutig rekonstruierbar (F-163). Verpixeln
  ist jetzt ein **Mosaik ohne Bezug zum Inhalt**: Jeder Block bekommt das Mittel des ganzen Bereichs, einmal und je
  Block um einen Zufallswert in [−24, 24] verschoben (für R, G und B derselbe Wert, gleichverteilt aus `crypto/rand`).
  Zwei Bereiche mit gleichem Mittel ergeben bei gleichem Zufall dasselbe Mosaik — es verrät nur noch die ungefähre
  mittlere Farbe. Die Blockgröße (12, mindestens 8) ist nur noch Optik.
- **Punkt 4 genauer.** Entfernt wird die alte Datei nur, wenn seit ihrem letzten gelungenen Schreiben geschwärzt wurde
  (`Dokument.SchwaerzStand`, F-169). `library.Entfernen` sperrt wie `Speichern` (F-165), meldet, ob es die Datei selbst
  entfernt hat, und unterscheidet „Datei bleibt“ von „nur die Vorschau bleibt“ (F-164); eine schon fehlende Datei
  heißt in der Meldung „nicht mehr da“, nicht „entfernt“ (F-167). Trägt die neue Aufnahme den Namen der alten (die alte
  war schon weg), wird nichts entfernt. Bleibt beim Zurückschreiben die alte Vorschau stehen, sagt die Warnung nach
  einer Schwärzung, dass sie den Inhalt noch zeigt (F-166).
- **Zuschnitt:** Der Rahmen bleibt auf die Ansicht begrenzt; was jenseits der Zuschnittkante liegt, steht weiter im
  Original (Zuschneiden ist nicht zerstörend, ADR-0008). Wie das mit dem Schwärzen zusammengeht, entscheidet Karte #38.
  *(Entschieden in ADR-0023 Punkt 8: Der Rahmen bleibt auf die Ansicht begrenzt.)*
- Ein vor dem Schwärzen mit Strg+C kopiertes Bild bleibt, wie es war, in der Zwischenablage (README).

## Nachtrag 2 (20.11.2026): Liegen gebliebene Fassungen bei späterer Schwärzung (F-221)

**Status:** Akzeptiert (Claude Opus 5.5, Nacharbeit zum Kritiker-Lauf Sprint-Abschluss 6, Karte #98).

- **Punkt 4 gilt über die ganze Sitzung, nicht nur für ein Speichern.** Bisher griff Entfernen oder Warnen nur, wenn
  Ausweichen und Schwärzen im selben Speichern zusammenfielen. Wer erst ausweicht (Zurückschreiben scheitert, ohne
  Schwärzung) und später schwärzt, ließ die alte Aufnahme ungeschwärzt liegen; ebenso blieb die Datei von außen ohne
  Warnung, wenn erst nach dem ersten Speichern geschwärzt wurde.
- **Umsetzung:** Die Speicherfunktion (`cmd/schnapp/editor.go`, `ablegenIn`) merkt sich jede Fassung, die neben der
  aktuellen Datei liegen bleibt, mit dem Schwärzungsstand, den sie zeigt (`SchwaerzStand`, F-169/F-187): die Datei von
  außen, die andere Fassung eines Konflikts und eine Aufnahme, deren Zurückschreiben scheiterte. Nach jedem gelungenen
  Speichern prüft `rueckblick`, ob seitdem geschwärzt wurde:
  - eigene Aufnahme der Bibliothek: entfernen wie F-126 (endgültig, mit Vorschau und Index, nur bei unverändertem
    Stand), Meldungen wie dort („entfernt“, „nicht mehr da“, „Vorschau bleibt“); scheitert das Entfernen, Warnung
    „Bitte selbst löschen“, und die Fassung bleibt vorgemerkt — eine weitere Schwärzung versucht es erneut;
  - Datei von außen: nie anfassen (ADR-0020), Warnung mit ihrem Namen ohne Ordner (F-155);
  - andere Fassung eines Konflikts: nie entfernen (gehört nicht Schnapp allein), Warnung.
  Gewarnt wird einmal je neuer Schwärzung, nicht bei jedem Speichern. Eine Schwärzung, die vor dem Speichern
  zurückgenommen wurde, löst nichts aus. Die gerade geschriebene Aufnahme wird nie entfernt.
- **Grenze:** Vorgemerkt wird nur innerhalb einer Editor-Sitzung. Wer den Editor schließt und dieselbe Aufnahme später
  wieder öffnet, hat nur noch diese Datei; was in einer früheren Sitzung liegen blieb, nannte schon deren Warnung.

## Nachtrag 3 (21.11.2026): Grenze für den Rückgängig-Verlauf (F-64, Karte #46)

**Status:** Akzeptiert (Claude, Andi informiert).

- **Problem:** Jede Schwärzung hält die alten Pixel ihres Bereichs (`vorher`) im Verlauf, ohne Obergrenze. Wer sehr
  oft oder über ganze Bilder schwärzt, füllt den Speicher.
- **Entscheidung:** Die Gesamtgröße der im Verlauf gehaltenen Pixelkopien ist auf `verlaufGrenze` begrenzt,
  512 MiB (`internal/document/verlauf.go`). Ein ganzes 4K-Bild hat 3840 × 2160 × 4 Byte = rund 31,6 MiB, 512 MiB
  reichen also für 16 Schwärzungen über das ganze Bild und für gut 3000 übliche Bereiche (400 × 100 Pixel = rund
  156 KiB). *Zahlen berichtigt 21.11.2026 abends (F-327): vorher „rund 33 MB“, „rund 15“, „160 KB“.*
  Nach jeder neuen Änderung fallen die **ältesten** erledigten Schritte weg, bis die Summe nicht mehr über der Grenze
  liegt; der
  **jüngste Schritt bleibt immer**, auch wenn er allein größer ist. Verworfene Schritte lassen sich nicht mehr
  zurücknehmen; `Stand()` bleibt stimmig (die Kennung des zuletzt verworfenen Schritts wird zur Basis).
- **Unverändert:** Die Zug-Sperre (F-18) bleibt; der Verlauf lebt nur im Speicher und wird nie gespeichert. Auch
  `SchreibeArbeitsdatei` leert den Verlauf nicht: Rückgängig nach dem Speichern ist gewollt; die Entscheidung 3 oben
  (Verlauf mit Schwärzung leeren) regelt der Editor nach einem gelungenen Speichern. Die Grenze kappt nur, sie hält
  nichts länger, und verworfene Kopien werden freigegeben.
- **Still, mit Absicht (Claudes Entscheidung, Andi informiert, 21.11.2026 abends, F-327):** Das Verwerfen meldet sich
  nicht. Es trifft nur Schritte, die mindestens 16 vollflächige Schwärzungen zurückliegen; eine Meldung in der
  Statusleiste käme bei jeder weiteren Änderung wieder und verdrängte die Hinweise zum Werkzeug. Verworfen werden auch
  Objekt- und Zuschnittschritte, die vor dem verworfenen Pixelschritt liegen, denn der Verlauf ist eine Folge; ein
  Test hält fest, dass Rückgängig danach nur echte frühere Stände erreicht (`TestVerlaufGrenzeGemischteFolge`, F-323).
