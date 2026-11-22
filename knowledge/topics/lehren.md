# Lehren aus der Arbeit an Schnapp

Stand: 2026-11-22 (ergänzt nach Sprint 10 und der ersten Sitzung von Sprint 11). Kurze Regeln aus Fehlern und Befunden; Einzelheiten in `docs/findings.md`.

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
  Start vor Aufnahme, Editor und Zwischenablage-Halter (`--version`, `--help` und `--list-screens` laufen ohne Prüfung;
  `--list-screens` fragt dann den echten Compositor nach Monitornamen, nimmt aber nichts auf) (`internal/probeguard.NurVirtuell`: Marke, privater Bus, Socket `schnapp-probe`, Laufzeitordner unter
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
  einem neuen Knopf alle Drehbücher in `scripts/drehbuecher/` fahren und per Foto prüfen — die `klick`-Zeilen auf
  Knöpfe ebenso wie `ziehe`/`doppelklick` auf der Leinwand: Bricht die Leiste in eine weitere Zeile um, rückt das Bild
  nach unten (1600 px: 22 px, Bild ab y=136), und ein Zug über der ersten Bildzeile startet still nichts (F-150, F-238). Beim Verschieben nicht mechanisch
  aufschlagen: Ein Zug, der schon vorher außerhalb lag, bleibt es (Schritt 8 von `rueckfall-zuschnitt-stauchen`, F-248);
  zwei byte-gleiche Fotos hintereinander (`cmp`) zeigen, dass ein Schritt nichts bewirkt hat.
- **Belege aus Sub-Agent-Worktrees sichern, bevor der Worktree abgeräumt wird (F-246, 20.11.2026):** `git worktree
  remove` nimmt das gitignorierte `tmp/` des Worktrees mit — Messprotokolle, Fotos, Gegenproben. Vor dem Abräumen
  nach `tmp/<sprint>/` im Hauptbaum kopieren oder den Agenten anweisen, Belege dort abzulegen.
- **Schutz messen, nicht annehmen (Sprint 6, F-163):** Verpixeln mit Blockmittel plus Zufallsrauschen sah sicher aus;
  der Kritiker hat es mit einem Toleranzvergleich gegen Kandidaten nachgerechnet und eine vierstellige Zahl in 193 von
  200 Läufen zurückgewonnen. Was vertraulichen Inhalt unkenntlich machen soll, trägt keine Information des Inhalts
  (Mosaik aus dem Gesamtmittel, ADR-0022 Nachtrag 1) — und der Test belegt das (gleiches Mittel → gleiches Mosaik).
- **Großzügig ist nicht immer die sichere Seite (Sprint 6, F-181):** Eine zu große Schätzung der Textfläche ist sicher
  fürs Rechenbudget, aber unsicher fürs Weglassen — der SVG-Export hielt einen unsichtbaren Text für sichtbar und schrieb
  ihn im Klartext hinein. Wo eine Grenze über Weglassen entscheidet, die echte Fläche messen (`internal/document/schrift.go`).
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
- **Register-Status nur aus dem Vokabular** (`sofort`, `eingeplant`, `backlog`, `akzeptiert`, `behoben`; dazu
  `teilweise behoben <Commit>: … Offen: …` für ein Finding mit benanntem Rest, das offen bleibt, sowie `hinfällig
  <Datum>: <Grund>` und `geschlossen <Datum>, <Grund>` für Funde ohne Code-Änderung — Nachtrag Sprint 10, F-326):
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
- **Register-Werkzeug und `|`:** `register-schreiben.mjs` bricht bei unmaskiertem `|` in einer Zelle ab. Mit `\|`
  maskiert ist die Tabelle heil, aber der Prüfbefehl falsch: Aus der Rohdatei kopiert führt `go test -run 'A\|B'` null
  Tests aus und meldet „ok“, `grep -E 'a\|b'` sucht einen wörtlichen Strich (nachgefahren 21.11.2026 an F-284). Prüfbefehle
  deshalb ohne Strich schreiben: gemeinsamer Test-Präfix oder zwei Aufrufe, bei grep mehrere `-e` (Hinweis im Kopf von
  `docs/findings.archive.md` für die älteren). Neuen Text als Datei unter `tmp/` ablegen (`--neu @datei`), nie über die
  Shell quoten.
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
  - *Doppelter Schutz macht den Rückbau einer Zeile grün* (Sprint 7, #106): Beim Schwärzen klemmen `klemmeAnsicht` und
    die Rundung in `schwaerzeRahmen` den Rahmen beide auf die Ansicht; nimmt die Gegenprobe nur eine zurück, bleibt der
    Test grün, ohne dass er wertlos wäre. Dann gegen den Stand vor dem Fix zurückbauen (`git show <fix>`), nicht gegen
    eine einzelne Zeile. Und: Ein Zug, der außerhalb des Fensters beginnt, startet nicht — Drehbuch und Test drücken im
    Bild und ziehen hinaus, nicht umgekehrt.
  - *Ein einzelner „wirkungsloser“ Klick im Foto ist noch kein Befund — wiederholen und eingrenzen:* In Sprint 7
    schien ein Druck direkt nach einem Zeigersprung an einem Leistenknopf nicht anzukommen (F-251); in Sprint 8 war das
    nicht nachzustellen (`scripts/drehbuecher/rueckfall-klick-ohne-zeiger.txt`, 5 von 5, `knowledge/messungen/f-251.md`).
    Die Ursache blieb unbelegt: `klick X Y` sendet selbst eine Bewegung vor dem Druck, und das schon damals (Fernbedienung
    unverändert seit 026b4ad); offen ist der Zeitabstand (15 ms). Deshalb vor einem Register-Eintrag mehrfach fahren, bei
    Wiederkehr zuerst `zeiger X Y` und `warte 200` vor dem Klick probieren (schadet nie, bleibt in den älteren
    Drehbüchern) und gegen den alten Stand eingrenzen per `git worktree add --detach tmp/<sprint>/<lauf>/alt <commit>` und
    dessen eigenem `scripts/probe.sh`.
  - *Eine Sperre prüft jeden Eingabeweg* (Sprint 7, F-257/F-261/F-263): Die Nacharbeit sperrte bei offenem Menü die
    Tasten (`key.Event`), der Zweig für getippten Text (`key.EditEvent`) lag davor und blieb offen; gefunden hat es erst
    der Verify-Lauf. Wer eine Eingabe sperrt, testet Taste, getippten Text und Maus einzeln; Gio-Tastenlogik ist mit
    `input.Router` ohne Fenster testbar (Vorbild `internal/ui/menue_tastatur_test.go`, Karte #110).

## Sprint 8 (20.11.2026)

- **Eine Sperre in Gio holt Ereignisse ab und verwirft sie — sie hört nicht auf zu fragen** (F-143, c28c793): Wer
  während einer Sperre `gtx.Event` gar nicht mehr aufruft, meldet in diesem Bild keine Filter. Gio nimmt einem Ziel
  am Bildende den Tastaturfokus, wenn es keinen `key.FocusFilter` mehr gemeldet hat (`keyQueue.Frame`,
  `third_party/gio/io/input/key.go`); nicht abgeholte Ereignisse verwirft `Router.Frame` am Bildende
  (`third_party/gio/io/input/router.go`). Also Filter wie immer anmelden, Ereignis abholen, dann `continue`. Der Test
  prüft nach dem Ende der Sperre, dass die Leinwand den Fokus noch hat und eine Taste wieder wirkt
  (`internal/ui/hintergrund_test.go`).
- **Zustand, der je Bild einmal gelesen wird, gilt nicht für das ganze Bild** (F-280): `an := !SpeichernLaeuft()` am
  Anfang der Knopfauswertung war schon falsch, als der erste Knopf im selben Bild das Speichern startete — der zweite
  Knopf änderte noch. Sperren je Ereignis frisch lesen, und die Sperre gehört in die Logikschicht (`canvas`), nicht
  nur in die Oberfläche.
- **Arbeit im Hintergrund nur an einem Abzug, und der Verlauf entscheidet mit** (ADR-0020 Nachtrag 6, ADR-0022
  Punkt 3): Speichern in einer Goroutine braucht eine unabhängige Kopie (`document.Dokument.Abzug`) und eine
  Bearbeitungssperre. Nach einem Speichern mit Schwärzung wird der Verlauf geleert; liefe daneben eine Bearbeitung,
  gälte das Leeren für einen anderen Stand, und ein Rückgängig holte das Geschwärzte zurück. Bei jedem Schritt ins
  Nebenläufige zuerst fragen: Welche Zusage hängt am „gespeicherten Stand“?
- **Aus synchron wird asynchron: Was wartet jetzt nicht mehr?** (F-281, F-283) Das synchrone Kopieren war vor dem
  Schließen fertig, das asynchrone nicht — die Kopie ging still verloren. Umgekehrt wartete das Schließen ohne Frist
  auf ein hängendes Speichern. Für jede Hintergrundarbeit festlegen: Wer wartet beim Schließen, beim Signal, beim
  Zerstören, und wie lange?
- **Ein Hardlink ist ein zweiter Name, kein Zwischenschritt** (F-48, F-274): Exklusives Anlegen per `os.Link` ist
  atomar, aber bis zum Entfernen zeigt die temporäre Datei auf denselben Inhalt. Bleibt sie liegen (Absturz,
  scheiterndes Entfernen), übersteht sie ein späteres Schwärzen, weil Speichern per Umbenennen einen neuen Inode
  anlegt. Wer einen [K]-Befund mit einer neuen Dateisystem-Technik behebt, prüft im selben Zug den [S]-Pfad:
  Welche Namen und Kopien des Inhalts gibt es jetzt, und wer räumt sie?
- **Ein schneller Leser urteilt wie der volle, oder er sagt, wo nicht** (F-272, F-273): `LeseMetadaten` nahm Dateien
  an, die das volle Lesen ablehnt, und meldete eine andere Größe. Gleiche Grenzen heißt: derselbe Prüfcode
  (`document.PruefeInhalt`), nicht ein Nachbau. Und: Eine angekündigte Länge reserviert nie Speicher, bevor die
  Bytes da sind — auch nicht in einem Leser, der „nur Köpfe“ liest.
- **„Alles verwaist“ ist fast immer ein fehlender Ordner** (F-270, F-271): `filepath.WalkDir` steigt an einer Wurzel,
  die ein Symlink ist, nicht ab, und ein ausgehängtes Laufwerk hinterlässt einen leeren Einhängepunkt. Listen, auf
  denen später gelöscht wird, liefern bei leerer Quelle einen Fehler statt „alles“ (`ErrAblageLeer`); die Wurzel
  vorher mit `filepath.EvalSymlinks` auflösen.
- **Ein Test, der sich selbst überspringt, belegt nichts**: `t.Skip`, wenn die Vorbedingung nicht eintritt („Lage
  geschätzt“), sieht in der Ausgabe wie grün aus. Die Vorbedingung herstellen (Knopfmitte messen) und mit `t.Fatal`
  verlangen; nach neuen Tests `go test -v` nach `SKIP` durchsehen.
- **Worktree-Agenten: Auch nach dem Push den Stand im Prompt nennen** (Ergänzung zur Lehre aus Sprint 6): Der Worktree
  lag in dieser Sitzung nicht immer auf dem aktuellen main. Im Prompt den erwarteten Commit nennen (`git log -1`
  als erste Handlung, bei Abweichung die genannten Commits per Cherry-Pick holen) und im Bericht die Basis ausweisen.
- **Heredoc-Verbot: am Diff prüfen, nicht am Bericht** (Ergänzung, Sprint 8): Sonnet-Agenten haben es in drei von fünf
  Läufen versucht, zwei haben es ausgeführt und selbst gemeldet, einen Versuch lehnte der Hook ab. Das Verbot im Prompt zweimal nennen (Auflagen und Arbeitsweise), den Abschnitt
  „Lokale Befehle“ nach `<<` durchsuchen und jede so entstandene Datei am Diff auf verstümmelte Zeichen prüfen.

## Sprint-Abschluss 8 und Sitzungs-QA (21.11.2026)

- **Eine Zusage im ADR gilt erst, wenn jeder Weg dorthin getestet ist:** „Schließen-Anfragen gehen weiter“ stimmte für
  den Fensterknopf, nicht für Strg+W und den Menüeintrag (F-299); „der Exit-Code ist ein Fehler“ stimmte für
  `schnapp DATEI`, nicht nach einer Aufnahme (F-302). Beim Schreiben eines ADR-Satzes die Wege aufzählen (Taste, Menü,
  Fensterknopf, Signal; jeder Aufrufweg in `cmd/schnapp`) und je Weg einen Test nennen.
- **Eine Frist gegen die schon gemessenen Höchstwerte halten:** Die 15-s-Frist für „Speichern antwortet nicht“ stand
  zwei Nachträge unter „Dokument am Budget: 35–45 s“ (ADR-0020 Nachtrag 3); 4K braucht 8,8 s. „Hängt“ ist nicht „dauert
  lange“ — eine Frist braucht ein Lebenszeichen (Fortschritt), keine Stoppuhr (F-298, Nachtrag 9). Vor jeder neuen Frist
  im selben ADR nach Zeitangaben suchen.
- **Meldet das Prüfwerkzeug einen Fehler, zuerst seine Frist gegen die Dauer halten:** „schnapp reagiert nicht auf
  SIGTERM“ kam von der festen 5-s-Grenze in `scripts/probe.sh`, nicht von Schnapp (F-249, F-296). Erst die Dauer messen
  (`SCHNAPP_DEBUG`, „Speichern im Hintergrund“), dann den Befund schreiben.
- **Mutationsnachweis je Zweig, nicht je Fix:** Ein neuer Zweig, den kein Produktpfad erreicht, bleibt beim Rückbau grün
  (F-306); ein Test, der eine Größe zählt, die der Fall nie ändert, ebenso (F-307). Für jeden neuen Zweig: Welcher
  Aufruf erreicht ihn, und welcher Test wird rot, wenn er fehlt?
- **Zahlen in Recap und Kartenkommentar aus dem Beleg zählen, und den Beleg aufheben:** In einer Sitzung dreimal falsch
  (Mutationen, Heredoc-Läufe, Kartenzahl im Recap). Karten per `jq`, Commits per `git log --oneline … | wc -l`; die
  Mutationsliste eines Verify-Laufs als Kommentar an die Folgekarte, bevor `tmp/<sprint>/` geräumt wird.
- **Versionsangaben vor dem Tag nachziehen:** v0.2.0 und v0.3.0 tragen im getaggten Baum die README der Vorversion,
  weil „docs: vX in README“ erst nach dem Tag kam. Reihenfolge: Sprint-Abschluss bestanden → README, README.de,
  CLAUDE.md auf die neue Version → dieser Commit wird getaggt (`docs/qa/sprint-abschluss.md`).
- **Anführungszeichen in ADR und Register nur für wörtlich Kopiertes:** Ein sinngemäßes „Zitat“ aus einem anderen
  Nachtrag fiel dem Verify auf (Nachtrag 8) und stand im nächsten Nachtrag wieder (Nachtrag 9). Sinngemäß ohne
  Anführungszeichen, wörtlich per Kopie. Ein neuer Nachtrag setzt kursive Vorwärtsmarker an den Stellen, die er ändert.
- **Grün heißt nicht ohne Befund:** Der CI-Lauf nach v0.3.0 war grün und trug vier Abkündigungs-Annotationen (Node 20,
  F-309). Nach einem Release-Push die Lauf-Seite auf Annotationen ansehen, nicht nur den Status (`gh` ist nicht
  angemeldet: Andi schaut im Browser).
- **Worktree-Agenten sehen `tmp/` des Hauptbaums nicht:** Vorlagen und Prüftests mit absolutem Pfad nennen und vor dem
  Start schreiben. ADR-Nachträge zweier Stränge landen beide am Dateiende — beim Cherry-Pick beide behalten.
- **Was der virtuelle KWin nicht herstellen kann, an der Logik mit Attrappe messen und das sagen:** Ein unsichtbares
  Fenster gibt die Fernbedienung nicht her; die Weckrufe wurden am `Wecker` mit Attrappe gezählt
  (`knowledge/messungen/f-258.md`). Die Messung nennt, was sie nicht belegt (das Verhalten von Gio bei unsichtbarem
  Fenster).
- **Board-JSON mit `jq` lesen, nicht mit `node -e`:** Schnapp hat kein JavaScript; die aBoard-Werkzeuge laufen mit Node,
  weil die Anleitung es so vorgibt — eigene Einzeiler brauchen es nicht (Andi, 21.11.2026).
- **Der Tabu-Hook prüft auch Text, der nur einen Prozesspfad erwähnt:** Ein Kartentext mit dem Pfad der Befehlszeile
  eines Prozesses ließ den Aufruf ablehnen, obwohl nichts gelesen wurde. Op-Nutzlasten mit dem Write-Werkzeug anlegen
  und solche Hinweise umschreiben („keine Befehlszeile lesen“).
- **Probelauf mit `--bild`: ein fremdes PNG hält das Fenster beim Schließen offen.** Es wird beim Speichern als neue
  Aufnahme abgelegt, und der Hinweis darauf verlangt eine zweite Schließen-Anfrage (Nachtrag 4) — ein Drehbuch, das
  „schließt von selbst“ prüft, braucht eine Datei aus der Probe-Bibliothek. Ein nach SIGKILL liegen gebliebenes
  `.schnapp-tmp-*` im selben Ordner löst ebenfalls einen Hinweis aus (`scripts/drehbuecher/rueckfall-schliessen-waehrend-speichern.txt`).
- **Messbilder bestimmen die Messung:** Ein Verlauf speicherte in 4K in 8,8 s, ein verrauschtes Bild in 19–24 s
  (`messungen/f-249.md`, `messungen/f-299-probe.md`). Im Messprotokoll die Art des Bildes nennen und für Fristen den
  ungünstigen Fall (Rauschen, Foto) messen.

## Sprint 9 (21.11.2026)

- **Dauern aus `scripts/probe.sh` sind Race-Zahlen.** Der Probe-Build läuft mit `-race`; Speichern in 4K braucht dort
  8–24 s, im Normalbau 0,4–1,5 s (Faktor 9–22, `messungen/f-297.md`). `f-249.md` nannte fälschlich „Bau ohne
  `-race`“. Dauern im Normalbau per Go-Benchmark messen (`go test -run '^$' -bench Speichern4K ./internal/library`) oder
  ausdrücklich als Race-Zahl führen; die Bauart steht im Messprotokoll, geprüft am Build-Befehl, nicht aus dem Gedächtnis.
- **CI-Annotationen gibt es über die öffentliche API**, ohne Anmeldung: `commits/<sha>/check-runs`, dann
  `check-runs/<id>/annotations`. Gegenprobe gegen den Vorlauf (F-309: vorher 1, nachher 0).
- **Fortschritt dort melden, wo die Zeit vergeht, nicht nur zwischen Schritten.** Zwischen zwei Speicher-Marken lagen im
  Probe-Build bis zu 8 s (eine PNG-Kodierung); erst Pulse aus Schreiber, Leser und Rasterschleife machen eine Frist
  von 15 s tragfähig (F-298, `messungen/f-298-probe.md`). Pulse in heißen Schleifen messen: je Pixel kostete 7 %, je
  64 Zeilen rund 3 % (1,87 statt 1,82 s).
- **Ein zweiter Puls überdeckt den ersten:** Nach dem Puls je Füllung und beim Dekodieren blieben die Mutationen „kein
  Zeilenpuls“ und „kein Leser beim Lesen der Datei“ grün, obwohl beide Wege gebraucht werden (hohe Füllung, langsames
  Medium). Jede Meldestelle braucht einen Test, den nur sie grün hält. Verify F-298: 8 von 20 Mutationen grün trotz
  „Gegenprobe je Zweig“ im eigenen Bericht — die eigene Gegenprobe deckte nur die Zweige, an die man gedacht hatte.
- **Vor einem Wächter-Test die Sperren dahinter zählen.** F-307: Hinter dem Enter-Wächter lagen zwei weitere Sperren
  (`an()` und `canvas.Sitzung.gesperrt`); ein Test kann die äußerste allein nicht rot machen. Dann die Wirkung über
  alle Ebenen prüfen und die Redundanz im Code benennen, statt eine Testnaht ins Produkt zu bauen.
- **bash rechnet eine Zahl mit führender Null oktal.** `case … *[!0-9]*` lässt „08“ durch, `$(( ))` scheitert daran
  oder liest „010“ als 8 (K9-1: `probe.sh --ende 08` schickte sofort SIGKILL). Nach der Prüfung `$((10#$wert))`.
- **Nach einem Umbau der Verzeichnisse auch die Prüfbefehle umziehen.** ADR-0027 stellte Beschreibungen und Code um,
  aber nicht die Prüfbefehle offener Findings, Aufrufbeispiele in Kommentaren und die Kontexte der Board-Karten
  (Kritik Sitzung Sprint 9: fünf Prüfbefehle, 16 Karten). Suche nach `\./<altes paket>\b` und `<altes paket>/` ohne
  `internal/` davor gehört in jeden Umbau.

## Sprint 10 (21.11.2026)

- **Lange TMPDIR-Pfade sprengen die Grenze für Unix-Sockets (107 Byte).** Ein Fernbedienungs-Test und Chromium
  brachen in Worktrees unter `.claude/worktrees/…` mit „bind: invalid argument“ bzw. SIGABRT ab. In Worktrees ein
  kurzes `TMPDIR` nehmen (`tmp/sprint10/<k>/g`, für Chromium relativ); seit F-319 zeigt `cd.py` die Ursache.
- **Schnapp liest keine Prozessverzeichnisse, auch nicht das eigene.** Der Halter fand geerbte Deskriptoren zuerst
  über das proc-Dateisystem; jetzt per fcntl (F-189, f4dacbd). Der Hook lehnt schon Befehle ab, die den Pfad nur nennen.
- **`spectacle -b -c` verliert unter Wayland die Zwischenablage**, weil das Programm sofort endet (F-220-Test). Für
  Vergleiche das Spectacle-Fenster offen lassen — derselbe Grund, aus dem Schnapp den Halter hat.
- **Sub-Agent-Worktrees zweigen vom gepushten Stand ab**, nicht von `main` im Arbeitsbaum. Vor dem Start pushen, sonst
  arbeitet der Agent auf altem Stand (Sprint 10: drei Agenten auf dem Stand vor der Paket-Umbenennung).
- **Sub-Agents setzen trotz Verbot Heredocs ab** (Sprint 10 fünfmal, harmlos; in der Sitzung vom 21.11. nachmittags
  bei vier von vier Sonnet-Karten, sechs Aufrufe, meist Python-Heredocs für Code-Änderungen, obwohl der Prompt es
  ausdrücklich und mit Begründung verbot; die Fable-Kritiker hielten sich daran). Berichte und Diffs darauf lesen (siehe
  „Heredoc-Verbot: am Diff prüfen“ oben); das Verbot steht im Prompt, wirkt aber nicht verlässlich — verlässlich wäre nur ein Hook, der den Aufruf abweist.
- **Wer „behoben“ schreibt, braucht den Beleg im Repo.** Eine Messung des Maintainers (F-295) oder ein Wegwerf-Prüfskript
  unter `tmp/` (F-29, F-319) ist nach dem Aufräumen weg; Zahlen und Ablauf gehören nach `knowledge/messungen/`.
- **Eine Quelle, die nur als Suchtreffer bekannt ist, ist nicht belegt.** Der Gio-Einreichungsweg stand zuerst nach
  Suchschnipseln im Wissen („PR ersatzweise“ klang gleichrangig); erst die geöffnete Seite zeigte, dass die Mailingliste
  der Hauptweg ist (Kritik Sitzungs-QA Sprint 10, C-1).

### Sitzung 21.11.2026 nachmittags (Karten #135, #136, #127, #125, #78, Sitzungs-QA)

- **Im Register den Commit auf `main` nennen, nicht den des Worktrees.** Nach `git cherry-pick` hat der Commit einen
  neuen Hash; der Worktree-Commit verschwindet mit dem Zweig und war nie auf `origin` (Archiv F-313, Doku-Kritik A-1).
  Den Status erst nach dem Cherry-Pick setzen und den Hash aus `git log --oneline -1` nehmen.
- **Backticks in `python3 -c "…"` wertet die Shell aus.** Ein ``` `O_NOFOLLOW` ``` im Ersetzungstext wurde still zu nichts.
  Ersetzungen mit Markdown-Code gehören in eine Skriptdatei (Write nach `tmp/<sprint>/…`, dann `python3 -I -B`), mit
  `assert s.count(alt) == 1` je Ersetzung.
- **Bitgleich zu Fremdcode heißt: dieselbe Rechenreihenfolge.** float32-Addition ist nicht assoziativ; x/image/vector
  akkumuliert auf amd64 per SIMD in Viererblöcken, eine skalare Nachbildung wich in einzelnen Pixeln ab (#78). Feste
  Testfälle fingen eine vertauschte Klammer nicht, erst 600 Zufallsfälle (Code-Kritik K-4). Bei jeder Nachbildung
  Wert-für-Wert-Test gegen die Referenz, mit Zufallsfällen, und die Mutation einer Klammer als Gegenprobe.
- **CPU-Abfragen über `golang.org/x/sys/cpu` folgen `GODEBUG`,** Assembler-Code, der CPUID selbst fragt, nicht (K-3).
  Wer das Verhalten eines Fremdpakets nachbildet, übernimmt dessen Bedingung, nicht eine bequemere.
- **Kritiker im Worktree: Zusatztests als Text in den Bericht.** Dateien, die ein Kritiker unter `tmp/` seines Worktrees
  anlegt, verschwinden mit dem Worktree, bevor der Orchestrator sie übernehmen kann (Zusatzprobe K-4).
- **Bei mehreren parallelen Läufen messen alle schlechter.** Benchmarks des Orchestrators liefen neben den Benchmarks
  eines Sub-Agents; die Vergleichszahlen für ADR-0028 sind deshalb auf `main` und auf dem Zweig zur selben ruhigen Zeit
  nachgemessen worden, nicht aus verschiedenen Zwischenständen zusammengesetzt.

### Sitzung 21./22.11.2026 (Karten #154, #155, #152, #133, #17, Sprint-Abschluss 10)

- **Ein Zustand aus mehreren Merkern verlangt einen Prüflauf je Runde und eine Grenze.** Der Kopierfehler-Halt (drei
  Merker: `kopierFehlerHaelt`, `schliessenNachKopie`, `schliessenNachSpeichern`) bekam in drei Verify-Runden je neue
  Reihenfolgen (F-346 bis F-359). Was half: die roten Prüfer-Tests als echte Tests übernehmen, je geänderte Zeile einen
  Mutanten in ein Skript (damals `tmp/sprint10/k154/mutanten.py`, heute `scripts/mutanten`, Gruppe `ui-kopieren`) und die Reste ab P2/P3 auf eine Karte
  statt in eine weitere Runde. Die Reste zeigen das fehlende Konzept („haltende Meldung“, Karte #156) — ein vierter
  Merker hätte es nicht gelöst.
- **Belege aus Agenten-Worktrees vor dem Abräumen sichern und vergleichen.** Prüfer-Tests und Mutantenskripte nach
  `tmp/<sprint>/verifyN/` kopieren, mit `cmp` gegen das Original prüfen, mit `git cherry main <zweig>` zeigen, dass kein
  Commit verloren geht; erst dann `git worktree remove` und `git branch -D` (ausgeschriebene Pfade).
- **Register-Status nur aus dem Vokabular** (`sofort`, `eingeplant (Sprint N, …)`, `backlog (…)`, `akzeptiert (…)`,
  `behoben …`). „umgesetzt …“ rotiert nie und zählt weder als offen noch als erledigt; `findings-rotate.mjs` meldet es
  mit Exit 2. `register-schreiben.mjs einfuege-nach` verlangt bei mehr als fünf neuen Zeilen `--erwarte-zeilen-delta N`.
- **Erst der Fix-Commit, dann das Register mit dessen Hash.** Ein Status „behoben (Commit mit diesem Eintrag)“ ist nicht
  prüfbar; zwei Commits kosten nichts.
- **Nichts im Namen des Maintainers behaupten, was er nicht getan hat.** Die vorbereiteten Gio-PR-Texte sagten „reviewed
  and tested by me“; Andi hatte den Code nicht durchgesehen. Vor dem Einreichen umformuliert auf das, was stimmt (KI
  schrieb Code, Tests und Text; Andi nutzt den gepatchten Gio täglich).
- **`gh` meldet Andi selbst im eigenen Terminal an** (`gh auth login`, interaktiv, sein Konto; im Chat per `!` nicht
  bedienbar). Danach liegt ein Token mit `repo`-Recht lokal. Ein Fork entsteht mit `gh repo fork --remote --remote-name
  fork` im Klon (mit Repo-Argument lehnt `gh` `--remote` ab). Gepusht wird über Andis SSH-Alias `github`
  (`git remote set-url fork github:Andiministrator/gio.git`); `git@github.com:` nimmt den falschen Schlüssel.
- **Einen Sprint planen, ohne ihn zu starten:** Karten in die Spalte Backlog der neuen Sprint-Swimlane legen. Eine Karte
  außerhalb von Backlog lässt den Sprint als „läuft, aber nicht aktiviert“ gelten (`/projekt-start`).
- **`git rev-parse --short` nimmt genau ein Argument**; mit zweien bricht es ab („Benötigte einen einzelnen Commit“),
  und die Ausgabe sieht aus wie ein Fehler des folgenden Befehls.
- **Zahlen aus dem Beleg, auch die eigenen** (Sitzungs-QA 22.11., Q-1, Q-6, Q-7): „25 Mutanten“ (es waren 23), „37
  Arbeitskarten“ (36; #77 war verschoben, nicht erledigt) und „2 offene [K]“ (3; F-361 kam im selben Recap dazu) standen
  in Commit, Archiv und Recap. Zahl unmittelbar vor dem Schreiben mit dem Prüfbefehl erheben (`grep -c`, `jq … | length`,
  `schulden-grenze.mjs`), nicht aus der Erinnerung an einen früheren Lauf.
- **Eine Testaussage in einem PR-Text braucht je Commit einen Beleg** (Q-11). „Tested on KDE Plasma with outputs at
  integer and fractional scales“ galt für den Patch auf v0.10.3 im täglichen Gebrauch, nicht für die von Hand auf Gio
  `main` umgesetzte Serie und nicht für das Zeigerbild (F-123-Echttest nie gefahren). Vor dem Einreichen jede Aussage
  im Text gegen die Akte halten; offene Andi-Tests einer geschlossenen Karte auf eine offene Karte umhängen.
- **Rote Prüfvorlagen offener Findings liegen nur einmal** (Q-3): unter `tmp/` gitignoriert, auf dem Laptop. Beim
  Kartenstart als Erstes in Tests übernehmen; `tmp/sprint10/verify5`, `kritiker`, `k154`, `gio-einreichung` erst
  räumen, wenn die zugehörigen Karten erledigt sind.
- **Wer eine Karte in einen neuen Sprint legt, zieht Titel, Beschreibung und agentContext nach** (Q-9): #77 hieß noch
  „… erwägen“, obwohl ADR-0029 akzeptiert war; die Entscheidung stand nur in Kommentaren, die ein Sub-Agent nicht sieht.
- **Nach jedem Commit hinter dem ▶-Kommentar diesen erneuern** (Q-8); er nennt Hash und Stand, die sonst veralten.
- **`gh` nur kurz anmelden** (Q-13, F-376, Rat von framework 22.11.): Auf dem Laptop laufen alle Sitzungen als `andi`; jede könnte den Token lesen und an einem Hook vorbei `curl` nehmen. Eine Sperre per Hook schützt nur vor Versehen. Darum: Andi meldet sich für einen konkreten PR an und danach ab; solange angemeldet, schreibt nur, wer Andis Freigabe hat (CLAUDE.md §Repos und Remotes). Auf dem 13 Pro laufen Sitzungen als eigene Benutzer — dort baut framework einen befristeten Token per KI-Widget und einen verwalteten Hook.
- **Tag-Nachricht aus dem README-Satz der Version bilden** (Q-16): v0.5.0 trägt nur „Schnapp 0.5.0 (Sprint 10)“, v0.1.0
  bis v0.4.0 nennen den Inhalt. Ein Tag lässt sich nicht ohne Force nachbessern.

## Sprint 11, erste Sitzung (22.11.2026)

- **Nach jedem Push den CI-Lauf ansehen, nicht erst am Sprint-Ende** (F-380): Der neue `-race`-Schritt (F-368) war ab
  827c9c0 sechs Pushes lang rot, ohne dass es auffiel. Status über die öffentliche API
  (`api.github.com/repos/<owner>/<repo>/actions/runs`, ohne Anmeldung); die Annotation sagt nur „exit code 1“, das
  Job-Log liefert die API nur angemeldet (403). Das Log steht ohne Anmeldung auf der öffentlichen Job-Seite
  (`github.com/<owner>/<repo>/actions/runs/<run>/job/<job>`), lesbar über Andis Chrome (nur lesen, Tab danach zu).
- **`NumGoroutine` direkt nach einem `<-fertig` zählt die Goroutine noch mit**, wenn sie mit `defer close(fertig)`
  endet: Sie lebt nach dem Signal noch einen Augenblick. Lokal fällt das nie auf, auf dem CI-Rechner mit `-race` schon.
  Kurzes Nachlauffenster (Abfrage bis 20 ms) und Gegenprobe ohne das Warten.
- **Tests, die ein Ergebnis-Struct als Ganzes vergleichen** (`erg != (ui.Gespeichert{})`, 28 Stellen), brechen bei
  jedem neuen Feld. Eine Zusatzangabe, die nur ein Aufrufer braucht (Pfad für die Rettungskopie), geht als Rückruf
  (`speichernMitNachher`), nicht als Feld.
- **Parallel arbeiten mit Worktree-Agenten trägt**, wenn die Konfliktprüfung vorher stimmt: Haiku (#160, #162),
  Sonnet (#151/#161, #159, #165) und Fable-Verify liefen neben der eigenen Arbeit; übernommen per Cherry-Pick, danach
  `git cherry` (nur „-“) und Worktree samt Zweig weg. Volle Testkette erst im Hauptbaum nach allen Cherry-Picks.
- **Verify-Reste derselben Karte sofort beheben** statt neue Karten: V6-1 bis V6-5 lagen alle in der Datei, die die
  Karte ohnehin änderte; als ein Finding (F-378) mit Hash geführt, Prüfer-Tests ins Repo übernommen.
- **Mutationsprobe ist jetzt ein Go-Werkzeug** (`scripts/mutanten`, F-371): arbeitet in einer Kopie unter
  `tmp/mutanten/`, nie im Arbeitsbaum. Nach Umbauten an `internal/ui/editor.go` zuerst `-trocken`: Muster, die nicht
  mehr genau einmal vorkommen, an die neue Stelle ziehen, Absicht beibehalten.
- **Sonnet-Agenten setzen trotz Verbot Heredocs ab** (zweimal in dieser Sitzung, folgenlos) — im Bericht nach
  „Heredoc“ suchen und Diffs lesen; dasselbe gilt für eigene Einzeiler (`python3 - <<` ist auch ein Heredoc).
- **Was eine Meldung zusagt, muss der Code halten** (Verify 7, V7-1): „kept for 30 days“ stand in der Statusleiste,
  aber das Schließen ohne Änderung löschte die Kopie und die erste neue überschrieb sie. Bei jeder Zusage in einem
  Oberflächentext den Weg suchen, der sie bricht (gleicher Dateiname, Aufräumen beim Schließen).
- **„Ruht während X“ braucht eine Obergrenze** (V7-2, V7-7): Eine Bündelung, die bei Zug oder Speichern einfach
  aussetzt, lässt eine fertige Änderung beliebig lange ungesichert. Lieber vor dem Ereignis sofort sichern.
- **Jede Stelle, die eine Meldung löscht, ist ein Weg, eine haltende Meldung zu verlieren** (Q-1): \ beim Export-Start lag neben allen geschützten Wegen. Bei einer Schutzregel nach allen Schreibern des
  Felds suchen (\), nicht nur nach \.
- **Ein Prüfwerkzeug prüft zuerst seine Ausgangslage** (Q-3): Ohne grünen Lauf ohne Mutation gilt jeder Mutant als
  getötet.
