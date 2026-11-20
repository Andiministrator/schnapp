# Projekt-Konventionen (Kompaktblatt für Sub-Agents und neue Sitzungen)

Primäre Quelle ist [`CLAUDE.md`](../../CLAUDE.md). Dieses Blatt fasst zusammen, was ein Sub-Agent ohne Memory wissen
muss.

## Commands
- `/projekt-start` — Session-Check (Board, Findings, offene Entscheidungen).
- `/chat-wechsel` — Übergabe als `▶ HIER STARTEN`-Kommentar an die Einstiegskarte.
- `/sprint-abschluss` — Abhängigkeits-Check, Kritiker, Verify, Triage; Gate vor jedem `v*`-Tag. Quelle des
  Auflagen-Blocks für Sub-Agent-Prompts.
- `/init-projekt`, `/retrofit-projekt` — Projekt anlegen bzw. nachrüsten.

## Konventionen
- **Sprache:** projektinterne Doku, Commits, Kommentare Deutsch (ADR-0002); Oberfläche Englisch als Ausgangssprache +
  Deutsch (PO-Dateien), README/CONTRIBUTING englisch mit `*.de.md` (ADR-0014); deutsche Oberfläche ohne Anrede, sonst „du“ (E-13).
  Go-Bezeichner je Paket einheitlich deutsch oder englisch.
- **Öffentliches Repo:** keine Kunden-/Firmenbezüge, keine lokalen Pfade, Mailadressen, Hausnetz-Angaben, echten Screenshots; Internes in `CLAUDE.local.md`.
- **Technik:** Go + Gio mit Patch (ADR-0003 Nachtrag 2, `topics/gio-wayland.md`), nur Wayland, KDE zuerst (ADR-0007),
  Fremdpakete nur mit Nutzen (ADR-0011), cgo nur für Gio. **Kein neues Python** im Repo (wird schrittweise entfernt).
- **Gestaltung:** `brand/corporate-design.md` ist verbindlich (Farben, Schrift Open Sans, Oberflächen-Regeln);
  Logo-Dateien sind erzeugt (`brand/werkzeug/`), nie von Hand ändern.
- **Dateiformat:** PNG mit zwei privaten Blöcken `snAP` (SVG) und `snAO` (Original), ADR-0008 mit Nachträgen 1 und 2 (beide Blöcke Pflicht, Grenzen, Export ohne Weggeschnittenes);
  Schwärzen überschreibt das Original — `[S]`-Pfad. Öffentliche Spezifikation folgt (E-15, englisch).
- **Bauen und Testen** im Wurzelmodul immer mit `GOFLAGS=-tags=novulkan,nox11` (Gio; sonst fehlt `vulkan/vulkan.h`),
  Programme mit `-trimpath`. Pakete: `i18n`, `po`, `document`, `fileformat`, `capture`, `library`, `settings`, `selection`,
  `canvas`, `render`, `ui`, `cmd/schnapp`; eigene Module `draw/` (früher `zeichnen/`), `prototype/`, `third_party/gio/`, `tmp/` (trennt den Arbeitsordner ab).
- **Pfade und eigene Dateinamen:** `schnapp` immer klein (`~/Bilder/schnapp/`, `schnapp.desktop`), im Text „Schnapp“.
- **Kommandozeile englisch:** `--region` (Voreinstellung), `--screen[=NAME]`, `--list-screens`, `--window`, `--repeat`,
  `--no-editor`, `DATEI`, `--ignore-limit DATEI` (ADR-0017 Nachträge 2–6); sie ist Oberfläche und künftig auch Schnittstelle für Agenten (E-14).
- **Autonomie:** selbst entscheiden; fragen nur bei den 3 K's (kritische Entscheidung, kritische Auffälligkeit,
  Klärungsbedarf). Vor jedem Push Diff auf Internes prüfen.
- **ADR-first:** Entscheidungen erst als ADR unter `docs/adr/`; nächste Nummer aus dem Verzeichnis.
- **QA:** Schritt-QA nach jedem Schritt, Sprint-Abschluss mit Kritiker- und Verify-Sub-Agent (`docs/qa/`).
- **Spurensicherung:** jede URL in `knowledge/sources-visited.md`.
- **Findings:** offene in `docs/findings.md` (Abschnitt `## Offen`), erledigte in `docs/findings.archive.md`
  (`findings-rotate.mjs`); jedes offene mit Board-Karte; schreiben nur über `register-schreiben.mjs`. Sub-Agents schreiben Findings nur als Text mit
  Kartenvorschlag in ihren Bericht, nie in die Datei.
- **Board:** alles selbst erledigen, auch Sprints anlegen (Andi); nie `aboard-sync.json` direkt schreiben; Ops über `node ~/.aboard-framework/tools/aboard-op.mjs … --dir .`.
- **Commits:** Conventional Commits, deutsch, `Co-Authored-By` für das ausführende bzw. prüfende Modell und den Orchestrator; gleiche
  Modelle nur eine Zeile, keine Leerzeile zwischen den Trailern (CLAUDE.md §Commit-Konvention, ADR-0005).
- **Sub-Agents:** kein Memory; Auflagen-Block wörtlich; keine gemeinsamen Dateien, kein `git stash/reset/checkout`,
  kein Build bei mehreren Strängen. Karten mit anderem Owner-Modell laufen als Sub-Agent im eigenen Worktree
  (`isolation: worktree`), Commit dort, Übernahme per `git cherry-pick`; den Worktree erst löschen, wenn
  `git cherry main <branch>` nur „-“ zeigt oder (nach Zusammenführen beim Cherry-Pick) die geänderten Zeilen
  nachweislich identisch sind; Agenten-Branches `worktree-agent-*` löscht Claude dann selbst (Andi, 20.11.2026). Commit-Nachricht per Datei (`git commit -F`), nie per Heredoc.
- **Echte Aufnahmen und Overlays** (ganzer Bildschirm, alle Monitore) nur nach Ankündigung bei Andi; Bilder nur im
  Speicher oder unter `tmp/`, Monitornamen nicht in Dateien, Tests oder Board.
- **Board-Checkpoints:** Board neu lesen bei Sitzungsstart, nach Kartenabschluss, vor „los", beim Sprint-Abschluss.
- **AKTION VOR ANTWORT:** Hook-Blöcke dieses Namens zuerst ausführen.
- **Dateien:** Arbeitsdateien nach `tmp/<lauf>/` im Projekt, nie `/tmp`. Kein `rm` außer mit ausgeschriebenem Pfad.
  Testdaten nie per Heredoc. Vor jedem Push auch Binärdateien mit `strings` prüfen (F-16); Python mit `-B`.
- **Remotes:** GitHub `origin` (öffentlich) + privates Zweit-Repo als zweite Push-URL (Einzelheiten nur in
  `CLAUDE.local.md`).
