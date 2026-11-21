# ADR-0027: Projektstruktur aufräumen

## Status
Vorgeschlagen (Claude Opus 5.5, 21.11.2026, Karte #124). Entscheidung durch Andi steht aus; umgebaut wird erst danach.

## Kontext
Andi, 21.11.2026: „die verzeichnis-struktur in unserem projekt sieht (für mich) ziemlich chaotisch aus“. Bestand am
21.11.2026 (Commit 49e5f90):

- **Hauptordner:** 15 Go-Pakete liegen flach neben Doku und Werkzeugen: `canvas`, `capture`, `document`, `export`,
  `fileformat`, `i18n`, `library`, `po`, `portal`, `render`, `selection`, `settings`, `ui`, `wlclip`, dazu `internal`
  (`probeschutz`, `speicherschritt`, `wlwire`) und `cmd`. Daneben `brand`, `data`, `docs`, `draw`, `knowledge`,
  `prototype`, `scripts`, `third_party`, `tmp` und acht Dateien. Wer das Repo öffnet, sieht zuerst 24 Ordner, ohne zu
  erkennen, was Programm, was Doku und was Werkzeug ist.
- **Testdateien nach Prozess statt Inhalt benannt** (12 Dateien, 53 Tests): `canvas/karte84_test.go`,
  `canvas/karte85_test.go`, `canvas/luecken_test.go`, `canvas/nacharbeit_test.go`, `cmd/schnapp/editor_luecken_test.go`,
  `cmd/schnapp/liegengeblieben_luecken_test.go`, `cmd/schnapp/nacharbeit_test.go`, `library/kritik_test.go`,
  `library/luecken_test.go`, `library/temp_rest_nacharbeit_test.go`, `library/vorschau_f115_test.go`,
  `ui/schliessen_nacharbeit_test.go`. Der Name sagt, wann der Test entstand, nicht was er prüft.
- **`prototype/`** (12 Dateien, eigenes Modul, eigener CI-Job): Wegwerf-Code der Gio-Prüfung (ADR-0003), seit dem
  Editor überholt.
- **`brand/`** (183 Dateien): `logo/` 94 erzeugte Dateien, `entwuerfe/` 72 Entwürfe, `werkzeug/` 8 Python-Werkzeuge,
  `cd/` 5, dazu `corporate-design.md/.pdf`, `farben.md`, `README.md`.
- **Nicht versioniert** (gitignoriert, nur auf diesem Rechner): `aboard.html`, `aboard-sync.json`, `aboard-sync.base*.json`
  (Ringpuffer von aBoard), `aboard-sync.inbox/`, `aboard-sync.offen.json` (Lesedatei, die die aBoard-Hooks laufend neu
  schreiben — gehört zu aBoard), `aboard.html.bak-20261119-070823Z` (Sicherung vom aBoard-Update am 19.11.);
  unter `tmp/` Reste alter Läufe: `echtlauf` (20 MB), `sprint7` (16 MB), `editor-test`, `sprint4`, `k62`, `papierkorb`,
  `probe-innen`; `probe` (125 MB) und `sprint9` sind in Gebrauch.

Bleibt, wie es ist: `cmd/` (Go-Konvention für Programme), `draw/` (eigenes Modul, das ein Schwesterprojekt per
`replace`-Pfad einbindet, ADR-0012 — verschieben nur nach Abstimmung dort), `third_party/gio/` (`replace`-Pfad und
`gio-patch.sh`, ADR-0015), `docs/`, `knowledge/`, `scripts/`, `data/`, `.github/`.

## Vorschlag

### 1. Programmpakete unter `internal/`
Alle Pakete, die nur Schnapp selbst nutzt, ziehen nach `internal/`: `internal/canvas`, `internal/capture`, … , `internal/po`,
`internal/wlclip`. `internal/` ist die Go-Konvention für Pakete, die kein anderes Modul importieren darf; das stimmt
hier, weil nur `draw/` öffentlich eingebunden wird. Der Modulpfad bleibt `github.com/Andiministrator/schnapp`, die
Importe werden `…/schnapp/internal/<paket>`. Paketnamen und Bezeichner bleiben (ADR-0019).

Hauptordner danach: `brand/ cmd/ data/ docs/ draw/ internal/ knowledge/ scripts/ third_party/` plus `README*`,
`CONTRIBUTING*`, `LICENSE`, `CLAUDE.md`, `install.sh`, `go.mod`, `go.sum`. Zwölf statt 24 Ordner, und die Rollen sind
erkennbar: Programm (`cmd`, `internal`), öffentliche Bibliothek (`draw`), Fremdcode (`third_party`), Doku (`docs`,
`knowledge`), Gestaltung (`brand`), Werkzeuge (`scripts`, `data`).

### 2. Testdateien nach Inhalt benennen
Reine Umbenennung mit `git mv`, Tests unverändert; Vorschlag:

| bisher | neu |
|---|---|
| `canvas/karte84_test.go` | `canvas/umschalt_raster_test.go` |
| `canvas/karte85_test.go` | `canvas/verschieben_grenze_test.go` |
| `canvas/luecken_test.go` | `canvas/rand_streifen_test.go` |
| `canvas/nacharbeit_test.go` | `canvas/rand_und_kante_test.go` |
| `cmd/schnapp/editor_luecken_test.go` | `cmd/schnapp/speichern_vorschau_test.go` |
| `cmd/schnapp/liegengeblieben_luecken_test.go` | `cmd/schnapp/liegengeblieben_rueckblick_test.go` (`liegengeblieben_test.go` gibt es schon) |
| `cmd/schnapp/nacharbeit_test.go` | `cmd/schnapp/ungesichert_exit_test.go` |
| `library/kritik_test.go` | `library/vorschau_neuaufbau_test.go` |
| `library/luecken_test.go` | `library/speichern_fehlerwege_test.go` |
| `library/temp_rest_nacharbeit_test.go` | `library/temp_reste_test.go` |
| `library/vorschau_f115_test.go` | `library/vorschau_veraltet_test.go` |
| `ui/schliessen_nacharbeit_test.go` | `ui/schliessen_waehrend_speichern_test.go` |

Wo eine Datei Tests zu zwei Themen mischt, kommt der Name vom Hauptthema; beim Umbau wird jede Datei gelesen und der
Name notfalls angepasst. Regel für später (in `CONTRIBUTING`): Testdateien heißen nach dem, was sie prüfen; Karten- und
Finding-Nummern stehen im Kommentar am Test, nicht im Dateinamen.

### 3. `prototype/` entfernen
Ordner und CI-Job `prototype` entfallen; der Stand bleibt in der git-Geschichte (letzter Commit mit `prototype/` wird
in ADR-0003 als Nachtrag genannt). Verweise in `README*`, `CLAUDE.md`, `docs/betrieb.md` und `knowledge/topics/` werden
angepasst. Nutzen: ein Modul, ein CI-Job und zwölf Dateien weniger, keine zweite Gio-Fassung,
die mitgepflegt werden will.

### 4. Ältere Logo-Entwürfe entfernen
`brand/entwuerfe/runde-1` bis `runde-5` (41 Dateien) und ihre Erzeuger `brand/werkzeug/runde1.py` bis `runde5.py`
entfallen; in der Geschichte bleiben sie auffindbar. `runde-6` (31 Dateien) bleibt: Dort liegt der gewählte Entwurf
e6, auf den `brand/logo/README.md`, `brand/README.md` und `brand/corporate-design.md` verweisen; die Verweise auf
„Runde 1 … 6“ werden auf Runde 6 und die Geschichte umgestellt. Übrige Werkzeuge bleiben, bis die Karte „Python aus
dem Projekt entfernen“ sie ersetzt.

### 5. Lokale Reste (nicht versioniert) löschen
`aboard.html.bak-20261119-070823Z` und unter `tmp/`: `echtlauf`, `sprint4`, `sprint7`, `k62`, `editor-test`,
`papierkorb`, `probe-innen` (zusammen rund 36 MB). Belege daraus, die noch gebraucht werden, stehen schon unter
`knowledge/messungen/`. Gelöscht wird je Ordner mit ausgeschriebenem Pfad nach einem Blick hinein (globale Löschregel).
`aboard-sync.offen.json` und die `aboard-sync.base.*.json` bleiben (gehören zu aBoard).

## Umsetzung (erst nach Entscheidung)
Ein Commit je Punkt, jeweils `go vet`/`go test` mit `novulkan,nox11`, mit Tag `probe` und in `draw/` grün, dazu ein
Probelauf mit `scripts/probe.sh`:
1. `git mv` der Pakete, Importe per `gofmt -r` bzw. Ersetzen des Importpfads, `.github/workflows/test.yml`
   (`./ui` → `./internal/ui`), `scripts/probe.sh`, `install.sh`, `CLAUDE.md` §Architektur und §Entwicklungsstand,
   `README.md`/`README.de.md` und `CONTRIBUTING.md`/`CONTRIBUTING.de.md` im selben Commit, `docs/betrieb.md`.
   Pfade in alten ADRs und archivierten Findings bleiben als Zeitstand; offene Findings und Karten bekommen die neuen Pfade.
2. Testdateien umbenennen, Regel in `CONTRIBUTING*`.
3. `prototype/` und CI-Job, Nachtrag ADR-0003.
4. Logo-Entwürfe Runde 1–5 samt Erzeugern, Verweise in `brand/`.
5. Lokale Reste.

Aufwand: rund 2 Stunden; Punkt 1 berührt jede Go-Datei mit Importen, daher zuletzt im Sprint und ohne parallele Arbeit.

## Konsequenzen
- Neue Pakete gehören nach `internal/`; was ein anderes Modul einbinden soll, wird wie `draw/` ein eigenes Modul.
- Die Kontrollen gegen heikle Begriffe und die CI laufen unverändert; `git log --follow` folgt den verschobenen Dateien.
- Offene Arbeit anderer Stränge muss vor Punkt 1 committet sein (sonst Konflikte in jedem Import).

## Entscheidung
*Steht aus.* Fragen an Andi: Punkte 1 bis 5 je einzeln annehmen oder ablehnen.
