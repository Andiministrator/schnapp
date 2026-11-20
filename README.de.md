<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="brand/logo/svg/schnapp-logo-dunkel.svg">
    <img alt="schnapp — screenshots for Linux" src="brand/logo/svg/schnapp-logo-hell.svg" width="520">
  </picture>
</p>

# Schnapp

*[English](README.md) · Deutsch*

Screenshot- und Beschriftungswerkzeug für Linux/Wayland — KDE Plasma zuerst. Vorbild ist die Arbeitsweise von Snagit:
schnell aufnehmen, beschriften (gerade und gebogene Pfeile, Kästen, Schritt-Nummern, Stempel), wirklich schwärzen, die
Mitte langer Seiten herausschneiden, teilen. Nebenbei ein einfaches Bildbearbeitungsprogramm.

## Stand

Frühe Arbeit. Schnapp nimmt unter KDE Plasma schon auf (Bereich, Monitor, Fenster; Taste **Druck**), legt jeden
Screenshot in der Bibliothek ab und öffnet ihn in einem Editor, in dem sich Pfeile (gerade oder gebogen), Kästen, Text
und Schritt-Nummern zeichnen sowie auswählen, verschieben und in der Größe ändern lassen, Bereiche endgültig schwärzen
oder verpixeln, zuschneiden, die Mitte eines langen Screenshots herausnehmen und das Ergebnis in die Zwischenablage
kopieren.
Schnapp entsteht in Go mit [Gio](https://gioui.org), dazu ein kleiner Patch an Gio (ein Vollbild-Overlay je Monitor,
gebrochene Skalierung, Mauszeiger bei gemischten Skalen, Bilder in der Zwischenablage). Die erste Fassung (0.1.0) ist in Arbeit.

## Geplante Funktionen

- Bereich, Fenster oder Bildschirm unter KDE Plasma (KWin) aufnehmen, mit eingefrorenem Vollbild-Overlay auf jedem
  Monitor.
- Beschriften mit Vektorebenen: Pfeile (auch gebogen), Kästen, Text, Schritt-Nummern, Stempel.
- **Wirklich** schwärzen und verpixeln — geschwärzte Pixel fehlen in jeder gespeicherten oder exportierten Datei.
- Lange Screenshots stauchen, indem die Mitte herausgeschnitten wird, ohne das Original zu zerstören.
- Bibliothek mit automatischem Speichern, Suche und Schlagworten.
- Oberfläche auf Englisch und Deutsch; sie folgt der Systemsprache, sonst Englisch.

## Bauen

Gio braucht cgo (`libwayland-dev`, `libegl-dev`, `libxkbcommon-dev` oder das Entsprechende deiner Distribution) und die Build-Tags `novulkan,nox11`; sie werden über `GOFLAGS` übergeben (die Go-Umgebung bleibt unverändert):

```sh
export GOFLAGS=-tags=novulkan,nox11
go build -trimpath -o schnapp ./cmd/schnapp
```

Vorhanden ist:

| Pfad | Was es ist | Prüfen |
|---|---|---|
| `.` (Wurzel) | Go-Modul `github.com/Andiministrator/schnapp`: das Programm `cmd/schnapp`, `capture` (Aufnahme über KWin), `selection` (Logik des Overlays), `canvas` (Logik des Editors), `document` (Ebenen, Zuschnitt, Schwärzen, Rückgängig), `fileformat` (Arbeitsdatei und Exporte), `render` (Ebenen zu Pixeln), `library` (Bibliothek), `settings` (`settings.ini`), `ui` (Overlay und Editor in Gio), `i18n` (Sprachwahl, gettext-PO-Leser) und `po` (eingebettete Übersetzungen, `po/de.po`); `data/` enthält die Vorlage der Desktop-Datei | `go vet ./... && go test ./...` (mit `GOFLAGS` wie oben) |
| `draw/` | Go-Modul `github.com/Andiministrator/schnapp/draw`: SVG lesen, Rasterer mit Kantenglättung, Rechenhilfen (nur Standardbibliothek) | `cd draw && go vet ./... && go test ./...` |
| `brand/` | Logo, Farben und Corporate Design ([`brand/corporate-design.md`](brand/corporate-design.md), mit PDF) | `python3 brand/werkzeug/paket.py` erzeugt die Logo-Dateien neu |
| `third_party/gio/` | Gio mit unserem Patch (Vollbild auf gewähltem Monitor, gebrochene Skalierung, Mauszeiger bei gemischten Skalen, Bilder in der Zwischenablage), eingebunden per `replace` in `go.mod` | `third_party/gio-patch.sh pruefen`, siehe [`third_party/README.md`](third_party/README.md) |
| `prototype/` | Wegwerf-Prototyp, der Gio geprüft hat (braucht Go ≥ 1.27, cgo, `libwayland-client`, `libwayland-egl`, `libwayland-cursor`, `libEGL`, `libxkbcommon`) | `prototype/bauen.sh`, siehe [`prototype/README.md`](prototype/README.md) |

## Benutzung

| Aufruf | Was passiert |
|---|---|
| `schnapp` oder `schnapp --region` | Bereich in einem eingefrorenen Vollbild-Overlay wählen (Voreinstellung, **Druck**) |
| `schnapp --screen[=NAME]` | einen Monitor aufnehmen (ohne Namen laut Einstellung; `--list-screens` nennt die Namen) |
| `schnapp --window` | ein Fenster wählen |
| `schnapp --repeat` | den letzten Bereich noch einmal aufnehmen |
| `schnapp --no-editor …` | aufnehmen, ohne danach den Editor zu öffnen |
| `schnapp DATEI` | ein PNG oder JPEG im Editor öffnen |
| `schnapp --ignore-limit DATEI` | die Datei auch öffnen, wenn ihre Beschriftungen so zahlreich oder groß sind, dass das Rendern das Rechenbudget überschreitet (für eigene Arbeit, die darüber hinausgewachsen ist) |

Jede Aufnahme landet in der Bibliothek (`~/Bilder/schnapp/JJJJ-MM/`), ihr Pfad steht auf der Standardausgabe, bevor
der Editor öffnet. Einstellungen stehen in `~/.config/schnapp/settings.ini`, etwa `[capture] open-editor = false`.

Im Editor, Werkzeuge: **V** Auswählen, **A** Pfeil (ziehen; **Umschalt** rastet auf 45°; mittleren Griff ziehen
biegt ihn), **R** Kasten (**Umschalt**: Quadrat), **T** Text (klicken, tippen, **Enter** bestätigt, **Umschalt+Enter**
neue Zeile, **Esc** verwirft; Doppelklick auf einen Text bearbeitet ihn), **N** Schritt-Nummer (zählt selbst hoch), **B** Schwärzen (über einen Bereich ziehen: schwarz),
**P** Verpixeln (ziehen; ein Mosaik nur aus der Durchschnittsfarbe des Bereichs und Zufall, es trägt nichts vom
Inhalt). Beides überschreibt die Pixel des Bildes selbst: Bis zum Speichern lässt es sich zurücknehmen, danach ist es
endgültig, und keine gespeicherte oder exportierte Datei enthält den verdeckten Inhalt. Ein vor dem Schwärzen mit
**Strg+C** kopiertes Bild bleibt, wie es war, in der Zwischenablage. **C** Zuschneiden (Rahmen ziehen; Doppelklick mit
dem Werkzeug zeigt wieder das ganze Bild), **X** Stauchen (quer über den Teil ziehen, der wegfallen soll — senkrecht
für Zeilen, waagrecht für Spalten; die Naht zeigt eine Zackenkante; Doppelklick auf eine Naht holt den Teil zurück).
Beides verbirgt nur: Das Original bleibt in der Arbeitsdatei vollständig, Geheimes also vor dem Wegschneiden
schwärzen — Exporte und Zwischenablage enthalten nur den sichtbaren Teil. Beschriftungen, die ganz in einem
verborgenen Teil liegen, sind mit verborgen (die Statusleiste nennt ihre Zahl) und kehren mit ihm zurück. Der
Knopf *Farbe* schaltet reihum durch vier Farben und färbt die ausgewählte Beschriftung um. Ein Zeichenwerkzeug bleibt
aktiv; **Esc** geht stufenweise zurück: Texteingabe → Ziehen → Auswahl → Auswählen. Beschriftung oder Griffe ziehen
verschiebt, ändert die Größe oder biegt; sie bleibt dabei immer teilweise im Bild. Das Mausrad zoomt zum Zeiger; mit
Auswählen verschiebt Ziehen auf freier Fläche die Ansicht, die mittlere Taste in jedem Werkzeug. Tasten: **Strg+Z**
rückgängig, **Strg+Umschalt+Z**/**Strg+Y** wiederholen, **Entf** löschen, **Strg+S** speichern,
**Strg+C** fertiges Bild in die Zwischenablage (es bleibt nach dem Schließen des Editors, bis etwas anderes
kopiert wird, höchstens 12 Stunden), **Strg+0** einpassen,
**Strg+1** 100 %, **Strg+W** schließen.

Änderungen werden beim Schließen gespeichert; **Strg+S** ohne Änderung schreibt nichts. Eine Datei außerhalb der
Bibliothek wird beim ersten Speichern zu einer neuen Aufnahme der Bibliothek (die Datei selbst bleibt unverändert; die
Statusleiste nennt die neue Datei, in die danach gespeichert wird). Lässt sich eine Datei der Bibliothek nicht
aktualisieren (gelöscht, ersetzt oder nicht beschreibbar), landen die Änderungen als neue Aufnahme. Scheitert das
Speichern, bleibt das Fenster offen und sagt es; erneutes Schließen verwirft die Änderungen. Strg+C im Terminal oder
Abmelden speichert vorher. Eine Datei, die zu aufwendig zu rendern ist, wird beim Öffnen abgelehnt; `--ignore-limit
DATEI` öffnet sie trotzdem.

## Installation

`./install.sh` baut Schnapp (gleiche Voraussetzungen wie unter „Bauen“; die Version kommt aus `git describe`) und installiert es für den aktuellen Benutzer:

| Datei | Ort |
|---|---|
| Programm | `$PREFIX/bin/schnapp` (Vorgabe `~/.local/bin/schnapp`) |
| Desktop-Datei | `$XDG_DATA_HOME/applications/schnapp.desktop` (Vorgabe `~/.local/share/applications/`) |
| Symbol | `$XDG_DATA_HOME/icons/hicolor/scalable/apps/schnapp.svg` |

Die Desktop-Datei trägt den absoluten Pfad des Programms (KDE Plasma 6.7 und älter binden die Aufnahme-Berechtigung daran) und das Standardkürzel **Druck** für „Bereich aufnehmen“. Spectacle belegt **Druck** ebenfalls: Sind beide installiert, schalte das Kürzel von Spectacle in den Systemeinstellungen (Kurzbefehle) ab. `~/.local/bin` muss in deinem `PATH` liegen.

Desktop-Werkzeuge aktualisieren dabei gegebenenfalls ihre Caches (Menü-Datenbank; den Symbol-Cache nur, wenn es schon einen gibt). Entfernen mit `./install.sh --uninstall` (löscht nur diese drei Dateien). `./install.sh --help` listet die Optionen.

## Dokumentation

Die Dokumente für Nutzer und Beitragende (diese README, `CONTRIBUTING.de.md`) gibt es auf Englisch und Deutsch. Die
Arbeitsdokumente des Projekts — Anforderungen, Architektur-Entscheidungen (`docs/adr/`), QA-Listen, Befunde,
Recherchen (`knowledge/`) und das Corporate Design (`brand/`) — sind auf Deutsch.

## Hinweis zu KI

Der größte Teil von Code und Doku in diesem Repository wird von einem KI-Modell (Anthropic Claude) unter Anleitung und
Prüfung des Maintainers geschrieben. Entscheidungen stehen in `docs/adr/`. Siehe [`CONTRIBUTING.de.md`](CONTRIBUTING.de.md).

## Lizenz

GPL-3.0-or-later, siehe [`LICENSE`](LICENSE). Das Logo verwendet Umrisse von Open Sans (SIL Open Font License 1.1). Fremder Code behält seine Lizenz: Der Gio-Patch
(`third_party/gio-ausgabe.patch`, angewendet in `third_party/gio/`) gilt für Gio (Unlicense OR MIT) und enthält aus wayland-protocols erzeugten Code (MIT).
