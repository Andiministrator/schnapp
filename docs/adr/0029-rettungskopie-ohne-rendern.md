# ADR-0029: Rettungskopie ohne Rendern (F-144)

## Status
Akzeptiert (Claude im Auftrag Andis, 21.11.2026: „#77 entscheidest du für mich“). Umgesetzt in Sprint 11, Karte #77 (Nachtrag 1).

## Kontext
Der Editor speichert beim Schließen, bei Strg+S und bei SIGTERM, SIGINT und SIGHUP (ADR-0020 Nachträge 2, 6 bis 9).
Ein Absturz, SIGKILL, ein leerer Akku oder ein Stromausfall verlieren dagegen alles seit dem letzten Speichern
(F-144). Ein Autospeichern der Arbeitsdatei wurde bewusst nicht gebaut (ADR-0020 Nachtrag 2): Speichern rendert das
Ansichtsbild und kodiert zwei Vollbild-PNGs, bei großen Dokumenten dauert das Sekunden bis Minuten (F-130, F-297)
und sperrt das Bearbeiten (Nachtrag 6).

Was eine Bearbeitung ausmacht, ist dagegen klein und ohne Pixel beschreibbar:

- Objekte (Pfeile, Kästen, Text, Nummern) — dieselben Angaben, die der Block `snAP` (SVG) trägt;
- Zuschnitt und Stauchstreifen — Angaben, kein Schnitt in Pixeln (ADR-0008, ADR-0023);
- Schwärzungen und Verpixelungen — Rechteck und Art (`document.Schwaerzung`); sie überschreiben Pixel, lassen sich
  aber auf demselben Ausgangsbild in derselben Reihenfolge wiederholen.

## Entscheidung

1. **Inhalt:** Eine Rettungskopie ist eine kleine JSON-Datei mit der Kennung der Ausgangsdatei, den Objekten, dem
   Zuschnitt, den Stauchstreifen und der Liste der Schwärzungen **seit dem letzten Speichern**, in ihrer Reihenfolge.
   Sie enthält keine Pixel, kein Ansichtsbild und kein Original.
2. **Kennung der Ausgangsdatei:** absoluter Pfad, Größe, Änderungszeit und SHA-256 des Dateiinhalts, wie er beim
   Öffnen bzw. beim letzten Speichern vorlag. Nur wenn alle vier passen, wird die Kopie angewendet.
3. **Ort:** `$XDG_STATE_HOME/schnapp/rescue/<SHA-256 des Pfads>.json` (ohne XDG_STATE_HOME `~/.local/state/…`), Rechte
   0600, Ordner 0700 — nicht im Bilderordner, damit die Bibliothek sie weder indiziert noch synchronisiert.
4. **Schreiben:** nach jeder Änderung, gebündelt (frühestens 2 s nach der letzten Änderung, höchstens alle 10 s), in
   einer Goroutine aus einem Abzug der Angaben (keine Pixel, daher Millisekunden), atomar per temporärer Datei und
   Umbenennen. Während eines Speicherns ruht sie.
5. **Löschen:** nach jedem erfolgreichen Speichern und beim Schließen ohne ungesicherte Änderung. Scheitert das
   Speichern, bleibt sie.
6. **Wiederherstellen:** Öffnet der Editor eine Datei, zu der eine passende Rettungskopie liegt, wendet er sie an
   (Schwärzungen wiederholen, dann Angaben setzen), markiert das Dokument als geändert und sagt in der Statusleiste
   „Restored unsaved changes from <Zeit>“. Rückgängig führt auf den gespeicherten Stand zurück. Passt die Kennung
   nicht (Datei inzwischen anders gespeichert), wird nichts angewendet; die Statusleiste nennt den Ort der Kopie, sie
   bleibt 30 Tage liegen und wird dann beim Aufräumen der Bibliothek entfernt.
7. **Grenzen:** dieselben wie beim Laden (`document.PruefeInhalt`, `render.Pruefe`); eine Kopie, die sie überschreitet,
   wird nicht angewendet.

## Folgen

- **Schwärzen bleibt endgültig (Prinzip 2):** Die Rettungskopie nennt nur Rechtecke, nie Pixel. Bis zum Speichern
  steht der ungeschwärzte Inhalt wie heute im Original der Datei; nach einem Absturz schwärzt das Wiederherstellen ihn
  wieder. Wer nach einem Absturz die Datei mit einem anderen Programm öffnet, sieht ihn — wie heute.
- **Verpixeln mit Mosaik** (ADR-0022 Nachtrag 1) zieht beim Wiederholen neue Zufallswerte: gleiche Fläche, anderes
  Muster. Unkenntlich bleibt es; bitgleich ist es nicht.
- **Kosten:** Ein Abzug der Angaben und ein JSON von wenigen KiB je Änderung. Kein Rendern, keine Sperre.
- **Nicht abgedeckt:** Text, der gerade getippt und noch nicht übernommen ist; Änderungen der letzten 2 bis 10 s;
  Pixelbearbeitungen, die es später geben könnte (dann braucht die Kopie ein eigenes Format).
- **Datenschutz:** Die Kopie enthält den Pfad und die Texte der Beschriftung — so viel wie die Arbeitsdatei selbst,
  im privaten Zustandsordner.

## Verworfen

- **Arbeitsdatei periodisch speichern:** zu teuer bei großen Dokumenten (siehe Kontext), sperrt das Bearbeiten.
- **Rückgängig-Verlauf auf Platte spiegeln:** enthält Pixelstände (Schwärzungen) und wäre ein Leck für Prinzip 2.
- **Rückfrage beim Öffnen** („Wiederherstellen?“): Schnapp hat keine Dialoge (ADR-0020 Nachtrag 2); Anwenden mit
  Rückgängig als Ausweg ist gleich sicher und schneller.

## Nachtrag 1 (22.11.2026): Umsetzung

**Status:** Akzeptiert (Claude Opus 5.5, Andi informiert). Karte #77, F-144.

- **Pakete:** `internal/rescue` (Kennung, Ort, JSON-Format 1, atomares Schreiben, Lesen mit Grenzen, Anwenden,
  Aufräumen, `Sicherung` für eine Editor-Sitzung); `document.SetzeAngaben` (Objekte, Zuschnitt und Streifen in einem
  Rückgängig-Schritt) und `document.SchwaerzungenImVerlauf`; `fileformat.AngabenSVG`/`LeseAngaben` (dasselbe SVG wie
  der Block snAP, ohne Metadaten); `ui.Rettung` (Bündelung in der Ereignisschleife, `internal/ui/rettung.go`);
  Anschluss in `cmd/schnapp/rettung.go`.
- **Schwärzungen seit dem letzten Speichern** sind genau die Schwärzungen im Rückgängig-Verlauf: Ein Speichern mit
  Schwärzung leert ihn (ADR-0022 Punkt 3). Verwirft die Grenze des Verlaufs (512 MiB, F-64) eine Schwärzung, wäre die
  Liste unvollständig; dann wird keine neue Kopie geschrieben, die zuletzt geschriebene bleibt.
- **Neue Ausgangsdatei nach dem Speichern:** Weicht das Speichern auf eine neue Aufnahme der Bibliothek aus (Datei von
  außen, Konflikt), bezieht sich die Rettungskopie danach auf diese Datei; die Kopie zur alten wird gelöscht.
- **Wiederherstellen** prüft erst an einem Abzug (Grenzen wie beim Laden, `render.Pruefe`, außer mit
  `--ignore-limit`); scheitert etwas, bleibt das Dokument unverändert und die Kopie liegen.
- **Löschen beim Schließen** nur, wenn das Dokument dem Gespeicherten gleicht. Hat der Nutzer nach einem gescheiterten
  Speichern das Verwerfen bestätigt, bleibt die Kopie (Punkt 5: „Scheitert das Speichern, bleibt sie“) — ein Netz etwa
  bei voller Platte.
- **Aufräumen** nach 30 Tagen geschieht beim Start des Editors (nicht beim Aufräumen der Bibliothek, das nicht bei
  jedem Start läuft); nur eigene Dateinamen im eigenen Ordner.
- **Meldungen** (englisch, deutsch in `de.po`): wiederhergestellt; nicht angewendet, weil die Datei geändert wurde;
  nicht anwendbar; nicht lesbar. Die Zeit erscheint als `JJJJ-MM-TT hh:mm` (Datumsformate je Sprache: Karte #28).

## Nachtrag 2 (22.11.2026): Nacharbeit aus Verify 7 (Fable)

**Status:** Akzeptiert (Claude Opus 5.5, Andi informiert). F-384.

- **Nicht angewendete Kopie wird beiseitegelegt** (Punkt 6): Passt die Kennung nicht, ist die Kopie unlesbar, einer
  neueren Fassung oder über den Grenzen, benennt das Öffnen sie um in `<SHA-256 des Pfads>.<Zeit>.alt.json`. Die
  Sitzung schreibt und löscht nur unter dem gewöhnlichen Namen; so bleibt die alte Kopie bis zum Aufräumen nach 30 Tagen,
  wie die Meldung sagt. Scheitert das Umbenennen, schreibt und löscht diese Sitzung keine Kopie.
- **Die Meldung nennt den Ort** relativ zum Zustandsordner (`schnapp/rescue/<Name>`), ohne Benutzerpfad (F-155).
- **Bündelung** (Punkt 4): Ein laufender Zug (Ziehen, Texteingabe) hält die fällige Kopie nicht auf; sie trägt das
  Objekt im Zug mit. Beim Beginn eines Speicherns wird eine noch nicht gesicherte Änderung sofort geschrieben, ohne
  Bündelung — hängt das Speichern und folgt ein SIGKILL, wäre sie sonst verloren.
- **Liste der Schwärzungen:** Nach jedem gelungenen Speichern gilt sie wieder als vollständig
  (`document.SchwaerzungenGesichert`, von `canvas` gerufen), auch wenn der Verlauf dabei nicht geleert wurde.
- **Blockgröße:** `document.Schwaerzen` klemmt eine Blockgröße über dem Bereich auf den Bereich (eine präparierte Kopie
  ließ sonst die Blockteilung überlaufen).
- **Offen:** Testlücken laut Mutationsprobe des Prüfers (F-385, Karte #77): Bündelung in der Schleife, `--ignore-limit`,
  neue Ausgangsdatei nach dem Speichern, Aufräumen beim Start, Grenzen beim Lesen. Zwei Instanzen auf derselben Datei
  sind nicht gegeneinander gesperrt (bewusst: Schnapp sperrt Dateien sonst auch nicht; F-110 schützt das Speichern).

