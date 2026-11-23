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


## Nachtrag 3 (22.11.2026): Nacharbeit aus dem Sprint-Abschluss 11 (F-409, F-412)

**Status:** Akzeptiert (Claude Opus 5.5, Andi informiert).

- **Zug (F-409, ersetzt den Satz „trägt das Objekt im Zug mit“ aus Nachtrag 2):** Ein laufender Zug hält die fällige
  Kopie weiter nicht auf, die Kopie trägt aber nur den Stand `Stand()`: Ein neues Objekt im Zug fehlt, ein gezogenes
  steht mit seiner Lage vor dem Zug darin (`document.ObjekteOhneZug`, auch im SVG-Schreiber). So kommt ein mit Esc
  verworfenes Objekt nach einem Absturz nie zurück, kein verworfener Text liegt auf der Platte, und leere oder entartete
  Zug-Objekte werden nie geschrieben. Ein übernommener Zug ändert den Stand und wird nach der Bündelung gesichert. Damit
  gilt wieder „Folgen“: *Nicht abgedeckt ist Text, der gerade getippt und noch nicht übernommen ist.* Verworfen: die
  Kopie im Zug zu schreiben und nach Zugende neu zu sichern — der verworfene Text läge dann bis zur nächsten Kopie auf
  der Platte, und die Schleife müsste das Zugende eigens erkennen.
- **Flächenbudget (F-412, ergänzt Punkt 7):** Vor dem Anwenden werden alle Schwärzungen ohne Pixel geprüft
  (`document.PruefeSchwaerzung`) und ihre Flächen im Bild summiert. Belegen sie mehr Pixelkopien, als der
  Rückgängig-Verlauf noch aufnimmt (512 MiB, 4 Bytes je Pixel, `document.VerlaufPlatz`), wird die Kopie nicht
  angewendet (`ErrZuGross`) und wie jede nicht anwendbare beiseitegelegt. Begründung: Unter dieser Grenze verwirft der
  Verlauf nichts; Rückgängig führt also auf den gespeicherten Stand (Punkt 6), die Sitzung schreibt danach weiter
  Kopien, und die Arbeit beim Öffnen bleibt bei rund 128 Mi Pixeln (unter einer Sekunde). Eine präparierte Kopie mit
  Tausenden vollflächiger Schwärzungen wird sofort abgelehnt. Die Höchstzahl von 10000 Schwärzungen je Kopie bleibt als
  Grenze beim Lesen. Abgelehnt heißt: nichts wird angewendet (Kernprinzip 2 bleibt gewahrt).
- **Probelauf:** Der Abzug bekommt nur noch die Angaben und die Prüfung (`render.Pruefe` liest keine Pixel). Die
  Schwärzungen laufen einmal, am Dokument, nachdem sie vorab geprüft sind.
- **Bekannte Lücke (F-431):** Ein einzelnes vollflächiges Schwärzen eines Bilds über 128 Mi Pixel (etwa 11600²) als
  letzter Schritt vor einem Absturz ergibt eine Kopie über dem Budget; sie wird beim Öffnen abgelehnt und gemeldet.

## Nachtrag 4 (22.11.2026): Rettungskopie-Reste aus Sprint 11 (F-410, F-413, F-417, F-431, F-432)

**Status:** Akzeptiert (Claude Opus 5.5, Andi informiert). Karte #179, Umsetzung 0513ced.

- **Pfad ohne gültiges UTF-8 (F-410):** Die Kennung trägt einen solchen Pfad zusätzlich roh (Base64) im Feld
  `pfad_roh`; `pfad` zeigt eine lesbare Fassung mit U+FFFD. Ohne `pfad_roh` gilt `pfad` — ältere Kopien lesen sich
  unverändert, das Format bleibt 1. Eine ältere Fassung legt eine Kopie mit `pfad_roh` als unlesbar beiseite; solche
  Kopien waren dort ohnehin nie anwendbar.
- **Zwei Fenster auf derselben Datei (F-413, ersetzt den letzten Satz von Nachtrag 2):** Jede Sitzung hält eine
  flock-Sperre auf `<SHA-256 des Pfads>.lock` im Ordner der Rettungskopien (`internal/rescue/sperre.go`). Ein zweites
  Fenster, das sie nicht bekommt, wendet keine Kopie an, legt keine beiseite, schreibt und löscht keine und sagt das in
  der Statusleiste. Nach Ende oder Absturz des ersten gibt der Kern die Sperre frei. Das Ende einer Sitzung entfernt
  die Sperrdatei; das Aufräumen entfernt eine alte nur, wenn niemand sie hält. Auf einem Dateisystem ohne flock läuft
  die Sitzung ohne Sperre wie bisher. Die Sperre steht nicht in der Kopie, ihr Format bleibt. Offen: F-436.
- **Keine Wartezeit auf der Ereignisschleife (F-417, ergänzt Punkt 4):** Schreiben, Löschen und das Setzen einer
  neuen Ausgangsdatei laufen als Kette von Aufträgen in Goroutinen, in der Reihenfolge der Aufrufe; kein Aufruf wartet
  auf die Platte, ein Löschen trifft nie eine später gewollte Kopie. Beim Schließen wartet die Sitzung höchstens 5 s;
  danach endet sie trotzdem (Diagnose), die zuletzt gewollte Kopie ist dann womöglich nicht geschrieben. Offen: F-437.
- **Flächenbudget beim Schreiben (F-431, ersetzt „Bekannte Lücke“ aus Nachtrag 3):** Die Sitzung schreibt keine Kopie,
  deren Schwärzungen mehr belegen, als ein frisch geöffnetes Dokument aufnimmt (512 MiB, dieselbe Rechnung wie beim
  Anwenden, `pruefeBudget`); die zuletzt geschriebene bleibt stehen, wie bei einem unvollständigen Verlauf
  (Nachtrag 1).
- **Dauer (F-432):** In Nachtrag 3 gilt statt „unter einer Sekunde“ „rund eine Sekunde“ (Mosaik knapp unter der
  Grenze gemessen 0,92 s).

## Nachtrag 5 (22.11.2026): Sperre strenger, fremdes Fenster übernimmt, NeueBasis in der Kette (F-436, F-440, F-446)

**Status:** Akzeptiert (Claude, Karte #198, Befunde der Sitzungs-QA Sprint 12, Fable).

- **Sperre (F-440, schärft Nachtrag 4):** Nur ein Dateisystem ohne flock (`ENOLCK`, `ENOTSUP`, `EOPNOTSUPP`) lässt die
  Sitzung ohne Sperre weiterlaufen; jeder andere Fehler beim Öffnen oder Sperren der Sperrdatei macht sie stumm (kein
  Anwenden, Beiseitelegen, Schreiben oder Löschen; eigene Meldung, `Wiederherstellung.Gestoert`). Die Sperrdatei muss
  regulär sein (`O_NOFOLLOW`, `O_NONBLOCK`), Ordner 0700, Datei 0600; Entfernen nur nach `os.SameFile`. Die Sperre
  hängt am angegebenen Pfad, nicht am aufgelösten: Kopie und Kennung sind pfadgebunden, sonst wären Sperre und Kopie
  nicht mehr dieselbe Datei.
- **Fremdes Fenster (F-436):** Es holt die Sperre beim nächsten Sichern nach (ein flock-Aufruf) und legt eine vom
  anderen Fenster hinterlassene Kopie vorher beiseite, statt sie zu überschreiben.
- **NeueBasis (F-446):** Die Entscheidung fällt in der Kette; ein `Sichere` direkt danach fällt nicht weg. Je Epoche
  gibt es höchstens eine wartende Kopie und einen Schreibauftrag (jüngste gewinnt). Wird die Sitzung dabei fremd,
  meldet das die Diagnose; die Statusleiste seit Nachtrag 6 (F-454).
- **Nicht umgesetzt (F-437, akzeptiert):** Nach SIGKILL zwischen Ersetzen der Datei und Löschen der Kopie bleibt die
  eigene alte Kopie liegen und wird beim nächsten Öffnen beiseitegelegt und gemeldet. Sie ist ohne Vermerk im
  Kopie-Format nicht von einer echten Kopie bei geänderter Datei zu unterscheiden; kein Schaden, nur eine unnötige
  Meldung.
- **Nachgezogen (Karte #200, F-456, F-457, F-460):** Aufträge der Kette lesen nur den Stand der Kette
  (`kettenBasis`, gesetzt mit der Sperre) oder einen Pfad, den ihr Aufruf mitgibt, nie `basis`. Die Übernahme der
  Sperre läuft nur für einen Auftrag der laufenden Epoche — sonst könnte ein verfallener Sichere-Auftrag die Sperre
  einer neuen Datei nehmen und der folgende `NeueBasis`-Auftrag die Kopie des anderen Fensters löschen (Rückschritt aus
  der ersten Fassung dieses Nachtrags, F-456). `gestoert` fällt wie `aus` erst nach einem ausstehenden Wechsel ins
  Gewicht (F-457).


## Nachtrag 6 (22.11.2026): Lage der Sperre in der Statusleiste, Ort der beiseitegelegten Kopie auf stderr (F-454, F-411)

**Status:** Akzeptiert (Karte #180, umgesetzt von Claude Sonnet 5.5, übernommen von Claude Opus 5.5).

- `rescue.Sicherung.Lage()` liefert `Fremd`, `Gestoert` und einen Zähler `Nr`, der mit jeder Änderung einer der beiden
  wächst. `ui.Rettung` hat dazu `Lage() (meldung string, fehler bool, nr uint64)`: gestört (Fehler), fremd (der
  Starttext) oder nach einer Übernahme „wieder geschützt“. Der Editor fragt nach jedem Ereignis und zeigt eine
  geänderte Lage über `meldeNebenbei`; eine haltende Meldung bleibt stehen (dann steht eine Fehlermeldung auf
  stderr). Weil die Kette die Sperre nebenläufig wechselt, weckt der Editor 300 ms nach Sichern und Speichern einmal.
- Eine Wiederherstellungs-Meldung mit Fehler (Kopie beiseitegelegt, `….alt.json`) steht zusätzlich auf stderr, samt
  Ort: In der einzeiligen Statusleiste war sie praktisch nie lesbar (F-411).

## Nachtrag 7 (22.11.2026): Lage mit Grund, Nachholen und Wecken im Takt, Ort vorn (F-472, F-474, F-475)

**Status:** Akzeptiert (Karte #204, umgesetzt von Claude Sonnet 5.5, Befunde von Claude Fable 5.2).

- `rescue.Lage` trägt den **Grund** der letzten Änderung (`LageStart`, `LageUebernahme`, `LageBasiswechsel`, `LageAus`)
  und `Aus`; der Zähler `Nr` wächst auch, wenn die Übernahme der Sperre an einer nicht beiseite zu legenden Kopie
  scheitert (die Sitzung ist dann stumm) oder wenn ein Basiswechsel dieses Stummsein aufhebt. Die Oberfläche sagt
  „wieder geschützt“ nur nach einer echten Übernahme der Sperre eines anderen Fensters (Text „hat die Datei
  losgelassen“); nach einem Speichern als andere Datei steht „Als andere Datei gespeichert … wieder geschützt“, bei `Aus`
  ein Fehler „nicht geschützt“.
- Eine Lage-Meldung ohne Fehler, die hinter einer haltenden Meldung nicht erscheinen kann, verbraucht den Zähler nicht:
  Sie kommt, sobald die haltende Meldung weicht. Der Editor weckt nach Sichern und Speichern im 300-ms-Takt bis 3 s
  (statt einmal), weil die Kette länger brauchen kann.
- Die Meldung zu einer beiseitegelegten oder unlesbaren Kopie beginnt mit dem Ort (`schnapp/rescue/<Name>`): Die
  einzeilige Statusleiste kürzt das Ende, und beim Start über „Öffnen mit“ sieht niemand die Fehlerausgabe.

## Nachtrag 8 (23.11.2026): Aussage vor Ort, Grund bei gleicher Datei, Fehler-Lage nachholen (F-486, F-487, F-488)

**Status:** Akzeptiert (Karte #206, umgesetzt von Claude Sonnet 5.5, Befunde aus dem Verify-Lauf von Sprint 12).

- **Ändert Nachtrag 7, letzter Punkt:** Der Ort steht nicht mehr vorn. Der Dateiname der Kopie ist ein SHA-256 (64 Hex-Zeichen
  plus Endung, über 100 Zeichen mit Pfad) und verdrängte so die Aussage aus der einzeiligen Statusleiste. Jetzt steht die
  kurze Aussage zuerst, dann `schnapp/rescue/<8 Hex-Zeichen>…`; der Kurzname ist im Zustandsordner eindeutig zu finden. Die
  volle Meldung steht bei Fehlern weiter auf der Fehlerausgabe.
- `NeueBasis` mit derselben Ausgangsdatei wie bisher (ein fremdes Fenster ließ los, dieses speichert unter demselben
  Namen) setzt den Grund `LageUebernahme`, nicht `LageBasiswechsel`: Der Text „hat die Datei losgelassen“ stimmt, „als
  andere Datei gespeichert“ nicht (F-486).
- Auch eine Lage mit Fehler hinter einer haltenden Meldung bleibt unverbraucht und wird nachgeholt; sie steht dabei je
  Änderung einmal auf der Fehlerausgabe (F-487).
