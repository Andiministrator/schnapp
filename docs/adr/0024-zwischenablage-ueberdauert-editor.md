# ADR-0024: Kopiertes Bild überdauert das Schließen des Editors

## Status
Akzeptiert (Claude Opus 5.5, 20.11.2026, Karte #91 in Sprint 6). Andi hat das Ziel entschieden (E-19: „ich würde das
schon gern komplett in der zwischenablage haben und behalten, wenn man den editor schließt“), den Weg nicht vorgegeben.
Ergänzt ADR-0015 Nachtrag 2.

## Kontext
Unter Wayland gehört der Inhalt der Zwischenablage dem Client, der ihn anbietet. Der Editor bot das kopierte PNG über
Gio an (ADR-0015 Nachtrag 2); schließt das Fenster, ist das Bild weg, sofern kein Zwischenablage-Verwalter es übernimmt.
Bei Andi übernimmt Klipper es nicht (20.11.2026).

Geprüfte Wege:

1. **Der Schnapp-Prozess hält die Auswahl nach dem Schließen weiter**, ohne Fenster. Das geht mit dem gewöhnlichen
   `wl_data_device` nicht: Gio baut beim Schließen die Verbindung ab, und eine neue Verbindung kann ohne Fokus und ohne
   Eingabe-Serial keine Auswahl setzen. Zudem warten Aufrufer (Druck-Taste, Skripte, `$(schnapp)`) auf das Prozessende.
2. **data-control-Protokoll** (`ext-data-control-v1`, älter `wlr-data-control-unstable-v1`): gedacht für
   Zwischenablage-Verwalter; ein Client setzt die Auswahl ohne Fenster und ohne Fokus. Gemessen im virtuellen KWin 6.8
   (`scripts/probe.sh --zwischenablage`): KWin bietet `ext_data_control_manager_v1` jedem Client an, das Bild kommt
   bytegleich zurück, eine neue Auswahl löst den Halter ab.
3. **`wl-copy` aufrufen**: ein Laufzeit-Fremdpaket ohne Not (ADR-0011).

## Entscheidung

Weg 2, als **abgelöster Halter-Prozess**.

- **Strg+C** rendert wie bisher das PNG und startet dasselbe Programm mit dem internen Argument
  `--internal-keep-clipboard` (nicht in `--help`; `main` prüft es vor allem anderen, ohne Gio, Einstellungen und
  Sprache). Der Editor schreibt das PNG auf die Standardeingabe des Halters und wartet höchstens 3 s auf dessen Meldung
  über den Dateideskriptor 3 (`ok <Manager>` oder `fehler: …`).
- Der Halter (Paket `wlclip`) spricht das Wayland-Leitungsformat selbst — **reines Go, kein cgo, keine libwayland**,
  nur die rund zehn nötigen Nachrichten. Er bindet den ersten `wl_seat` und `ext_data_control_manager_v1`, sonst
  `zwlr_data_control_manager_v1` (je Version 1), bietet genau `image/png` an und setzt die Auswahl.
- Der Halter läuft in **eigener Sitzung** (`setsid`), mit Arbeitsordner `/`, ohne Terminal-Ausgaben. Der Editor holt
  ihn ab, sobald er endet (kein Zombie). Aufrufer warten nicht auf ihn.
- **Fehlt data-control** oder meldet sich der Halter nicht rechtzeitig, beendet der Editor den eigenen, noch nicht
  abgeholten Kindprozess und bietet das Bild **selbst über Gio** an wie bisher (dann gilt wieder: nur solange das
  Fenster offen ist). Der Grund steht im Debug-Log (`SCHNAPP_DEBUG=1`), die Statusleiste meldet „Copied to clipboard“.

**Wächter-Regeln** (globale Vorgaben für Hintergrundprozesse) — der Halter greift in nichts ein, er beantwortet nur
Einfüge-Anfragen:

- **Ziel beim Start fest:** nur die eigene Auswahl; er sucht keine Prozesse und beendet keine. Der Editor beendet nur
  sein eigenes Kind über das Prozess-Handle, bevor es abgeholt ist (die PID kann nicht wiederverwendet sein).
- **Harte Zeitgrenze:** 12 Stunden nach der Wanduhr (`wlclip.Haltezeit`, Nachtrag 1); danach gibt er die Zwischenablage frei und endet.
- **Genau ein Werkzeug:** Bytes in die Pipe eines Einfügenden schreiben (höchstens 10 s ohne Fortschritt, wie F-176).
  Er schreibt keine Dateien und löscht nichts.
- **Endet von selbst**, sobald eine andere Anwendung etwas kopiert (`cancelled`), das Gerät endet (`finished`), der
  Compositor die Verbindung schließt (Abmelden) oder ein Protokollfehler kommt — Ausfall in Richtung Nichtstun.
- **Sichtbar:** Das Debug-Log des Editors nennt PID, Protokoll und Größe. In der Prozessliste heißt er `schnapp` mit
  dem Argument `--internal-keep-clipboard`; beenden mit `kill <PID>` oder indem man etwas anderes kopiert.

Je Strg+C entsteht ein neuer Halter; der vorige wird dabei abgelöst und endet. Es läuft also höchstens einer (kurz zwei).

## Konsequenzen

- Das kopierte Bild bleibt nach dem Schließen des Editors, bis etwas anderes kopiert wird, höchstens 12 Stunden.
  Ein Absturz des Editors nimmt es nicht mehr mit.
- Ein Halter belegt den Speicher eines PNG (bei 7680 × 2160 einige MB) und eine Wayland-Verbindung.
- Geschwärztes ist im kopierten PNG wie in jedem Export endgültig (ADR-0008); der Halter bekommt nur das fertige PNG
  ohne Bearbeitungsdaten.
- Die Wayland-Leitung in `wlclip` ist eigener Code am Protokoll: Tests mit nachgebautem Compositor über ein Socket-Paar
  (`wlclip/halter_test.go`), Probelauf im virtuellen KWin (`scripts/probe.sh --zwischenablage BILD.png`).
- Grundlage für „Aufnahme direkt in die Zwischenablage“ (ohne Editor): derselbe Halter.
- Unter einem Compositor ohne data-control (etwa GNOME) bleibt es beim bisherigen Verhalten; das Portal-Thema
  (ADR-0007) entscheidet dort später.

## Nachtrag 1 (20.11.2026): Nacharbeit aus der Kritik (Fable)

- **Zeitgrenze nach der Wanduhr:** Fristen in Go laufen auf der monotonen Uhr, die im Standby steht. `Diene` setzt die
  Frist in Scheiben von einer Minute neu und vergleicht mit der Wanduhr; die Frist gilt für Lesen und Schreiben (ein
  hängender Compositor hält den Halter nicht fest). Ein Einfügen, das bei Ablauf gerade läuft, darf zu Ende laufen
  (je 64-KiB-Stück höchstens 10 s ohne Fortschritt).
- **Meldeweg nur über eine Pipe:** Der Halter schreibt auf fd 3 nur, wenn fd 3 beim Start eine Pipe ist. Von Hand ohne
  fd 3 aufgerufen, bekäme sonst der Wayland-Socket die Nummer 3 und die Meldung landete darin.
- **Rückfall auf Gio ohne Halter**, wenn `WAYLAND_SOCKET` gesetzt ist (geerbte Verbindung; der Halter fände nicht
  sicher denselben Compositor), und für den Rest der Editor-Sitzung, sobald der Halter „kein data-control“ meldet.
- README nennt die Einschränkung „mit data-control, etwa KDE Plasma“. Offen als F-189 (Karte #94): geerbte
  Deskriptoren ohne CLOEXEC und das Verhalten in einer systemd-Scope.

## Nachtrag 2 (20.11.2026): Bild im Klipper-Verlauf (Meta+V)

Andi fragte, warum das Bild nicht im Verlauf der Zwischenablage (Meta+V) erscheint. Ursache in Klipper
(`plasma-workspace/klipper/historymodel.cpp`): Ein Inhalt mit Bild und ohne Text kommt nur in den Verlauf, wenn die
Einstellung „Include in history: Image data copied explicitly“ (`SaveImages`, Voreinstellung **aus**) eingeschaltet
ist **oder** der Inhalt den MIME-Typ `x-kde-force-image-copy` trägt. Spectacle setzt diesen Typ
(`spectacle/src/ExportManager.cpp`, Kommentar „ensures that the image is copied to Klipper even when … is not
checked“).

Entscheidung: Der Halter bietet zusätzlich `x-kde-force-image-copy` an und liefert dafür null Byte (wie Spectacle
einen leeren Wert setzt). Damit erscheint das kopierte Bild im Klipper-Verlauf, ohne dass Andi eine Einstellung ändern
muss. Der Rückfall über Gio bietet weiter nur `image/png` an (ADR-0015 Nachtrag 2) — dort erscheint es im Verlauf nur
mit eingeschalteter Einstellung.

## Nachtrag 3 (20.11.2026): Nacharbeit aus der Sitzungs-QA (Kritik Fable, Teil B)

- Die Haltezeit gilt nach der Wanduhr in Scheiben; die Scheibe ist nur eine Lesefrist. Für das Antworten an den
  Compositor gilt je Ereignis eine eigene Schreibfrist `Stau` (10 s), damit ein gesunder Halter nach Ablauf einer
  Scheibe nicht an einem `destroy` scheitert.
- Nach dem Ende wartet der Halter auf laufende Sendungen höchstens `Stau`; die Frist gilt je 64-KiB-Stück, nicht für
  das Ganze.
- Die Rückfall-Entscheidung des Editors ist testbar: `wlclip.HalterMoeglich` (Fehler bei `WAYLAND_SOCKET`) und
  `errors.Is(err, wlclip.ErrKeinDataControl)` (`Uebergib` bildet den Text des Halters auf den Sentinel ab).
- Die Wayland-Leitung liegt seit Karte #95 in `internal/wlwire` (gemeinsam mit der Probe-Fernbedienung).
- `TestZeitgrenze` prüft Unter- und Obergrenze mit einspeisbarer Uhr, einschließlich Wanduhr-Sprung (Standby).

## Nachtrag 4 (20.11.2026): Haltezeit auch nach der monotonen Uhr (F-232)

**Status:** Akzeptiert (umgesetzt von Claude Sonnet 5.5, Karte #102; übernommen von Claude Opus 5.5).

- Die 12-Stunden-Grenze des Halters galt nur nach der Wanduhr; eine zurückgestellte Uhr verlängerte sie um den Sprung.
  `Diene` rechnet jetzt mit der früheren von Wanduhr- und monotoner Grenze (`monoSeit`, im Test einspeisbar); ein
  Vorstellen der Uhr verkürzt sie wie bisher. Test `TestZeitgrenzeUhrZurueck` (Gegenprobe ohne monotone Grenze rot).

## Nachtrag 5 (21.11.2026): Geerbte Deskriptoren (F-189, Karte #94)

**Status:** Akzeptiert (Claude, Andi informiert; umgesetzt von Claude Sonnet 5.5).

- `HelferLauf` schließt als Erstes alle Deskriptoren ab 4, die ohne `FD_CLOEXEC` geerbt wurden (`schliesseGeerbte`: Nummern per fcntl bis zur weichen Grenze, höchstens 65536, durchprobiert — bewusst ohne das proc-Dateisystem, Schnapp liest keine Prozessverzeichnisse, auch nicht das eigene; Nachbesserung Opus 21.11.2026). Die Go-Laufzeit öffnet ihre eigenen Deskriptoren immer mit CLOEXEC, ein gezieltes Schließen der Nicht-CLOEXEC-Deskriptoren trifft sie also nicht (kein pauschales `close_range`). Deskriptoren 0 bis 3 bleiben. Test `TestUebergibSchliesstGeerbte` (Gegenprobe ohne Schließen rot).
- **systemd-Scope bleibt offen, nicht umgesetzt:** Startet KDE Schnapp in einer `app-*.scope`, bleibt der Halter trotz `setsid` in dieser cgroup (setsid löst keine cgroup). Ein Verlassen bräuchte `StartTransientUnit` über D-Bus an den Benutzer-systemd oder `systemd-run --user` als Fremdprogramm (ADR-0011) — beides eine neue Abhängigkeit; als eigenes Finding geführt (F-320, Karte #133: erst Andis Messung, dann Entscheidung).
- *Gemessen 21.11.2026 abends (F-325):* Im virtuellen KWin (`scripts/probe.sh`) schließt der Halter nach Strg+C drei
  geerbte Deskriptoren (20, 21, 37); das Schließen ist also nötig. Mit `SCHNAPP_DEBUG` meldet der Halter die Zahl
  (`[wlclip] Halter: …`); seine Fehlerausgabe geht dafür nur an ein Terminal oder eine Datei, nie in eine Pipe.

## Nachtrag 6 (21.11.2026): Warnung beim Schließen ohne Halter (F-295, F-321, F-329)

**Status:** Akzeptiert (Claude, Andi informiert). Umgesetzt in Sprint 10 (F-295, F-321 von Claude Sonnet 5.5, F-329
Karte #143).

- Gibt es keinen Halter (kein data-control, `WAYLAND_SOCKET`, Startfehler), bietet das Fenster selbst die Kopie an
  (Rückfall). Schließt es, ist das Bild aus der Zwischenablage weg. Das Schließen hält deshalb einmal an und sagt in
  der Statusleiste „The clipboard holds the image only while this window is open. Close again to discard it.“
  (deutsch: „Die Zwischenablage hält das Bild nur, solange dieses Fenster offen ist. Erneutes Schließen verwirft es.“);
  die nächste Anfrage schließt. Ein Signal schließt immer.
- Das gilt auch, wenn das Schließen vorher speichert (F-321), und auch, wenn beim Schließen noch ein Kopieren läuft:
  Das Schließen wartet auf dessen Antwort (höchstens bis zur Kopierfrist), weil erst sie sagt, ob das Fenster die Kopie
  hält (F-329).
- Scheitert das Kopieren, auf das eine Schließen-Anfrage wartete (F-281, F-329), schließt das Fenster nicht still: Der
  Fehler steht in der Statusleiste und zusätzlich auf der Fehlerausgabe, das Fenster bleibt einmal offen, erst die
  nächste Anfrage schließt. Ein Signal schließt in jedem Fall; dann meldet nur die Fehlerausgabe (F-337, Karte #148).
- Offen: Übernimmt ein anderes Programm die Zwischenablage, meldet Gio `data_source.cancelled` nicht nach außen; die
  Warnung kommt dann unnötig (F-328, Karte #142, braucht einen Gio-Patch).
