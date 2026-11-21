# Betrieb — Bauen, Prüfen, Installieren, Ausliefern

Kurzanleitung für die wiederkehrenden Handgriffe. Begründungen stehen in den ADRs, die Regeln für Arbeit und QA in
`CLAUDE.md` und `docs/qa/`. Diese Datei sammelt nur, **was man tut**, und verweist für das Warum.

## Bauen und Testen

Wurzelmodul (Gio braucht cgo und die Bibliotheken `libwayland-dev libegl-dev libxkbcommon-dev`, ADR-0003 Nachtrag 2):

```bash
export GOFLAGS=-tags=novulkan,nox11
go vet ./... && go test ./...
go build -trimpath -o schnapp ./cmd/schnapp
```

Probe-Builds (Fernbedienung und Probe-Tests sieht der Normalbau nicht):

```bash
GOFLAGS=-tags=novulkan,nox11,probe go vet ./cmd/... ./internal/ui
GOFLAGS=-tags=novulkan,nox11,probe go test ./cmd/schnapp-fernbedienung ./cmd/schnapp ./internal/probeschutz
```

Weitere Module, je mit eigenem `go.mod`:

| Modul | Prüfung |
|---|---|
| `draw/` (Zeichenpakete, ADR-0012) | `cd draw && go vet ./... && go test ./...` |
| `third_party/gio/` (gepatchtes Gio, ADR-0015) | `third_party/gio-patch.sh pruefen` |

**Probeläufe der Oberfläche** ohne echten Bildschirm: `scripts/probe.sh` startet einen virtuellen KWin, Drehbücher
liegen unter `scripts/drehbuecher/` (Anleitung in `knowledge/topics/lehren.md`). Probe-Builds laufen nur dort
(`internal/probeschutz`, F-229) und werden nie ausgeliefert.

## CI

GitHub Actions, `.github/workflows/test.yml` (ADR-0010), läuft bei jedem Push und Pull-Request. Drei Jobs:

| Job | Inhalt |
|---|---|
| `wurzel` | gofmt, vet, test, Programm bauen; dazu vet/test mit Tag `probe` |
| `draw` | gofmt, vet, test im Modul `draw/` |
| `gio-patch` | Patch passt zu `third_party/gio/` |

Nach einem Push den Lauf ansehen, auch auf **Annotationen** (Abkündigungen kommen dort, solange der Lauf grün ist;
F-309; öffentliche Check-Run-API je Job, `knowledge/topics/lehren.md` Sprint 9). Die Aktionen sind per Commit-SHA gepinnt (F-43).

## Installieren

```bash
./install.sh              # baut und installiert nach ~/.local (PREFIX, XDG_DATA_HOME einstellbar)
./install.sh --uninstall  # entfernt genau die drei installierten Dateien
```

Installiert werden Programm, Desktop-Datei (aus `data/schnapp.desktop.in`, ADR-0017) und Symbol. Die Version kommt aus
`git describe`; ohne Tag ist sie bisher der nackte Commit-Hash (Soll `0.0.0-dev+<hash>`, offen: F-113).

## Ausliefern (Release)

Versionen nach SemVer mit `v`-Präfix (ADR-0006). Release-gebundene Arbeit läuft in einer Sprint-Swimlane.

1. Sprint-Abschluss nach `docs/qa/sprint-abschluss.md` bestehen, einschließlich Kritiker-Lauf. Blockierend: alle P0,
   alle [S], [K] P0/P1, alle [B].
2. **Vor dem Tag** `README.md`, `README.de.md` und `CLAUDE.md` auf die neue Version bringen; genau dieser Commit wird
   getaggt. Prüfung: `git show vX.Y.Z:README.md | grep -c 'Version X.Y.Z'` → 1.
3. Den Tag schlägt Claude vor, gesetzt wird er nur auf ausdrückliche Ansage des Maintainers.
4. Vor jedem Push den Diff auf Pfade, Namen, Mailadressen und interne Angaben prüfen — das Repo ist öffentlich
   (Prüfmuster außerhalb des Repos, F-1).
5. Nie `git push --all`, `--mirror` oder `--tags`; den Tag einzeln pushen.

## Zuständigkeiten

| Was | Wer |
|---|---|
| Code, Tests, Doku, Commits, Pushes, Board-Pflege | Claude (selbstständig) |
| Probeläufe im virtuellen KWin | Claude |
| Echte Läufe: Portal-Dialog, Klipper, fremde Programme, echte Monitore | Maintainer, per `!`-Befehl nach Anleitung |
| CI-Läufe und Annotationen ansehen | Claude, über die öffentliche API (`commits/<sha>/check-runs`, dann `check-runs/<id>/annotations`) |
| Release-Tag setzen, Force-Push, Löschen von Dateien oder Branches | nur auf Ansage des Maintainers |
| Änderungen an öffentlichen Schnittstellen von `draw/` | nach Abstimmung mit dem einbindenden Schwesterprojekt |
