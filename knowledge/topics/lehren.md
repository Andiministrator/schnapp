# Lehren aus der Arbeit an Schnapp

Stand: 2026-11-20 (ergänzt in Sprint 6). Kurze Regeln aus Fehlern und Befunden; Einzelheiten in `docs/findings.md`.

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
  (nicht `-rf`); in Skripten statt Löschen nach `tmp/papierkorb/` verschieben (`third_party/gio-patch.sh`).
- **Vollbild- und Aufnahme-Tests** vorher ankündigen — sie liegen über allen Monitoren und nehmen echte Inhalte auf;
  Aufnahmen nur im Speicher oder unter `tmp/`, nie im Repo.
- **Rechte-Prüfung lehnt ab, was Berechtigungen erweitert** (z. B. eine `.desktop`-Datei mit
  ScreenShot2-Freigabe): nicht umgehen, Vorlage im Repo ablegen und Andi fragen.
- **Board:** alles selbst erledigen (auch Sprints anlegen) — nie „Andi: im Browser anlegen“.
- **Echte Läufe mit Andi (Overlay, Sprint 4; geändert 19.11.2026 abends):** Andi startet selbst — Claude gibt einen
  fertigen `!`-Befehl und schreibt **davor** nummeriert, was er tun und worauf er achten soll (je Lauf eine Sache) und
  wie er beendet. Grund: Ein von Claude gestartetes Fenster lief schon, bevor er die Anleitung gelesen hatte, das
  Ergebnis blieb unsicher (F-78). Weil ein `!`-Befehl nach 120 s in den Hintergrund wandert, den Lauf mit
  `timeout 110` begrenzen und stderr in eine Datei unter `tmp/` leiten; die wertet Claude danach aus. Inhalte seiner
  Aufnahmen nie ansehen, nur Größe (`file`).
- **Probeläufe ohne Andi (20.11.2026, Andi: „ich muss eigentlich arbeiten“):** Was ohne Augen prüfbar ist (Aufnahme
  über D-Bus, Overlay-Weg, Ablegen, stdout/stderr, Exit-Codes, Bibliothek), fährt Claude selbst mit
  `scripts/probe.sh [--monitore N] [--skala S] [-- ARGUMENTE]`: virtueller KWin auf privatem D-Bus; KWin und Schnapp
  mit HOME, XDG-Ordnern und Laufzeitordner unter `tmp/probe/` (die echte KWin-Konfiguration bleibt unberührt), Build-Tag
  `probe` (Overlay wählt den ersten Monitor ohne Eingabe; nie ausliefern), Zeitgrenze `timeout -k 5 90`. Der echte
  Desktop wird dabei nie aufgenommen; der innere Teil bricht ab, wenn er nicht auf dem privaten Bus läuft. Ergebnisse
  werden vor jedem Lauf geleert — ein Ergebnis ohne frischen Lauf war ein angreifbarer Beleg (Kritik Fable, F-175).
  Die Probe-Programme selbst (Schnapp mit Tag `probe`, Fernbedienung) verweigern außerhalb des virtuellen KWin den
  Start (`internal/probeschutz.NurVirtuell`: Marke, privater Bus, Socket `schnapp-probe`, Laufzeitordner unter
  `tmp/probe/run`, alle zugleich; F-229) — ein liegen gebliebenes `tmp/probe/schnapp` nimmt so nie den echten Monitor
  auf. Trotzdem nie von Hand starten. `probe.sh` biegt auch `XDG_STATE_HOME` um und entfernt `WAYLAND_SOCKET` (F-230).
  Andi bleibt für das, was nur auf seinem Rechner geht: Aussehen, echte Monitore mit gemischter Skala, Klipper,
  Tastatur im Editor — gebündelt (Karte #92), nicht Lauf für Lauf.
- **Tests, die Claude selbst fahren kann, fährt Claude selbst (Andi, 20.11.2026, zum zweiten Mal gefragt: „kannst du
  die tests nicht selbst machen?“ — „merke dir mal, dass das geht“):** Vor jeder Prüfliste für Andi jeden Punkt
  einteilen in „fahre ich selbst“ und „braucht Andi“; die eigenen zuerst fahren und auswerten, Andi bekommt nur den
  Rest. Auch der Editor ist selbst prüfbar: Der virtuelle KWin bietet `org_kde_kwin_fake_input` an, und KWin winkt
  `authenticate` ohne Prüfung durch (`kwin/src/backends/fakeinput/fakeinputbackend.cpp`, „TODO: make secure“). Damit
  spielt die Probe-Fernbedienung (Karte #95, `scripts/probe.sh --drehbuch`) Zeiger, Klicks, Ziehen, Tasten und Text
  gegen den echten Editor; Bildschirmfotos (ScreenShot2) sieht Claude mit dem Read-Werkzeug an und prüft Pixel per
  Einzeiler. `wl-paste`/`wl-copy` gehen gegen die Zwischenablage des virtuellen KWin (`--zwischenablage`), eigene
  Prüfskripte laufen mit `--innen SKRIPT`. Für Andi bleibt, was der virtuelle KWin nicht hat: Desktop-Portal-Dialoge,
  Klipper und Plasma, Einfügen in andere Programme, seine echten Monitore und Skalen, rein subjektives Aussehen.
  **Fernbedienung (Karte #95):** `scripts/probe.sh --drehbuch DATEI [--bild BILD] [--breite W --hoehe H] [--zeit S]`
  spielt ein Drehbuch (`warte`, `zeiger`, `klick`, `doppelklick`, `ziehe`, `taste Strg+Z`, `schreibe`, `rad`, `foto`;
  Beispiel `scripts/drehbuecher/beispiel-schwaerzen.txt`) gegen den Editor, der im Probe-Build im Vollbild öffnet
  (Fensterkoordinaten = Bildschirmkoordinaten, Bild bei 100 % mittig); Fotos unter `tmp/probe/fotos/<Zeitstempel>/`.
  ⚠️ Der virtuelle KWin erbt sonst das Tastaturlayout des Rechners (de: Y/Z vertauscht, Strg+Z blieb still wirkungslos);
  `probe.sh` setzt `XKB_DEFAULT_LAYOUT=us`, die Codes der Fernbedienung sind US-evdev. Die Oberfläche erscheint in der
  Sprache des Rechners. Kein Portal-Dialog, keine fremden Programme im virtuellen KWin.
  **Rückfallproben:** die Drehbücher des Echtlaufs #92 liegen als `scripts/drehbuecher/rueckfall-*.txt` mit dem
  synthetischen `testbild-1200x800.png` im Repo (Aufruf steht im Kopf jeder Datei) — nach Umbauten an `frame()`,
  `zeiger()` oder der Statusleiste fahren. **Drehbuch-Fallen (Echtlauf #92):** ein Zug, der außerhalb des Bilds beginnt, startet absichtlich nichts (F-150) —
  Bänder und Rahmen im Bild beginnen; `taste Esc` verwirft einen laufenden Text, bestätigt wird mit `taste Enter`;
  Rahmen unter 6 Fensterpixeln tun nichts (`MinObjekt`); im 4K-Lauf (Race-Build) braucht ein Neuaufbau über eine
  Sekunde — `warte 3000` vor dem Foto, sonst zeigt es den alten Stand. Ein „falsches“ Foto erst gegen das Drehbuch
  prüfen, dann gegen den Code; umgekehrt zeigt ein Foto, das nach einer Taste plötzlich stimmt, ein fehlendes
  Neuzeichnen (F-216, F-217). Was die Fernbedienung nicht kann: Maustaste halten und fotografieren (Rahmen beim
  Ziehen). Ein `rad`-Schritt zoomt nur um rund 3,5 % (30 Schritte: 99 % → 34 %). **Feste Klickkoordinaten veralten
  mit jeder Änderung an der Leiste** (nach „Ganzes Bild“ trafen zwei Drehbücher „Einpassen“ statt „Exportieren“): nach
  einem neuen Knopf alle `klick`-Zeilen in `scripts/drehbuecher/` fahren und per Foto prüfen, dass der gemeinte Knopf
  getroffen wird.
- **Schutz messen, nicht annehmen (Sprint 6, F-163):** Verpixeln mit Blockmittel plus Zufallsrauschen sah sicher aus;
  der Kritiker hat es mit einem Toleranzvergleich gegen Kandidaten nachgerechnet und eine vierstellige Zahl in 193 von
  200 Läufen zurückgewonnen. Was vertraulichen Inhalt unkenntlich machen soll, trägt keine Information des Inhalts
  (Mosaik aus dem Gesamtmittel, ADR-0022 Nachtrag 1) — und der Test belegt das (gleiches Mittel → gleiches Mosaik).
- **Großzügig ist nicht immer die sichere Seite (Sprint 6, F-181):** Eine zu große Schätzung der Textfläche ist sicher
  fürs Rechenbudget, aber unsicher fürs Weglassen — der SVG-Export hielt einen unsichtbaren Text für sichtbar und schrieb
  ihn im Klartext hinein. Wo eine Grenze über Weglassen entscheidet, die echte Fläche messen (`document/schrift.go`).
- **Sub-Agent-Karten bekommen einen Kritiker (Sprint 6):** #38 kam grün und mit vielen Tests zurück und hatte trotzdem
  einen [S]-Rückschritt (F-181); erst der Fable-Lauf fand ihn. Grün heißt nicht geprüft.
- **Löschen nur nach Stand-Prüfung (Sprint 6, F-167):** Mit fester Uhr bekam eine neue Aufnahme den Namen einer
  verschobenen alten Datei; nur der Vergleich von Gerät, Inode, Größe und Zeit hat verhindert, dass `Entfernen` die neue
  Datei löscht. Wer löscht, prüft die Identität der Datei, nicht ihren Namen.
- **Nebenläufigkeit erst im echten Lauf sichtbar:** Unit-Tests ohne Fenster fanden weder den geteilten Text-Setzer
  noch den Wettlauf beim Schließen; ein `go build -race`-Lauf fand ihn sofort. Zeiten aber nie mit `-race` messen
  (Faktor 10–30).
- **Rastern nur im Ausschnitt** (Renderer #58, 19.11.2026): `vector.NewRasterizer` in Bildgröße je Objekt kostete
  beim Speichern Sekunden; das umschließende Rechteck je Füllung reicht (4K, 20 Objekte ~80 ms). Ein Benchmark
  (`BenchmarkGrossesBild`) hält das fest. Speichern beim Schließen läuft nach dem Fensterschluss weiter — ein zu
  knapper `timeout` im Testbefehl bricht es ab, ohne dass man es sieht.
- **Worktree-Agenten können `HOME` nicht setzen** (Sandbox): Installations- und Bibliothekstests dort über
  `PREFIX`/`XDG_DATA_HOME` bzw. eigene Umgebung im Testcode, Läufe mit Test-HOME macht der Orchestrator.
- **Register-Status nur aus dem Vokabular** (`sofort`, `eingeplant`, `backlog`, `akzeptiert`, `behoben`):
  `findings-rotate.mjs` rotiert „widerlegt“ nicht; die Nuance gehört in den Text. Senkrechte Striche im Prüfbefehl
  zerreißen die Tabelle — umformulieren.
- **Kritiker mit anderem Modell lohnt sich** (Fable, Sitzungs-QA Sprint 4 Teil 2): fand einen Datenverlust (Datei
  während des Editierens weg) und eine Speicherfalle (Schriftgrad 1e5) trotz grüner Tests — Proben im eigenen
  `tmp/`-Modul mit `GOPROXY=off` statt Änderungen im Repo.

## Sprint-Abschluss 4 (19.11.2026)

- **Ein Schutz darf keine echte Arbeit blockieren:** Ein Rechenbudget gegen präparierte Dateien, das beim *Speichern*
  abbrach, lehnte schon 9 lange Pfeile auf zwei 4K-Monitoren ab — mit dem Speichern beim Schließen (F-107) drohte
  Verlust (F-131). Schutz gegen fremde Eingaben gehört an den Eingang (Öffnen), nie an den Ausgang einer Bearbeitung.
  Testfälle für „realistisch“ müssen alle Objektarten enthalten, nicht nur die billigen.
- **Ein Kostenmodell braucht Grundkosten:** Grad² je Glyphe und Fläche je Füllung übersehen feste Kosten (winzige
  Glyphen ~0,5 µs, Polygone mit riesigen Koordinaten). Kostenmodell und Rendern über denselben Code (Trockenlauf)
  führen, sonst laufen sie auseinander.
- **Jede Nacharbeit an [S]/[K] braucht einen eigenen Verify-Lauf:** Kritiker → Nacharbeit → Verify → Nacharbeit →
  Verify fand in jeder Runde echte Fehler in der *Nacharbeit* (Vorschub bei Emoji, Budget-Lücken, zu enges Budget).
  Die Kette endet erst, wenn eine Runde nur Doku/Tests anfasst — im Sprint-Abschluss 4 brauchte es vier Runden
  und eine offene fünfte (F-136).
- **„Gegenproben rot“ je Fix-Zeile belegen:** Ein Sammelsatz im Commit deckte drei von sieben Fix-Zeilen nicht
  (Verify V-4, V2-6). Gegenprobe heißt: genau die eine Zeile zurückbauen, prüfen, dass der Test den Zweig betritt.
- **`Co-Authored-By` ohne Leerzeile dazwischen:** Eine Leerzeile beendet den Trailer-Block; git liest dann nur die
  letzte Zeile. Prüfen: `git log -1 --format='%(trailers:key=Co-Authored-By,valueonly)'`. Ungepushte Commits lassen
  sich mit gleichem Baum per `git commit-tree` neu bauen.
- **`eigenerOrdner` heilt Rechte:** Ein Test, der einen Monatsordner auf 0500 setzt, um das Zurückschreiben scheitern
  zu lassen, sieht das Ausweichen trotzdem gelingen — `Ablegen` bringt den Ordner wieder auf 0700 (ADR-0016).
- **Messfälle auf winzigen Bildern unterschätzen sichtbare Arbeit:** Glyphen auf 10×10 kosten nur das Rastern der
  Maske; sichtbar aufs Bild gezeichnet kosten sie bis 20- bis 30-mal mehr (Sitzungs-QA). Kostenmodelle immer auch mit
  sichtbaren Objekten auf großem Bild messen.
- **Ein Trockenlauf zählt nur, was im Zeichner liegt:** Die Abbildung `ZuAnsicht` mit vielen Streifen kostet je Punkt
  O(Streifen) und taucht im Modell nicht auf (F-136). Kosten vorgelagerter Schritte gesondert prüfen.
- **Agenten schreiben Testdateien gern per Heredoc** (`cat > datei <<EOF`), obwohl der Prompt es verbietet: Das
  Verbot ausdrücklich auch für Quelltext nennen und den Weg (Write-Werkzeug) dazu.
- **… und `sed -i` auf Quelltext ebenso** (Sprint 5: in fast jedem Agentenlauf trotz Verbot). Im Prompt beides
  nennen samt erlaubtem Weg (Edit/Write), im Bericht den Abschnitt „Lokale Befehle“ daraufhin lesen.

## Sprint 5 (20.11.2026)

- **Ein Echtlauf belegt nur den Stand, auf dem er lief.** Nach einer Nacharbeit an der Gio-Schicht (Tasten, Fokus,
  Statusleiste) den Lauf wiederholen oder den Prüfpunkt als Karte führen — „behoben, Echtlauf offen“ im Archiv geht
  verloren (F-151, F-153).
- **Belege offener Befunde gehören ins Repo.** Ein Prüfbefehl, der auf `tmp/…` zeigt, ist nach dem Aufräumen wertlos
  (F-140, wieder F-150/F-156/F-157/F-159). Roten Beleg-Test eines Kritikers im Fix-Commit als Test übernehmen oder den
  Prüfbefehl als Verhalten formulieren; kein `<Name>`-Platzhalter.
- **`git cherry` zeigt „+“ nach Zusammenführen.** Musste der Cherry-Pick mit anderer Arbeit zusammengeführt werden,
  bleibt die Patch-ID anders. Nachweis dann: geänderte Zeilen beider Commits vergleichen
  (`diff <(git show --format= ALT | grep '^[+-]' | grep -v '^+++\|^---') <(… NEU …)` leer).
- **Echtlauf-Anleitung: stderr nicht verstecken.** `2> tmp/…log` verbirgt genau die Meldungen, die der Lauf prüfen soll
  (Ablageort, Warnungen). Dann `2>&1 | tee tmp/…log` nehmen oder Andi sagen, dass die Meldung in der Datei steht.
- **Strg+C-Proben nur im eigenen Terminal.** Ein per `!` gestartetes Programm hat kein eigenes Terminal; Signalproben
  (F-152) fährt Andi in einer eigenen Konsole. Im Test: injizierter Kanal statt echter Signale.
- **Speichern an anderem Ort muss am Ort der Handlung sichtbar sein.** „Datei von außen wird neue Aufnahme“ stand in
  ADR und README und verwirrte trotzdem („Bild war weg“, F-155). Jeden Speicherweg (Strg+S, Schließen, Signal) einzeln
  durchgehen (F-158).
- **Register-Werkzeug und `|`:** `register-schreiben.mjs` bricht bei unmaskiertem `|` in einer Zelle ab (auch in
  `-run 'A|B'`). `\|` schreiben; neuen Text als Datei unter `tmp/` ablegen (`--neu @datei`), nie über die Shell quoten.
- **Eine Karte, die auf eine Entscheidung wartet, startet nicht** — oder die Entscheidung wird ausdrücklich getroffen
  und in ADR und `docs/open-decisions.md` als „Claudes Entscheidung, Andi informiert“ vermerkt (#85).
- **Schulden-Grenze im Blick:** Kritiker- und Verify-Läufe erzeugen P3-[K] im Dutzend (Sprint 5: Stand 10/10). Vor dem
  Eintragen prüfen, ob [K] oder eher [V]/[D], und `schulden-grenze.mjs` nach jedem Register-Commit laufen lassen.
- **Lange verkettete Bash-Befehle mit Push oder Löschen** lehnt die Berechtigungsprüfung ab: Push und Löschen je in
  einem eigenen Aufruf, vor dem Push die Heikle-Begriffe-Prüfung.
- **Sitzungs-QA Sprint 6, zweiter Teil (20.11.2026, Kritik Fable A/B, F-190 bis F-208):**
  - *Worktree-Agenten starten vom gepushten Stand* (origin/main), nicht vom lokalen HEAD: vor jedem Worktree-Agenten
    pushen, sonst fehlt ihm der eigene letzte Commit (#95, erster Anlauf abgebrochen).
  - *Eine Gegenprobe deckt nur den Fehlerweg, den sie nachbaut.* „Scheibe rot“ prüfte die Obergrenze; das vorzeitige
    Ende des Halters (1 min statt 12 h) blieb grün (F-199). Zeitgrenzen immer mit Unter- und Obergrenze und
    einspeisbarer Uhr testen.
  - *Wächter: Identität beim Start, nicht nach der Wartezeit,* und vor jedem Eingriff den Namen gegen das Literal
    prüfen (F-200). Ein Werkzeug, das Eingaben einspeist, prüft selbst, dass es im virtuellen KWin läuft — der Schutz
    der Hülle allein reicht nicht, das gebaute Programm bleibt liegen (F-201).
  - *„Der Dialog bestätigt das Überschreiben“ gilt nur für den Namen, den er zurückgibt* — wer danach eine Endung
    anhängt, muss selbst prüfen (F-190). Feste Dateirechte (`Chmod 0644`) umgehen die umask (F-192).
  - *Geschlossene Kanäle aus Bibliotheken* (godbus schließt Signalkanäle beim Verbindungsende) mit `v, ok := <-ch`
    lesen, sonst Dauerschleife (F-191).
  - *D-Bus-Signale tragen als Absender immer den eindeutigen Namen* (`:1.x`), nie den Wellknown-Namen — eine
    Absenderprüfung braucht den Besitzer, und der kann beim Anmelden noch fehlen (Aktivierung erst durch den Aufruf):
    nachschlagen statt verwerfen (F-209). Der Verify-Lauf nach der Nacharbeit fand 6 neue Befunde in 16 Behebungen —
    er ist nicht verzichtbar.
