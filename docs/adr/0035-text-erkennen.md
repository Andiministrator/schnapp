# ADR-0035: Text aus dem Bild erkennen (OCR) über ein lokales tesseract

## Status
Akzeptiert (Claude Opus 5.5, 24.11.2026, Karte #195 in Sprint 12, im Auftrag Andis: „arbeite selbstständig“).
Grundlage: Wunsch Andis vom 22.11.2026 (Ideenliste, Idee 7: „Text aus dem Bild kopieren“). Recherche in
`knowledge/sources-visited.md`, Abschnitt „OCR“. Ergänzt ADR-0011 (Fremdpakete), ADR-0015/ADR-0024 (Zwischenablage),
ADR-0026 (Menü und Kürzel).

## Kontext
Auf Screenshots steht oft Text, den man weiterverwenden will (Fehlermeldung, Pfad, Seriennummer). Spectacle kann das
seit Plasma 6.6 über Tesseract; Snagit hat „Grab Text“. Kernprinzip 3 verlangt, dass nichts den Rechner verlässt, also
nur lokale Erkennung. Eine reine Go-Erkennung in brauchbarer Güte gibt es nicht; Tesseract als Bibliothek (gosseract)
bräuchte cgo und bände Schnapp an die Bibliotheksversion — Spectacle zeigt, wie zerbrechlich das ist (OCR grau, weil
die Bibliothek nicht gefunden wird). Das Programm `tesseract` liest seit 5.x ein Bild von stdin und schreibt den Text
auf stdout; eine Probe auf diesem Rechner (tesseract 5.5.4, Bild 1600 × 1000) dauerte rund 1 s.

## Entscheidung

1. **Externes Programm, kein cgo, kein Fremdpaket:** Schnapp ruft `tesseract` auf, wenn es im `PATH` steht. Fehlt es,
   ist der Menüeintrag grau und seine Erklärung in der Statuszeile nennt das Paket („install tesseract“). Neues Paket
   `internal/ocr` (ohne Gio): Programm suchen (`exec.LookPath`, einmal im Hintergrund beim Öffnen des Editors), Sprachen
   ermitteln (`tesseract --list-langs`, einmal), erkennen.
2. **Aufruf:** `tesseract - - -l <sprachen> --psm 3` mit fester Argumentliste (kein Shell), Bild als PNG über stdin,
   Text über stdout; keine temporäre Datei. Frist 30 s, stdout höchstens 1 MiB, stderr verworfen (höchstens 64 KiB
   gelesen). Abbruch (Esc, Schließen des Editors) beendet den eigenen Kindprozess über den Kontext. Ausgabe: gültiges
   UTF-8 erzwingen, `\f` am Ende und Leerzeilen am Rand weg, Zeilenenden `\n`.
3. **Sprachen:** die Sprache der Oberfläche (ISO 639-1 → Tesseract-Code, z. B. `de` → `deu`) und Englisch, soweit
   installiert, als `deu+eng`; ist keine davon installiert, die erste installierte außer `osd`; gibt es gar keine,
   ist der Eintrag grau. Keine neue Einstellung (ADR-0034 verlangte sonst eine Tafelzeile); eine Wahl der Sprachen
   kann als eigener Schlüssel nachkommen.
4. **Was erkannt wird:** das **sichtbare Bild, wie es exportiert würde** (`render.Bild`: Ansicht mit Zuschnitt,
   Stauchung, Schwärzungen und Beschriftungen), im gewählten Bereich. Nie das Original: Geschwärztes darf auch als
   Text nicht herauskommen (Kernprinzip 2). Ist der Bereich klein (kürzere Seite unter 600 px), wird er vor der
   Erkennung auf das Doppelte vergrößert (bilinear, `x/image/draw`), weil Tesseract kleine Bildschirmschrift sonst
   häufiger verliest (Umlaute in der Probe). Über 40 Mio. Pixel lehnt Schnapp mit Meldung ab (Ergänzt F-633: nur unvergrößert gemessen, siehe unten).
5. **Bedienung:** Menü Bearbeiten → „Copy text from image“/„Text aus dem Bild kopieren“, Kürzel **Strg+Umschalt+T**
   (frei; T wie Text). Danach ist ein einmaliger Bereichsmodus aktiv (Zeiger Fadenkreuz, Statuszeile „Drag over the text
   to copy; Enter takes the whole image; Esc cancels“): Ziehen wählt den Bereich in der Ansicht, Enter nimmt das ganze
   sichtbare Bild, Esc bricht ab. Nach dem Loslassen läuft die Erkennung im Hintergrund (Statuszeile „Recognizing
   text…“, Bearbeiten bleibt möglich), danach steht das vorige Werkzeug wieder. Ergebnis: Text in die Zwischenablage,
   Statuszeile „Copied N characters“; ohne Text „No text found“, die Zwischenablage bleibt dann unverändert; bei Fehler
   die Meldung über `meldeNebenbei` (ADR-0024 Nachtrag 7).
6. **Zwischenablage:** Text wie das Bild (ADR-0024): im Fenster über Gio, nach dem Schließen hält `wlclip` ihn weiter,
   mit den Typen `text/plain;charset=utf-8`, `text/plain` und `UTF8_STRING`. Der Halter bekommt dafür einen Textmodus
   (Nachtrag zu ADR-0024 bei der Umsetzung).
7. **Flatpak (später, vor Flathub):** Die Sandbox sieht das `tesseract` des Systems nicht. Der Flatpak-Bau bündelt
   tesseract, leptonica und die Sprachdaten `eng` und `deu` als Module; weitere Sprachen über eine Erweiterung. Bis
   dahin ist der Eintrag im Flatpak grau. Kein `flatpak-spawn --host` (bräuchte eine weite Freigabe).
8. **Agenten** (`schnapp mcp`, `--annotate`) bekommen die Erkennung in diesem Schritt nicht.

## Erwogene Alternativen
- **gosseract (cgo, libtesseract):** schneller Aufruf, aber cgo gegen ADR-0011, bricht mit jeder Bibliotheksversion
  und erschwert Flatpak und Kreuzbau.
- **Reines Go über WebAssembly (gogosseract: Tesseract als WASM im Go-Laufzeitsystem wazero, Sprachdaten per
  `go:embed`):** kein cgo und kein externes Programm, liefe auch im Flatpak ohne Zusatzmodul. Dagegen: laut Besprechung
  rund sechsmal langsamer als nativ (bei 1 s nativ also mehrere Sekunden), nur ein Betreuer mit Fork für wazero,
  Pflegestand unklar, Programm um Laufzeit, WASM-Modul und Sprachdaten (je Sprache einige MiB) größer; ein Fremdpaket
  ohne spürbaren Nutzen gegenüber dem installierten Programm (ADR-0011). Bleibt die Rückfallebene, falls das Bündeln im
  Flatpak (Punkt 7) scheitert. Eine eigene Erkennung in Go (Netz und Gewichte selbst) steht in keinem Verhältnis.
- **ocrs (Rust, eigenes Modell) oder PaddleOCR (Python):** selten installiert, große Modelle; Tesseract ist in jeder
  Distribution ein Paket.
- **Ganzes Bild immer:** einfacher, aber man will meist eine Stelle; Enter deckt das ganze Bild trotzdem ab.

## Konsequenzen
- Neues Paket `internal/ocr` mit Tests über eine Attrappe des Programms (Shell-Skript im Test-`PATH`, eigenes HOME),
  eine echte Erkennung nur als übersprungener Test, wenn `tesseract` fehlt.
- ADR-0026: Menüeintrag und Strg+Umschalt+T; ADR-0024: Textmodus des Halters.
- `docs/anforderungen.md`: OCR als optionale Abhängigkeit in der README nennen („install tesseract for text
  recognition“).

## Ergänzt bei der Umsetzung (24.11.2026, Oberfläche, Karte #195)

Wo die Oberfläche von den Punkten 1 bis 6 abweicht oder sie genauer fasst (`internal/ui/ocr.go`,
`internal/canvas/textbereich.go`, ADR-0024 Nachtrag 9, ADR-0026 Nachtrag 8):

- **Auf Weiß (Punkt 4):** Das Bild aus `render.Bild` wird vor der Erkennung auf Weiß gelegt. Der Spalt einer Stauch-Naht
  ist durchsichtig (ADR-0023 Nachtrag 7); wie tesseract Alpha liest, hängt vom Leser ab, Weiß ist eindeutig.
- **Bereich (Punkt 5):** Der Bereich gilt in Ansichtskoordinaten (unabhängig vom Zoom, nach Zuschnitt und Stauchen), wird
  nach außen auf ganze Pixel gerundet und auf das sichtbare Bild begrenzt (`canvas.SichtPixel`). Ein Klick unter der
  Ziehschwelle oder ein Zug ganz neben dem Bild wählt nichts, der Modus wartet weiter. Beim Werkzeug Zuschneiden
  (Vollansicht) zählt nur der Teil im Zuschnitt. Die mittlere Taste und das Rad verschieben und zoomen wie sonst; ein
  Werkzeugwechsel beendet den Modus. Der Modus ändert das Werkzeug nie, „danach steht das vorige Werkzeug wieder“ gilt
  damit von selbst.
- **Esc (Punkt 5):** Im Bereichsmodus beendet Esc den Modus (auch mitten im Zug). Während der Erkennung bricht Esc sie ab
  („Text recognition cancelled.“), solange Esc nichts mit Vorrang zu tun hat (Bereichsmodus, Zug, Texteingabe); danach
  wirkt Esc wieder in seinen Stufen.
- **Grau (Punkt 1):** Der Eintrag ist grau, solange die Suche läuft („Looking for tesseract…“), ohne Programm („Install
  tesseract to recognize text.“), ohne Sprachdaten („Install language data for tesseract to recognize text.“), bei einem
  anderen Fehler der Suche und während einer Erkennung. Die Erklärung steht beim Überfahren und bei Tastaturfokus in der
  Fußzeile (neues Muster, ADR-0026 Nachtrag 8) und als Meldung, wenn das Kürzel gedrückt wird.
- **Sprache (Punkt 3):** Die Sprache der Oberfläche kommt aus dem aktiven Katalog (`i18n.Sprache`, neu).
- **Statuszeile:** „Recognizing text…“ steht als Meldung und als Hinweis, damit es auch nach einer Bearbeitung während der
  Erkennung sichtbar bleibt. Fehler „area too large“ und „took too long“ sind eigene übersetzte Texte, andere Fehler
  stehen als „Could not recognize text: …“ (ohne Paketpräfix).
- **Testnaht:** Die Oberfläche spricht tesseract über eine eigene Schnittstelle an (`textErkenner`, Suche über die
  Variable `sucheErkenner`); `internal/ocr` braucht dafür keine exportierte Naht.

- **Ergänzt (F-633, 24.11.2026) zu Punkt 4:** Verdoppelt wird nur, wenn das Ergebnis noch passt: Fläche höchstens 40 Mio. Pixel und jede Seite höchstens 32767 (`ocr.MaxSeite`, Grenze von Leptonica). Sonst geht der Bereich unvergrößert weiter, ein 590 × 18000 großer Streifen wird also erkannt, nicht abgelehnt. Mit `ErrZuGross` ablehnen nur noch Bereiche, die schon unvergrößert über 40 Mio. Pixel oder über 32767 auf einer Seite liegen.

## Nachtrag 1 (24.11.2026, Karte #242): eigenes Hilfsprogramm schnapp-ocr, wenn tesseract fehlt

Entscheidung Andis vom 24.11.2026 („ja, so“): erst das installierte tesseract, dann ein eigenes Hilfsprogramm. Messung
(Karte #242, synthetisches Bild, 6 Zeilen mit Umlauten): Güte der WASM-Fassung gleich (deu 98–99 %, zeichengleich mit
tesseract), Tempo 4- bis 17-mal langsamer (1–6 s statt 0,2–0,6 s je Bereich, Start 0,5 s). Brauchbar als Ersatz.

1. **Programm `schnapp-ocr`** im eigenen Go-Modul `ocr/` (wie `draw/`, damit die Fremdpakete nur dort liegen):
   tesseract (nur LSTM) als WebAssembly über `github.com/danlock/gogosseract` (Apache-2.0) und wazero, ohne cgo.
   Schnittstelle ist die Teilmenge von tesseract, die `internal/ocr` braucht: `--list-langs`, `- - -l LANG --psm N`,
   dazu `--progress` (Zeilen `progress N` auf stderr) und `--version`. Je Aufruf **eine** Sprache: Die WASM-Fassung
   lädt Sprachdaten aus dem Speicher, das geht nur für ein Modell; `Erkenner.Sprachen` liefert dann nur die erste
   Sprache der Wahl aus Punkt 3 (deutsche Oberfläche: `deu`, das liest englischen Text in der Messung gleich gut).
2. **Sprachdaten** `deu` und `eng` aus `tessdata_fast` (Apache-2.0, Stand 87416418, 2024-08-01; zusammen 5,6 MB) liegen
   in `ocr/tessdata/` mit Lizenz und sind per `go:embed` eingebaut. Weitere Sprachen bleiben dem installierten
   tesseract vorbehalten.
3. **Suche** (Punkt 1 erweitert): `tesseract` im PATH hat Vorrang. Fehlt es oder hat es keine Sprache, sucht
   `internal/ocr.Suche` `schnapp-ocr` neben dem eigenen Programm (`os.Executable`; dorthin legt `install.sh` es), dann
   im PATH. Fehlen beide, bleibt die Erklärung „Install tesseract to recognize text.“.
4. **Frist** (Punkt 2): für `schnapp-ocr` 3 Minuten statt 30 s. Der Fortschritt steht in der Statuszeile
   („Recognizing text… 42 %“); tesseract meldet keinen, dort bleibt der Text ohne Zahl.
5. **Abbruch:** gogosseract startet wazero ohne `CloseOnContextDone`, ein Kontext erreicht die laufende Erkennung im
   Programm nicht. Abgebrochen wird wie bei tesseract, indem `internal/ocr` den Kindprozess beendet.
6. **wazero 1.7.3:** Ab wazero 1.8 scheitert das von gogosseract mitgelieferte WASM (`emscripten_stack_get_current not
   exported`; mit wazero-emscripten-embind 1.5.2 und wazero 1.12 geprüft). Neuere wazero-Fassungen erst mit einem neu
   übersetzten WASM; bis dahin bleibt 1.7.3 fest. Das WASM ist vertrauter Code aus dem Modul, keine fremde Eingabe.
7. **Flatpak (Punkt 7 ersetzt):** statt tesseract und leptonica als Module zu bündeln, kommt `schnapp-ocr` neben
   `schnapp` in das Paket; das tesseract des Systems ist in der Sandbox ohnehin nicht sichtbar.
8. **Bau:** `install.sh` baut beide Programme in Nebendateien und tauscht erst, wenn beide gebaut sind; `--uninstall`
   entfernt vier Dateien. Die CI hat einen eigenen Job `ocr` (gofmt, vet, Test mit echter Erkennung in deu und eng).

Verworfen: Sprachdaten zur Laufzeit herunterladen (verlässt den Rechner, Kernprinzip 3), `tessdata_best` (rund 4-mal
größer, langsamer, die Messung zeigt keinen Bedarf), zwei Läufe für `deu+eng` (verdoppelt die ohnehin lange Dauer).
