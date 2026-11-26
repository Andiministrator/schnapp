<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="brand/logo/svg/schnapp-logo-dunkel.svg">
    <img alt="schnapp — screenshots for Linux" src="brand/logo/svg/schnapp-logo-hell.svg" width="520">
  </picture>
</p>

# Schnapp

*[English](README.md) · Deutsch*

Screenshot- und Beschriftungswerkzeug für Linux/Wayland — KDE Plasma zuerst. Die Arbeitsweise kurz:
schnell aufnehmen, beschriften (gerade und gebogene Pfeile, Kästen, Schritt-Nummern, Stempel), wirklich schwärzen, die
Mitte langer Seiten herausschneiden, teilen. Nebenbei ein einfaches Bildbearbeitungsprogramm.

## Stand

Version 0.6.0 (nur KDE Plasma) bringt ein neues Editorfenster — Menüzeile, Werkzeugleiste links, Leiste rechts, Fußzeile — mit Ebenen, Schnellstilen, Sprechblasen, Lupe, Navigator, Einrasten, Farbwahl mit Deckkraft, Texterkennung (OCR), einer Leiste der letzten Aufnahmen, einer vollen Ansicht der Bibliothek und einem Einstellungsdialog; Skripte und KI-Agenten können Dateien ohne Fenster beschriften und schwärzen (`--annotate`) und den MCP-Server `schnapp mcp` nutzen. Seit 0.5.0 öffnet Schnapp PNG und JPEG aus dem Dateimanager („Öffnen mit“, JPEG gedreht wie in seiner EXIF-Angabe), setzt auf Wunsch alle Monitore zu einem Bild zusammen, speichert schneller und setzt Text im Editor, in PNG/JPG und in SVG gleich. Seit 0.4.0 wartet Schließen oder Beenden, solange ein langsames Speichern vorankommt. Seit 0.3.0 speichert und kopiert Schnapp im Hintergrund: Das Fenster friert bei großen Bildern nicht mehr ein (das Bearbeiten ruht, bis gespeichert ist; Kopieren und Schließen gehen weiter), und das Menü geht ganz per Tastatur. Schnapp nimmt unter KDE Plasma schon auf (Bereich, Monitor, Fenster; Taste **Druck**), legt jeden
Screenshot in der Bibliothek ab und öffnet ihn in einem Editor, in dem sich Pfeile (gerade oder gebogen), Kästen, Text
und Schritt-Nummern zeichnen sowie auswählen, verschieben und in der Größe ändern lassen, Bereiche endgültig schwärzen
oder verpixeln, zuschneiden, die Mitte eines langen Screenshots herausnehmen und das Ergebnis in die Zwischenablage
kopieren.
Schnapp entsteht in Go mit [Gio](https://gioui.org), dazu ein kleiner Patch an Gio (ein Vollbild-Overlay je Monitor,
gebrochene Skalierung, Mauszeiger bei gemischten Skalen, Bilder in der Zwischenablage). Ein Teil bis an
den Bildrand wird abgeschnitten und kommt mit *Bild → Ganzes Bild* zurück.

## Geplante Funktionen

- Stempel (fertige Zeichen wie Haken und Warnhinweise).
- Andere Desktops als KDE Plasma, über das Desktop-Portal.
- Export als PDF.
- Ein Flatpak-Paket.

## Bauen

Gio braucht cgo (`libwayland-dev`, `libegl-dev`, `libxkbcommon-dev` oder das Entsprechende deiner Distribution) und die Build-Tags `novulkan,nox11`; sie werden über `GOFLAGS` übergeben (die Go-Umgebung bleibt unverändert):

```sh
export GOFLAGS=-tags=novulkan,nox11
go build -trimpath -o schnapp ./cmd/schnapp
```

Vorhanden ist:

| Pfad | Was es ist | Prüfen |
|---|---|---|
| `.` (Wurzel) | Go-Modul `github.com/Andiministrator/schnapp`: das Programm `cmd/schnapp` und seine Pakete unter `internal/`, darunter `capture` (Aufnahme über KWin), `selection` (Logik des Overlays), `canvas` (Logik des Editors), `document` (Ebenen, Zuschnitt, Schwärzen, Rückgängig), `fileformat` (Arbeitsdatei und Exporte), `render` (Ebenen zu Pixeln), `library` (Bibliothek), `settings` (`settings.ini`), `ui` (Overlay und Editor in Gio), `wlclip` (hält ein kopiertes Bild nach dem Schließen des Editors in der Zwischenablage), `portal` (Dateidialog des Desktop-Portals), `export` (Export aus dem Editor), `i18n` (Sprachwahl, gettext-PO-Leser) und `po` (eingebettete Übersetzungen, `internal/po/de.po`; Vorlage `schnapp.pot`); `data/` enthält die Vorlage der Desktop-Datei | `go vet ./... && go test ./...` (mit `GOFLAGS` wie oben) |
| `draw/` | Go-Modul `github.com/Andiministrator/schnapp/draw`: SVG lesen, Rasterer mit Kantenglättung, Rechenhilfen (nur Standardbibliothek) | `cd draw && go vet ./... && go test ./...` |
| `brand/` | Logo, Farben und Corporate Design ([`brand/corporate-design.md`](brand/corporate-design.md), mit PDF) | `cd brand/werkzeug && go run . paket` erzeugt die Logo-Dateien neu |
| `third_party/gio/` | Gio mit unserem Patch (Vollbild auf gewähltem Monitor, gebrochene Skalierung, Mauszeiger bei gemischten Skalen, Bilder in der Zwischenablage), eingebunden per `replace` in `go.mod` | `third_party/gio-patch.sh pruefen`, siehe [`third_party/README.md`](third_party/README.md) |

## Benutzung

| Aufruf | Was passiert |
|---|---|
| `schnapp` oder `schnapp --region` | Bereich in einem eingefrorenen Vollbild-Overlay wählen (Voreinstellung, **Druck**) |
| `schnapp --screen[=NAME]` | einen Monitor aufnehmen (ohne Namen laut Einstellung, `[capture] screen = all` setzt alle Monitore zu einem Bild zusammen; `--list-screens` nennt Namen, Größen und Skalierung) |
| `schnapp --window` | ein Fenster wählen |
| `schnapp --repeat` | den letzten Bereich noch einmal aufnehmen |
| `schnapp --no-editor …` | aufnehmen, ohne danach den Editor zu öffnen |
| `schnapp DATEI` | ein PNG oder JPEG im Editor öffnen (das startet auch „Öffnen mit“ im Dateimanager; ein JPEG mit EXIF-Ausrichtung wird gedreht oder gespiegelt, wie Bildbetrachter es zeigen, die EXIF-Angabe selbst bleibt nicht erhalten) |
| `schnapp --ignore-limit DATEI` | die Datei auch öffnen, wenn ihre Beschriftungen so zahlreich oder groß sind, dass das Rendern das Rechenbudget überschreitet (für eigene Arbeit, die darüber hinausgewachsen ist) |
| `schnapp … --output=bild.png` | die Aufnahme nach `bild.png` (oder `.jpg`, `.svg`) schreiben statt in die Bibliothek, ohne Editor |
| `schnapp --export=aus.png DATEI` | DATEI so exportieren, wie der Editor sie zeigt, ohne Fenster |
| `schnapp --annotate=liste.json DATEI` | Pfeile, Kästen, Text, Schritt-Nummern und Schwärzungen aus einer JSON-Liste auf DATEI legen und wie der Editor speichern, ohne Fenster (siehe [`docs/agents.md`](docs/agents.md), englisch) |
| `schnapp --list-screens` | die Monitore mit Name, Größe und Skalierung auflisten |
| `schnapp mcp` | als MCP-Server für KI-Agenten laufen (aus, bis unter `[agents]` erlaubt, siehe [`docs/agents.md`](docs/agents.md)) |
| `schnapp … --json` | das Ergebnis (Pfad und Größe oder die Monitorliste) als eine Zeile JSON ausgeben |
| `schnapp --version` | die Version ausgeben |

Ohne `--output` landet jede Aufnahme in der Bibliothek (`~/Bilder/schnapp/JJJJ-MM/`), ihr Pfad steht auf der Standardausgabe, bevor
der Editor öffnet. Skripte und KI-Agenten: siehe [`docs/agents.md`](docs/agents.md) (englisch), auch zum Beschriften und Schwärzen einer Datei ohne Fenster (`--annotate`) und zum MCP-Server `schnapp mcp` (aus, bis du ihn unter `[agents]` erlaubst). Einstellungen stehen in `~/.config/schnapp/settings.ini`, etwa `[capture] open-editor = false`, `[capture] seam = wave` (Kante an Stauch-Nähten: `tear`, `wave` oder `zigzag`) oder
`[general] language = en` (Voreinstellung `auto`: die Sprache des Systems). Die Schnellstile, die du im Editor selbst speicherst (das „+“ in der rechten Leiste), liegen daneben in `~/.config/schnapp/styles.ini`; Schnapp schreibt diese Datei jeweils ganz neu, bearbeite Stile also im Editor, nicht von Hand. Dieselben Einstellungen ändert auch der Editor: *Datei → Einstellungen…* (**Strg+,**) schreibt nur die geänderten Schlüssel und lässt deine Kommentare stehen; der Reiter *Agenten* schreibt die Datei, die `schnapp mcp` liest.

Im Editor, Werkzeuge: **V** Auswählen, **A** Pfeil (ziehen; **Umschalt** rastet auf 45°; mittleren Griff ziehen
biegt ihn; *Pfeil-Stil* in der rechten Leiste wählt die Spitze am Anfang und am Ende aus einer Liste mit Vorschau — gefüllt, schlank, eingekerbt, offen, halb, Punkt, Raute, Quadrat, Querstrich oder *keine*; keine an beiden Enden ergibt eine Linie), **R** Kasten (**Umschalt**: Quadrat), **T** Text (klicken, tippen, **Enter** bestätigt, **Umschalt+Enter**
neue Zeile, **Esc** verwirft; Doppelklick auf einen Text bearbeitet ihn), **K** Sprechblase (von der Stelle, auf die sie zeigt, dorthin ziehen, wo der Text stehen soll, dann tippen; ein Klick setzt eine; der Griff an der Spitze verschiebt nur sie; der Schalter „Spitze“ in der rechten Leiste macht jeden Text zur Sprechblase und zurück), **N** Schritt-Nummer (zählt selbst hoch), **M** Lupe (über eine Stelle ziehen: daneben erscheint sie vergrößert, mit einer Linie verbunden; Quelle und Vergrößerung lassen sich getrennt verschieben, die rechte Leiste stellt 2×, 3× oder 4× ein; sie zeigt die geschwärzten Pixel und nichts Weggeschnittenes), **B** Schwärzen (über einen Bereich ziehen: schwarz),
**P** Verpixeln (ziehen; ein Mosaik nur aus der Durchschnittsfarbe des Bereichs und Zufall, es trägt nichts vom
Inhalt). Beides überschreibt die Pixel des Bildes selbst: Bis zum Speichern lässt es sich zurücknehmen (der Rückgängig-Verlauf hält höchstens 512 MiB Pixelkopien, ältere Schritte fallen still weg), danach ist es
endgültig, und keine gespeicherte oder exportierte Datei enthält den verdeckten Inhalt. Ein vor dem Schwärzen mit
**Strg+C** kopiertes Bild bleibt, wie es war, in der Zwischenablage. **C** Zuschneiden (das ganze Bild erscheint mit einem Rahmen;
Kanten oder Ecken nach innen ziehen oder einen neuen Rahmen ziehen — der Teil außerhalb bleibt abgedunkelt sichtbar,
eine Kante lässt sich also wieder hinausschieben; *Bild → Ganzes Bild* oder ein Doppelklick mit dem Werkzeug zeigt
wieder das ganze Bild), **X** Stauchen (quer über den Teil ziehen, der
wegfallen soll — senkrecht für Zeilen, waagrecht für Spalten; die Naht zeigt einen durchsichtigen Spalt zwischen zwei Papier-Abrisskanten (in JPEG weiß) — **Kantenform** im Menü *Bild* (sobald es eine Naht gibt) wechselt zu Welle oder Zacken, `[capture] seam` legt die Form neuer Aufnahmen fest; Doppelklick auf eine
Naht holt den Teil zurück (jede Naht verdeckt auf beiden Seiten rund 14 Pixel Bild, bei sehr großen Aufnahmen entsprechend mehr, und eine Naht näher als rund 24 Pixel am Rand wird abgeschnitten); ein Teil bis an den Bildrand wird stattdessen abgeschnitten und kommt mit *Ganzes Bild*
zurück).
Beides verbirgt nur: Das Original bleibt in der Arbeitsdatei vollständig, Geheimes also vor dem Wegschneiden
schwärzen — Exporte und Zwischenablage enthalten nur den sichtbaren Teil. Beschriftungen, die ganz in einem
verborgenen Teil liegen, sind mit verborgen (die Fußzeile nennt ihre Zahl) und kehren mit ihm zurück.
Eine der vier Farben wählst du in der rechten Leiste oder in der Liste des Farbfelds der Fußzeile bzw. unter den Werkzeugen; sie färbt die ausgewählte Beschriftung um. Die Leiste nimmt auch jede andere Farbe: in der Farbfläche und an den Reglern für Farbton und Deckkraft ziehen, einen Hex-Wert (`#RRGGBB`; `#RRGGBBAA` setzt auch die Deckkraft) oder die Deckkraft in Prozent eintippen, eine der vier zuletzt benutzten Farben nehmen oder mit der Pipette (Knopf rechts in der Farbzeile) eine Farbe aus dem Bild aufnehmen (sie liest das Bildschirmfoto, nicht die Beschriftungen). Jede Beschriftung hat zwei Farben, wie in Bildprogrammen: den **Vordergrund** (Linie von Pfeil und Kasten, Schrift des Texts, Kreis der Schritt-Nummer) und die **Füllung** (Fläche des Kastens, Schild hinter dem Text; *Keine* lässt sie leer). Die beiden Felder unter den Werkzeugen und die Zeile über der Farbfläche zeigen beide; ein Klick macht ein Feld zu dem, das die Farbwahl ändert (ein Klick auf ein Feld unter den Werkzeugen klappt außerdem den Abschnitt Farbe der Leiste auf), **⇄** (*Bearbeiten → Farben tauschen*, **Strg+Umschalt+X**) tauscht sie. Jedes Zeichenwerkzeug merkt sich sein eigenes Paar, solange das Fenster offen ist: neuer Text beginnt weiß auf Rot, die anderen Werkzeuge rot ohne Füllung. Schwärzen bleibt immer deckend. Für Text wählt die Leiste die Schrift (die eingebauten Go und Go Mono oder eine in den üblichen Schriftordnern installierte mit normalem, fettem oder kursivem Schnitt), die Größe, fett, kursiv, unterstrichen, durchgestrichen und die Ausrichtung der Zeilen; eine Datei mit einer Schrift, die auf einem anderen Rechner fehlt, zeigt den Text dort in Go und behält den Namen. Auf großen Bildschirmfotos (ab 4K) beginnen Striche, Text und Schritt-Nummern dicker und größer. Ein Zeichenwerkzeug bleibt
aktiv; **Esc** geht stufenweise zurück: Texteingabe → Ziehen → Auswahl → Auswählen. Beschriftung oder Griffe ziehen
verschiebt, ändert die Größe oder biegt; sie bleibt dabei immer teilweise im Bild. Beim Ziehen (auch beim Aufziehen eines neuen Pfeils, Kastens oder einer Nummer) rasten Kanten und Mitten an den Kanten und Mitten anderer Beschriftungen und des Bilds ein, gezeigt als dünne orange Hilfslinien; mit gedrückter **Alt**-Taste rastet beim Ziehen nichts ein, ganz ausschalten geht über *Ansicht → An Objekten einrasten*. Zoomst du so weit hinein, dass das Bild nicht mehr ins Fenster passt, erscheint unten rechts auf der Leinwand ein kleiner **Navigator**: das ganze Bild in Miniatur mit einem orangen Rahmen für den sichtbaren Teil. Ein Klick darauf springt an die Stelle, Ziehen verschiebt die Ansicht, das Mausrad darüber zoomt; *Ansicht → Navigator* blendet ihn aus. Das Mausrad zoomt zum Zeiger; mit
Auswählen verschiebt Ziehen auf freier Fläche die Ansicht, die mittlere Taste in jedem Werkzeug. Tasten: **Strg+O** (*Datei → Öffnen …*) öffnet ein anderes PNG oder JPEG über den Dateidialog im selben Fenster (das offene Bildschirmfoto wird vorher gespeichert; eine Datei von außerhalb der Bibliothek wird mit ihrer ersten Änderung hineinkopiert), **Strg+Z**
rückgängig, **Strg+Umschalt+Z**/**Strg+Y** wiederholen, **Entf** löschen, **Strg+S** speichern,
**Strg+Umschalt+E** (*Datei → Exportieren …*) als PNG, JPG oder SVG über den Dateidialog des Desktops weitergeben (ohne Bearbeitungsdaten, Geschwärztes bleibt weg; scheitert das Ersetzen einer vorhandenen Datei auf halbem Weg und lässt sich der alte Inhalt nicht wiederherstellen, nennt die Fußzeile die Datei als beschädigt),
**Strg+C** fertiges Bild in die Zwischenablage (mit dem data-control-Protokoll, etwa unter KDE Plasma, bleibt
es nach dem Schließen des Editors, bis etwas anderes kopiert wird, höchstens 12 Stunden; sonst nur, solange der Editor offen ist; das Schließen hält dann einmal an und warnt, dass die Zwischenablage das Bild nur hält, solange das Fenster offen ist), **Strg+0** einpassen,
**Strg+1** 100 %, **Strg+W** schließen. Jedes Kürzel geht auch mit der Maus (nur Umschalt oder Alt beim Ziehen, für Pfeile in 45°-Schritten und Quadrate bzw. zum Ausschalten des Einrastens, und Umschalt+Enter für den Zeilenumbruch im Text bleiben Tasten). Das Editorfenster hat oben eine **Menüzeile** (das „s“ des Logos, dann *Datei, Bearbeiten, Ansicht, Bild, Werkzeuge, Ebene, Hilfe*; rechts steht der Dateiname), links eine **Werkzeugleiste** mit den elf Werkzeugen als Symbolen und darunter dem Farbfeld (in niedrigen Fenstern werden die Knöpfe kleiner), rechts eine **Seitenleiste** (*Ansicht → Seitenleiste*, **Strg+Umschalt+B**; in Fenstern unter etwa 750 px Breite bleibt sie eingeklappt, in niedrigen Fenstern rollt sie mit dem Mausrad) mit dem Werkzeug oder der ausgewählten Beschriftung, ihrer Farbe, der Strichstärke von Pfeil und Kasten (− / +) und beim Stauchen der Kantenform, und unten eine **Fußzeile**: links die Meldung oder der Hinweis zum Werkzeug, rechts Felder für die Größe der Auswahl, den Zeigerort im Bild, die Farbe mit Deckkraft (Klick: Liste der Farben), die Schriftgröße, die Bildgröße (mit „gestaucht“, wenn Teile gestaucht sind), den Zoom (Klick: Liste mit Einpassen und 25 bis 400 %) und „gespeichert“ oder „ungesichert“. Die Menüs führen alle Aktionen mit ihren Kürzeln (darunter **Gestauchtes zurückholen**, das alles Gestauchte wieder zeigt); über einem Werkzeugknopf erscheinen Name und Buchstabe. **F1** (*Hilfe → Tastenkürzel*) zeigt alle Kürzel. Die Menüs gehen auch ohne Maus: **F10** öffnet das erste Menü, mit **Tab** erreichst du die Menütitel, **Enter** oder **Leertaste** öffnet eines, **Pfeil links**/**rechts** wechselt das Menü, **Pfeil hoch**/**runter**, **Pos1** und **Ende** wählen einen Eintrag, **Enter** führt ihn aus, **Esc** schließt. Die Listen der Fußzeile gehen genauso: mit **Tab** zum Feld, **Enter** öffnet die Liste.

**Ebenen**: Jede Beschriftung ist eine eigene Ebene. Der Abschnitt *Ebenen* unten in der rechten Leiste listet sie, die oberste zuerst, je mit kleiner Vorschau, Name, Auge (zeigen/ausblenden) und Schloss; über der Liste die Deckkraft der gewählten Ebene (0–100 %). Eine Zeile ziehen ändert die Reihenfolge, Doppelklick oder **F2** benennt um; die Knöpfe darunter schieben nach oben oder unten, doppeln oder löschen. Das Menü *Ebene* enthält diese Aktionen mit Tasten (Löschen steht unter *Bearbeiten*, **Entf**): **Strg+D** doppeln, **F2** umbenennen, **Strg+H** ausblenden/zeigen, **Strg+L** sperren/entsperren, **Strg+Bild↑**/**Strg+Bild↓** nach vorn/nach hinten, **Alt+Bild↑**/**Alt+Bild↓** die Ebene darüber/darunter wählen. Ausgeblendete Ebenen und Ebenen mit 0 % fehlen in Exporten und in der Zwischenablage, bleiben aber in der Arbeitsdatei — Ausblenden ist kein Schwärzen. Gesperrte Ebenen lassen sich auf der Leinwand weder wählen noch verschieben.

**Schnellstile**: Der Abschnitt *Schnellstile* der rechten Leiste zeigt fertige Ansichten für das aktuelle Werkzeug (oder die gewählte Beschriftung) als kleine Kacheln; die passende ist orange umrandet. Ein Klick auf eine Kachel (oder **Tab** dorthin, Pfeiltasten, **Leertaste** oder **Enter**) wendet sie an. **+** speichert die aktuelle Ansicht als eigenen Stil (höchstens 16 je Art, in `styles.ini`), Rechtsklick oder **Entf** entfernt einen eigenen.

**Text aus dem Bild kopieren** (Menü *Bearbeiten*, **Strg+Umschalt+T**) erkennt Text auf dem Bildschirmfoto und legt ihn in die Zwischenablage: über den Text ziehen (**Enter** nimmt das ganze sichtbare Bild, **Esc** bricht ab); die Erkennung läuft im Hintergrund, die Fußzeile sagt, wie viele Zeichen kopiert wurden. Gelesen wird das Bild, wie es exportiert würde (mit Schwärzungen, nie das Original), und zwar lokal mit [Tesseract](https://github.com/tesseract-ocr/tesseract). Ist das Paket `tesseract` installiert (mit den nötigen Sprachdaten, etwa `tesseract-data-deu`/`tesseract-ocr-deu` und `…-eng`), nimmt Schnapp es mit der Sprache der Oberfläche und Englisch, soweit installiert. Fehlt es oder hat es keine Sprachdaten, springt das eigene Hilfsprogramm `schnapp-ocr` ein (legt `install.sh` neben `schnapp`): dieselbe Erkennung als WebAssembly mit eingebautem Deutsch und Englisch, eine Sprache je Lauf, mehrfach langsamer; die Fußzeile zeigt den Fortschritt.

**Letzte Aufnahmen** (*Ansicht → Letzte Aufnahmen*, **Strg+Umschalt+R**): Eine Leiste unter dem Bild zeigt die neuesten Bildschirmfotos der Bibliothek, das neueste links, mit Uhrzeit (heute), „gestern“ oder Wochentag und Datum (aus einem anderen Jahr mit Jahr) und der Größe in Pixeln; das offene ist grün umrandet. Fährt der Zeiger über eine Vorschau (oder nach langem Druck), erscheint oben rechts ein **✕**, das dieses Bildschirmfoto in den Papierkorb legt (von dort wiederherstellbar; beim offenen wird es erst gespeichert, dann öffnet sich das nächst ältere — beim ältesten das nächst jüngere — und erst danach kommt das bisher offene in den Papierkorb; mit Strg+C Kopiertes bleibt in der Zwischenablage; das einzige Bildschirmfoto der Bibliothek bleibt); **Entf** an einer Vorschau mit Fokus tut dasselbe. Ein Klick auf eine Vorschau (oder **Tab** bis dahin, **Links**/**Rechts**, **Pos1**, **Ende**, dann **Enter** oder **Leertaste**) öffnet sie im selben Fenster: Das offene Bildschirmfoto wird vorher gespeichert wie mit **Strg+S**; scheitert das, bleibt es offen, und die Fußzeile sagt warum. Das Mausrad rollt die Leiste. Aufnahmen von KI-Agenten stehen dort nicht. In niedrigen Fenstern blendet sich die Leiste aus; ob sie gezeigt wird, merkt sich `[library] bar = yes|no`.

**Bibliothek** (*Datei → Bibliothek*, **Strg+Umschalt+L** oder der Bibliotheksknopf links in der Leiste): die ganze Bibliothek im Editorfenster statt des Bilds und seiner Leisten; Menüzeile und Fußzeile bleiben. Die Bildschirmfotos stehen nach Monaten, das neueste zuerst, je mit Vorschau, Datum und Uhrzeit, Größe und Schlagworten; das offene ist grün umrandet, der Tastaturfokus orange. Oben: Suche nach einem Schlagwort (**Strg+F**; ein Teil des Worts genügt), der Zeitraum (heute, letzte 7 oder 30 Tage, dieses Jahr, alle), ein Schalter, der Aufnahmen von KI-Agenten zeigt (bei jedem Öffnen aus), und **Zurück zum Bild**. Die **Pfeiltasten** wandern zwischen den Bildschirmfotos, **Pos1**/**Ende** springen, **Bild↑**/**Bild↓** blättern, das Mausrad rollt. **Enter** oder ein Doppelklick öffnet ein Bildschirmfoto im selben Fenster (das offene wird vorher gespeichert, wie bei der Leiste), **Entf** oder das **✕** an der Vorschau unter dem Zeiger legt es in den Papierkorb (von dort wiederherstellbar; das offene wie in der Leiste, die Ansicht bleibt offen), **F2** bearbeitet seine Schlagworte unter der Vorschau (durch Komma getrennt, **Enter** speichert, **Esc** bricht ab; nicht beim offenen), **Esc** führt zurück zum Bild. Solange die Bibliothek offen ist, wirken Werkzeugtasten und die übrigen Kürzel für das Bild nicht; das Menü *Bearbeiten* führt dann *Öffnen*, *Schlagworte bearbeiten*, *In den Papierkorb* und *Nach Schlagwort suchen*.

Änderungen werden beim Schließen gespeichert; **Strg+S** ohne Änderung schreibt nichts. Gespeichert wird im Hintergrund: Das Fenster reagiert weiter und zeigt „Speichern …“, das Bearbeiten ruht, bis es fertig ist. Eine Datei außerhalb der
Bibliothek (`schnapp DATEI`, *Öffnen mit*, *Datei → Öffnen…*) öffnet sich, ohne kopiert zu werden; die Fußzeile sagt,
dass sie nicht in der Bibliothek liegt. Mit der ersten Änderung speichert der Editor sie als neue Aufnahme in die
Bibliothek und arbeitet danach an dieser Kopie: Sie steht links in den letzten Aufnahmen, die Fußzeile nennt sie, die
Datei selbst bleibt unverändert (wer in der Kopie schwärzt, wird beim Speichern gewarnt, dass das Original den Inhalt
noch zeigt). Eine Datei, die man nur ansieht, kommt nie in die Bibliothek. Dieselbe unveränderte Datei noch einmal
geöffnet, öffnet diese Aufnahme samt Beschriftungen. Lässt sich eine Datei der Bibliothek nicht
aktualisieren (gelöscht, ersetzt oder nicht beschreibbar), landen die Änderungen als neue Aufnahme. Scheitert das
Speichern, bleibt das Fenster offen und sagt es; erneutes Schließen verwirft die Änderungen. Strg+C im Terminal oder
Abmelden speichert vorher. Eine Datei, die zu aufwendig zu rendern ist, wird beim Öffnen abgelehnt; `--ignore-limit
DATEI` öffnet sie trotzdem.

### Rettungskopie

Während du bearbeitest, legt Schnapp eine kleine *Rettungskopie* (englisch „rescue copy“) deiner ungespeicherten Arbeit an, damit ein Absturz, ein beendeter Prozess oder ein Stromausfall sie nicht vernichtet. Sie enthält deine Beschriftungen mit ihren Texten, den Zuschnitt, die Stauchstreifen und die Schwärzungen seit dem letzten Speichern, aber keine Pixel. Sie liegt in `~/.local/state/schnapp/rescue/` (`$XDG_STATE_HOME/schnapp/rescue/`, falls gesetzt), in einem Ordner, den nur du lesen kannst (Rechte 0700, Dateien 0600), je geöffneter Datei eine, benannt nach dem SHA-256 des absoluten Pfads der bearbeiteten Datei. Die Kopie selbst hält diesen Pfad samt Größe, Änderungszeit und SHA-256 der Datei fest; daran erkennt der Editor, dass sie noch passt. Nach einem gelungenen Speichern und beim Schließen ohne Änderungen wird sie gelöscht. Öffnest du dieselbe, unveränderte Datei später wieder, stellt der Editor sie wieder her und sagt es in der Fußzeile; Rückgängig führt auf den gespeicherten Stand zurück. Hat sich die Datei inzwischen geändert, wird die Kopie nicht angewendet, sondern als `….alt.json` im selben Ordner beiseitegelegt, und die Fußzeile nennt sie. Jede Kopie, deren eigene Änderungszeit mehr als 30 Tage zurückliegt, wird beim Start des Editors entfernt, auch eine beiseitegelegte oder nie angewendete. Ein zweites Fenster auf derselben Datei schreibt keine Rettungskopie und sagt es; eine Sperrdatei `<SHA>.lock` im selben Ordner trennt die Fenster. Während die Bibliotheksansicht die Schlagworte einer Aufnahme ändert, liegt daneben kurz eine Marke `<SHA>.um`, damit ein Fenster, das diese Aufnahme öffnet, einen Moment wartet und die Kopie danach wiederherstellt; eine Marke, die ein Absturz hinterlassen hat, hält niemanden auf und wird beim Aufräumen zum Start des Editors entfernt, sobald sie älter als eine Minute ist.

## Installation

`./install.sh` baut Schnapp (gleiche Voraussetzungen wie unter „Bauen“; die Version kommt aus `git describe`, ohne erreichbaren Tag lautet sie `0.0.0-dev+<Hash>`, bei geänderten Dateien `0.0.0-dev+<Hash>.dirty`) und installiert es für den aktuellen Benutzer:

| Datei | Ort |
|---|---|
| Programm | `$PREFIX/bin/schnapp` (Vorgabe `~/.local/bin/schnapp`) |
| Texterkennung | `$PREFIX/bin/schnapp-ocr` (springt ein, wenn `tesseract` fehlt) |
| Desktop-Datei | `$XDG_DATA_HOME/applications/schnapp.desktop` (Vorgabe `~/.local/share/applications/`) |
| Symbol | `$XDG_DATA_HOME/icons/hicolor/scalable/apps/schnapp.svg` |

Die Desktop-Datei nennt PNG und JPEG, damit Dateimanager Schnapp unter „Öffnen mit“ anbieten. Sie trägt den absoluten Pfad des Programms (KDE Plasma 6.7 und älter binden die Aufnahme-Berechtigung daran) und das Standardkürzel **Druck** für „Bereich aufnehmen“. Spectacle belegt **Druck** ebenfalls: Sind beide installiert, schalte das Kürzel von Spectacle in den Systemeinstellungen (Kurzbefehle) ab. `~/.local/bin` muss in deinem `PATH` liegen.

Desktop-Werkzeuge aktualisieren dabei gegebenenfalls ihre Caches (Menü-Datenbank; den Symbol-Cache nur, wenn es schon einen gibt). Entfernen mit `./install.sh --uninstall` (löscht nur diese vier Dateien). Scheitert der Bau von `schnapp-ocr` (etwa ohne Netz für seine Go-Module), wird Schnapp trotzdem installiert und das Skript warnt. `./install.sh --help` listet die Optionen.

## Dokumentation

Die Dokumente für Nutzer und Beitragende (diese README, `CONTRIBUTING.de.md`) gibt es auf Englisch und Deutsch. Die
Arbeitsdokumente des Projekts — Anforderungen, Architektur-Entscheidungen (`docs/adr/`), QA-Listen, Befunde,
Recherchen (`knowledge/`) und das Corporate Design (`brand/`) — sind auf Deutsch.

Das Format der Arbeitsdatei (PNG mit den privaten Blöcken `snAP` und `snAO`) ist auf Englisch in [`docs/format.md`](docs/format.md) spezifiziert, mit Beispieldateien in `testdata/format/`.

## Hinweis zu KI

Der größte Teil von Code und Doku in diesem Repository wird von einem KI-Modell (Anthropic Claude) unter Anleitung und
Prüfung des Maintainers geschrieben. Entscheidungen stehen in `docs/adr/`. Siehe [`CONTRIBUTING.de.md`](CONTRIBUTING.de.md).

## Lizenz

GPL-3.0-or-later, siehe [`LICENSE`](LICENSE). Das Logo verwendet Umrisse von Open Sans (SIL Open Font License 1.1). Die in `schnapp` eingebauten Go-Module stehen mit ihren Lizenzen in [`THIRD-PARTY-LICENSES/`](THIRD-PARTY-LICENSES/README.md). Fremder Code behält seine Lizenz: Der Gio-Patch
(`third_party/gio-ausgabe.patch`, angewendet in `third_party/gio/`) gilt für Gio (Unlicense OR MIT) und enthält aus wayland-protocols erzeugten Code (MIT); `internal/raster16` stammt aus golang.org/x/image/vector (BSD-3-Clause, siehe `internal/raster16/LICENSE`). Das Hilfsprogramm zur Texterkennung `schnapp-ocr` (Modul `ocr/`) enthält Tesseract, Leptonica
und weitere Fremdteile, aufgeführt mit ihren Lizenzen in [`ocr/THIRD-PARTY-LICENSES/`](ocr/THIRD-PARTY-LICENSES/README.md);
this software is based in part on the work of the Independent JPEG Group (Vermerk nach der IJG-Lizenz).
