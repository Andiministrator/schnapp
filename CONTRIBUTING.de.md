# Mitarbeit an Schnapp

*[English](CONTRIBUTING.md) · Deutsch*

Danke für dein Interesse. Schnapp steht am Anfang; bitte vor größeren Änderungen ein Issue eröffnen, damit wir die
Richtung vorher abstimmen.

## Wie dieses Projekt entsteht

Der größte Teil von Code und Doku wird von einem KI-Modell (Anthropic Claude) unter Anleitung und Prüfung des
Maintainers geschrieben. Beiträge von Menschen sind genauso willkommen. Wer selbst KI-Werkzeuge nutzt, sagt das im
Pull Request.

## Grundregeln

- **Nur Wayland.** Keine X11-Wege (`docs/adr/0007-*`).
- **Schwärzen ist endgültig.** Geschwärzte oder verpixelte Inhalte dürfen aus keiner gespeicherten oder exportierten
  Datei wiederherstellbar sein. Änderungen hier brauchen einen Test, der das belegt.
- **Keine Spuren.** Erzeugte Dateien enthalten keine lokalen Pfade, Benutzer-, Rechner- oder Programmnamen.
- **Keine echten Screenshots** in Tests oder Doku — sie zeigen private Daten. Testbilder synthetisch erzeugen.
- **Neue Abhängigkeiten** nur mit spürbarem Nutzen; Zweck, Lizenz (GPL-3.0-verträglich) und Alternative in der
  Commit-Nachricht nennen (`docs/adr/0011-*`).
- **Kein cgo** über das hinaus, was Gio (mit unserem Patch) braucht — diese Ausnahme ist `docs/adr/0003-*` Nachtrag 2;
  alles andere braucht eine eigene Architektur-Entscheidung (`docs/adr/0011-*`).

## Sprachen

- Oberflächentexte: Englisch ist die Ausgangssprache; jeder neue Text braucht auch die deutsche Übersetzung
  (`po/de.po`, sobald es sie gibt). Weitere Sprachen sind willkommen.
- Nutzer-Dokumente (README, CONTRIBUTING) gibt es auf Englisch und Deutsch — beide im selben Pull Request pflegen.
- Arbeitsdokumente (`docs/`, `knowledge/`), Commit-Nachrichten und Code-Kommentare sind deutsch. Englisch ist im Pull
  Request in Ordnung; der Maintainer passt es an.

## Gestaltung

Logo, Farben, Schrift und Regeln für die Oberfläche: [`marke/corporate-design.md`](marke/corporate-design.md) (auch als
PDF); Logo-Dateien in [`marke/logo/`](marke/logo/README.md).

## Commits und Prüfungen

- Conventional Commits: `<typ>(<bereich>): <beschreibung>`, Typen `feat fix docs refactor chore style test perf`.
- Vor einem Pull Request: `gofmt`, `go vet ./...` und `go test ./...` in jedem berührten Modul.
- `extern/gio/` ist Gio mit unserem Patch (`docs/adr/0015-*`). Wer dort ändert, ruft `extern/gio-patch.sh erzeugen` auf
  und committet den neuen `extern/gio-ausgabe.patch` mit; die CI prüft, dass beides zusammenpasst.
- Jede selbst geschriebene Quelldatei beginnt mit `SPDX-License-Identifier: GPL-3.0-or-later` (als Kommentar).
  Patches an fremdem Code (z. B. Gio) behalten dessen Lizenz.

Mit einem Beitrag stimmst du zu, dass er unter GPL-3.0-or-later steht.
