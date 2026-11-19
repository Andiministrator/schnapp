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
- **Sprache:** Deutsch (ADR-0002); Go-Bezeichner je Paket einheitlich deutsch oder englisch.
- **Öffentliches Repo:** keine Kunden-/Firmenbezüge, keine lokalen Pfade, Mailadressen, Hausnetz-Angaben, echten Screenshots; Internes in `CLAUDE.local.md`.
- **Technik:** Go + Gio (ADR-0003), nur Wayland, KDE zuerst (ADR-0007), Fremdpakete nur mit Nutzen (ADR-0011).
- **Dateiformat:** PNG mit SVG-Block; Schwärzen überschreibt auch das Original (ADR-0008) — `[S]`-Pfad.
- **Autonomie:** selbst entscheiden; fragen nur bei den 3 K's (kritische Entscheidung, kritische Auffälligkeit,
  Klärungsbedarf). Vor jedem Push Diff auf Internes prüfen.
- **ADR-first:** Entscheidungen erst als ADR unter `docs/adr/`; nächste Nummer aus dem Verzeichnis.
- **QA:** Schritt-QA nach jedem Schritt, Sprint-Abschluss mit Kritiker- und Verify-Sub-Agent (`docs/qa/`).
- **Spurensicherung:** jede URL in `knowledge/sources-visited.md`.
- **Findings:** in `docs/findings.md`, jeder mit Board-Karte. Sub-Agents schreiben Findings nur als Text mit
  Kartenvorschlag in ihren Bericht, nie in die Datei.
- **Board:** nie `aboard-sync.json` direkt schreiben; Ops über `node ~/.aboard-framework/tools/aboard-op.mjs … --dir .`.
- **Commits:** Conventional Commits, deutsch, `Co-Authored-By` für jedes beteiligte Modell (ADR-0005).
- **Sub-Agents:** kein Memory; Auflagen-Block wörtlich; keine gemeinsamen Dateien, kein `git stash/reset/checkout`,
  kein Build bei mehreren Strängen.
- **Board-Checkpoints:** Board neu lesen bei Sitzungsstart, nach Kartenabschluss, vor „los", beim Sprint-Abschluss.
- **AKTION VOR ANTWORT:** Hook-Blöcke dieses Namens zuerst ausführen.
- **Dateien:** Arbeitsdateien nach `tmp/<lauf>/` im Projekt, nie `/tmp`. Kein `rm` außer mit ausgeschriebenem Pfad.
  Testdaten nie per Heredoc.
- **Remotes:** GitHub `origin` (öffentlich) + privates Zweit-Repo als zweite Push-URL (Einzelheiten nur in
  `CLAUDE.local.md`).
