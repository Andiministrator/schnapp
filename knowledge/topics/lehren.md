# Lehren aus der Arbeit an Schnapp

Stand: 2026-11-27 (laufend ergänzt, zuletzt Sprint 14, Karten #361 und #334 Teil 1 mit Kritik (F-887 bis F-893); davor Karten #313 Teil 2 und #369 mit Sitzungs-QA (F-873 bis F-880); davor #313 Teil 1 und Sitzungs-QA (F-860 bis F-872); davor #363 Teil 2 und Sitzungs-Kritik (6); davor #353 bis #364 und Sitzungs-Kritik (4); davor #349 bis #352, #358). Kurze Regeln aus Fehlern und Befunden; Einzelheiten in `docs/findings.md`.

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

## Sitzung 27.11. (#361, #334 Teil 1, Kritik F-887 bis F-893)

- **Was ein asynchroner Ablauf beim Auslösen einfriert, wird beim Abholen gegen den aktuellen Stand geschnitten**
  (F-887): Der Papierkorb mehrerer merkte die übrigen markierten beim Entf und warf sie nach Speichern und Laden der
  Nachbarin weg — auch wenn der Nutzer die Markierung inzwischen mit Esc aufgehoben hatte. Jede Liste, die über einen
  Wechsel (Speichern, Laden, Goroutine) getragen wird, beim Abholen mit dem gezeigten Stand schneiden oder die Eingabe
  so lange sperren.
- **Tasten und Klicks an derselben Stelle über dieselbe Quelle führen** (F-888): Der Klick nahm `gezeigt[i]` (Stand
  des letzten Bilds), die Taste `Eintraege()[i]` (Stand nach dem Neulesen); zwischen beiden kann der Index neu gelesen
  sein.
- **Gios `widget.Clickable.Update` liefert die Umschalter** (`Click.Modifiers`), `Clicked` nicht. Wer Strg/Umschalt am
  Knopf braucht, liest `Update` in einer Schleife. Und: Der Leertasten-Zweig in `tasten` muss Umschalter ausschließen,
  sonst wirkt Strg+Leertaste wie Enter.
- **Doppelter Schutz macht einen Mutanten grün** (MK25): Die Markierung der Ansicht wurde beim Öffnen und beim Schließen
  geleert; der Mutant „Öffnen leert nicht“ überlebte. Einen der beiden Wege streichen oder den Test über den anderen Weg
  führen (hier: Schließen per Strg+Umschalt+L statt Esc, weil Esc die Markierung zuerst leert).
- **Ein Mutant mit unbenutzter Variable baut nicht** (MA21, MK4): `x + 0*luecke` bzw. `false && …` statt die Zeile zu
  streichen.
- **Bevor eine gewohnte Bedienung kippt, die Vorbilder nennen** (ADR-0039 Nachtrag 1): Ziehen im Bild wurde zum
  Auswahlrahmen; Inkscape, Krita, GIMP, Photoshop und Figma verschieben die Ansicht dann mit Leertaste+Ziehen. Andi
  entschied nach dem Vergleich in einem Satz.
- **Halbe Arbeit über einen Chat-Wechsel auf einem eigenen Zweig sichern** (`wip-334-mehrfachauswahl`), nicht halb auf
  main: Die Logik veränderte schon das Ziehen im Bild, das Zeichnen des Rahmens fehlte noch — auf main wäre das ein
  sichtbarer Rückschritt gewesen.

## Sitzung 26.11. spät (#313 Teil 2, #369, F-873 bis F-880)

- **`gofmt -l` vor jedem Commit.** Ein neues Feld in einem ausgerichteten Struct-Block (`editor`) verschiebt die
  Spalten; `go vet` und `go test` merken das nicht, die CI bricht im ersten Schritt ab (f4c6b9e, rot gemeldet an 47a0c1e). Die Prüfliste vor dem
  Commit beginnt mit `gofmt -l $(git ls-files '*.go' | grep -v '^third_party/')`.
- **Ein Rechtsklick, der Fokus setzt, und eine offene Liste vertragen sich nur, wenn Enter/Leertaste bei offener Liste
  auf deren Knöpfe beschränkt sind** (F-873). Die Leertaste bedient Gio an jedem fokussierten `widget.Clickable` selbst;
  der Editor erreicht sie nur an den Knöpfen in `leertastenZiele`. Deshalb nimmt eine per Maus geöffnete Liste anderen
  Knöpfen den Fokus (F-883). Wer einem Knopf hinter einer Liste den Fokus gibt, prüft Enter und Leertaste.
- **Mutanten auf Platzrechnungen überleben, wenn kein Test die Lage misst** (FZ13, F359-1): Ein unsichtbarer Reiter
  nahm nur Breite weg. Bei neuen Zeilen der Leiste eine Lage-Prüfung (füllt die Zeile, endet vor X) mitschreiben.
- **Gios `widget.Clickable` setzt bei einem Mausklick nie den Tastaturfokus**; wer nach einem Klick Tasten erwartet (die
  Richtungsscheibe, F-875), schickt `key.FocusCmd` selbst — aber **nie während einer Texteingabe**: Dann landen getippte
  Zeichen beim Knopf und gehen verloren (F-881). Ebenso bestätigt jeder Rechtsklick, der ein Menü öffnet, die
  Texteingabe (F-882).
- **`math.Mod` und `math.Round` behalten −0**; was in eine Datei geschrieben wird, vorher auf +0 bringen (F-874).
- **Kritiker-Gegenproben laufen zuverlässig über „sichern → anhängen → testen → zurückkopieren“**; ein Mutantenlauf
  zeigt dagegen nur, was die Mutantenliste kennt — die Kritik fand fünf Zweige ohne Mutanten.
- **Verify nach jeder Nacharbeit lohnt sich auch bei P3** (Sitzungs-QA (9)): Der Fix F-875 (Fokus per Klick) brachte
  den P2-Rückschritt F-881; ein „gleichwertig“ markierter Mutant (FZ25) war es nicht, weil der Zwischenstand während des
  Ziehens sichtbar ist. „Gleichwertig“ erst nach einem Gegenbeispiel-Versuch am Zwischenzustand setzen.

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
- **`gh` nur kurz anmelden** (Q-13, F-376, Rat von framework 22.11.): Auf dem Laptop laufen alle Sitzungen unter demselben Benutzerkonto; jede könnte den Token lesen und an einem Hook vorbei `curl` nehmen. Eine Sperre per Hook schützt nur vor Versehen. Darum: Andi meldet sich für einen konkreten PR an und danach ab; solange angemeldet, schreibt nur, wer Andis Freigabe hat (CLAUDE.md §Repos und Remotes). Auf dem 13 Pro laufen Sitzungen als eigene Benutzer — dort baut framework einen befristeten Token per KI-Widget und einen verwalteten Hook.
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
- **Jede Stelle, die eine Meldung löscht, ist ein Weg, eine haltende Meldung zu verlieren** (Q-1): `e.meldung =
  canvas.Meldung{}` beim Export-Start lag neben allen geschützten Wegen. Bei einer Schutzregel nach allen Schreibern des
  Felds suchen (`grep 'e.meldung ='`), nicht nur nach `melde(`.
- **Backticks in einem doppelt gequoteten Einzeiler führt die Shell aus** (22.11., eigener Fehler, folgenlos):
  Markdown mit Code-Spannen nie über `python3 -c "…"` schreiben, sondern mit dem Write- oder Edit-Werkzeug oder aus einer
  Skriptdatei unter `tmp/`.
- **Ein Prüfwerkzeug prüft zuerst seine Ausgangslage** (Q-3): Ohne grünen Lauf ohne Mutation gilt jeder Mutant als
  getötet.

## Sprint 11, zweite Sitzung (22.11.2026)

- **Ein Test-Auslöser am ersten Pixelzugriff wackelt, sobald jemand das Bild nebenläufig liest** (CI-Rot zu e9b4cef,
  5ff47d0): Die Vorschau-Goroutine (F-310) las `sichtbar` mitunter vor dem Lesen der Altdatei. Zeitpunkte im Ablauf
  des Speicherns über `saveprogress.SetzeHaken` auf eine Marke legen, nicht über Seiteneffekte eines Datenzugriffs.
- **Verdrahtung in der Gio-Schleife ist ohne Fenster prüfbar, wenn sie in einer Methode steht** (F-385): `neuerEditor`
  und `nachEreignis` herausgelöst, dann trugen Tests die Mutanten „Aufruf fehlt“. Eine Zeile mitten in `laufen()`
  oder `Editor()` erreicht nur der virtuelle KWin.
- **Speicherbefunde messen statt rechnen** (F-361): `TotalAlloc` mit abgeschaltetem GC um den Dekoder; ein von Hand
  gebautes JPEG (nur DC-Scan, Koeffizienten 0) braucht wenige Bytes je Block. Die Schätzung im Code dann gegen dieselbe
  Messung im Test halten (Spiel 2 % + 64 KiB), sonst überleben Mutanten am Rechenweg.
- **Ein Befehl mit Platzhalter geht nicht an den Browser** (22.11., eigener Fehler, folgenlos): Eine URL mit `RUNID`
  statt der Nummer wurde aufgerufen. Werte erst holen, dann einsetzen.
- **Eine Schutzprüfung fällt geschlossen aus** (F-389, Kritiker A): Kann der eigene Leser die Eingabe nicht deuten,
  die Bibliothek dahinter aber schon, muss die Prüfung mit dem ungünstigsten Fall rechnen, nicht durchwinken. Gegenprobe:
  „Was macht der nachgelagerte Dekoder mit dem, was mein Leser ablehnt?“ — Go überspringt Fremdbytes vor Markern.
- **Probe-Skripte schützen sich selbst** (F-390): Ein Skript, das nur in einer Sandbox laufen darf, prüft die Sandbox
  (`PROBE_INNEN`, HOME und XDG unter `tmp/probe`) vor dem ersten Schreiben, trägt kein Ausführbit und fällt ein Urteil
  mit Exit-Code statt nur `ls`-Ausgaben.
- **Nach jeder Übernahme die Trockenprobe über alle Mutantengruppen** (K14/K20 nach F-391): Eine Änderung für eine
  Gruppe verschiebt Muster einer anderen; die CI prüft `-trocken` über alle und wäre rot geworden.
- **Erwartete Werte gegen die Referenz prüfen, nicht gegen das Gedächtnis** (brand/werkzeug, QB-4): Ein eigener
  Testwert widersprach dem Code; fontTools selbst gab dem Code recht (SVGPathPen merkt „M“ bis zum nächsten Befehl).
- **Kritiker aus anderer Familie lohnen sich auch nach eigenen Mutationsproben** (Sitzungs-QA 22.11.): Die eigene
  Gruppe `fileformat-jpeg` war 9/9 rot und F-361 trotzdem mit einem Byte umgehbar — Mutanten prüfen die Tests gegen den
  Code, nicht den Code gegen die Wirklichkeit.

## Sprint 11, dritte Sitzung (22.11.2026, Karten #169, #43, #28, #172)

- **Eine Ansicht nach dem Zuschnitt ist ein verschobener Ausschnitt der Vollansicht** (F-386, ADR-0023 Nachtrag 5):
  Streifen vor dem Zuschnittsanfang verschieben nur. Statt eines zweiten Koordinatensystems reicht ein Versatz
  (`VollVersatz`) und eine größere Bildfläche; ein Zufallstest über Zuschnitt und Streifen (auch Kanten mitten im
  Streifen) belegt die Gleichung, bevor die Oberfläche darauf baut.
- **Neue Griffe verschieben den Sinn alter Tests** (F-386): Fünf Tests zogen einen „neuen Rahmen“ ab der Bildecke —
  genau dort sitzt jetzt der Griff. Nicht die Tests „reparieren“, bis sie grün sind, sondern jeden Fall auf die neue
  Regel umschreiben und den alten Sinn mit einem Griff-Zug abdecken.
- **Der virtuelle KWin ist das echte KWin**: `kwin_wayland --virtual` hat dieselben Protokolle. Den Rückfall ohne
  Halter erzwingt eine geerbte Verbindung (`WAYLAND_SOCKET`, kleiner Python-Starter unter `tmp/`, nicht im Repo); so
  ließ sich F-392 bis auf Klipper vorab prüfen. Was nur der echte Desktop hat (Klipper, echte Monitore, Portal), bleibt
  Andis Handprobe.
- **`strings` findet keine Texte mit Umlauten**: Es sucht nur ASCII-Folgen. Für einen Nachweis im Programm
  `grep -a -c` mit einem ASCII-Stück des Texts.
- **Ein Diagnosebefehl gehört vor die Handprobe, nicht danach**: Ohne Zeile für `cancelled` hätte die Handprobe F-392
  nur „Warnung ja/nein“ geliefert, nicht wann KWin meldet. Nach dem Ergänzen das Probe-Programm neu bauen.
- **Eigene Abweichungen dieser Sitzung** (folgenlos, als Erinnerung): `rm -f schnapp` mit relativem Pfad im
  Projektordner (gelöscht war womöglich ein altes, gebautes Programm); Commit-Nachrichten per Heredoc statt per Datei;
  Register per Python-Splice statt mit `register-schreiben.mjs`; Karte #43 erst nach Arbeitsbeginn übernommen;
  Probe-Skript mit Ausführbit angelegt (F-390).
- **Sitzungs-QA 22.11. abends (Kritiker Fable, F-394 bis F-405):** Esc setzte das Werkzeug an `SetzeWerkzeug` vorbei
  — jede Zustandsumschaltung, die an einem Werkzeug hängt, braucht genau einen Weg (`grep '\.Werkzeug = '`). Ein neuer
  Schalter wie `--export=ZIEL DATEI` braucht den Fall „Ziel ist die Quelle“ (`os.SameFile`, auch Symlink/Hardlink).
- **Ein Mutant streicht eine Prüfung mit `if false && …`, nicht durch Löschen**: Sonst bleiben Variablen unbenutzt,
  der Bau bricht, und das Werkzeug meldet FEHLER statt ROT.
- **Eine Probe darf nicht von Resten früherer Läufe leben** (F-403): Jede Probe schreibt ihre `settings.ini` selbst
  und hat eine Positivkontrolle für jede „bleibt gleich“-Prüfung.
- **Öffentliche Doku verweist nicht auf `tmp/`** (F-404): Ein Referenzbild für einen ADR gehört nach
  `docs/adr/bilder/`, sonst existiert die Vorgabe nur lokal.
- **Sammelbefehle mit `git commit`/`git push` und `aboard-op` in einer Kette lehnt die Rechteprüfung ab**: Schritte
  einzeln ausführen.

## Sprint 11 Abschluss und Sprint 12, erste Sitzung (22.11.2026 abends, Karten #96, #176, #179, #183)

- **Worktree-Agenten starten auf `origin/main`, nicht auf dem lokalen HEAD:** Zwei Agenten bekamen den vorletzten
  Commit, weil der jüngste noch nicht gepusht war. Vor jedem Worktree-Agenten pushen, die erwartete Basis in den
  Prompt schreiben und `git log --oneline -1` prüfen lassen. Gesperrte Worktrees nach dem Lauf mit
  `git worktree unlock`, dann `remove --force`; das Material unter `tmp/` vorher in den Hauptbaum kopieren.
- **Muster für Andi aus dem echten Renderer, ohne den Baum zu berühren:** eine Wegwerf-Testdatei unter `tmp/…`
  per `go test -overlay <json> ./internal/render -run …` einblenden; sie sieht Paketinterna, Schriften
  (`golang.org/x/image/font/gofont`) und `document`. Namen mit Präfix (`mu…`), sonst kollidieren sie mit dem Paket.
  So entstanden Oberflächen-, Ecken- und Naht-Muster (`tmp/sprint12/muster/`) und ein fairer Vergleich A/B.
- **Fragen an Andi nur mit Bild:** „Wie nehme ich das ab?“ — eine Gestaltungsfrage ohne Musterbild ist nicht
  beantwortbar. Je Frage ein Bild mit Pfad an der Karte, die Frage auf dem Bild selbst.
- **Fix-Commits zu Kritiker-Befunden tragen das Kritiker-Modell im Trailer** (F-435): Der Fix-Prompt gibt die
  Trailer-Zeilen wörtlich vor; nachträglich umschreiben hieße Force-Push.
- **Register per Einzeiler ändern, dann `register-schreiben.mjs pruefe --gegen-git HEAD`:** `einfuege-nach` mit
  `--erwarte-aenderung` scheitert, wenn die Änderungen schon vorher im Arbeitsbaum stehen (Mengenprobe gegen die
  Datei, nicht gegen git). „Rest …“ in einer Statuszelle liest der Abgleich als offen — „Folgebefund F-…“ schreiben.
- **Agenten brechen das Heredoc-Verbot trotz Prompt** (leere oder Python-Heredocs für eigene Dateien): im Bericht
  ausdrücklich nachsehen; der Prompt nennt das Write-Werkzeug als erlaubten Weg.
- **Durchsicht ändert Bildpfade:** Sobald `render.Bild` Alpha liefert, jeden Ausgang prüfen (PNG, Zwischenablage,
  JPEG auf Weiß, Vorschau vormultipliziert mitteln, SVG-Maske) — ein Spalt mit Alpha 0 ist sonst je nach Ausgang
  schwarz oder weiß.
- **Nebenläufige Ketten: Zustand, der beim Aufruf gesetzt und erst im Auftrag gelesen wird, ist ein Rückschritt-Muster**
  (F-446 → F-456): Die Nacharbeit zu einem Befund baute genau diese Klasse an anderer Stelle neu ein; erst ein zweiter
  Verify-Lauf fand sie. Jeder Auftrag liest den Stand der Kette oder einen mitgegebenen Wert, nie das Feld, das der
  Aufrufer gerade umsetzt — und der Verify-Prompt fragt danach ausdrücklich.
- **Overlay-Gegenproben, die ganze Dateien ersetzen, veralten mit dem Code:** Ein „FAIL“ nach einer Änderung kann ein
  Baufehler sein und belegt dann nichts. Verbindlich sind Mutanten im Mutanten-Werkzeug (`if false &&`), die an der
  aktuellen Zeile hängen.

- **Protokoll-Spezifikationen vor der Umsetzung frisch lesen** (Karte #173): Die MCP-Revision 2026-07-28 hat den
  `initialize`-Handshake abgeschafft (`server/discover`, `_meta` je Anfrage). Aus dem Gedächtnis gebaut, hätte der
  Server nur die Altform gesprochen. Quelle in `knowledge/sources/`, beide Epochen getestet.
- **Probe-Skripte prüfen JSON nicht über die Feldreihenfolge und Oberflächentexte nicht wörtlich:** Go schreibt
  `content` vor `isError`, und der virtuelle KWin läuft in der Systemsprache (deutscher Benachrichtigungstitel).
  Muster je Feld einzeln prüfen (`urteil` in `scripts/drehbuecher/agenten-mcp.sh`).
- **Ein Werkzeug, das bei EOF auf stdin seine laufende Anfrage abbricht, braucht im Probe-Skript offenes stdin:**
  `{ printf …; sleep N; } | schnapp mcp`, sonst endet jede lange Anfrage als „Cancelled“.
- **Benachrichtigungen im virtuellen KWin:** Der private D-Bus hat keinen Dienst; `scripts/fake-notifications`
  stellt ihn mit eigener Zeitgrenze, so prüft der Lauf auch die echte D-Bus-Signatur von `Notify`.
- **Eine Anzeige „wieder X“ braucht den Auslöser der Änderung, nicht nur einen Zähler** (F-472): Ein Zähler erzählte in
  drei Lagen die falsche Geschichte („anderes Fenster hat losgelassen“ nach Speichern als andere Datei, bei `aus`).
- **Einmal nach festem Abstand wecken ist eine Annahme über die Dauer der Kette** (F-474): im Takt bis zu einer Grenze
  wecken.
- **Nach jedem Umbau einer Produktzeile `-trocken` fahren:** Vier Mutanten-Muster (M5, M12b, T5, Q5) waren durch die
  Nacharbeit ungültig; das Werkzeug lehnt seit F-478 auch unbekannte Schlüssel und ungültige `timeout` ab.
- **Sicherheitsgrenzen eines Servers, den ein Agent startet, hängen nie an dessen Umgebung** (F-461): `HOME` und
  `XDG_*` setzt der Client; Einwilligung und Ablageort kommen aus dem Heim der Benutzerdatenbank.
- **Mutanten eines Kritikers können Dateien in den Quellbaum schreiben**, wenn ein Test relative Pfade nutzt und die
  Prüfung mutiert ist (`rel.png`, F-471): Tests mit relativen Pfaden laufen in `t.Chdir(t.TempDir())`.
- **Ein Dialog-Kind testet man, indem `TestMain` das Testprogramm selbst als Kind spielen lässt** (Umgebungsvariable,
  Naht `kindProgramm`, Karte #203): ohne Wayland, ohne Tracing.
- **Pipe-Tests gegen einen Zeilen-Server:** Kehrt das Schreiben der übernächsten Zeile zurück, ist die vorletzte
  verarbeitet — das ersetzt Wartezeiten; EOF erst nach der Antwort, sonst bricht der Server den Aufruf ab.
- **Attrappen in Tests bekommen eine eigene Ausweichzeit:** Ein Mutant, der einen Test hängen lässt, lieferte sonst
  FEHLER statt ROT (seit F-478 eigener Status HÄNGT).

## Sprint 12, zweite Sitzung (23.11.2026, Karten #205, #206, #207)

- **Ein Sicherheitsnetz hinter dem eigentlichen Schutz verdeckt dessen Mutanten** (M41, F-483): Seit der Server eine
  nach dem Abbruch geschriebene Datei wieder entfernt, überlebte der Mutant, der die Prüfung *vor* dem Schreiben
  strich. Der Test muss den Schreibversuch selbst beobachten (Naht), nicht nur das Ergebnis im Dateisystem.
- **Unix-Sockets in Tests nie unter `t.TempDir()` mit absolutem Pfad:** Im Mutantenlauf ist `TMPDIR` lang, der Pfad
  sprengt die 108 Zeichen von `sun_path` („bind: invalid argument“). Relativ in `t.Chdir(t.TempDir())`.
- **Den Dialog im Probelauf vor dem Urteil sichtbar machen** (F-491): Nach der Sitzungsbindung scheiterte das Kind an der
  Probe-Sperre (absoluter `WAYLAND_DISPLAY`); D und F („Esc lehnt ab“, „Tastatur erlaubt nicht“) waren trotzdem grün,
  weil auch ein gescheitertes Kind „nicht erlaubt“ heißt. Erst E (Erlauben) fiel. Ablehnungs-Proben brauchen einen
  Beleg, dass der Dialog da war (Diagnosezeile `[bestaetigen] erstes Bild`, Foto).
- **Zeitkritische Drehbuch-Schritte am Ereignis ausrichten, nicht an einer Wartezeit:** Teil G klickt, sobald das Kind
  sein erstes Bild meldet, und zählt nur als bestanden, wenn der Dialog ein Drücken *in* der Sperre meldet — sonst
  „nicht aussagekräftig“ statt still grün.
- **Agenten-Fehlertexte nicht über die Oberflächen-Übersetzung führen** (F-492): `nutzerFehler` übersetzt; was an einen
  Agenten geht, braucht eigene englische Sätze ohne Pfade (`agentGrund`).
- **Worktree eines Agenten mit eigenem HOME lässt sich nicht entfernen,** wenn darin ein Go-Modulcache liegt (Go legt ihn schreibgeschützt an): erst `GOMODCACHE=<voller Pfad> go clean -modcache`, dann `git worktree remove --force`.
- **Agenten setzen trotz Verbot leere Heredocs ab** (vier von fünf Berichten am 23.11.); harmlos, aber im Bericht nachsehen. Einmal lag dadurch eine leere Datei im Worktree-Wurzelordner.
- **Der virtuelle KWin ist nicht die echte Plasma-Sitzung** (F-500): `probe.sh` startet `kwin_wayland --virtual` ohne
  `kwin_wayland_wrapper`. Eine Prüfung, die am Prozess hinter dem Wayland-Socket hängt, bestand jede Probe und lehnte
  jede echte Sitzung ab. Was an Prozess- oder Socket-Struktur hängt, braucht eine Gegenprobe in der echten Sitzung
  (lesend: verbinden ohne Byte, Bus-Abfrage) — der Kritiker hat sie gemacht, nicht die Umsetzung.
- **Einen Pfad nicht zweimal auflösen:** Wer einen Socket prüft und dann einem Kind den Pfad gibt, prüft etwas anderes,
  als das Kind benutzt (`..`, Symlink, Umhängen, F-501). Die geprüfte Verbindung vererben (`WAYLAND_SOCKET`).
- **Eine Tafel über der Leinwand braucht dieselbe Tastensperre wie das Menü** (F-508): sonst wirken Entf und
  Werkzeugbuchstaben unsichtbar dahinter.
- **Kritiker-Proben, die Formeln nachrechnen, bleiben nach dem Fix rot bzw. grün wie vorher** — sie messen das alte
  Layout. Nach der Nacharbeit mit einer Sichtprobe oder einem Test gegen den echten Code prüfen.


## Drehbücher nach der neuen Oberfläche (Karte #210, 23.11.2026)

- **Pfeiltasten im Menü überspringen graue Einträge.** Wer im Drehbuch mit „Runter“ zählt, prüft die Reihenfolge im
  Foto; zwei Runter statt einem löschten den Text (`rueckfall-menue-tastatur.txt`).
- **Ein Knopf-Klick ohne Zeiger beweist nur etwas an einem Knopf, der noch nicht aktiv ist** (Stauchen war schon
  gewählt; jetzt Verpixeln).
- **4K-Drehbücher im Race-Build brauchen `--ende 90`**, sonst beendet `probe.sh` Schnapp nach 5 s mitten im Speichern.
- **Kopfkommentare gegen das Foto halten:** `rueckfall-schwaerzen.txt` behauptete „4 px → schwarz“; tatsächlich lehnt
  der Editor unter `MinObjekt` ab und meldet „zu klein“.
- **Gio nimmt die Tab-Reihenfolge aus der Anmeldung der Klickflächen im Bild,** nicht aus der Zeichenreihenfolge —
  Felder von rechts legen, Klickflächen von links anmelden geht ohne Layoutänderung. Im Test bewegt
  `key.Event{Name: key.NameTab}` den Fokus nicht, nur `MoveFocus(key.FocusForward)` (Karte #213).
- **Ein Zustandswechsel und eine Taste im selben Bild sehen denselben Gio-Fokus:** Pfeil rechts wechselt das Menü,
  Enter wertet danach noch den alten Fokus-Index aus. Was im selben Bild auf dem Fokus aufbaut, braucht eine
  „gerade gewechselt“-Marke (`menue.gewechselt`, F-512).
- **Gios Strich (`clip.Stroke`) kennt nur runde Ecken.** Spitze Ecken im Editor gehen nur über gefüllte Flächen
  (`document.EckigerRahmen`, F-498).
- **Ein Test, der die gerade zu ändernde Eigenschaft festschreibt, wird nicht gelöscht,** sondern bekommt die
  Voraussetzung der alten Eigenschaft (`TestKastenEckeRund` jetzt mit Radius 6).
- **API-Abbrüche (529) treffen Agenten kurz vor dem Commit:** Beim Fortsetzen zuerst committen lassen, dann prüfen
  und berichten (Karte #211, zwei Abbrüche).
- **Nach jeder Übernahme den Mutanten-Trockenlauf fahren, nicht nur `go test`** (`cd scripts/mutanten && go run . -wurzel ../.. -trocken`):
  #213 änderte den Code unter O6/O7, die Muster fanden nichts mehr, und die CI war von 93ae780 bis 4ce56c3 rot. Die
  Orchestrator-Prüfkette ist die CI-Kette (`.github/workflows/`), auch `go test -race ./internal/...`.
- **Vor einer Umbenennung `git grep` auch über `scripts/mutanten/*.json`, `draw/` und `docs/format.md`** — Mutanten-Muster
  tragen Bezeichner als Text (Karte #182).
- **Gleichfarbige Halbtransparenzen lassen sich in überlappungsfreie Stücke mit zusammengesetzter Deckkraft
  (1 − Π(1 − aᵢ)) umrechnen;** gleich breite Stücke teilen dann einen SVG-Pfad per `<use>` (F-422: Abriss-Naht +557 kB
  → +85 kB).
- **Chromium/Aboard: leere Board-Seite ohne Konsolenmeldung** kam von gespeicherten Ordner-Handles, deren
  `queryPermission` nie antwortete (alle 25, auch lokale; frisch gewählter Ordner ging). Abhilfe: IndexedDB
  `aboard-handles` gezielt entfernen, Ordner neu wählen; hält über Neustart (23.11.2026, aBoard F-3106/#1179).
- **Messen vor Optimieren, und den Benchmark über alle Grenzen mitlaufen lassen** (Karte #212): Der Verdacht
  „Rasterkern“ war falsch, teuer waren das wiederholte Pfad-Rastern gleicher Nähte (gemerkte Masken lösen es) und
  das Mischen der Randpixel. Eine verallgemeinerte Viererschleife machte `BenchmarkNaehteAlleGrenzen` von 14 s auf
  21,6 s langsamer, weil der Schnellpfad „volle Deckung, deckende Quelle“ wegfiel.
- **Ein Cache über Pfadkoordinaten ist nur bitgleich, wenn der Schlüssel vollständig verglichen wird** (Hash nur zur
  Vorsortierung). Eine Maske in Spaltenstreifen abspielen ist bitgleich; eine Füllung in Kacheln rastern nicht
  (float32-Summen im Rasterkern).
- **~~Gio mischt halbdurchsichtig linear~~ — überholt seit 24.11.2026** (ADR-0020 Nachtrag 14, F-520): Das gepatchte
  Gio mischt im sRGB-Raum wie `render`. Vorrastern für Farbgleichheit ist nicht mehr nötig (die Nähte bleiben
  vorgerastert, F-407). Deckkräfte der Oberfläche gelten jetzt als Anteil der Zahlenwerte: 60 % Dunkel ist `0x99`, nicht
  mehr „deckender, weil Gio linear mischt“.

## Sitzungs-QA 23.11.2026 nachmittags (Kritiker A/B, Fable)

- **Nach einer Übernahme die berührten Mutanten-Gruppen scharf fahren, nicht nur trocken:** O2 endete seit b2bb31e
  scharf mit Panik („kein FAIL-Name“) statt ROT; der Trockenlauf der CI sieht das nicht. Ein Panik-FEHLER ohne
  Testnamen ist meist ein nil-Rückruf in einer Hintergrund-Goroutine — der Testlauf-Helfer braucht Attrappen.
- **Herausgelöste Teilfunktionen getestet heißt nicht Verdrahtung getestet:** Nach F-407 waren `KantenOhneSchatten`,
  `kantenBildStand` und `fuehreNahtAblaufAus` geprüft, aber `bildNeu`/`leinwand` riefen sie ungeprüft auf (Q2/Q3,
  E14 bis E17). Den Bau ohne Gio herauslösen (`leinwandBild`, `bildPruefen`) und die Entscheidung testen.
- **Kritiker-Worktrees ohne Änderung verschwinden sofort beim Ende des Agenten** — Belege unter `tmp/` sind dann weg.
  Zusatztests und Mutationsmuster gehören wörtlich in den Bericht (beide Kritiker haben es so gemacht); der
  Orchestrator schreibt die Muster aus dem Bericht nach `tmp/<sprint>/qa-…/`.
- **Teilüberdeckung halbdurchsichtiger Rechtecke zeigt sich nur mit halben Pixeln:** Die 16-Bit-Maske klemmt
  Überlappung bei voller Deckung weg; ein Test auf doppelte Mischung braucht Koordinaten auf .5.
- **Mutanten, die eine Variable unbenutzt lassen, bauen nicht** (`declared and not used` → FEHLER): im Muster
  `_ = x` mitschreiben (L5, M32).

## CI zu e08a477 (23.11.2026 abends, F-527)

- **Ein Test, der stdin nach einem Ereignis schließt, muss auf die erwartete Antwort warten, nicht auf das Ereignis.**
  Der Server darf ein EOF vor dem Abholen des Werkzeug-Endes vertragsgemäß als Ende nehmen; lokal 300× grün, in der CI
  unter `-race` rot. Nachstellen gelang erst mit einer erzwungenen Pause im Test-Werkzeug (Fenster vergrößern statt
  Wiederholungen zählen).
- **Testhilfen mit träge angelegtem Schloss sind selbst ein Datenrennen** (`if p.mu == nil { p.mu = … }`). Ein
  `sync.Mutex` als Nullwert braucht keine Anlage. Aufgefallen erst, als der Test den Puffer während des Laufs las.

## Rechte Leiste (Karte #187 b, 23.11.2026)

- **Eine neue, einklappbare Leiste verschiebt die Leinwand aller Drehbücher.** Statt 17 Drehbücher neu auszumessen,
  klappen sie die Leiste am Anfang per Taste ein (Strg+Umschalt+B); damit gilt die alte Lage, und ihr Prüfgegenstand
  bleibt derselbe. Die neue Leiste bekommt eigene Drehbücher. Gegenprobe: alle Drehbücher mit Testbild gefahren, Fotos
  der lagerempfindlichen (Schwärzen, Stauchen, Kantenform) angesehen.
- **Neue Aufklapplisten über die vorhandene Menü-Mechanik führen** (eine Nummer hinter den Menüs statt eines zweiten
  Listensystems): Tastatur, Klick daneben und Sperre beim Speichern kommen mit, nur Lage, Rückgabe des Fokus und die
  Pfeile links/rechts brauchen eine Ausnahme.
- **Gio-Knöpfe nehmen beim Mausklick keinen Fokus** (`widget.Clickable`): Ein Klick auf eine Farbe während der
  Texteingabe lässt die Eingabe weiterlaufen. Vor einem Fokus-Umbau im Gio-Quelltext nachsehen statt vermuten.

## Sitzungs-QA 23.11.2026 spätnachmittags (Kritiker A/B, Fable)

- **Ein Fix, der etwas Sichtbares wegnimmt, braucht einen Rückweg für jeden Fehlerweg** (F-528): Das Schließen des
  vorigen Hinweises vor dem Dialog ließ einen Agenten den Hinweis seiner Aufnahme mit einem Fehlversuch verschwinden
  lassen. Erst nach dem letzten Ablehnungspunkt schließen und bei jedem späteren Fehler wiederherstellen.
- **Ein Echtlauf vor dem Fix belegt den Befund, nicht den Fix** (F-490 wieder offen): Nach dem Fix gehört derselbe
  Echtlauf wiederholt, bevor das Finding ins Archiv geht.
- **Skripte, die echte Einstellungen ändern, stellen nach jedem Teil zurück und hinterlassen eine Laufmarke**
  (F-529): Wartet ein Skript zwischen zwei Teilen auf Enter, steht die Erlaubnis sonst unbegrenzt offen; ein harter
  Abbruch hinterließe sie dauerhaft, und der nächste Lauf sicherte den veränderten Stand als „Original“.
- **Merker für eine Liste nur für diese Liste auswerten** (F-534): `listeAmFarbKnopf` galt auch für die Zoomliste.
- **Mutanten in `canvas`, deren Tests in `ui` liegen, brauchen `paket: ./internal/ui`**: Das Werkzeug fährt nur die
  Tests des genannten Pakets; sonst „überlebt“ der Mutant, obwohl ein Test ihn fängt.
- **Doppelter Schutz (UI-Sperre und canvas-Sperre) macht den Mutanten einer Schicht grün**: Einen Einzelmutanten der
  äußeren Schicht nicht als Lücke werten; geprüft wird die innere Schicht (wie F-307).
- **Ein Skript-Test mit eigenem `HOME` braucht `GOMODCACHE` und `GOCACHE` bei JEDEM Aufruf** (23.11.): Ein zweiter
  Aufruf ohne sie lud alle Module neu in den Sandkasten (schreibgeschützt, `rm` scheitert); Abräumen mit
  `GOMODCACHE=<voller Pfad> go clean -modcache`, dann `rm`. Billige Prüfungen (Symlink, Laufmarke) gehören vor den Bau.


## Farbwähler und Schrift (Karten #189, #190, 23.11.2026 abends)

- **Gio gibt ein Tastenereignis dem ersten passenden `gtx.Event`-Aufruf des Bildes** (Tastenereignisse tragen kein
  Ziel; maßgeblich ist die Reihenfolge der Aufrufe). Ein eigenes Eingabefeld holt Enter, Esc und die Rücktaste deshalb
  **vor** der allgemeinen Tastenauswertung ab (`feldTasten` vor `tasten`), sonst löscht die Rücktaste das gewählte
  Objekt. Getippter Text kommt unter Wayland getrennt als `EditEvent` an das fokussierte Ziel; die zugehörige Taste
  läuft trotzdem durch die Kürzel — Buchstaben-Kürzel müssen bei fokussiertem Feld ausdrücklich übergangen werden.
- **Tab fährt den Fokus nur, wenn kein Filter die Taste nimmt**: Ein Feld darf keinen Filter ohne Namen
  (`key.Filter{Focus: f}`) anmelden, sonst bleibt der Fokus im Feld stecken.
- **Im Test lässt sich `pointer.Drag` nicht einspeisen** (Gio-Router: Panik „unsupported pointer event type“); Ziehen
  ist `Move` mit gedrückter Taste, Gio macht daraus `Drag`. Ein Druck genau auf `Max` eines Rechtecks liegt außerhalb.
- **Gleiche Maße in Editor und Export gibt es nur mit demselben Satz** (#190): Der Gio-Shaper setzt mit eigenem Shaping,
  und `opentype.Face.Kern` skaliert die Unterschneidung nicht mit der Größe; bei fremden Schriften liefen Schild und
  Glyphen auseinander. Gemeinsam gilt jetzt `document.KernPaar` (GPOS, sonst kern, im Grad). Der Editor zeichnet
  Text deshalb aus den Glyphenumrissen derselben Datei (`internal/ui/textsatz.go`) mit denselben Orten wie der Renderer.
- **Systemschriften ohne fontconfig**: `sfnt.ParseReaderAt` liest nur Tabellenverzeichnis und Namen; auf dem
  Entwicklungsrechner 297 Familien in rund 0,6 s. Gios eigene Systemschrift-Suche (go-text/fontscan) legt einen Index
  im Cache an — deshalb `text.NoSystemFonts()` und eigene Suche (Kernprinzip 3).
- **Ein Testablauf kann einen Mutanten zufällig verdecken** (F189-6): Eine Beschriftungsfarbe, die fälschlich in die
  zuletzt benutzten Farben rutschte, wurde vom weiteren Ablauf wieder hinausgeschoben. Zwischenzustände prüfen, nicht
  nur das Endergebnis.
- **Im Auto-Modus lehnt die Freigabe ein rekursives Löschen von Ordnern mit Bildschirmfotos ab**: Solche Ordner löscht
  Andi selbst per `!`-Befehl (am 23.11. so geschehen); nicht wiederholt versuchen.

## Ebenen-Leiste (Karte #191, 23.11.2026 abends)

- **Ein `FocusCmd` wirkt nur auf ein Ziel, dessen `key.FocusFilter` Gio schon kennt.** Ein Feld, das erst in diesem Bild
  gezeichnet wird (Namensfeld beim Umbenennen), bekommt den Fokus deshalb erst im nächsten Bild, und zwar dort, wo seine
  Filter angemeldet werden (`feldTasten`, per `defer` nach dem Anmelden). Vorher liefert Gio dem frisch angemeldeten Ziel
  ein `FocusEvent{Focus: false}` — das darf nicht als „Fokus verloren“ gelten, sonst endet das Umbenennen sofort
  (`nameHatteFokus`).
- **Eine Deckkraft je Objekt ist Gruppendeckkraft**, nicht ein Faktor auf jede Farbe: sonst sind Pfeilspitze über dem
  Schaft und Schrift auf dem Schild dunkler. Renderer: Objekt in ein durchsichtiges Zwischenbild, dann mit einer Deckung
  einmischen (`render.ebene`); Gio: `paint.PushOpacity`. Prüfen an einer Stelle, wo sich zwei Teile überlagern.
- **Vorschauen verkleinern Striche bis zur Unsichtbarkeit**: in der Vorschau eine Mindestbreite in Vorschau-Pixeln
  setzen, nicht im Dokument.
- **Drehbuch-Koordinaten der Liste hängen an der Zahl der Zeilen** (Doppeln schiebt alles): Leiste per Tastatur bedienen,
  wo es geht, und Koordinaten aus dem Foto davor lesen.
- **Ein Zwischenbild für ein Objekt muss so groß sein wie das GEZEICHNETE, nicht wie die abgebildeten Grenzen**
  (F-583): Text und Nummer werden starr ab einem abgebildeten Anker gezeichnet; hinter einer Stauch-Naht liegt ihre Tinte
  weiter rechts/unten als `ZuAnsicht(Grenzen.Max)`. Prüfen mit „Tinte bei 100 % ⊆ Tinte bei 50 %“ auf gestauchtem Bild.
- **Platz in der Leiste: zuerst den größten Block einklappen, nicht Kleinkram weglassen** (F-586): Das Weglassen des
  Hinweises half in keiner geprüften Fensterhöhe; der überlebende Mutant zeigte, dass der Code tot war — entfernt.
- **Kritik-Worktrees ohne Commit räumt die Laufzeit selbst weg**: Testvorlagen der Kritik sind danach verloren. Im
  Prompt verlangen, dass jede belegende Testvorlage als Code-Block im Bericht steht.
- **`register-schreiben.mjs einfuege-nach`**: Anker muss eindeutig sein (bei gleichen Statusspalten das Zeilenende
  mehrerer Zeilen gleich → längeren Anker nehmen); endet der Anker am Zeilenende, beginnt `--neu` mit einem
  Zeilenumbruch; eine führende Leerzeile zerreißt die Tabelle (Abgleich Exit 2).
- **Bei Platzmangel nicht den meistgenutzten Block opfern** (Verify V23-2): Das erste Auto-Einklappen nahm die Farbe weg
  und sperrte den Farbwähler unter 690 dp aus; richtig ist, den seltener gebrauchten Abschnitt einzuklappen und einen
  Klick auf die Überschrift als Vorrang zu werten. Schwellen in dp mit `gtx.Dp(unit.Dp(a+b))` rechnen, nie
  `a*gtx.Dp(1)` (rundet 1,25 auf 1 und 1,5 auf 2).
- **`sfnt.Font.Bounds` rechnet in int32**: mit ppem = unitsPerEm läuft es bei großen Schriften über; festen ppem nehmen.

## Füllfarbe, `--annotate`, Einrasten, Glyphkosten (Karten #219, #202, #196, #224, 23.11.2026 nachts)

- **Speicherfunktion vor der ersten Änderung anlegen** (#202): `ablegenIn` merkt sich den Schwärzungsstand der gelesenen
  Datei beim Anlegen. Wer sie erst nach dem Anwenden der Schwärzungen baut, verliert die Warnung „Ausgangsdatei zeigt
  noch den Klartext“ — ein Test mit einer Datei von außen hat es gefangen.
- **`encoding/json` füllt `[2]float64` still auf** (`[1]` wird `[1, 0]`) und vergleicht Feldnamen ohne Groß-/
  Kleinschreibung; doppelte Schlüssel gewinnt der letzte. Strenge Eingaben brauchen eigenes `UnmarshalJSON` und eine
  Schlüsselprüfung über die Rohdaten.
- **Neuer Code darf alte Mutanten-Muster nicht verdoppeln**: `-trocken` meldet „Muster 2-mal gefunden“, wenn eine neue
  Funktion eine vorhandene Zeile wörtlich wiederholt (cmd-mcp M38, M66). Nach jedem größeren Commit `-trocken` fahren.
- **Ein „aktives Feld“ braucht eine einzige Quelle**: `Fuellbar()` (Werkzeug des Paars) und `Seitenleiste().Fuellung`
  (aktuelles Werkzeug) liefen auseinander; „Tauschen“ war bei Schwärzen an (Kritik #219 B-4).
- **Ganz durchsichtig ist ein Sonderfall der Farbwahl**: HSV von Alpha 0 ist Schwarz; Regler und Hex auf „Keine“ müssen
  sichtbar machen, Zuletzt-Farben nehmen Alpha 0 nicht auf.
- **Gio `gpu/headless` (EGL) läuft auf dem Arbeitsrechner** (#224): Editor-Zeichnen lässt sich ohne Fenster testen
  (`internal/ui/zeichnen_headless_test.go`, `EGL_PLATFORM=surfaceless`; `SCHNAPP_HEADLESS_PFLICHT=1` macht das Überspringen zum Fehler). Für F-520 und Hilfslinien nutzbar.
- **Alt beim Ziehen unter KDE**: Plasma 5 verschob Fenster mit Alt+Ziehen, Plasma 6 nimmt Meta als Voreinstellung — Alt
  erreicht Schnapp, solange niemand KWin umstellt (Annahme, ADR-0021 Nachtrag 12); der Menüschalter bleibt der Ausweg.
- **Agenten-Berichte sichern, bevor der Chat wechselt**: Kritiker ohne Commit verlieren ihren Worktree; Berichte gehören
  vor einem Chat-Wechsel nach `tmp/sprint12/berichte/` (gitignoriert) und die Übergabe nennt die Pfade. Langen Berichten
  vorbeugen: Ausführliches in eine Datei im Worktree, Antwort an den Orchestrator kurz.
- **Ein Sub-Agent kann von einem Sicherheitsfilter der API abgebrochen werden** (Sonnet, #224), auch bei harmloser
  Arbeit. Commit und Bericht waren schon da; nichts wiederholen, Lage im Bericht an Andi weitergeben.

## Nachlauf Sprint 12 (Karten #226, #227, #228, #225, #220, #197, #233, 23./24.11.2026 nachts)

- **Was nur während eines Zugs zu sehen ist, fotografiert das Drehbuch mit `druecke` … `foto` … `loslassen`** (F-599,
  `einrasten-196-halten.txt`). Alt loslassen wirkt mitten im Zug sofort, ohne Zeigerbewegung.
- **Ein vermuteter Hänger bei Signalen gehört mit echten Signalen im Kindprozess geprüft** (F-603): Die Nähte
  `meldeAn`/`meldeAb` ersetzen gerade das, worum es geht (`signal.Stop` stellt das Standardverhalten wieder her). Muster:
  `cmd/schnapp/beschriften_signal_kind_test.go` (Kind über Umgebungsvariable, Heim vom Elternprozess, Ende per
  `WaitStatus.Signal()`); die Gegenprobe ohne Abmelden muss hängen.
- **SKIP in der CI ist kein PASS**: Der headless-Zeichentest sprang im Runner, weil `libGLESv2.so.2` fehlte (Paket
  `libgles2`), obwohl EGL da war. Der Grund steht nur im Log mit `-v`; die Pflicht-Variable macht es zum Fehler (F-602).
- **Worktree-Agenten können nicht in den Hauptbaum schreiben** (Isolation): Berichte landen in ihrem Worktree unter
  `tmp/…` und müssen vor dem Abräumen herüberkopiert werden; im Prompt den Worktree-Pfad statt des Hauptbaums nennen.
- **Messungen nicht neben anderen Läufen**: Eine Zeitmessung parallel zu Tests oder Mutanten (#227) streut um 20 %;
  Zahlen für ein Versprechen nur in ruhiger Zeit, mehrfach (`-count 3`).
- **Grüne Sonde, falscher Fix** (#225, F-598): Die Sonde schickte ein einziges großes Radereignis; das echte Rad kommt in
  15-px-Stufen, und nach fünf Stufen lag ein Knopf unter dem Zeiger, der das Rad schluckte (Gio `hitTest`: eine
  Anmeldung in einem eigenen, sofort abgelegten Clip-Ausschnitt verliert gegen jeden undurchlässigen Bereich darüber).
  Erst die Sichtprobe im virtuellen KWin fand es. Eingaben im Test so zerlegen, wie das System sie liefert.
- **Ein globales `op.Offset` um einen gerollten Inhalt** hält Zeichnung, gemerkte Rahmen und Ereignislagen deckungsgleich;
  Fensterkoordinaten (Wachen über dem ganzen Fenster) müssen umgerechnet werden.
- **Gio setzt den Fokus eines Tags, das aus dem Bild verschwindet, ohne FocusEvent auf nil** (#220): Wer ihn halten will,
  merkt sich das Feld und prüft im Folgebild (`farbwaehler.hatteFokus`). Tab im Test als `input.SystemEvent` schicken und
  bei nicht behandeltem Ereignis `MoveFocus` rufen wie `app/window.go`; ein nacktes `key.Event` bewegt den Fokus nicht.
- **Ein Filter mit `Focus: tag` fängt eine Taste nur, solange dieses Tag den Fokus hat** — so bleibt Tab in Feldern der
  Leiste frei. Wer Tasten freigibt, prüft, wohin die nächsten Zeichen gehen (F-608).
- **Gio reicht ein Zeigerereignis an Ziele weiter, die im selben offenen `clip`-Scope davor angemeldet sind** (#197):
  Ein schwebendes Element über der Leinwand (Navigator) wird **nach** dem `Pop()` des Leinwand-Ausschnitts angemeldet,
  sonst erreicht der Druck auch die Werkzeuge. Test prüft, dass kein Objekt entsteht.
- **Ein Test „Klick zentriert“ ist leer, wenn die Ansicht vorher schon mittig stand**: Zielpunkt abseits der Mitte wählen.

## Sitzungs-QA und Aufräumen (24.11.2026 früh)

- **Ein Fix, den die Kritik „halb“ nennt, braucht eine zweite Prüfung** (F-609 → F-620): Das Budget zählte die Quelle,
  aber viele gleiche Lupen unter dem Budget bauten den Export trotzdem auf Gigabyte. Grenzen gehören an die Stelle, die
  den Speicher belegt (Export), nicht nur ans Rechenmodell.
- **Eine Meldung vor dem Beenden kann selbst blockieren** (QS-1): Wer beim zweiten Signal erst auf stderr schreibt und
  dann beendet, hängt an einer vollen Pipe. Meldung mit Frist (200 ms), Beenden unabhängig davon.
- **Mutanten, die Speicher fressen, früh abbrechen lassen**: Ein Mutant der Exportgrenze wollte 9 GiB belegen; der
  Test begrenzt jetzt selbst und bricht ab, bevor der Rechner leidet.
- **Worktree-Agenten mit eigenem HOME legen den Go-Modulcache schreibgeschützt in den Worktree**: HOME für go-Befehle
  nicht umsetzen; vor dem Abräumen `GOMODCACHE=<pfad> go clean -modcache`, sonst scheitert `git worktree remove`.
- **Prüfmuster F-1 trifft auch Bezeichner** (eine Testvariable mit dem Kürzel aus dem Muster, `-i`): Variablen in Tests so benennen, dass das Muster sauber bleibt.
- **Cherry-Pick-Konflikt am Ende eines ADR**: Zwei Stränge hängen Nachträge an dieselbe Stelle; beide behalten,
  Nachtragsnummern prüfen.
- **Wer eine Grundannahme umstellt, sucht ihre Spuren im ganzen Repo, nicht nur im Code** (Sitzungs-QA 24.11., F-635):
  Nach der Umstellung der Gio-Mischung auf sRGB standen „Gio mischt linear“ noch in einer Lehre, zwei Kommentaren und
  einem ADR, und auf Sicht abgestimmte Deckkräfte (Overlay, Einstellungstafel) waren falsch. Vor dem Commit `git grep -i`
  nach dem Stichwort über `internal`, `docs`, `knowledge`, `third_party/README.md`, und jede auf Sicht abgestimmte
  Konstante einmal ansehen.
- **Ein Agent auf alter Basis stimmt Werte auf den alten Stand ab**: Die Einstellungstafel entstand parallel zur
  sRGB-Umstellung und bekam eine „deckendere“ Abdunklung, die nach dem Zusammenführen zu dunkel war. Nach jedem
  Cherry-Pick, der eine Grundannahme kreuzt, die Werte des Agenten gegen den neuen Stand halten (Sichtprobe).
- **Ein überlebender Mutant an einer Zwischenstufe kann echt äquivalent sein**: Eine sRGB-Zwischenebene kodiert beim
  Schreiben und dekodiert beim Lesen; das hebt sich auf, nur die Enden einer Kette ändern das Bild (F-621). Erst die
  Umrechnung zu Ende denken, dann Tests nachschärfen.
- **Ein neues Kürzel oder ein neuer Menüeintrag ist erst fertig mit README en/de, Glossar, Vermerk im eingelösten
  Alt-ADR und den Zählern in CLAUDE.md** („Nachträge 2 bis N“, ADR-Spanne) — in der Sitzung vom 24.11. fehlte das für
  die Einstellungstafel und ADR-0018.
- **Regeln in CLAUDE.md nennen fremde Skripte mit Herkunft**: `install.sh` gibt es im Projekt und im Framework; eine
  Regel ohne Herkunft trifft das falsche Skript.
- **Mutanten-Muster nicht mit einem Tabulator-Escape direkt vor einem Wort beginnen**: Das Escape plus Wortanfang kann
  das Prüfmuster aus CLAUDE.local.md als Wort treffen (Fehlalarm in schnellstile.json); Muster ohne führenden Tabulator
  sind ebenso eindeutig.
- **Parallele Kritiker brauchen getrennte Probeorte**: Drei Kritiker liefen gleichzeitig nur lesend im Hauptbaum; jeder
  bekam `tmp/sprint12/kritik-<x>/` und blendete Probetests per `go test -overlay` ein, statt Dateien in Pakete zu legen.
  Kein `./...`-Build und kein `probe.sh` bei parallelen Kritikern.
- **Einstellungen mit Sicherheitsbezug müssen am Ort des Lesers geschrieben werden** (F-625): Der MCP-Server liest
  bewusst nie aus `XDG_*`; eine Tafel, die nach `XDG_CONFIG_HOME` schreibt, schaltet bei abweichender Umgebung die
  Agenten-Einwilligung nur scheinbar ab. Wer schreibt, nimmt den Ort dessen, der die Einstellung auswertet.

## Sprint 12, Sitzung 24.11. nachmittags (Karten #242, #241, #157, #248)

- **Eine Werkzeugoption, die nie geprüft wurde, wirkt womöglich nicht** (F-636): `probe.sh --skala` reichte
  `--scale` an `kwin_wayland --virtual` weiter, das die Option übergeht und die Skala aus
  `tmp/probe/home/.config/kwinoutputconfig.json` nimmt. Aufgefallen erst, als ein Foto „Skala 1.00“ meldete. Bei einer
  Probe die Messgröße im Ergebnis gegenlesen (Foto-Zeile, `preferred_scale` im `WAYLAND_DEBUG`), nicht nur den Aufruf.
- **WASM-Bausteine binden die Laufzeitfassung** (Karte #242): Das mit gogosseract gelieferte tesseract-WASM läuft nur
  bis wazero 1.7.3; ab 1.8 fehlt der Export `emscripten_stack_get_current`. Ein `go get …@latest` baut, scheitert aber
  erst zur Laufzeit; deshalb ein Test mit echter Erkennung im Modul, nicht nur `go build`.
- **Maintainer-Wünsche zu Form gehen vor eigene Gewohnheit** (Karte #157): Gio will keine Co-Authored-By-Trailer und
  keine Werkzeughinweise in Commits und Kommentaren, und kurze Kommentare. Die Offenlegung steht im PR-Text, nicht in
  jedem Commit.
- **Sitzungs-QA mit drei Kritikern nach drei Karten an einem Nachmittag** (F-640 bis F-666, 27 Befunde, keiner P0):
  Die Kritiker fanden genau dort Fehler, wo die eigenen Tests an der falschen Stelle prüften — `ocrHinweis` statt der
  gezeichneten Fußzeile (F-640), „Text ≠ englisch“ statt Sollwert je Fehlerwert (F-660), Wechsel ohne Sperren für
  Export, Texteingabe und Vorschau (F-651 bis F-654). Lehre: Tests gegen das prüfen, was der Nutzer sieht oder was auf
  der Platte landet, nicht gegen eine Zwischenfunktion; je neuer Zustandswechsel (Laden, Wechsel) eine Liste „was darf
  währenddessen nicht passieren“ mit je einem Test.
- **Agenten-Übernahmen formatieren und Muster prüfen** (#174 Teil 1): Ein Agent lieferte gofmt-fremde Dateien; nach jeder
  Übernahme `gofmt -l` über alle Pakete und danach den Mutanten-Trockenlauf, weil Ausrichtung Muster verschiebt.
- **Zwei parallele Fix-Agenten konfliktfrei:** Dateien vorher fest aufteilen (probe.sh nur bei einem), gemeinsame
  Dateien (README, CLAUDE.md, Register) nur der Orchestrator; `de.po`/`schnapp.pot` ließen sich per Cherry-Pick ohne
  Konflikt zusammenführen, danach `TestVorlage` ohne `-update` als Probe.

## Sitzung 24.11. abends (Karten #174 Teil 2, #249, #250, #254 bis #257, #217, F-672, Sitzungs-QA)

- **Echtlauf-Heim vorher mit einem Ladetest prüfen:** Ein Probe-Heim ohne `.config/user-dirs.dirs` hat als Bilderordner
  `Pictures`, `scripts/probelibrary` legt aber nach `Bilder` ab (wie `probe.sh`). Der Index zeigte dann auf Dateien,
  die Schnapp nicht findet; im Echtlauf „gingen nur die grün umrandeten auf“. Vor einem Echtlauf mit eigenem Heim die
  `user-dirs.dirs` anlegen und eine Probe-Datei per `--export` laden.
- **Echtläufe finden nie gerufene Funktionen:** `library.Abgleich` war getestet, aber im Programm nirgends aufgerufen
  (F-672). Beim Echtlauf einmal eine Datei außerhalb von Schnapp löschen/verschieben und die Oberfläche ansehen.
- **Register-Status nur aus dem Vokabular:** `widerlegt …` rotiert nie (findings-rotate meldet „HINSEHEN“); „behoben …
  (widerlegt durch Messung)“ schreiben.
- **`register-schreiben einfuege-nach`: Anker ohne Zeilenumbruch** (`printf '%s' "$(grep …)" > anker.txt`), sonst „Anker
  kommt nicht vor“.
- **aboard-op:** neue Karte per `card.upsert` braucht `card.id` (UUID selbst setzen); `--value @datei` liest keine
  Datei, Freitext über `--payload @datei.json` (`{"cardId","field","value"}`).
- **Testnotbremse bei hängenden Pfaden:** Ein Test, der prüft, dass etwas NICHT in der Ereignisschleife wartet, braucht
  eine Goroutine, die das Tor nach wenigen Sekunden öffnet; sonst hängt der Mutant bis zum Timeout (WR1: 242 s statt
  17 s).
- **Mutant überlebt → Testfall betritt den Zweig nicht:** W1 (Reihenfolge der Modi: der letzte gewann zufällig), BS2
  (Groß/Klein: Testwort ohne Großbuchstaben im Schlagwort). Testdaten so wählen, dass nur der richtige Code besteht.
- **Shell:** `cat > datei` ohne Eingabe blockiert den Aufruf bis zum Timeout; Dateien mit dem Write-Werkzeug anlegen.
- **Agenten-Worktree-Abgabe:** Bericht und Fotos vor `git worktree remove` in den Hauptbaum kopieren und mit
  `cmp`/`diff -rq` gegenprüfen; `git cherry main <branch>` muss nur „-“ zeigen, dann Branch löschen.
- **Lizenzprüfung eines eingebetteten WASM:** Quellpfade im Debug-Modul (`strings … | grep system/lib`, `ports/`)
  zeigen die Bestandteile; Meldungstexte im Release-Modul bestätigen sie. Gekürzte Lizenztexte sind kein Original.

- **Sitzungs-QA 24.11. abends (18 Befunde F-673 bis F-690, drei Fable-Kritiker):** Neue Funktionen öffnen alte
  Lücken: `SetzeSchlagworte` und `InPapierkorb` gab es schon, erst die Ansicht machte sie per Taste erreichbar —
  und damit fehlende Standvergleiche (F-673), Verweis-Ordner (F-675) und Prozesswettläufe (F-686). Lehre: Wird eine
  Bibliotheksfunktion neu von der Oberfläche gerufen, ihre Schutzregeln gegen `Speichern` halten (Standvergleich,
  `Relativ`, Sperre), nicht nur gegen ihre eigenen Tests.
- **Ein neuer Aufrufer macht eine harmlose Funktion gefährlich:** `Abgleich` war ungenutzt und deshalb harmlos; mit
  dem Aufruf bei jedem Start leerte eine nicht eingehängte Ablage den ganzen Index (F-685). Vor dem ersten Aufruf
  einer bisher toten Funktion fragen: was tut sie, wenn ihre Eingabe (Ordner, Datei, Netz) gerade fehlt?
- **Sperre der Ansicht über JEDEN Eingabeweg:** Klick gesperrt, aber F10 → Tab → Enter öffnete die Fußliste (F-674).
  Die Sperre gehört an die Stelle, die die Wirkung auslöst (`menueOeffne`), nicht an einen Eingabeweg.
- **`sed` mit Freitext:** Kommt im Ersatztext der Trenner vor (`/` in „Farb-/Zoom“, `#` in „Karte #259“), bricht
  `sed` ab, und `register-schreiben` schreibt nichts. Registerzeilen mit Freitext aus Python setzen.
- **`ohnePaketpraefix` entfernt alle Präfixe:** Ein Vergleich mit `err.Error()` des Fehlerwerts scheitert dann; beide
  Seiten gleich kürzen (F-678/fehlergruende).
- **Kritiker-Arbeitskopien und Probetests:** Kritiker legen Probetests in ihre Kopie; vor `git worktree remove` nach
  `tmp/…/proben/` kopieren — sie sind die Vorlage der Regressionstests.

## Sprint-Abschluss 12 (24.11.2026 nachts, F-691, F-692 bis F-740, Tag v0.6.0)

- **Kritiker nach Bereichen teilen:** Über 360 Commits und rund 29 000 Zeilen Produktcode trägt ein Kritiker nicht.
  Vier parallele Fable-Kritiker (Modell/Format/Render, Programm/MCP/Bibliothek, Oberfläche, Doku/CI/Repo) mit je
  eigenem Laufordner unter `tmp/sprint12/abschluss/<Teil>/` fanden 35 Befunde, davon sechs Release-Blocker.
- **Belege per `go test -overlay`:** Kritiker und Verify-Läufe legen Probetests und geänderte Kopien unter ihren
  Laufordner und blenden sie mit `-overlay` ein. Der Arbeitsbaum bleibt unberührt, mehrere Läufe können parallel lesen,
  und die Probetests taugen als Vorlage der Regressionstests.
- **Jede [S]-Behebung verlangt einen weiteren Verify-Lauf — die Kette endet erst mit einer Entscheidung:** V1 fand am
  Fix F-692 ein Restleck (F-728), V2 einen alten Nachbarfall auf dem Pixelweg (F-735), V3 dessen Nachbarn (F-739).
  Ein P3, das im Editor genauso sichtbar ist (kein verdecktes Leck), wird als bewusste Grenze im ADR festgehalten
  (Ausnahmepfad), sonst läuft die Kette weiter. Ein Fix am SVG ist zu Ende gedacht erst, wenn Renderer und Editor
  dieselbe Regel haben („eine Rechnung“ gilt auch für Sichtbarkeit).
- **Signaturänderung bricht Mutanten:** `ablegenIn`/`beschrifteDatei` bekamen einen Parameter; die `alt`-Muster von
  C202-2 und K202-M3 passten nicht mehr. Nach jeder Signaturänderung `go run . -wurzel ../.. -trocken` vor dem Commit.
- **Prüfbefehle im Register roh lesbar halten:** In der Tabelle wird `|` als `\|` maskiert; unter `grep -E` ist `\|`
  ein wörtlicher Strich, der Befehl schlug nie an (F-737). Prüfbefehle ohne Alternation schreiben (`[a-z]+` statt
  `(yes|no)`) oder BRE nehmen.
- **Behauptungen über Trailer und Zahlen veralten im nächsten Commit:** „die folgenden Commits tragen beide Zeilen“
  stimmte einen Commit später nicht mehr (F-738). Nacharbeit an Befunden eines Fable-Laufs trägt die Fable- und die
  Opus-Zeile, auch reine Doku-Nachzüge.
- **Abschnitts-Sammelkarten statt einer Karte je Befund:** 35 Befunde → 4 Nacharbeits- und 12 Backlog-Karten, nach
  gemeinsamer Handlung gebündelt; der Abgleich `findings-karten-abgleich.mjs` blieb grün.
- **Worktree-Agent und Cherry-Pick mit Konflikt in CLAUDE.md:** Beide Seiten änderten dieselbe lange Zeile. Die Seite
  des Agenten nehmen und die eigenen Zahlen nachtragen; danach weicht der Branch ab und fällt nicht unter die
  Löschregel ohne Rückfrage.

## Sprint 13 (24.11.2026 abends, Karten #269 bis #284)

- **Headless-Zeichentests nie pixelgenau gegen sich selbst verschoben vergleichen:** Die CI rastert mit llvmpipe; an
  anderer Lage glättet Gio Kanten um eine Stufe anders (lokal mit GPU exakt gleich). Toleranz 2 Stufen je Kanal; lokal
  nachstellen mit `LIBGL_ALWAYS_SOFTWARE=1 GALLIUM_DRIVER=llvmpipe SCHNAPP_HEADLESS_PFLICHT=1`.
- **„Je Bereich zeichnen“ vervielfacht die Arbeit über vielen Nähten:** Bis `MaxStreifen` Nähte heißt bis zu 1025 Bereiche
  je Objekt. Der Prüflauf beim Öffnen (F-136, `TestPruefeVieleStreifenUndPfeile`) wurde in der CI unter `-race` zu langsam.
  Je Bereich nur, wo die Rechnungen außerhalb des Kantenbands wirklich verschieden sind (Text, Nummer, schräge Kanten
  eines Dreiecks); Kosten im Trockenlauf einmal zählen, nicht je Bereich.
- **CI-Protokoll lesen ohne `gh`:** Die API liefert Job-Logs nur mit Admin-Rechten; Schritte und Ergebnis gibt
  `api.github.com/repos/<o>/<r>/actions/jobs/<id>` öffentlich. Den Text liefert im Browser
  `github.com/<o>/<r>/commit/<sha>/checks/<job>/logs` (leitet auf eine Blob-Adresse um), gefiltert per JavaScript auf
  `--- FAIL`/`DATA RACE`; Tab danach schließen.
- **`git --work-tree=… checkout REV -- pfad` ändert auch den Index:** Danach `git restore --staged pfad`. Für einen alten
  Stand zum Vergleich besser `git show REV:datei > tmp/…` je Datei oder `go test -overlay` mit einer Ersetzungsliste.
- **govulncheck findet echte Treffer:** `golang.org/x/image` v0.26.0 hatte fünf erreichbare Schwachstellen (SFNT, TIFF,
  F-744). Ein neuer Prüfschritt in der CI ist erst fertig, wenn sein erster Lauf ausgewertet ist.
- **Worktree-Agenten parallel nach Dateien schneiden:** sechs Sonnet-Karten in Worktrees, je eine Welle ohne gemeinsame
  Dateien; Cherry-Picks liefen ohne Konflikt. Sonnet hielt sich trotz Verbot nicht immer an „keine Heredocs“ — es meldete
  es aber; der Diff war jeweils sauber.
- **Trailer-Block braucht eine Leerzeile davor:** Hängen die `Co-Authored-By`-Zeilen direkt am Fließtext (abe287c, aus
  einem Worktree-Commit übernommen), erkennt git keinen Trailer, und der Modell-Abgleich aus dem Repo greift nicht.
  Nach jedem Cherry-Pick prüfen: `git log -1 --format='%(trailers:key=Co-Authored-By,valueonly)'`.
- **Sitzungs-Kritik nach Sprint 13 (drei Fable-Kritiker, 23 Befunde):** „je Bereich zeichnen“ hatte ein Loch bei genau
  einem Bereich (Text endet im Streifen, [S]) und vervielfacht Aufwand, den das Budget nicht zählt (Zeilen × Bereiche,
  Glyphen je Bereich). Eine Behauptung im ADR („außerhalb des Kantenbands gleich“) gilt nur für die gemessenen Fälle —
  Grenzfälle (lange Pfeile, breite Striche, Rundung) mitmessen, bevor sie im ADR steht. CI-Shell ohne `shell: bash`
  heißt `bash -e` ohne pipefail.

## Sprint 13, Restkarten und Abschluss-Kritik (25.11.2026, Karten #285 bis #295)

- **Rasterkern nicht verschiebungsfest (F-764):** Der Festkomma-Weg aus `x/image/vector` summiert x je Zeile mit
  Abschneiden; `dxdy` in float32 hängt von der Lage des Ausschnitts ab. Dieselbe Form wich je nach Ausschnitt um bis
  22 % Deckung entlang langer Kanten ab — sichtbar erst, als ein beschnittener Bereich mit dem ungestauchten Bild
  verglichen wurde. Erster Verdacht war die eigene Änderung; die Gegenprobe mit reinem Zuschnitt auf dem alten Stand
  (`go test -overlay` mit `git show HEAD:… > tmp/…`) zeigte die Altlast. Zufallsformen fanden sie nicht; der Test
  braucht den gemessenen Fall und Koordinaten wie im Renderer (float64 verschieben, dann float32).
- **Referenz für Bitgleichheit muss denselben Weg rechnen:** `vector` wählt den Gleitkomma-Weg über Breite **oder Höhe**
  > 512. Mit gleicher Breite und Höhe ≥ 513 rechnet die Referenz in derselben Speicheranordnung; eine größere Breite
  verschiebt den Übertrag über Zeilenenden und rundet anders.
- **Mutanten nach jedem Umbau trocken prüfen:** Ein Umbau verschiebt Muster anderer Gruppen (`schrift`, `text-naht`); `cd
  scripts/mutanten && go run . -wurzel ../.. -trocken` über alle Gruppen findet sie in Sekunden (eigenes Modul: `go run
  ./scripts/mutanten` aus der Wurzel findet es nicht). Ein Mutant, dessen Ersetzung eine Variable
  ungenutzt lässt, baut nicht (FEHLER, nicht ROT) — im Muster `_ = x` ergänzen.
- **Überlebende Mutanten zeigen Testlücken genau:** N3/N3b (Text über einer Naht nie pixelweise verglichen), P5/P6
  (Editor-Pfeil und -Kasten nur in Fällen getestet, in denen die alte Abbildung zufällig gleich aussah). Testfall so
  wählen, dass alt und neu außerhalb des Kantenbands sichtbar verschieden sind (breiter Strich, Rundung im Streifen).
- **Eine bestehende Datei nicht mit Write überschreiben:** `naht_aufwand_test.go` gab es schon; Write meldete „updated“
  statt „created“ — vor dem Anlegen `ls` auf den Namen.
- **Integration von Agenten-Commits braucht die volle Kette:** `TestSpeichernDateiWegWaehrendEditor` (cmd/schnapp) fing
  eine Folge des F-755-Fixes, die der Agent im Paket `library` nicht sehen konnte; der gofmt-Schritt der CI fing eine
  unformatierte Testdatei aus einem Agenten-Worktree. Vor dem Push `gofmt -l` über beide Build-Varianten wie in der CI.
- **Sichtprobe statt Raten bei Drehbuch-Koordinaten:** Die Fußzeile („bei X, Y“) im Foto zeigt, wo der Zeiger im Bild
  lag. Im 1600 breiten Probefenster steht ein 1200 breites Bild ab x = 226; nach dem Stauchen wird es neu zentriert
  (1010 breit ab x = 321). Klicks vor dem Stauchen rechnen mit 226, danach mit der neuen Lage (Kritik 2 C, C2-1: der
  erste Eintrag hier verwechselte beides).
- **Eine Datei lesen und neu schreiben, in dieser Reihenfolge (F-778):** `open(a,'w').write(open(a).read()+…)` öffnet
  zum Schreiben (und kürzt), bevor gelesen wird — das Finding-Archiv schrumpfte so auf 5 Zeilen und wurde gepusht.
  Erst lesen, dann schreiben; nach jedem Register-Schreiben `grep -c '^| F-'` gegen den Stand davor halten
  (`register-schreiben.mjs pruefe` prüft nur `docs/findings.md`, nicht das Archiv).
- **Verify über die Nacharbeit findet Regressionen in „behoben“-Fixes:** Das Entdoppeln der Indexzeilen (F-773) machte
  „gleiche Zeilenzahl“ zu einem falschen Beleg für „nichts zu tun“ (F-781). Wer eine Bedingung um einen Fall erweitert,
  prüft jeden Ausstieg, der auf dem alten Zusammenhang beruht.
- **Hochrechnungen ohne Grundlast täuschen:** Die Kritik rechnete „mit Streifen“ auf 46 s hoch; rund 2,5 s davon waren
  die Kanten der 1023 Nähte selbst, die auch ohne Text anfallen (F-766). Erst die Grundlast ohne Objekte messen.
- **Editor-Zeit über ganze Bilder messen, nicht über den Befehlsaufbau:** 4 ms Befehlsaufbau waren 3,5 s je Bild mit
  llvmpipe (F-765). Gio rechnet jede beschnittene Füllung einzeln; ein Pfad mit vielen kleinen, selbst beschnittenen
  Trapezen statt vieler Schnitte war 40-mal schneller.
- **Trailer bei Nacharbeit nach Kritik:** Fix-Commits, die Befunde eines Fable-Kritikers beheben, tragen Fable mit
  (CLAUDE.md §Commit-Konvention, dritter Fall) — auch wenn ein Agent den Commit schreibt: im Agenten-Prompt vorgeben
  (F-780, F-786).

## Sprint 13, Verify 2 bis 5, Karten #299 bis #306 und Sitzungs-Kritik (25.11.2026, F-787 bis F-816)

- **Fehlermails der CI sind ein Befund, kein Rauschen** (F-791, F-812): Der Job `wurzel` war am 25.11. von 09:40 bis
  16:08 Uhr bei sieben Pushes in Folge rot (gofmt, viermal eine feste Zeitgrenze von 2 s unter `-race` mit 4,8–5,7 s,
  F-798, F-802); schon am 24.11. gab es zehn rote Läufe, seit dem 24.11. 18 von 89. Danach wackelte noch Lauf #389
  bei unverändertem Code (Grenzfall 2,12 s gegen 2 s auf dem Runner, F-812) — „grün ab …“ erst nach mehreren Läufen
  schreiben, und jeden roten Lauf eintragen, auch zwischen zwei grünen. Zeitgrenzen gehören in
  Konstanten mit Build-Tag `race`/`!race` (Muster `zeitgrenze_test.go` in `render`, `fileformat`, `ui`); nach jedem Push
  die check-runs ansehen, nicht nur bei Verdacht. Lokal nachfahren ist schneller als das Protokoll zu holen.
- **Erst messen, wohin die Zeit geht** (F-787): 90 % der Bildzeit über vielen Kacheln war das Abspielen einer leeren
  Füllung (Alpha 0) in 4096 Bereichen, nicht der vermutete Strich.
- **Bänder eines Rasters vergleicht man mit ±1, nicht bitgleich**: Der Rasterizer rechnet float32 relativ zum Ausschnitt.
- **Eine Zahl im ADR veraltet mit dem nächsten Umbau** (F-795): Die Budgetgrenze „Strich 41“ galt nur bis Nachtrag 21;
  vor dem Berichtigen auf dem aktuellen Stand nachmessen und beide Stände nennen.
- **Mutanten-Namen in Commit, ADR und Testkopf wörtlich aus der JSON übernehmen** (F-794), nicht als Bereich „A1 bis A8“.
- **Grenzfälle der Geometrie gezielt treffen** (F-792): sechs Testfälle verfehlten genau Strich = 2·Radius; Fälle
  an den Umschaltpunkten der Rechnung (Gehrung erreicht Mittelpunkt) gehören in die Tabelle.
- **Ein abgelaufener Login beendet Agenten mitten im Lauf**: Sie lassen sich mit SendMessage an derselben Stelle
  fortsetzen; vorher Hauptbaum und Worktree auf Zwischenstände prüfen.
- **Eine summierende Füllung deckt nur dann wie die Fläche, wenn sich die Teile nicht überlappen** (F-788, F-796):
  Rechteck plus Kreis je Gelenk addiert die Glättung an jedem Gelenk (bis 0,63 Abweichung). Gilt für jeden Strich.
- **Eine billigere Zerlegung senkt die gezählten Kosten**: Dann nimmt `Pruefe` mehr Objekte an (breite Pfeile 171 → 429);
  die Zeit an der Grenze nachmessen, nicht nur je Objekt.
- **Zeittests knapp unter ihrer Grenze sind Wackelkandidaten** (F-802): 0,8–1,0 s unter `-race` gegen 1 s kippt unter
  Last. Jede Rechenzeit-Grenze gehört in `zeitgrenze_test.go`/`zeitgrenze_race_test.go`; einen rot gewordenen Lauf
  immer mit `tee` sichern, sonst bleibt die Ursache Vermutung.
- **Vorzeichen-Entscheidung und Atan2 widersprechen sich bei −0** (F-803): `sin < 0` ist für −0 falsch, `Atan2(−0, −1)`
  liefert −π. An exakten Sonderfällen (Kehrtwende) Seite und Winkel aus einer Entscheidung setzen und alle vier
  Achsrichtungen testen.
- **Ein Fang gegen Splitter zieht nur die Punkte, die ihn brauchen** (F-804): Den Zielpunkt auf den wahren Rand legen,
  nicht auf einen bequemen Punkt (Mittelpunkt), sonst wandert eine exakte Kante mit; analytische Tests brauchen Fälle
  im Band des Fangs, sonst bleibt selbst `mittelpunktFang = 2.0` grün.
- **Jede Verify-Runde nach [K]-Nacharbeit fand neue Fehler der Nacharbeit** (Runden 2 bis 4): Die Kette endet erst,
  Fixes kleiner schneiden verkürzt die Kette; wie sie endet, steht im nächsten Punkt.
- **Die Verify-Kette endet mit einer Entscheidung, nicht mit null Befunden** (Verify 5): Ein [K]-P3-Altbestand, den die
  Nacharbeit verkleinert hat und der höchstens ein Pixel betrifft, wird als bewusste Grenze mit Messwert ins ADR und als
  „akzeptiert“ ins Register geschrieben, statt einen geprüften Pfad ein weiteres Mal umzubauen (F-807, F-809).
- **Ein Mutant, den `klemme` o. ä. verdeckt, braucht einen Geometrietest statt eines Deckungstests** (F-811): Die
  Deckung blieb ohne die Regel in der Toleranz; erst die Prüfung der kürzesten Teilkante machte ihn rot.
- **Jedes Finding bekommt eine Karte, auch eines, das im selben Zug behoben oder akzeptiert wird** (Sitzungs-Kritik,
  F-815): F-807 bis F-811 gingen ohne Karte direkt ins Archiv; der Abgleich prüft nur offene Findings und blieb grün.
- **Agenten-Prompts nennen die Trailer-Zeile des Befund-Modells** (F-816, wie F-780): Fix-Commits zu Fable-Befunden
  trugen nur „Opus“ bzw. „Sonnet“; gepusht ist das nicht mehr heilbar.
- **Öffentliche Release-Notizen vergleichen mit dem vorigen Tag, nicht mit Zwischenständen des Sprints** (F-813): Ein
  im Sprint eingebauter und wieder behobener Fehler (8 s je Bild, fehlende Kappe) ist keine Verbesserung gegenüber
  v0.6.0; jede Zeile gegen `git log v<vorher>..v<neu>` und den Code des vorigen Tags prüfen.

- **„Funktioniert nicht“ heißt erst nachstellen, dann deuten** (F-817, Sprint 14): Im virtuellen KWin kamen alle Klicks
  aufs Farbpaar an; der Mangel war die Wirkung (kleine Liste statt Farbwähler). Drehbuch mit Fotos je Klick klärt in
  Minuten, ob Zeigerziel oder Verhalten fehlt — und das Ergebnis gehört als Frage in den Schlussblock an Andi.
- **Was der virtuelle KWin nicht fotografiert, ist ungeprüft** (F-818): Zeigerformen sind in Screenshots unsichtbar;
  grüne Tests von `canvas.Sitzung.Zeiger` belegen die Entscheidung, nicht die Anzeige. Solche Karten gehen erst nach
  Andis Echtlauf auf erledigt — oder mit dem Vermerk „Echtlauf offen“.
- **`… | grep … | head; echo $?` misst `head`, nicht `grep`** (Sprint 14): zweimal ging rot als grün durch. Vor dem
  Commit `go test ./...` ohne Pipe-Ende oder mit `PIPESTATUS` auswerten; Pakete einzeln testen ersetzt den vollen Lauf
  nicht (TestVorlage in i18n und TestRepoRegeln in cmd/schnapp prüfen andere Pakete mit).
- **Eine Änderung der Voreinstellung zieht alle eingebauten Stile und Vorgaben mit** (#316): Strich 4 → 6 berührte
  `StandardStil`, die Schnellstil-Tabelle, `settings.stilBreite` und ~20 Testerwartungen; Tests mit „eine Breite, die
  keine Kachel hat“ (6) wurden still zu Doppelten.
- **Ein Klick-Mutant, der den Zweig nur über den Nebeneffekt eines anderen Tests betritt, überlebt** (OF4, ZF9):
  Vorher stehende Meldungen und gleichzeitig wechselnde Pixel verdecken die geprüfte Wirkung; Meldung vor dem Schritt
  leeren bzw. den Fall so bauen (starker Zoom), dass nur die geprüfte Größe sich ändert.

## Sprint 14, Karten #347, #315, #345 und Sitzungs-Kritik (26.11.2026, F-825 bis F-831)

- **Wer einen Ablauf umlenkt, prüft die Schutzwarnungen des alten Wegs** (K-1/D-1, #345): Die Übernahme beim Öffnen
  machte jede Datei von außen zur Aufnahme — die Warnung „das Original zeigt das Geschwärzte noch“ hing am alten Weg und
  fiel still weg. Beide Kritiker fanden es unabhängig. Vor dem Commit nach `aussen`, `Warnung`, [S]-Pfaden im
  verlassenen Zweig suchen.
- **Ein Schalter, den `TestMain` für alte Tests abschaltet, lässt den Standardweg ungetestet** (K-8): `importBeimOeffnen
  = false` hielt die Rückfall-Tests richtig, aber nur zwei Tests fuhren das ausgelieferte Verhalten. Die wichtigsten
  Altfälle (Schwärzen, Budget) auch mit eingeschaltetem Schalter fahren.
- **Ein frisch gezeichnetes Objekt bleibt gewählt, und die Leiste ändert es** (#315 Sichtprobe): Ein Drehbuch, das nach
  dem Zeichnen die Spitzenart für den nächsten Pfeil wählt, änderte den eben gezeichneten. Vor jeder neuen Wahl `Esc`.
- **gofmt richtet Zeilenkommentare an der Nachbarzeile aus** (IM5): Ein Mutanten-Muster mit `… //` traf nach dem
  Formatieren nicht mehr. Muster ohne Zeilenkommentar wählen oder den Kommentar auf eine eigene Zeile setzen.
- **„Kürzer als der Kopf“ ist nach Abstand, nicht nach Bogenlänge, gemessen** (K-4): Bei einer Haarnadel verschwand der
  ganze Schaft. Geometrie-Rückfälle mit gebogenen und gefalteten Kurven prüfen, nicht nur mit geraden.
- **Ein neues Maß ersetzt alle alten Obermaße** (K-5): Die Ebenenvorschau wurde auf `Form().Huelle` umgestellt, der
  Rahmen des Zwischenbilds im Renderer nicht; bei Deckkraft unter 100 % fehlten Schenkel. Nach einer Maßänderung nach
  dem alten Ausdruck greppen (`max(8, 4`).
- **Ein Drehbuch-Lauf mit `--bild` darf keine Daten im Probe-Heim hinterlassen, die den nächsten Lauf ändern** (D-5):
  Die Übernahme legte das Testbild samt Beschriftungen ab, der zweite Lauf hätte es wiedergefunden. Der Probe-Build
  schaltet die Übernahme im Drehbuch-Lauf ab.

## Sprint 14, Karten #348, #351 und Sitzungs-Kritik (26.11.2026)

- **Eine Warteschleife mit `pgrep -f MUSTER` findet sich selbst** (#351): Die Befehlszeile der wartenden Shell enthält
  das Muster, die Schleife endet nie (zwei liefen fünf Stunden). Auf einen Hintergrundlauf nicht pollen, er meldet sich
  selbst; sonst auf eine Ergebniszeile in einer Datei warten, mit Zeitgrenze. Hängende Aufgaben mit TaskStop beenden.
- **Ein neuer Abschnitt in der rechten Leiste verschiebt jedes Drehbuch, das beim selben Werkzeug weiter unten klickt —
  und niemand merkt es** (#315 → #351): `farbwaehler-189.txt` klickte seit #315 um 90 px daneben; ein Drehbuch prüft
  nichts, der Lauf blieb „grün“. Nach jeder Änderung der Leistenhöhe `git grep -l -E '^(klick|ziehe) 1[3-5][0-9][0-9] '
  scripts/drehbuecher`, jedes Drehbuch mit demselben Werkzeug fahren und die Fotos ansehen; Maße im Kopfkommentar mit
  Karte und Datum der letzten Messung.
- **Zwei gleichartige Felder mit verschiedenen Voreinstellungen brauchen verschiedene Nullwerte** (#351,
  `Pfeil.Ende`/`Anfang`): Ende ohne Angabe = Dreieck, Anfang ohne Angabe = keine. Mit demselben Typ trüge jeder alte
  Pfeil (Nullwert) zwei Spitzen, und 40 Test-Literale samt Prüfsummen kippten. Lösung: eigener Typ `Anfangskopf`, um
  eins verschoben, Zugriff nur über `KopfAnfang()`/`MitKoepfen()`, gehalten von Mutant PF23. Vor jedem neuen Feld an
  einem gespeicherten Objekt fragen: Was zeigt der Nullwert in alten Dateien, Tests und Literalen?
- **Ein Vermerk „abgelöst durch Nachtrag N“ gehört an jede Stelle, die den alten Nachtrag zitiert** (#348): ADR-0020,
  ADR-0036 und E-16 trugen noch „seit Nachtrag 10 … beim Öffnen“. Nach einem ablösenden Nachtrag nach der alten Nummer
  greppen (`git grep -n 'Nachtrag 10' docs/`).
- **Was von selbst speichert, darf nichts Endgültiges auslösen** (Kritik K-1, K-2): Das automatische Speichern nach der
  ersten Änderung leerte bei einer Schwärzung den Verlauf und machte sie sofort unumkehrbar; die haltende Warnung beim
  Schließen wurde zur flüchtigen Meldung. Ein automatischer Auslöser prüft vorher, ob der Schritt Folgen hat, die der
  Nutzer sonst bewusst bestätigt (hier: `VerlaufMitSchwaerzung`).
- **Ein Ansatz, der eine Datei elffach vergrößert, ist kein Fix** (#349, F-826): Den Pfeil im SVG als einen Pfad aus den
  Teilen des Renderers zu schreiben, beseitigte die Fuge, blähte `arrow-heads.snap.svg` von 3 auf 38 KB und band
  `render` in `fileformat` ein (Importzyklus in render-Tests). Vor dem Bau die Größe als Grenze festlegen und messen.


## Sprint 14, Karten #349 bis #352 und #358 (26.11.2026, F-826 bis F-833)

- **Kopien eines Baums an wechselnden Orten füllen den Go-Build-Cache.** Ohne `-trimpath` steht der absolute Quellpfad in
  jedem Cache-Schlüssel; jeder Mutationslauf (neuer Ordner `lauf-…`, darin auch `third_party/gio` mit cgo) baute alles neu
  und legte es neu ab. `~/.cache/go-build` war am 26.11. 116 GB groß, `/home` voll (F-833). Seitdem: `-trimpath` und ein
  eigener Cache `tmp/mutanten/gocache`, über 8 GiB vor dem Lauf mit `go clean -cache` geleert (kein `rm`). Gemessen: zwei
  Läufe hintereinander, der Cache bleibt bei 181 MB. Lehre allgemein: Wer Werkzeug-Läufe in wechselnden Ordnern baut,
  prüft den Cache nach zwei Läufen (`du -sh $(go env GOCACHE)`), nicht erst, wenn die Platte voll ist.
- **Eine Wartebedingung im Test muss das Ende des ganzen Ablaufs treffen.** `!laedt` war schon vor dem Laden wahr (erst
  speichert der Wechsel); der alte Test bestand nur, weil das Speichern schnell war. Fertig ist ein Wechsel erst, wenn
  `wechsel.ziel` leer ist.
- **Ein Mutant, der in einer Goroutine eine nil-Funktion ruft, endet als Panik ohne Testnamen** (FEHLER statt ROT). Vor
  dem Aufruf im Test auf nil prüfen und mit `t.Fatal` benennen (N5 in `scripts/mutanten/rescue.json`).
- **Doppelter Schutz macht einen Mutanten gleichwertig** (BV21 in `scripts/mutanten/bibliothek-ansicht.json`: `menueOeffne` sperrt die Listen selbst): als äquivalent mit
  Grund vermerken, nicht den Test verbiegen.
- **Ein Merker richtet sich nach dem Ergebnis der Anfrage, nicht nach einem Zustandsvergleich danach** (Kritik 26.11.
  K-6): `if e.wechsel.ziel == x` nach `wechselAnfrage(x)` war auch wahr, wenn die Anfrage abgelehnt wurde und schon ein
  Wechsel zu x lief. Seitdem liefert `wechselAnfrage` ein `bool`.
- **Eine Grenze, die nur gegen endliche Daten getestet ist, ist nicht getestet** (K-8): `io.LimitReader` in
  `dateiSumme` überlebte die Mutation, bis ein Test mit endlos schreibender FIFO dazukam.
- **Ein gemeinsamer Cache mehrerer gleichzeitiger Läufe braucht eine Sperre vor dem Leeren** (K-2): geteilte flock je
  Lauf, Leeren nur mit exklusivem Griff ohne Warten (`scripts/mutanten`, `gocache.lock`).
- Vermutung widerlegt statt umgebaut (K-7): Gio liefert Loslassen auch an ein Ziel, das nach dem Einklappen nicht mehr
  gezeichnet wird; der Test `TestStrichReglerLeisteVerschwindetImZug` hält das fest.

## Sprint 14, Karten #353 bis #364 und Sitzungs-Kritik (4) (26.11.2026)

- **Wer ein Dokument über einen Wechsel hinweg aufhebt, braucht den Dateistand der eigenen Speicherfunktion, nicht den
  der Platte** (F-834, [S]): Der Stand beim Verlassen von der Platte zu lesen, nahm einen fremden Stand mit (ein anderes
  Fenster hatte geschwärzt), und der Rückwechsel hätte das Ungeschwärzte zurückgeschrieben. Der Stand, gegen den F-110
  schützt, ist der, den das eigene Schreiben zuletzt gesehen hat (`Geoeffnet.Stand`); nur wenn die Platte ihn noch
  trägt, ist das gemerkte Dokument aktuell.
- **Ein Zwischenspeicher wird erst verbraucht, wenn sein Inhalt wirklich übernommen ist** (F-838): beim Start eines
  Ladens herausnehmen verlor ihn bei jedem Fehlschlag. Holen ohne Entfernen, Entfernen in der Übernahme.
- **„Äquivalent“ nur nach dem Gegenbeispiel-Versuch:** V2 (Schwärzung im Verlauf) schien äquivalent, weil Speichern den
  Verlauf leert — aber Schwärzen und Rückgängig ohne Speichern ist ungeändert und hält die Schwärzung im Verlauf (F-835).
  Vor dem Markieren den Zustand suchen, in dem die Klausel allein entscheidet.
- **Eine Schleife, die ihren eigenen Gegenstand ändern kann, hört danach auf** (F-836): `for j := range f.farben` lief
  nach dem Entfernen mit der alten Länge weiter (Index hinter dem Ende). Nach einer Änderung verfallen weitere Klicks
  desselben Bilds.
- **Fehlerwerte eines geborgten Helfers tragen dessen Text** (F-837): `leseStileRoh` lieferte `ErrStileKeineDatei`, und
  `fehlerGrund` meldete „the styles file …“ für `colors.ini`. Ursache als Text anhängen, nicht mit `errors.Join`.
- **`testEditor` umgeht `neuerEditor`:** Was nur dort geschieht (`stileLaden`, `farbfelderLaden`), haben Tests nicht.
  Zustand, den das Zeichnen braucht, bekommt einen Ausweichwert am Ort des Lesens (`farbfeldLeiste.liste()`) und eine
  Naht für die Datei (`ladeFarben`/`aendereFarben`), statt im Test `neuerEditor` nachzubauen.
- **Ein neuer Schreiber im Paket `settings` braucht seine Ausnahme im Graph-Test** `TestMCPErreichtDenSchreiberNicht`
  (jeder Aufruf `settings.Schreibe…`/`Aendere…` außerhalb von `einstellungen.go` fällt sonst durch); die Ausnahme dort
  begründen (nur Farben, keine Einwilligung eines Agenten).
- **Vor einem neuen Begriff `git grep -i` über Code, ADRs und de.po:** „Farbfeld“ hieß in ADR-0021 Nachtrag 23 das Feld
  des Farbpaars, in Nachtrag 24 die eigene Palette. Glossar-Eintrag im selben Commit.
- **Oberflächentexte englisch amerikanisch** („color“, wie die übrigen msgids); die Nutzer-Doku darf britisch bleiben.
  Deutsch einheitlich „ungesichert“ und „ignoriert“ wie in den vorhandenen Einträgen.
- **Eine Leiste, die wächst, hat Nachbarn mit Schwellen:** Die Farbfelder verlängern den Farbabschnitt; die Schwelle für
  das Einklappen der Ebenen (`farbwaehlerHoehe`) muss mitwachsen, und Drehbücher mit Leistenkoordinaten gehören neu
  gefahren.
- **Testvorrichtungen müssen neue Pflichtangaben mitliefern:** Seit `Geoeffnet.Stand` merkt der Editor ohne Stand nichts;
  die Leisten-Tests bekamen `mitStand`, sonst wären die Verlaufs-Tests still grün geworden, ohne etwas zu prüfen.

## Sprint 14, Karte #362 Strichart (26.11.2026)

- **Wer Vielecke einer Zerlegung in Gio füllt, richtet sie gleich herum aus:** Der Renderer dreht jedes Vieleck auf
  positive Fläche (`render.raster`), `fuelleTeile` im Editor tat es nicht. Sechseck des Linienstrichs und Kappe bzw. Kopf
  laufen gegeneinander; Gio hob die gemeinsame Kante auf, am Ansatz jeder Pfeilspitze stand im Editor eine helle Fuge —
  unbemerkt, bis ein Pixelvergleich Editor gegen Renderer mit vollem Strich lief. Ein neuer Zeichenweg bekommt diesen
  Vergleich (`TestEditorZeichnetStrichartWieRenderer`), nicht nur Deckungstests im Renderer.
- **Gemusterte Striche aus derselben Rechnung wie der volle:** Die Stücke rechnet `document`, Renderer, Editor und SVG
  machen daraus Teile. Im SVG ausdrücklich statt `stroke-dasharray` (die Bogenlänge einer Bézier misst jeder Betrachter
  anders). Eine Lücke im zusammengesetzten Linienstrich ist eine Strecke ohne Teil (`strecke` −1), so bleibt die
  Bereichssuche über Nähte dieselbe.
- **Ein Mutant, der eine geänderte Zeile trifft, gehört mitgezogen** (Trockenlauf zeigt FEHLER, hier N13 in
  `text-naht.json`): Muster auf die neue Zeile setzen, Absicht des Mutanten behalten.

## Sprint 14, Karte #363 Teil 1 und Sitzungs-Kritik (5) (26.11.2026, F-840 bis F-848)

- **Geometrie, die aus Dateiwerten wächst, braucht eine Obergrenze vor dem Budget** (F-840, [S]): Gemusterte Striche
  erzeugten je Objekt so viele Stücke, wie die Länge hergab (±1e6 Koordinaten: 1,4 Mio. Scheiben, 0,35 s), und das schon
  im Trockenlauf von `render.Pruefe`, der nur die sichtbaren Teile zählte. Regel: Was eine Rechnung aus einer Länge
  vervielfacht, hat eine feste Obergrenze je Objekt (`MaxMusterStuecke`), und seine Kosten zählen, bevor gezeichnet wird.
- **Der Schreiber hält die Grenzen des Lesers** (F-841): Das SVG der Arbeitsdatei konnte über `maxSVG` wachsen; die
  Datei ließ sich danach nicht mehr öffnen. Jede Lesegrenze bekommt eine Prüfung im Schreibweg, bevor die alte Datei
  ersetzt wird — und Darstellung, die der Leser nicht braucht, bleibt in der Arbeitsdatei knapp.
- **Jede neue Angabe, die aus der Ausdehnung eines Objekts folgt, verrät, was die Ausdehnung verrät** (F-842, Klasse
  F-728): Die Filterfläche des Schattens trug die Breite des verborgenen Schilds. Bei einem neuen Attribut mit Maßen
  prüfen, ob der Export das Objekt nur teilweise schreibt (`ohneText`), und die Maße aus dem Geschriebenen nehmen.
- **Ein Zwischenspeicher je Objekt hängt nicht am Stand des ganzen Dokuments** (F-843): Sonst rechnet jede Änderung an
  einem Objekt alle Einträge neu. Schlüssel = das Objekt selbst plus genau das, was sein Bild außerdem bestimmt; dazu
  eine Speichergrenze und Leeren beim Wechsel der Aufnahme (Klasse F-799).
- **Kastenfilter-Näherungen haben eine Untergrenze** (F-846): Drei Kästen der Breite 1 tun nichts; unter σ = 2 rechnet
  ein echter Gaußkern.
- **Commit-Nachrichten per Datei, nie per Heredoc** (Projektkonvention, Kompaktblatt): In dieser Sitzung liefen die
  Commits mit `git commit -F -` und Heredoc; die Inhalte stimmen, die Regel nicht. Nachricht mit dem Write-Werkzeug nach
  `tmp/<lauf>/` schreiben und `git commit -F` darauf.
- **Wer einen Schlüssel schmaler macht, prüft, was der alte nebenbei abdeckte** (Verify V-2): `D.Stand()` im Schlüssel
  der Schattenbilder war zu grob (F-843), deckte aber Kantenform und Rückgängig einer Schwärzung mit ab, an denen der
  Lupeninhalt hängt. Vor dem Entfernen jede Eingabe der gemerkten Rechnung aufzählen.
- **Eine Speichergrenze verdrängt einzeln, nie das gerade Gebrauchte** (V-3): „über der Grenze alles leeren“ machte aus
  einem Speicher eine Dauer-Neuberechnung, sobald ein Bild allein über der Grenze lag.
- **Eine Obergrenze gilt erst, wenn jeder Weg sie zählt** (V-1, Rest F-840): Pfeil und runder Kasten zählten ihre
  Stücke, eckige Rahmen nicht, weil `fuelle` nur Teile im Bild zählt. Beim Schließen einer Budgetlücke alle Aufrufer der
  vervielfachenden Rechnung durchgehen (hier `RahmenTeile`: Kasten, beide Lupenrahmen, Editor).


## Sprint 14, Karte #363 Teil 2 und Sitzungs-Kritik (6) (26.11.2026, F-854 bis F-859)

- **Eine übernommene Sperrbedingung gilt nur für die Fälle des Vorbilds** (F-854): `zug == zugText` im Strichregler war
  harmlos, weil Text keinen Strich hat; im Schattenregler machte dieselbe Zeile den Regler beim Text stumm. Wer ein Muster
  kopiert, prüft jede Sperre gegen die Arten, für die das neue Element gilt.
- **Ein gemerkter Modus gehört dem Ziel, für das er gewählt wurde** (F-855): Ein Feld wie „Schatten“, das nur unter einer
  Bedingung wirkt, kehrt sonst bei einem anderen Objekt ungefragt zurück, sobald die Bedingung dort eintritt. Das Ziel
  mitmerken (`feldSchattenID`), statt nur den Modus.
- **Vor einer Teiländerung die Sperre prüfen** (F-858): Wer erst eine Nebenwirkung setzt (Feld zurück) und dann die
  eigentliche Änderung versucht, die abgelehnt wird, lässt Anzeige und Zustand auseinanderlaufen.
- **Eine neue Zeile in der rechten Leiste verschiebt alle Drehbücher darunter** — und kann Abschnitte in den Platzmangel
  schieben (Ebenen). Nach jeder Höhenänderung die Drehbücher mit Leistenkoordinaten fahren, nicht nur umrechnen: Ein
  verschobener Klick traf bei `ebenen-191` die Überschrift und klappte den Abschnitt zu.

## Sprint 14, Karte #313 Teil 1 und Sitzungs-QA (26.11.2026, F-860 bis F-872)

- **Ein neuer Eingabeweg erbt nicht die Wächter der alten** (F-862, F-863, F-864): Der Linksdruck beendet Pipette,
  Bereichsmodus und Ablehnung in `Druecken`; ein Rechtsklick, der daneben einsteigt, muss dieselben Modi selbst
  beenden, sonst wirkt der nächste Klick im alten Modus auf das neue Ziel. Und ein Zeiger, den die Leinwand vom
  Rechtsklick noch hält, liefert einen Linksdruck an die Leinwand, auch wenn die Liste offen ist.
- **Ein neuer Tastenfilter mit Umschalter trifft auch ältere Zweige, die den Namen ohne Umschalter prüfen** (F-860):
  `k.Name == key.NameF10` im Zweig der Bibliothek fing Umschalt+F10 ab, sobald der Filter dafür existierte. Beim
  Hinzufügen einer Tastenkombination alle Stellen suchen, die nur den Namen vergleichen.
- **Testhelfer, die „Ruhe“ abwarten, brauchen jede Hintergrundarbeit** (F-871): eine fehlende Flagge lässt eine
  Goroutine nach Testende in `t.TempDir` schreiben, und das Aufräumen scheitert nur gelegentlich (CI, 59515e4).

## Sprint 14, Karte #334 Oberfläche und Sitzungs-QA (12) (27.11.2026, F-905 bis F-911)
- **Gio prüft die Umschalter eines `key.Filter` auch beim Loslassen** (F-905): Wer einen Zustand „Taste gehalten“
  über Press/Release führt, erlaubt im Filter alle Umschalter (`Optional`), wertet Release immer und filtert erst beim
  Press. Sonst bleibt der Zustand hängen, sobald jemand beim Loslassen Strg hält.
- **Ein gehaltener Zustand muss jeden Eingabeweg umlenken, nicht nur den Hauptweg** (F-906): `Druecken` kannte die
  Leertaste, `Doppelklick` nicht — ein zweiter Druck hob einen Zuschnitt auf. Beim Einführen eines Modus alle Aufrufer
  der Zeigerbehandlung ablaufen (Druck, Doppelklick, Merker für den nächsten Doppelklick).
- **Fokusgebundene Filter wirken erst, wenn jemand den Fokus setzt** (F-907): Die Leinwand bekam ihn bisher nur per
  Klick. Ein Kürzel an `Focus: e` braucht einen Anfangsfokus beim Öffnen und nach jedem Dokumentwechsel.
- **Zeichentests prüfen den Ort, nicht nur die Farbe** (F-909): Ein Pixelzähler übersieht versetzte Rahmen und eine
  fehlende Kontur; ein Pixel an der erwarteten Stelle fängt beides.
