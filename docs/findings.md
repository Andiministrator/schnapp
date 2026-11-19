# Findings

Typen: `[S]` Sicherheit · `[K]` Stabilität · `[B]` Blockierend · `[V]` Verbesserung · `[D]` Konsistenz.
Priorität: P0 (sofort) · P1 (diesen Sprint) · P2 (nächsten Sprint) · P3 (irgendwann).
Schulden-Grenze: > 10 offene `[K]`-Findings → Hygiene-Sprint einplanen.

**Jedes offene Finding hat eine offene Board-Karte** mit seiner F-Nummer in Titel oder Beschreibung.

**Pflichtangaben für offen eingetragene Befunde:**
- **Prüfbefehl** — ein Aufruf mit erwartetem Wert, an dem sich ablesen lässt, ob die Lücke noch besteht. Am besten
  eine Verhaltensprobe (Test, Aufruf, Messung); ein Wort-`grep` nur für reine Doku-Befunde.
- **Widerlegungsbedingung** — nur bei Befunden, deren Fix-Richtung Code ändert: woran sich zeigt, dass der notierte Weg
  nicht trägt. Für `[D]` freiwillig.

Im selben Zug gefundene und behobene Befunde brauchen beides nicht; dort steht der Test im Fix-Commit.

| ID | Typ | P | Sprint | Beschreibung | Status |
|---|---|---|---|---|---|
| F-1 | [S] | P1 | — | **Git-Geschichte nicht veröffentlichungsreif** (Karte #8): private Mailadresse als Autor, interne Firmen-, Projekt- und Hausnetz-Angaben in Docs und Commits. Behoben 19.11.2026 (Variante A, Andi): Docs neutralisiert, Internes in `CLAUDE.local.md` (gitignoriert), Geschichte neu begonnen, Autor GitHub-noreply. **Prüfbefehl:** `git log --format=%ae \| sort -u` (erwartet nur die noreply-Adresse) und `git log -p \| grep -ciE "<Liste in CLAUDE.local.md>"` (erwartet 0 bis auf den akzeptierten Projektnamen aus F-6 im ersten Commit). | behoben |
| F-2 | [S] | P1 | — | Lokales Ref `refs/original/refs/heads/main` (Rest von filter-branch) und Zweig `alt-main` tragen die alte Geschichte mit privater Mailadresse; `git push --all/--mirror` würde sie veröffentlichen (Kritiker K1). Abgesichert durch deny-Regeln in `.claude/settings.json` (F-3); Ref löschen nur mit Andis Freigabe. **Prüfbefehl:** `git for-each-ref refs/original` (erwartet: leer). **Widerlegungsbedingung:** Andi will das Ref als zusätzliche Sicherung behalten — dann akzeptiert. | behoben 19.11.2026 (Andi: „ja“; Ref gelöscht, `git for-each-ref refs/original` leer; `alt-main` bleibt als Sicherung) |
| F-3 | [S] | P2 | — | `.claude/settings.json` erlaubte `git push*` ohne Ausnahme für `--all`/`--mirror`/`--tags`/`alt-main` (K2). Behoben: deny-Regeln mit Platzhalter vor dem Flag (`git*push*--all*` usw., auch `git -C … push`). Nach zweitem Verify-Lauf auch Force (`--force`, `-f`, `+ref`), `--delete` und `:ref` an beliebiger Stelle gesperrt. Fehlsperren bewusst: `refs/`-Refspecs, Zweignamen mit `-f`. Restrisiko akzeptiert: Aufruf über Pfad (`/usr/bin/git`), `env …` oder Git-Alias. | behoben 19.11.2026 |
| F-4 | [D] | P2 | — | Prüfbefehl von F-1 verwies auf eine nicht vorhandene Begriffsliste (K3). Behoben: Abschnitt „Heikle Begriffe“ in `CLAUDE.local.md`. | behoben 19.11.2026 |
| F-5 | [D] | P2 | — | E-05 in `docs/open-decisions.md` veraltet (K4). Behoben. | behoben 19.11.2026 |
| F-6 | [S] | P3 | — | Name eines anderen privaten Projekts in `knowledge/` (K5). Im Baum behoben; im ersten veröffentlichten Commit bleibt er stehen (ohne Force-Push nicht entfernbar, nur ein Projektname ohne Inhalt). | akzeptiert 19.11.2026 (Baum bereinigt; erster Commit bleibt so) |
| F-7 | [D] | P3 | — | Karte #5 verwies auf nicht vorhandenen „ADR-0012 Nachtrag 1“ (K6). Behoben (Kartentext). | behoben 19.11.2026 |
| F-8 | [D] | P3 | — | Karte #8 nannte alten ADR-Dateinamen (K7). Behoben. | behoben 19.11.2026 |
| F-9 | [D] | P3 | — | Formfehler: Klammer in `CLAUDE.md`, Typenzeile an falscher Stelle, `knowledge/sources/` fehlte, Status `blocked` für HTTP 405 (K8). Behoben. | behoben 19.11.2026 |
| F-10 | [V] | P3 | — | Framework-Verweise (ADR-0043/0062, `~/.aboard-framework`) in öffentlicher `CLAUDE.md` unerklärt (K9). Behoben: Hinweis am Kopf. | behoben 19.11.2026 |
| F-11 | [V] | P3 | — | `TestRepr` ohne Grenzfälle (K10). Behoben: 10 Grenzfälle mit Sollwerten aus Python `repr()`. | behoben 19.11.2026 |
| F-12 | [V] | P3 | — | `TestRasterproben` nur Rauchtest (K11). Behoben: Sollwerte (Pixelzahl, Kanalsummen) je Probe; Gegenprobe mit verfälschtem Sollwert rot. | behoben 19.11.2026 |
| F-13 | [D] | P3 | — | Kommentar in `zeichnen/geo/geo.go` verwies auf eine Funktion des Schwesterprojekts (K12). Behoben. | behoben 19.11.2026 |
| F-14 | [V] | P3 | — | Board ohne `kind:sprint`-Swimlane; Sprint-Abschluss kann keinen Recap in eine 🏁-Karte legen (K13). Meta-Karten legt nur der Browser an. **Prüfbefehl:** im Board eine nicht archivierte Swimlane mit `kind: sprint`. Spätestens vor dem ersten `v*`-Tag nötig (ADR-0006). | behoben 19.11.2026 (Swimlane „Sprint 1 — Prototyp Gio“ per Op angelegt; die Meta-Karten legt der Browser beim nächsten Laden an) |
| F-15 | [K] | P2 | — | **Aufnahme über ScreenShot2 klappt ohne installierte `.desktop`-Datei** (Prototyp, 19.11.2026), obwohl KWin laut Recherche nur Programmen mit Berechtigungseintrag erlaubt. Ursache unbekannt — auf anderen Systemen kann die Sperre greifen, dann scheitert Schnapp still. Klären vor der ersten Fassung (KWin-Quelltext der Berechtigungsprüfung; Verhalten bei Start über `.desktop`). **Prüfbefehl:** `~/.local/bin/schnapp-prototyp` ohne Datei unter `~/.local/share/applications` → Protokollzeile „Ausgabe …: … Aufnahme“ erwartet, solange der Befund gilt. **Widerlegungsbedingung:** KWin-Quelltext zeigt eine dokumentierte Ausnahme (z. B. Abschaltung der Prüfung) — dann Wissen festhalten, Befund schließen. | offen |
