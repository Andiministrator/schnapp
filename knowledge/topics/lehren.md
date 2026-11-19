# Lehren aus der Arbeit an Schnapp

Stand: 2026-11-19 (ergänzt nach Sprint 1). Kurze Regeln aus Fehlern und Befunden; Einzelheiten in `docs/findings.md`.

## Öffentliches Repo

- **Vor dem ersten öffentlichen Push die ganze Geschichte prüfen**, nicht nur den Arbeitsbaum: Autor-Mailadresse
  (`git log --format=%ae`), Namen, Pfade, Hausnetz (F-1). Ein privates Zweit-Repo mit Schreibsperre gegen Force-Push
  macht spätere Korrekturen unmöglich — dort erst pushen, wenn die Geschichte veröffentlichungsreif ist.
- **Interne Angaben gehören in `CLAUDE.local.md`** (gitignoriert), nicht in versionierte Dateien. Auch Namen anderer
  eigener Projekte zählen dazu (F-6).
- **Binärdateien vor dem Push eigens prüfen** (F-16): `git diff` zeigt sie nur als „Binary files differ“, ein `grep`
  über den Diff findet darin nichts. Python-Bytecode (`.pyc`), PDFs und Bilder können Pfade und Namen tragen —
  `git diff --cached --name-only` ansehen, Binäres mit `strings` gegen die Begriffsliste prüfen; Bytecode und
  Build-Reste gehören in `.gitignore`.
- **`git filter-branch` hinterlässt `refs/original/…`** mit der alten Geschichte (F-2). Push-Varianten `--all`,
  `--mirror`, `--tags`, Force und Refspecs sind in `.claude/settings.json` gesperrt (F-3).

## Go-Tests

- **Konstante Ausdrücke rechnet Go exakt:** `0.1 + 0.2` ist im Test `0.3`, nicht `0.30000000000000004`; ebenso
  `-0.0` ist `0`. Für Gleitkomma-Grenzfälle den Wert direkt als Literal oder per `math.Copysign` angeben.
- **Ein Rauchtest beweist wenig:** Für Raster-Ergebnisse Sollwerte (Pixelzahl, Kanalsummen) festhalten und mit einem
  verfälschten Sollwert gegenprüfen, dass der Test rot wird (F-12).

- **Fehlerfälle bis auf den geprüften Punkt gültig machen** (Sprint 3, datei): Ein Testfall „zu tief“ oder „Entität“
  scheiterte schon an einer fehlenden Pflichtangabe — die Gegenprobe (Grenze entfernt) blieb grün. Je Fehlerfall ein
  gültiges Gegenstück daneben, das nur den einen Punkt nicht hat.
- **Verpixeln einer einfarbigen Fläche behält die Farbe** — für Schwärzen-Tests ein Muster mit Struktur (Schachbrett)
  nehmen, sonst meldet der Test ein Leck, das keins ist.

- **Fehlschläge im eigenen Code nachbauen** (Sprint 3, bibliothek): Systemaufrufe als Paketvariable halten
  (`var umbenenne = os.Rename`) und im Test durch eine Attrappe ersetzen, die gezielt scheitert (z. B. EXDEV). So
  lassen sich „atomar ohne Restdatei“ und „Rückweg ohne Löschen“ prüfen, ohne Prozesse zu beobachten.
- **Einem Index als Cache nie trauen:** Vor jedem Eingriff in eine Datei die maßgeblichen Angaben aus der Datei selbst
  lesen; ein Test mit absichtlich veraltetem Index belegt das.
- **`O_WRONLY|O_APPEND` erlaubt kein `ReadAt`** — wer vor dem Anhängen das letzte Byte prüft, braucht `O_RDWR`.

- **Kritiker nach Autor aufteilen** (Sitzungs-QA Sprint 3): Je Teil ein Kritiker aus einer anderen Modellfamilie als
  der Autor, parallel und mit getrennten Dateien. Gefunden wurden so zwei echte Lecks, die alle eigenen Tests bestanden
  hatten: der SVG-Export trug das ganze Original samt weggeschnittenen Bereichen (F-60), und ein Vorschaubild überlebte
  seine Aufnahme (F-51).
- **Weggeschnitten ist beim Weitergeben auch ein [S]-Fall**, nicht nur geschwärzt: Jeder Export-Weg braucht einen Test,
  dass Pixel außerhalb der Ansicht fehlen.
- **Ein [S]-Test, der den Bereich zusätzlich wegstaucht, prüft die sichtbaren Wege nicht** — sie sind dann trivial
  sauber. Die Gegenprobe (Schwärzen wirkungslos) muss in jedem geprüften Weg rot werden.

- **Eine Gegenprobe, die nicht baut, ist kein roter Test** (Sitzungs-QA Sprint 3): Wer nur `--- FAIL` zählt,
  übersieht Baufehler (ungenutzte Variable nach dem Rückbau) und hält einen ungeprüften Fix für belegt — oder umgekehrt.
  Die Ausgabe lesen; die Mutation so wählen, dass sie baut (z. B. `return true` statt Zeile löschen).
- **Ein Filter auf geschätzten Grenzen braucht ein Obermaß** (F-70): Wer anhand einer Schätzung löscht oder weglässt,
  muss die Schätzung großzügig nach außen runden.

## Recherche

- **KWin ScreenShot2** gibt Pixel roh über eine Pipe; eine Berechtigung (installierte `.desktop`-Datei mit absolutem
  `Exec=`) verlangt KWin nur bis Plasma 6.7 — ab 6.8 ist die Prüfung entfernt (F-15, `topics/kwin-screenshot.md`).
  Lehre: Weicht ein Messwert von der Doku ab, im Quelltext-Verlauf (Commits, Tags) suchen, nicht nur in der Doku.
- **Plattformregeln zu KI-Code ändern sich schnell** (Codeberg Verbot Juli 2026, Flathub Offenlegung Okt. 2026) — vor
  jeder neuen Plattform aktuell prüfen (`topics/plattformen-und-ki-regeln.md`).

## Prüfen und Belegen (Sprint 1)

- **Zahlen in ADRs und Findings nur mit Zählbefehl** eintragen und den Befehl daneben nennen (F-20 → F-27: der
  Patch-Umfang war zweimal falsch, weil aus dem Gedächtnis eines Zwischenstands übernommen).
- **Messgröße passend zur Frage wählen:** Bildrate beim Ziehen von Hand enthält Mauspausen; Ruckeln zeigt der Stau
  (Ereignisse je Frame) — `topics/gio-wayland.md`.
- **Statusspalte im Findings-Register beginnt mit dem Status** („behoben …“, „akzeptiert …“, „backlog …“) — die
  Werkzeuge (`findings-karten-abgleich`, `findings-rotate`) erkennen ihn nur am Anfang. Register nur über
  `register-schreiben.mjs` ändern; das Register braucht die Abschnitte `## Offen` (und optional `## Behoben / Akzeptiert`).
- **Verify-Kette beenden:** Berührt die Behebung eines Verify-Befunds wieder einen `[S]`-/`[K]`-Weg, ist ein weiterer
  Lauf Pflicht; Kleinstbefunde danach lieber ohne Code-Änderung triagieren (backlog mit Karte), statt eine neue Runde
  auszulösen.

## Arbeiten am Rechner des Maintainers

- **Löschen:** eigener Werkzeugaufruf, vorher `ls`, Ziel im Chat nennen, `rm -rv --one-file-system -- /voller/pfad`
  (nicht `-rf`); in Skripten statt Löschen nach `tmp/papierkorb/` verschieben (`extern/gio-patch.sh`).
- **Vollbild- und Aufnahme-Tests** vorher ankündigen — sie liegen über allen Monitoren und nehmen echte Inhalte auf;
  Aufnahmen nur im Speicher oder unter `tmp/`, nie im Repo.
- **Rechte-Prüfung lehnt ab, was Berechtigungen erweitert** (z. B. eine `.desktop`-Datei mit
  ScreenShot2-Freigabe): nicht umgehen, Vorlage im Repo ablegen und Andi fragen.
- **Board:** alles selbst erledigen (auch Sprints anlegen) — nie „Andi: im Browser anlegen“.
- **Echte Läufe mit Andi (Overlay, Sprint 4):** selbst starten mit `timeout 90 … > tmp/…out 2> tmp/…err` statt ihn
  einen `!`-Befehl tippen zu lassen — dessen Ausgabe kommt erst am Ende, nach 120 s wandert er in den Hintergrund und
  hängt dort unbemerkt weiter. Vorher in einem Satz sagen, **was** er tun und **worauf** er achten soll (je Lauf eine
  Sache); sonst übersieht er Einzelheiten und der Lauf muss wiederholt werden. Inhalte seiner Aufnahmen nie ansehen,
  nur Größe (`file`).
- **Nebenläufigkeit erst im echten Lauf sichtbar:** Unit-Tests ohne Fenster fanden weder den geteilten Text-Setzer
  noch den Wettlauf beim Schließen; ein `go build -race`-Lauf fand ihn sofort. Zeiten aber nie mit `-race` messen
  (Faktor 10–30).
