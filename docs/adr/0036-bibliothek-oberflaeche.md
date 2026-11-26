# ADR-0036: Bibliothek in der Oberfläche — Leiste der letzten Aufnahmen und volle Ansicht

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #174 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Gestaltung nach Muster `tmp/sprint12/muster/bilder/174-bibliothek-{a-leiste,b-ansicht}.png`; Andi nimmt sie mit der
fertigen Umsetzung ab (gebündelt am Sprint-Ende). Grundlage: Wunsch Andis vom 22.11.2026 („die bibliothek sollte per
button aufgehen, aber auch (per schalter de-/aktivierbar) unten als leiste zu sehen sein (so wie bei snagit)“), dazu
aus Karte #173: Agenten-Aufnahmen per Schalter einblendbar. Setzt A-02 in der Oberfläche um; ergänzt ADR-0016
(Bibliothek), ADR-0018/ADR-0034 (Einstellungen), ADR-0026 (Menü und Kürzel), ADR-0031 (Oberfläche).

## Kontext
Die Bibliothek (ADR-0016) legt jede Aufnahme ab, führt einen Index und Vorschaubilder, kann suchen und Schlagworte
setzen — aber nur im Code. Wer eine frühere Aufnahme wieder öffnen will, muss den Dateimanager nehmen. Snagit zeigt
die letzten Aufnahmen als Leiste („Tray“) unter dem Bild und eine volle Bibliothek.

## Entscheidung

1. **Leiste der letzten Aufnahmen** unten im Editor zwischen Leinwand und Fußzeile, so breit wie die Leinwand
   (Werkzeugleiste und rechte Leiste bleiben voll hoch), Höhe 104 dp (*seit Zusatz Karte #311:* 118 dp). Links der Knopf „Library“ (öffnet Punkt 3),
   daneben die Vorschaubilder der Bibliothek, **neueste links**, je 120 × 68 dp mit Zeitangabe darunter (heute: Uhrzeit,
   gestern: „yesterday“, sonst Wochentag und Datum, Sprache über `date format`). Die offene Aufnahme ist grün umrandet.
   Mausrad rollt waagerecht, Tastatur über die Fokusfahrt. Agenten-Aufnahmen stehen nicht in der Leiste.
2. **Schalter:** Ansicht → „Recent captures“/„Letzte Aufnahmen“, **Strg+Umschalt+R**, gemerkt in
   `[library] bar = yes|no` (Voreinstellung **yes**; Nachtrag zu ADR-0018 und Zeile in der Einstellungstafel nach
   ADR-0034 Punkt 2, Reiter Library). In Fenstern unter 420 dp Leinwandhöhe blendet sich die Leiste von selbst aus
   (wie die rechte Leiste unter 752 dp Breite), ohne die Einstellung zu ändern.
3. **Volle Ansicht** im Editor-Fenster: Menü Datei → „Library“, **Strg+Umschalt+L**, oder der Knopf in der Leiste.
   Sie ersetzt Werkzeugleiste, Leinwand und rechte Leiste; Menüzeile und Fußzeile bleiben. Kopf: Suchfeld
   (Schlagwort, `library.Anfrage.Schlagwort`), Zeitraum (Today, 7 days, 30 days, This year, All), Schalter
   „Show agent captures“ (aus bei jedem Öffnen, nicht gemerkt; `Anfrage.OhneAgenten`), Knopf „Back to image“. Darunter
   ein Raster nach Monaten (Überschrift mit Zahl der Aufnahmen), neueste zuerst; je Kachel Vorschau, Datum und Uhrzeit,
   Größe und Schlagworte. Grün umrandet die offene Aufnahme, orange der Tastaturfokus.
4. **Bedienung der Ansicht:** Pfeiltasten wandern im Raster, Pos1/Ende springen, Bild↑/↓ blättern; Enter oder
   Doppelklick öffnet; **Entf** legt die Aufnahme in den Papierkorb (wie das Aufräumen, freedesktop, rückholbar; die
   offene Aufnahme ist ausgenommen, der Eintrag ist dann grau); **F2** bearbeitet die Schlagworte (Feld unter der
   Kachel, Komma trennt, Enter übernimmt über `SetzeSchlagworte`); Esc oder „Back to image“ zurück. Strg+F setzt den
   Fokus ins Suchfeld. Solange die Ansicht offen ist, wirken die Werkzeugtasten nicht.
5. **Öffnen einer anderen Aufnahme (Leiste oder Ansicht) im selben Fenster:** Zuerst wird die offene Aufnahme
   gespeichert wie beim Schließen (Speichern im Hintergrund mit den Fristen aus ADR-0020 Nachträgen 6 bis 9; scheitert
   es, bleibt sie offen und die Statuszeile meldet es — gewechselt wird dann nicht). Danach lädt der Editor die gewählte
   Datei wie `schnapp DATEI` (`ladeFuerEditor` mit Budget-Prüfung, Rettungskopie nach ADR-0029, eigene Sitzung,
   eigener Rückgängig-Verlauf; der Verlauf der vorigen ist danach weg). Fenster, Zoomstufe-Voreinstellung, Zustand
   der Leisten bleiben. Ist eine Datei beschädigt oder neuer, öffnet sie als einfaches Bild wie bisher.
6. **Vorschaubilder** nur aus dem Vorschau-Cache (`library.VorschauPfad`), nie aus dem Original; fehlt eine, wird sie
   im Hintergrund erzeugt (wie heute beim Ablegen) und bis dahin ein graues Feld gezeigt. Geladen werden nur die
   sichtbaren Kacheln (plus eine Zeile Vorlauf); höchstens 64 MiB dekodierte Vorschauen im Speicher, ältere fallen
   heraus. Index und Vorschauen werden im Hintergrund gelesen; die Oberfläche wartet nie auf die Platte.
7. **Aktualität:** Die Leiste liest den Index beim Öffnen des Editors, nach jedem eigenen Speichern und beim Öffnen der
   Ansicht neu (`library.Laden`/`Suchen`); kein Dateiwächter (inotify) — eine Aufnahme aus einem zweiten Fenster
   erscheint beim nächsten dieser Anlässe.
8. **Nicht in diesem Schritt:** Ordner oder Sammlungen, mehrere auswählen, Ziehen in andere Programme, Umbenennen,
   Vorschau einer fremden Datei außerhalb der Bibliothek, Suche im Text (OCR, ADR-0035).
9. **Umsetzung:** Logik ohne Gio in `internal/canvas` oder einem eigenen Paket (Raster, Fokus, Zeitraum, Zeitangabe,
   Speicher der Vorschauen), Zeichnen in `internal/ui/bibliothek.go` und `internal/ui/leiste_unten.go`, Wechsel des
   Dokuments in `cmd/schnapp/editor.go` und `internal/ui` (der Editor bekommt eine Anfrage „öffne Pfad“, die die
   Hauptschleife nach dem Speichern ausführt). Texte über `i18n`, Mutanten-Gruppe `bibliothek-ui`, Sichtproben per
   Drehbuch.

## Erwogene Alternativen
- **Neues Fenster je geöffneter Aufnahme (`schnapp DATEI` als eigener Prozess):** einfacher, aber viele Fenster und
  kein Bezug zur Leiste; Snagit wechselt im Fenster.
- **Bibliothek als eigenes Programmfenster:** wie oben, dazu eine zweite Oberfläche ohne Bild.
- **Dateiwächter (inotify) für die Leiste:** sofort aktuell, aber ein weiterer Hintergrundvorgang für einen seltenen
  Fall.

## Konsequenzen
- ADR-0018 Nachtrag 5: `[library] bar`; ADR-0034: Zeile in der Tafel; ADR-0026: zwei Menüeinträge mit Kürzeln.
- Die Leiste nimmt der Leinwand 104 dp Höhe (*seit Zusatz Karte #311:* 118 dp); Drehbücher mit festen Koordinaten setzen sie am Anfang aus (wie die
  rechte Leiste, ADR-0031 Nachtrag 3) — über den Menüeintrag, nicht über einen Umschalter, sobald die Einstellung
  gemerkt wird (Lehre aus F-539).

## Ergänzt bei der Umsetzung (Teil 1, 24.11.2026, Karte #174, Claude Opus 5.5)

Teil 1 setzt die Punkte 1, 2, 5, 6, 7 und die Leistenseite von 9 um; die volle Ansicht (Punkte 3 und 4) folgt in
Teil 2. Was die Entscheidung offen ließ oder anders kam:

1. **Kein Knopf „Library“ in Teil 1.** Ohne volle Ansicht hätte er kein Ziel; ein grauer Knopf mit Erklärung nähme
   der Leiste nur Platz. Die Vorschauen beginnen links am Rand; Teil 2 setzt den Knopf davor (Muster 174 a).
2. **Schwelle:** Die Leiste erscheint, wenn neben ihr mindestens 420 dp Leinwandhöhe bleiben (bei Skala 1 ein Fenster
   ab 584 dp Höhe), wie die rechte Leiste ihre 400 dp Mindest-Leinwand rechnet (`leistePasst`).
3. **Zeitangabe:** Uhrzeit, „yesterday“ und Datum kommen über den Kontext **`date part`** (Go-Muster „15:04“, „Jan 2“,
   „Jan 2, 2006“; deutsch „15:04“, „02.01.“, „02.01.2006“), die Wochentage über den Kontext `weekday` („Mon“ … „Sun“,
   deutsch „Mo“ … „So“). `date format` verlangt laut Prüfung (KA-7) ein Muster, das sich minutengenau zurücklesen
   lässt; Teilmuster ohne Jahr oder Datum fielen dort durch. In einem anderen Jahr steht das Jahr dabei.
4. **Länge der Leiste:** höchstens die 200 neuesten Aufnahmen (`libview.MaxEintraege`); ältere erreicht die volle
   Ansicht. Geladen werden die sichtbaren Vorschauen und je eine Vorschau Vorlauf links und rechts (die Leiste hat
   keine Zeilen), die jüngste Anforderung zuerst. Fehlt eine Vorschau, schreibt `library.ErzeugeVorschau` sie im
   Hintergrund aus dem sichtbaren Bild der Datei (wie der Neuaufbau, nie aus dem Original). Eine Datei im
   Vorschau-Cache über 4 MiB oder über 1024 px Seitenlänge, ein Verweis oder ein FIFO wird nicht dekodiert (grau).
5. **Speichern vor dem Wechsel** läuft wie Strg+S (Speichern im Hintergrund, `canvas.Sitzung.SpeichernJetzt`), nicht
   als Schließen-Anfrage: Scheitert es, bleibt die Aufnahme offen, und die Statuszeile sagt „Could not save: … The
   other capture was not opened.“ Ein Hinweis beim Speichern (etwa „als neue Aufnahme gespeichert“) hält den Wechsel
   einmal an, damit er gelesen wird, wie beim Schließen; der nächste Klick wechselt dann ohne Speichern. Ohne Änderung
   wird nicht gespeichert. Ein Signal oder eine Schließen-Anfrage während des Speicherns geht vor; der Wechsel entfällt.
6. **Laden im Hintergrund:** Die Ladefunktion aus `cmd/schnapp` (`oeffnerFuer`: `ladeFuerEditor` mit Budget,
   Kantenvorgabe `[capture] seam` für ein einfaches oder beschädigtes Bild, eigene Rettungskopie) läuft in einer
   Goroutine; die Statuszeile sagt „Opening…“. Wird die offene Aufnahme in der Zwischenzeit geändert, bleibt sie offen,
   und die geladene gibt ihre Rettungskopie frei. Ein Fehler beim Laden steht übersetzt in der Statuszeile.
   `--ignore-limit` gilt nur für die beim Start geöffnete Datei; nach einem Wechsel prüft auch der Export wieder das
   Budget. Während eines Exportdialogs oder einer Texterkennung wechselt ein Klick nicht (Hinweis in der Statuszeile);
   ein Klick auf die offene Aufnahme tut nichts.
7. **Was bleibt:** Fenster, Leisten, Einrasten und Navigator. Neu sind Sitzung, Verlauf, Werkzeug (Auswählen) und die
   eingepasste Ansicht; die eigenen Schnellstile werden aus `styles.ini` neu gelesen. Die alte Rettungskopie endet
   (gespeichert: verworfen, Sperre frei), die neue wird angewendet wie beim Öffnen.
8. **Grüner Rand:** Der Editor kennt die Datei, die gerade den Stand trägt (`ui.Geoeffnet.Pfad`, nach einem Speichern
   als neue Aufnahme die neue). Eine Datei von außen hat keinen grünen Rand, bis sie als Aufnahme gespeichert ist (ADR-0016 Nachtrag 11: Das gilt bis zur ersten Änderung; sie stößt das Speichern an, danach ist die neue Aufnahme grün umrandet. Nachtrag 10 hatte vorübergehend schon beim Öffnen übernommen.).
9. **Drehbücher:** Statt jedes Drehbuch die Leiste per Menü ausschalten zu lassen, startet sie im Drehbuch-Lauf aus und
   merkt ihren Schalter nicht (`SCHNAPP_PROBE_LEISTE_AUS`, nur der Probe-Build liest es, `scripts/probe.sh` setzt es).
   So bleiben alle Drehbücher mit festen Koordinaten gültig, und ein Umschalten kippt keinen späteren Lauf (Lehre aus
   F-539); die Sichtprobe `bibliothek-174-leiste.txt` schaltet sie mit Strg+Umschalt+R ein.
10. **Nummern:** Der Nachtrag zu ADR-0018 ist Nachtrag 6 (5 war schon belegt), zu ADR-0026 Nachtrag 9, zu ADR-0034
    Nachtrag 2 (die Tafel wirkt für diesen Schlüssel sofort). Die Logik ohne Gio steht im Paket `internal/libview`.
11. **Mausrad:** Senkrechtes und waagerechtes Rad rollen beide die Leiste.

## Zusatz (24.11.2026, Sitzungs-QA, Karte #252, Claude Opus 5.5)

Wo die Entscheidung oben und die Ergänzungen sich widersprechen, **gelten die Ergänzungen** (F-659):

- **Punkt 1, „Sprache über `date format`“:** überholt durch Ergänzung 3 (Kontexte `date part` und `weekday`).
- **Konsequenzen, „ADR-0018 Nachtrag 5“:** gemeint ist Nachtrag 6 (Ergänzung 10).
- **Konsequenzen, „über den Menüeintrag, nicht über einen Umschalter“:** überholt durch Ergänzung 9: Im Drehbuch-Lauf
  startet die Leiste aus (`SCHNAPP_PROBE_LEISTE_AUS`) und merkt ihren Schalter nicht. Die Einstellung bleibt dabei
  unberührt, die Lehre aus F-539 also gewahrt.
- **Punkt 1 und 3, Knopf „Library“ in der Leiste:** erst mit Teil 2 (Ergänzung 1).

Nachgeschärft nach der Sitzungs-QA (Kritiker B):

- **Ergänzung 6, Sperre in beiden Richtungen (F-652, F-653):** Solange die andere Aufnahme lädt, ruht das Bearbeiten
  wie beim Speichern (`canvas.Sitzung.SetzeRuhe`): kein Export-Dialog, kein Bereich der Texterkennung, keine Texteingabe,
  kein Zug. Speichern, Schließen und Signale bleiben möglich und halten den Wechsel an. Ist beim Abholen doch ein
  Export-Dialog, eine Texteingabe oder ein Zug offen, bleibt die offene Aufnahme. „Wird die offene Aufnahme in der
  Zwischenzeit geändert“ meint damit nur noch Änderungen an der Ruhe vorbei.
- **Ergänzung 6, Fehler beim Laden:** übersetzt, soweit die Ladefunktion den Fehler kennt; sonst englisch ohne
  Paketpräfix und ohne Pfad (ADR-0019 Nachtrag 4, F-663).
- **Ergänzung 6, „ein Klick auf die offene Aufnahme tut nichts“:** auch wenn einer der Pfade über einen Verweis führt;
  verglichen wird wie beim grünen Rand über `library.Relativ` (F-657).
- **Ergänzung 4, `ErzeugeVorschau`:** läuft unter der Sperre der Bibliothek wie Speichern und Entfernen und schreibt nur,
  solange die Datei den geprüften Stand behält (vor dem Lesen, vor und nach dem Umbenennen); nur reguläre Dateien, kein
  Verweis, kein Ordner-Verweis aus der Ablage (F-651, Kernprinzip 2).
- **Tastaturfokus:** Kommt beim Neulesen vorn eine Aufnahme dazu, bleibt der Fokus bei derselben Aufnahme (F-656); ohne
  Fokus in der Leiste zieht das Neulesen ihn nicht hinein.
- **Ergänzung 9, Sichtprobe:** Die Probe-Bibliothek legt `go run ./scripts/probelibrary -heim tmp/probe/home` an
  (Aufruf im Kopf von `bibliothek-174-leiste.txt`).

## Ergänzt bei der Umsetzung (Teil 2, 24.11.2026, Karte #174, Claude Opus 5.5)

Teil 2 setzt die Punkte 3 und 4 und die Ansicht-Seite von 6, 7 und 9 um (`internal/ui/bibliothek.go`, Logik ohne Gio in
`internal/libview/raster.go` und `ansicht.go`, Papierkorb `library.InPapierkorb`). Gestaltung nach Muster 174 b. Was
die Entscheidung offen ließ oder anders kam:

1. **Knopf „Library“** links in der Leiste (100 dp breit, 10 dp vom Rand; *seit Zusatz Karte #311:* 40 dp, Symbol), die Vorschauen beginnen dahinter; er gehört
   zur Fokusfahrt und öffnet die Ansicht wie Datei → „Bibliothek“ (Ergänzung 1 von Teil 1 eingelöst).
2. **Kopf:** In Fenstern unter 920 dp Breite steht er in zwei Zeilen (Suchfeld und „Back to image“ oben, Zeitraum und
   Schalter darunter) und ohne Titel, damit nichts überlappt. Suche, Zeitraum und Schalter beginnen bei jedem Öffnen
   leer bzw. bei „All dates“ und aus (Punkt 3 nennt das nur für den Schalter; ein gemerkter Filter, der beim nächsten
   Öffnen Aufnahmen verbirgt, wäre verwirrender als ein Neubeginn).
3. **Suche:** genau ein Schlagwort wie `library.Anfrage.Schlagwort` (Groß- und Kleinschreibung egal, kein Teilwort);
   gesucht wird bei jeder Eingabe im Hintergrund, ein veraltetes Ergebnis wird verworfen. **Zeitraum:** „Today“,
   „Last 7 days“ und „Last 30 days“ zählen ab Mitternacht und schließen heute ein, „This year“ ab dem 1. Januar
   (`libview.Zeitraum.Von`); die Auswahl läuft über die Liste des Menüs (`listeZeitraum`).
4. **Raster:** Kachel 206 × 128 dp Vorschau, darunter Datum mit Uhrzeit (im laufenden Jahr ohne Jahr, Kontext `date part`)
   und „Breite × Höhe · Schlagworte“; Spalten nach der Fensterbreite, Monatsüberschrift „November 2026 · 13 captures“
   (Monatsnamen über den Kontext `month`). Geladen werden die Vorschauen der sichtbaren Zeilen und je einer Zeile darüber
   und darunter; der Speicher der Vorschauen ist der der Leiste (ein `libview.Vorschauen`, 64 MiB). Die Ansicht zeigt
   alle gefundenen Aufnahmen, nicht nur die 200 der Leiste. Rollbalken eckig wie die rechte Leiste; nur das senkrechte
   Rad rollt.
5. **Tastatur:** Die Ansicht hält den Tastaturfokus selbst wie die Einstellungstafel (ADR-0034); **Tab** kreist zwischen
   Suchfeld, Zeitraum, Schalter, „Back to image“ und Raster, Umschalt+Tab rückwärts; Runter oder Enter im Suchfeld geht
   ins Raster. Hoch und runter halten die Spalte, auch über Monatsgrenzen (in einer kürzeren Zeile die letzte Kachel);
   Bild↑/↓ blättern so viele Zeilen, wie ganz sichtbar sind. **F10** öffnet das Menü; von den übrigen Kürzeln wirken
   nur die der bedienbaren Menüeinträge (ADR-0026 Nachtrag 10). Werkzeugtasten, Leinwand-Kürzel, getippter Text außer in
   den Feldern, Zeiger auf der Leinwand und die Felder Farbe und Zoom der Fußzeile wirken nicht.
6. **Öffnen:** Enter oder Doppelklick (zweiter Druck auf dieselbe Kachel innerhalb von 400 ms, wie auf der Leinwand)
   geht über `wechselAnfrage` wie die Leiste; die Ansicht schließt erst, wenn die neue Aufnahme übernommen ist. Scheitert
   Speichern oder Laden, bleibt sie offen, und die Fußzeile sagt warum. Enter auf der offenen Aufnahme schließt nur die
   Ansicht.
7. **Papierkorb (Entf):** `library.InPapierkorb` legt die Aufnahme wie das Aufräumen in den freedesktop-Papierkorb
   (Umbenennen samt `.trashinfo`, rückholbar, Vorschau und Indexeintrag fallen weg), aber **ohne** dessen Schutzregeln
   (Schlagworte, Bearbeitung, Alter): Der Nutzer will genau diese Aufnahme loswerden. Nur reguläre Dateien in der
   Ablage, kein Verweis; auf einem anderen Dateisystem bleibt sie liegen (Meldung). Die offene Aufnahme ist ausgenommen
   (Hinweis in der Fußzeile, der Menüeintrag ist grau mit Erklärung).
8. **Schlagworte (F2):** Das Feld unter der Kachel beginnt mit den bisherigen Schlagworten („a, b“); Komma trennt,
   Leerraum an den Enden und doppelte (ohne Rücksicht auf Groß- und Kleinschreibung) fallen weg
   (`libview.ZerlegeSchlagworte`). Enter schreibt im Hintergrund über `SetzeSchlagworte`, Esc oder ein Klick woanders
   bricht ab. **Abweichung:** An der offenen Aufnahme geht F2 nicht (Hinweis, Menüeintrag grau). Ihre Datei schreibt der
   Editor beim Speichern selbst; ein zweiter Schreiber dazwischen änderte ihren Stand, und das nächste Speichern legte
   eine neue Aufnahme an.
9. **Hintergrund:** Index, Papierkorb und Schlagworte laufen in Goroutinen; während einer solchen Arbeit, eines
   Speicherns oder eines Wechsels wirken Enter, Entf und F2 nicht. Danach liest die Ansicht und die Leiste den Index neu
   (Punkt 7), die Vorschau der Aufnahme wird vergessen.
10. **Menü:** Datei → „Library“ steht nach „Settings…“; bei offener Ansicht bedient das Menü „Bearbeiten“ die Ansicht
    (Öffnen, Schlagworte, Papierkorb, Suche), die übrigen Bild-Einträge sind grau mit Erklärung (ADR-0026 Nachtrag 10).

Getestet in `internal/libview/raster_test.go`, `ansicht_test.go`, `internal/library/papierkorb_test.go` und
`internal/ui/bibliothek_test.go`; Mutanten `scripts/mutanten/bibliothek-ansicht.json`; Sichtprobe
`scripts/drehbuecher/bibliothek-174-ansicht.txt` (Probe-Bibliothek mit Schlagworten aus `scripts/probelibrary`).

## Zusatz (24.11.2026, Karte #249, F-637, Claude Opus 5.5)

**Ende der Rettungskopie im Hintergrund.** Beim Wechsel (Ergänzung 7 von Teil 1) und wenn eine geladene Aufnahme
verworfen wird (Zusatz zu Ergänzung 6), wartete `Rettung.Ende` in der Ereignisschleife bis zu 5 s auf die
Schreibkette (F-417); so lange zeichnete das Fenster nicht. Jetzt läuft `Ende` in einer Goroutine
(`rettungEndeImHintergrund` in `internal/ui/wechsel.go`), mehrere nacheinander; der Kanal `rettungEnden` schließt nach
dem letzten. Ein folgender Wechsel lädt erst danach (in seiner Lade-Goroutine, die Oberfläche wartet nicht): Führt er
zurück zu derselben Datei, ist deren Sperre dann frei, und die neue Sicherung läuft nicht als fremd. Das Zerstören des
Fensters wartet höchstens `wechselFrist` (5 s) auf diese Enden. Tests `TestWechselWartetNichtAufAlteRettung`,
`TestWechselVerworfenEndetImHintergrund`, `TestZerstoerenWartetAufAlteRettung`; Mutanten
`scripts/mutanten/wechsel-rettung.json` (WR1–WR4).

## Zusatz (24.11.2026, Karte #256, F-669, Claude Opus 5.5)

**Weggerollte Vorschauen werden nicht mehr geladen.** Zu Punkt 6: `libview.Vorschauen` zählt Runden; die Leiste und
die volle Ansicht rufen je gezeichnetem Bild einmal `Runde()` und fragen danach die sichtbaren Vorschauen ab. Ein
wartender Auftrag, der seit zwei Runden nicht mehr abgefragt wurde, verfällt in der Lade-Goroutine (`veraltetNach` =
1); eine spätere Abfrage fordert ihn neu an. Ohne Neuzeichnen verfällt nichts. Tests
`TestVorschauenWeggerollteVerfallen`, `TestVorschauenOhneRundeVerfaelltNichts`; Mutanten
`scripts/mutanten/vorschauen-runden.json`. Den Aufruf von `Runde()` in der Oberfläche deckt kein Test (ohne ihn
verfiele nur nichts, wie vorher).

## Zusatz (24.11.2026, Karte #257, F-671, Claude Opus 5.5)

**Suche nach Teilwort.** Abweichend von Ergänzung 3 aus Teil 2 trifft das Suchfeld der vollen Ansicht jetzt jeden Teil
eines Schlagworts („ogi“ findet „login“), ohne Groß-/Kleinschreibung und Leerraum an den Enden: neues Feld
`library.Anfrage.Teilwort`, das `libview.Anfrage` statt `Schlagwort` setzt. Das MCP-Werkzeug `library_search` behält den
genauen Vergleich (`Schlagwort`, ADR-0030), damit sich das Verhalten für Agenten nicht still ändert. Dass ein spätes
Ergebnis einer überholten Suche verworfen wird (`indexNochmal`), prüft jetzt `TestBibliothekUeberholteSuche`. Mutanten
`scripts/mutanten/bibliothek-suche.json` (BS1–BS5).

## Zusatz (24.11.2026, F-672, Echtlauf Andi, Claude Opus 5.5)

**Index mit der Ablage abgleichen.** `library.Abgleich` (Einträge ohne Datei fallen weg, ihre Vorschau auch) rief das
Programm bisher nirgends auf: Eine im Dateimanager gelöschte oder verschobene Aufnahme stand weiter in Leiste und
Ansicht, Öffnen und Papierkorb scheiterten. Zu Punkt 7: Die Lese-Goroutine gleicht jetzt vorher ab, wenn die Leiste
startet, die Ansicht öffnet oder ein Öffnen bzw. Papierkorb gescheitert ist (`untenLeiste.abgleich`,
`bibAnsicht.abgleich`, `abgleichAnfordern`, `gleicheAb`); das Lesen nach jedem eigenen Speichern gleicht nicht ab
(Kosten je Speichern, F-638). Tests `TestLeisteFehlendeDateiVerschwindet`, `TestBibliothekOeffnenGleichtAb`,
`TestBibliothekPapierkorbFehlendeDatei`, `TestLeisteStartGleichtAb`; Mutanten `scripts/mutanten/bibliothek-abgleich.json`.

## Zusatz (24.11.2026, Sitzungs-QA, F-673, F-675, F-678, F-685, F-689, Claude Opus 5.5)

**Bibliothek gehärtet, Folgen für die Ansicht** (Einzelheiten ADR-0016 Nachtrag 6). Zu Punkt 7: `InPapierkorb` und
`SetzeSchlagworte` lehnen jetzt auch einen Monatsordner ab, der als Verweis aus der Ablage führt (`ErrPfad`).
`SetzeSchlagworte` überschreibt keine Datei, die ein anderes Fenster inzwischen gespeichert hat (`ErrGeaendert`, F2 +
Enter meldet dann einen Fehler statt die Schwärzung zurückzunehmen). `InPapierkorb` liefert `ErrIndex` nur nach
gelungenem Verschieben; die Ansicht soll das als Erfolg mit Hinweis melden (Aufrufer `bibliothekArbeitFertig`). Der
Abgleich beim Start und beim Öffnen der Ansicht (Zusatz F-672) leert den Index nicht mehr, wenn die Ablage fehlt oder
leer ist (`ErrAblageLeer`, nur Diagnose), und baut einen fehlenden Index neu auf. Das Suchfeld faltet wie das ganze
Schlagwort (Zusatz F-671).

## Zusatz (24.11.2026, Karte #273, F-706, F-708, F-710, Claude Sonnet 5.5)

**Oberflächenzustand beim Wechsel und beim Schließen** (Abschluss-Kritik C, Sprint 12). (1) `uebernimm` beendet einen
laufenden Zug der Ebenenliste (`e.ebenen.druck`); ein späteres Loslassen ordnet in der neuen Aufnahme nichts um
(F-706). (2) `zerstoert` wartet nach dem Kopieren höchstens `wechselFrist` (5 s, wie bei den Rettungskopien) auf eine
laufende Arbeit der Ansicht (Schlagworte, Papierkorb, `bibliothekArbeitAbwarten`); das Ergebnis wird nicht mehr
gemeldet, das Schreiben reißt aber nicht mit dem Prozessende ab (F-708). Das Fenster bleibt dafür nicht offen: Das
Muster der Rettung ist Warten mit Frist, und `schliessen()` hat für diese Arbeit keine Rückfrage an Andi. (3) Ein
laufendes Kopieren sperrt den Wechsel (`wechselAnfrage`, `kopierLaeuft`): der Klick gilt nicht, genau wie bei laufender
Arbeit der Ansicht (F-676). So vergleicht die Antwort nie mit dem Stand der neuen Sitzung. Der Wechsel wird nicht
vorgemerkt; nach der Antwort genügt ein weiterer Klick. Tests `internal/ui/wechsel_zustand_test.go`, Mutanten
`scripts/mutanten/ui-wechsel-zustand.json`.

## Zusatz (25.11.2026, Karte #288, F-759, Claude Sonnet 5.5)

Der gesperrte Klick bleibt nicht mehr still. `wechselAnfrage` meldet bei laufendem Kopieren „Copying is still running. Please wait a moment, then try again.“ in der Fußzeile, über `meldeNebenbei` (eine haltende Meldung wie die Verwerfen-Warnung bleibt stehen). Tests `internal/ui/wechsel_kopieren_meldung_test.go`, Mutanten WZ6 und WZ7 in `scripts/mutanten/ui-wechsel-zustand.json`.

## Zusatz (25.11.2026, Karte #295, F-774, F-775, Claude Sonnet 5.5)

`wechselAnfrage` prüft die offene Aufnahme (`istOffen`) vor allen Sperrfällen: Ein Klick auf sie meldet nie etwas, auch nicht während des Kopierens (F-774). Die übrigen gesperrten Klicks sagen es in der Fußzeile über `meldeNebenbei`, auch bei laufender Arbeit der Ansicht („The library is still working…“), beim Laden einer anderen Aufnahme („Another capture is still loading…“) und beim Speichern („Saving is still running…“, jeweils „Please wait a moment, then try again.“); beim Schließen des Fensters (Signal, erzwungen) bleibt der Klick still (F-775). Tests `internal/ui/wechsel_klick_meldung_test.go`, Mutanten WZ8 bis WZ11 in `scripts/mutanten/ui-wechsel-zustand.json`.

## Zusatz (25.11.2026, Karte #311, Claude Opus 5.5)

Andi (25.11.2026): „geht die "Bibliothek" Kachel unten links platzsparender und sinnvoller? … unten drunter sollte
nicht nur das änderungsdatum, sondern auch die auflösung stehen.“

- **Knopf „Library“** ist ein schmaler Symbolknopf (`leisteKnopfB` 40 statt 100 dp, so hoch wie eine Vorschau) mit dem
  Symbol `icons.Bibliothek`; Name und Kürzel stehen im Hinweis der Statuszeile wie bei den Werkzeugen, Fokusring orange.
  Die Vorschauen beginnen 60 dp weiter links.
- **Zwei Zeilen unter jeder Vorschau:** Zeitangabe wie bisher, darunter im Nebentext die Auflösung der Ansicht aus dem
  Index (`leisteAufloesung`, Kontext `status bar` „%d × %d“). Die Leiste wird dafür 118 statt 104 dp hoch (Punkt 1 und
  Abschnitt „Konsequenzen“); die Schwelle von 420 dp Leinwandhöhe bleibt. Die volle Ansicht zeigte die Auflösung schon
  (`libview.KachelAngaben`).
- Tests `internal/ui/leiste_knopf_aufloesung_test.go`, Drehbuch `bibliothek-174-leiste.txt` auf die neuen Lagen nachgezogen.

## Zusatz (25.11.2026, Karte #312, Claude Opus 5.5)

Andi (25.11.2026): „wenn ich mit der maus über den vorschau-bildern bin (oder dem finger lang drücke oder so) dann sollte
ein x oben rechts erscheinen, mit dem ich das vorschaubild (bzw. das ganze bild) löschen kann – dies sollte auch in der
bibliotheks-ansicht selbst so sein.“

- **✕ oben rechts** (20 dp, 4 dp vom Rand) in jeder Vorschau der Leiste und jeder Kachel der vollen Ansicht, sobald der
  Zeiger darüber steht oder nach einem **langen Druck** (`langDruckZeit` 500 ms, Finger oder Maus; in der Leiste über die
  Druck-Historie des Knopfs, der lange Druck öffnet dann nicht). Rot unter dem Zeiger, Hinweis „Move to trash (Del)“.
- **Löschen heißt Papierkorb** wie Entf in der vollen Ansicht (Punkt 4): `library.InPapierkorb`, rückholbar; ein
  gemeinsamer Weg `papierkorbAnfrage` für Ansicht und Leiste. Die **offene Aufnahme** zeigt kein ✕, Entf an ihr meldet (abgelöst durch den Zusatz zu Karte #347)
  nur. Während Speichern, Wechsel oder einer Arbeit der Ansicht wartet die Anfrage nicht, die Fußzeile sagt warum.
- **Entf an einer Vorschau der Leiste** legt sie ebenfalls in den Papierkorb (vorher wirkungslos).
- *Kritik Fable 25.11. (K-2, K-3):* Nach einem Löschen gilt `loeschRuhe` (Doppelklickzeit + 200 ms) kein ✕ und kein
  Klick auf die Vorschau der Leiste, sonst träfe der zweite Klick eines Doppelklicks die nachgerückte Aufnahme;
  `langDruck` beginnt bei −1 und fällt nach jedem Neulesen der Liste weg.
- Tests `internal/ui/papierkorb_kreuz_test.go`, Mutanten `scripts/mutanten/papierkorb-kreuz.json` (PK1–PK9), Sichtprobe
  `scripts/drehbuecher/papierkorb-kreuz-312.txt`.

## Zusatz (26.11.2026, Karte #347, Claude Opus 5.5)

Andi (26.11.2026): „das x in der bibliothek(sleiste) sollte auch beim aktuellen bild da sein - ein strg-c (was ich
vorher gedrückt habe) soll dadurch aber nicht das bild in der zwischenablage verlieren“ und „wenn ich ein bild lösche in
der leiste, rückt das nächste nach - dort ist das x erst da, wenn ich die maus weg und wieder auf das bild bewege“.
Löst die Ausnahme der offenen Aufnahme aus dem Zusatz zu Karte #312 ab (dort „Die offene Aufnahme zeigt kein ✕“).

- **✕, Entf und Bearbeiten → „Move to trash“ gelten auch an der offenen Aufnahme**, in Leiste und voller Ansicht. Der Weg
  ist ein Wechsel (Punkt 5) mit Merker `wechselZustand.papierkorb`: erst Speichern wie Strg+S (der letzte Stand samt
  Beschriftungen landet im Papierkorb und ist rückholbar), dann Laden der **Nachbarin** (`nachbarStelle`: die nächst
  ältere, an der ältesten die nächst jüngere), Übernehmen, erst dann `papierkorbAnfrage` für die Datei, die den Stand
  trug (`Geoeffnet.Pfad` vor der Übernahme, nach einem Speichern als neue Aufnahme die neue). Das Fenster zeigt so nie
  eine weggeworfene Datei, und ein Schließen mitten im Ablauf speichert nicht in den Papierkorb hinein.
- Scheitert Speichern oder Laden, oder hält ein Speicher-Hinweis den Wechsel auf, bleibt die Aufnahme offen und im
  Bestand; die Meldung sagt zusätzlich „The capture was not moved to the trash.“ Die **einzige** Aufnahme bleibt offen
  (Meldung, Menüeintrag grau mit Erklärung): Der Editor hat keinen leeren Zustand.
- Aus der **vollen Ansicht** weggeworfen, bleibt die Ansicht offen (`editor.wechselBehaeltAnsicht`, grüner Rand über
  `bibliothekAktuell`); ein gewöhnlicher Wechsel schließt sie weiter.
- **Zwischenablage:** Kopiert hält der Halter `wlclip` die Bytes in einem eigenen Prozess (ADR-0024), im Rückfall das
  Fenster (`fensterHaeltKopie`); beide hängen nicht an der Datei, und der Wechsel fasst sie nicht an. Ein laufendes
  Kopieren sperrt den Wechsel wie bisher (F-710).
- **Ruhe-Ende weckt:** Während `loeschRuhe` bestellt das Zeichnen von Leiste und Ansicht ein Bild zu
  `loeschRuheBis` (`loeschRuheWecken`); vorher erschien das ✕ der nachgerückten Aufnahme unter dem stehenden Zeiger erst
  mit der nächsten Mausbewegung.
- Tests `internal/ui/papierkorb_kreuz_test.go` (`TestLeisteKreuzAnDerOffenen`, `…LadenScheitert`, `…EinzigeBleibt`,
  `…NachRuheOhneBewegung`), `TestBibliothekPapierkorb`; Mutanten PK1, PK4, PK10–PK16.

## Zusatz (26.11.2026, Karte #350, F-827, F-828, Claude Opus 5.5)

Sitzungs-Kritik 26.11. K-9 und K-10.

- **Mehrere Nachbarinnen (F-827).** Statt einer Nachbarin (`nachbarStelle`) gibt es eine Reihe nach Abstand
  (`nachbarStellen`: die nächst ältere, die nächst jüngere, dann die übernächsten; höchstens `nachbarGrenze` = 8; die offene
  selbst nie, auch wenn sie über einen Verweis-Ordner zweimal in der Liste steht, `nachbarnIn`). Lässt sich die erste
  nicht laden (beschädigt, über dem Budget), lädt der Editor die nächste (`wechselZustand.weitere`); die offene ist dann
  schon gespeichert. Nicht weiter versucht wird nach einer Änderung während des Ladens, bei erzwungenem Schließen und nach
  einem Signal. Erst wenn alle scheitern, sagt die Meldung „The capture was not moved to the trash.“
- **Gefilterte Ansicht (F-827).** Zeigt die volle Ansicht wegen Suche oder Zeitraum keine andere Aufnahme, gelten die
  Nachbarinnen der Leiste der letzten Aufnahmen (ungefiltert, `bibNachbarn`). Gibt es auch dort keine, sagt die Meldung
  „No other capture to switch to: clear the search and the period, then try again.“ statt „only capture“ (`einzigText`).
- **Öffnen… der offenen Aufnahme (F-828).** `Anbindung.Zuordnen` (`cmd/schnapp` `zuordnerFuer`, ohne Nebenwirkung) bildet
  die im Dateidialog gewählte Datei in dessen Goroutine auf die Aufnahme ab, die das Laden öffnen würde: eine früher
  übernommene Datei von außen auf ihre Aufnahme (ADR-0016 Nachtrag 10 Punkt 2). Ist das die offene, wird weder gespeichert
  noch neu geladen (der Rückgängig-Verlauf bleibt); die Fußzeile sagt „This file is already open.“

Tests `TestLeisteKreuzAnDerOffenenZweiteNachbarin`, `TestLeisteKreuzZweiteNachbarinNichtNachAenderung`,
`TestAnsichtGefiltertNimmtNachbarinDerLeiste`, `TestNachbarinnenOhneOffene`, `TestOeffnenDerOffenenUeberZuordnung`
(`internal/ui`), `TestZuordnerFindetUebernommeneAufnahme` (`cmd/schnapp`); Mutanten PK-N1–PK-N6, OE-F828-1/2,
IO-F828-1–3.
