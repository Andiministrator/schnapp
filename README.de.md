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

Frühe Arbeit — **noch nichts benutzbar**. Ein Prototyp hat die Oberflächen-Bibliothek bestätigt: Schnapp entsteht in Go
mit [Gio](https://gioui.org), dazu ein kleiner Patch an Gio (ein Vollbild-Overlay je Monitor, gebrochene Skalierung).
Die erste Fassung (0.1.0) wird geplant.

## Geplante Funktionen

- Bereich, Fenster oder Bildschirm unter KDE Plasma (KWin) aufnehmen, mit eingefrorenem Vollbild-Overlay auf jedem
  Monitor.
- Beschriften mit Vektorebenen: Pfeile (auch gebogen), Kästen, Text, Schritt-Nummern, Stempel.
- **Wirklich** schwärzen und verpixeln — geschwärzte Pixel fehlen in jeder gespeicherten oder exportierten Datei.
- Lange Screenshots stauchen, indem die Mitte herausgeschnitten wird, ohne das Original zu zerstören.
- Bibliothek mit automatischem Speichern, Suche und Schlagworten.
- Oberfläche auf Englisch und Deutsch; sie folgt der Systemsprache, sonst Englisch.

## Bauen

Bisher nimmt Schnapp unter KDE Plasma einen Bereich, einen Monitor oder ein Fenster auf (eingefrorenes Vollbild-Overlay mit Lupe) und legt den Screenshot in der Bibliothek ab; der Editor fehlt noch. Gio braucht cgo (`libwayland-dev`, `libegl-dev`, `libxkbcommon-dev` oder das Entsprechende deiner Distribution) und die Build-Tags `novulkan,nox11`; sie werden über `GOFLAGS` übergeben (die Go-Umgebung bleibt unverändert):

```sh
export GOFLAGS=-tags=novulkan,nox11
go build -trimpath -o schnapp ./cmd/schnapp
```

Vorhanden ist:

| Pfad | Was es ist | Prüfen |
|---|---|---|
| `.` (Wurzel) | Go-Modul `github.com/Andiministrator/schnapp`: das Programm `cmd/schnapp` (bisher nur Optionen), `capture` (Aufnahme über KWin), `document` (Ebenen, Zuschnitt, Schwärzen, Rückgängig), `fileformat` (Arbeitsdatei und Exporte), `library` (Bibliothek), `ui` (Schicht zu Gio), `i18n` (Sprachwahl, gettext-PO-Leser) und `po` (eingebettete Übersetzungen, `po/de.po`); `data/` enthält die Vorlage der Desktop-Datei | `go vet ./... && go test ./...` (mit `GOFLAGS` wie oben) |
| `zeichnen/` | Go-Modul `github.com/Andiministrator/schnapp/zeichnen`: SVG lesen, Rasterer mit Kantenglättung, Rechenhilfen (nur Standardbibliothek) | `cd zeichnen && go vet ./... && go test ./...` |
| `brand/` | Logo, Farben und Corporate Design ([`brand/corporate-design.md`](brand/corporate-design.md), mit PDF) | `python3 brand/werkzeug/paket.py` erzeugt die Logo-Dateien neu |
| `third_party/gio/` | Gio mit unserem Patch (Vollbild auf gewähltem Monitor, gebrochene Skalierung), eingebunden per `replace` in `go.mod` | `third_party/gio-patch.sh pruefen`, siehe [`third_party/README.md`](third_party/README.md) |
| `prototype/` | Wegwerf-Prototyp, der Gio geprüft hat (braucht Go ≥ 1.27, cgo, `libwayland-client`, `libwayland-egl`, `libwayland-cursor`, `libEGL`, `libxkbcommon`) | `prototype/bauen.sh`, siehe [`prototype/README.md`](prototype/README.md) |

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
