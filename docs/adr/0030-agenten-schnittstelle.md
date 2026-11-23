# ADR-0030: Schnittstelle für Agenten und Skripte — Kommandozeile (Fassung 1), MCP-Server (Fassung 2)

## Status
Akzeptiert (Claude Opus 5.5, 22.11.2026, Karte #43; Andi hat E-14 am 19.11.2026 entschieden und die Karte am
22.11.2026 freigegeben: „E-14 ist freigegeben“). Fassung 1 umgesetzt in Sprint 11; Fassung 2 (Punkt 6 bis 10) ist
Entscheidung ohne Umsetzung, eigene Karte.

## Kontext
E-14 (`docs/open-decisions.md`): Skripte, KI-Agenten und lokale Modelle sollen Schnapp bedienen können — aufnehmen,
die Bibliothek lesen, exportieren, später beschriften und schwärzen. Zwei Wege: eine skriptfähige Kommandozeile (die
erste, billige Schnittstelle) und ein MCP-Server (Model Context Protocol, JSON-RPC über stdio). Beides ist
sicherheitsrelevant: Wer die Schnittstelle bedient, sieht den Bildschirm und die Bibliothek.

Die Kernschichten kennen Gio nicht (`capture`, `document`, `fileformat`, `render`, `export`, `library`); die
Kommandozeile gibt es seit Sprint 2. Ihr fehlte bisher eine maschinenlesbare Ausgabe: Die Monitorliste ist übersetzt
und mit „ד gesetzt (F-98), und eine Aufnahme landet immer in der Bibliothek, oft gefolgt vom Editor.

## Entscheidung — Fassung 1: Kommandozeile

1. **`--json`** schreibt bei Erfolg genau eine Zeile JSON auf stdout, Schlüssel englisch und stabil, unabhängig von der
   Sprache. Erlaubt bei Aufnahmen, `--list-screens`, `--export` und `--version`; mit `--help` oder `schnapp DATEI`
   (Editor) ist es ein Aufruffehler.
   - Aufnahme und Export: `{"path":"/abs/datei.png","width":800,"height":600}` — `path` absolut, Größe der Ansicht.
   - `--list-screens` (F-98): `{"screens":[{"name":"DP-1","width":2560,"height":1440}]}`; der Name ist genau der Wert
     für `--screen=NAME`.
   - `--version`: `{"version":"0.5.0"}`.
   Fehler stehen weiter als Text auf stderr (übersetzt); maßgeblich für Skripte ist der Exit-Code: 0 Erfolg,
   1 Fehler, 2 Aufruffehler, 3 vom Nutzer abgebrochen. Ein JSON-Fehlerobjekt gibt es nicht: Jeder Fehlerweg hätte es
   tragen müssen, und der Exit-Code sagt dasselbe verlässlich.
2. **`--output=DATEI`** bei einer Aufnahme schreibt das Bild nach DATEI, Format nach der Endung (`.png`, `.jpg`/`.jpeg`,
   `.svg`, wie beim Export, ADR-0025), **statt** es in der Bibliothek abzulegen; der Editor öffnet nicht, das Aufräumen
   der Bibliothek läuft nicht. Grund: Ein Skript, das hundert Bilder an seinen Ort schreibt, soll die Bibliothek nicht
   füllen — jede weitere Kopie des Bildschirminhalts ist eine zu viel (Kernprinzip 3). Eine unbekannte Endung ist ein
   Aufruffehler, bevor aufgenommen wird. Die Datei wird atomar geschrieben (ADR-0025 Punkt 5) und ersetzt ein
   vorhandenes Ziel, wie `cp`. Ohne Metadaten, ohne `snAP`/`snAO` (Kernprinzip 3). `--repeat` merkt sich den Bereich
   weiter.
3. **`--export=ZIEL DATEI`** öffnet DATEI (Arbeitsdatei, PNG, JPG) ohne Fenster und schreibt das sichtbare Bild —
   Zuschnitt, Stauchen, Beschriftungen, Schwärzungen — nach ZIEL, Format nach der Endung wie bei `--output`. Das
   Rechenbudget gilt wie beim Öffnen im Editor (ADR-0020 Nachtrag 3); `--ignore-limit` überspringt die Prüfung. Die
   Datei selbst bleibt unverändert.
4. **Werte nur mit Gleichheitszeichen** (`--output=DATEI`, `--export=ZIEL`), wie `--screen=NAME`: Die Form ohne `=`
   meldet einen Aufruffehler mit der richtigen Schreibweise, statt den Wert still als Datei zum Öffnen zu nehmen.
5. **Doku:** `docs/agents.md` (englisch, kurz): Aufrufe, JSON-Formen, Exit-Codes, Grenzen, Datenschutz.

## Entscheidung — Fassung 2: MCP-Server (umzusetzen auf eigener Karte)

6. **`schnapp mcp`**, nur stdio — nie ein Socket, nie das Netz. Der Server läuft, solange der aufrufende Agent die
   Verbindung hält.
7. **Einwilligung in den Einstellungen**, getrennt und voreingestellt aus (ADR-0018):
   `[agents] capture = no` (Aufnehmen) und `[agents] library = no` (Bibliothek suchen und lesen). Ohne Einwilligung
   listet der Server das Werkzeug, lehnt den Aufruf aber mit einem Text ab, der sagt, welcher Schalter fehlt. Die
   Einwilligung gibt nur der Mensch in der Datei; der Server schreibt die Einstellungen nie.
8. **Sichtbarer Hinweis bei jeder Aufnahme über MCP:** eine Desktop-Benachrichtigung
   (`org.freedesktop.Notifications`, „Schnapp hat für einen Agenten aufgenommen“), vor dem Ergebnis an den Agenten.
   Scheitert der Hinweis, scheitert die Aufnahme (Ausfall in Richtung Nichtstun).
9. **Werkzeuge** auf denselben Kernschichten wie die Kommandozeile: aufnehmen (Monitor, alle Monitore, letzter Bereich —
   kein Overlay, das braucht einen Menschen), Monitore auflisten, Bibliothek suchen, Bild aus der Bibliothek lesen,
   exportieren; Beschriften und Schwärzen einer Arbeitsdatei folgen, sobald die Kommandozeile sie kann. Bilder gehen
   als Datei-Pfad zurück, auf Wunsch als Bild-Inhalt (Base64, mit Größengrenze).
10. **Kein Fremdpaket für MCP** (ADR-0011): Die nötige Teilmenge — `initialize`, `tools/list`, `tools/call`, JSON-RPC 2.0
    zeilenweise über stdio — ist klein und spezifiziert; ein Fremdpaket brächte mehr Fläche als Nutzen. Neu abwägen,
    wenn Fassung 2 Ressourcen, Prompts oder Streaming braucht.

## Konsequenzen
- Skripte und Agenten bekommen eine stabile Form, bevor das Repo veröffentlicht wird; die JSON-Schlüssel sind ab jetzt
  Oberfläche und ändern sich nur mit einem Nachtrag.
- `--output` umgeht die Bibliothek bewusst: Was so aufgenommen wird, taucht dort nicht auf und wird nicht aufgeräumt.
- F-98 ist mit `--list-screens --json` behoben.
- Beschriften und Schwärzen über die Kommandozeile sind noch offen; sie kommen mit Fassung 2 auf eigener Karte.

## Nachtrag 1 (22.11.2026): Signalwache beim Schreiben von `--output`/`--export` (F-396)

**Status:** Akzeptiert (Claude, Karte #176; Befund aus der Sitzungs-QA 22.11., Fable).

Das Schreiben der Ausgabe von `--output` und `--export` läuft unter einer Signalwache (`mitSignalwache` in
`cmd/schnapp/agenten.go`, SIGTERM/SIGINT/SIGHUP). Kommt ein Signal vor dem ersten Byte, bricht Schnapp ab, die
temporäre Datei fällt weg, das Ziel bleibt unberührt, Exit 1. Kommt es nach dem Schreiben, ist die Datei vollständig,
der Aufruf meldet trotzdem den Abbruch (Exit 1). Das Rendern liegt außerhalb der Wache; der Editor-Weg ist nicht
betroffen. Restfall: SIGKILL oder Absturz (in `docs/agents.md` genannt). Test `cmd/schnapp/signalwache_test.go`,
Mutant in `scripts/mutanten/cmd-agenten.json`; Rechte 0600 von `--output` prüft das Drehbuch
`scripts/drehbuecher/agenten-kommandozeile.sh` im virtuellen KWin (F-405).
- **Nachgezogen (Karte #199, F-441 bis F-443):** Die Wache liegt in `cmd/schnapp/signalwache.go`. Kam das Signal nach
  dem ersten Byte, gibt Schnapp die Ergebniszeile (Pfad bzw. JSON) trotzdem aus und meldet „vollständig geschrieben,
  durch ein Signal unterbrochen“ (Exit 1). Ein zweites Signal bei hängendem Schreiben räumt die Temp-Datei
  (`.schnapp-export-*`) auf und endet mit Exit 1. Beim Start ignorierte Signale (`nohup`) fängt die Wache nicht.
- **Rückfall (Karte #201, F-458):** Im Rückfall aus ADR-0025 Nachträgen 3/4 (direktes Schreiben ins vorhandene Ziel)
  gibt es keine temporäre Datei: Ein zweites Signal beendet den Aufruf mit Exit 1 und der Meldung, das Ziel könne
  unvollständig sein; die Wache fasst das hängende Schreiben nicht an (kein zweiter Schreiber auf dieselbe Datei) und
  stellt nichts wieder her. Ohne Namen einer temporären Datei meldet sie, dass das Schreiben nicht begonnen hat und das
  Ziel unverändert ist; ist die temporäre Datei schon umbenannt, dass die Datei vollständig war.


## Nachtrag 2 (22.11.2026): Fassung 2 im Einzelnen — Protokoll, Einstellungen, Bestätigung, Ablage (Karte #173)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #173; Entscheidungen Andi vom 22.11.2026 im Kommentar an der Karte:
Einstellungen `[agents]`, Bild direkt an den Agenten, Bestätigungsdialog, eigener Unterordner; Frist des Dialogs von
Claude gesetzt). Ersetzt die Punkte 7 bis 9 dort, wo sie abweichen.

Quelle zum Protokoll: `knowledge/sources/mcp-2026-07-28.md` (Spezifikation 2026-07-28, ohne `initialize`).

1. **Protokoll, zwei Epochen.** `schnapp mcp` spricht JSON-RPC 2.0 zeilenweise über stdin/stdout, sonst nichts auf
   stdout; Meldungen nur auf stderr. Neue Form (2026-07-28): jede Anfrage trägt `_meta` mit
   `io.modelcontextprotocol/protocolVersion` und `…/clientCapabilities`; `server/discover` meldet
   `supportedVersions`, `capabilities: {"tools":{}}` und `serverInfo`; fehlt ein Pflichtfeld, `-32602`; unbekannte
   Version `-32022` mit `supported`/`requested`. Altform (2025-11-25, 2025-06-18, 2025-03-26, 2024-11-05): eine
   `initialize`-Anfrage wählt sie, die Antwort nennt die angefragte Version, wenn bekannt, sonst 2025-11-25; dazu
   `ping`. Jedes Ergebnis trägt `resultType: "complete"`; `tools/list` trägt `ttlMs` (1 h) und `cacheScope`
   (`"private"`). JSON-Arrays (Batching) lehnt der Server mit `-32600` ab, unbekannte Methoden mit `-32601`,
   unlesbare Zeilen mit `-32700`; Benachrichtigungen bekommen keine Antwort. Zeilen über 1 MiB gelten als unlesbar.
   Anfragen laufen nacheinander. Bei EOF auf stdin endet der Server; SIGTERM/SIGINT beenden ihn ebenso (ein
   laufender Dialog wird geschlossen, eine laufende Aufnahme verworfen).
2. **Einstellungen `[agents]`** (ADR-0018), alle global, gelesen **bei jedem Werkzeugaufruf** (ein Widerruf in der
   Datei wirkt sofort, ohne Neustart des Servers):
   - `capture = no` — Aufnehmen erlaubt (Voreinstellung aus).
   - `library = no` — Bibliothek suchen und Bilder daraus lesen (Voreinstellung aus).
   - `confirm = yes` — vor jeder Aufnahme ein Bestätigungsdialog (Punkt 4).
   - `image = yes` — das Bild geht als Bildinhalt (PNG, Base64) an den Agenten; mit `no` nur Pfad und Größe.
   Ein ungültiger Wert gilt als Voreinstellung (mit Warnung auf stderr) — für `capture`/`library` also als „aus“.
   Der Server schreibt die Datei nie.
3. **Werkzeuge** (Namen und JSON-Schlüssel sind Oberfläche; die Liste ist immer vollständig, auch ohne
   Einwilligung — die Spezifikation verlangt dieselbe Liste für jede Verbindung):
   - `list_screens {}` → `{"screens":[{"name","width","height"}]}` (wie `--list-screens --json`), keine Einwilligung.
   - `capture_screen {screen?, all?, output?}` — ein Monitor nach Name, alle mit `all: true`; ohne beides gilt
     `[capture] screen = all` als „alle“, sonst der aktive Monitor (KWin `CaptureActiveScreen`). Kein Overlay.
   - `capture_repeat {output?}` — der zuletzt gewählte Bereich (`--repeat`).
   - `library_search {tag?, from?, to?, limit?, agents?}` — neueste zuerst, `limit` 1–50 (Voreinstellung 20),
     `from`/`to` als RFC 3339 oder `JJJJ-MM-TT`, `agents: false` blendet Agenten-Aufnahmen aus. Antwort je Eintrag
     `path` (absolut), `created`, `width`, `height`, `tags`, `agent`. Braucht `library`.
   - `library_read {path}` — das sichtbare Bild eines Eintrags; `path` absolut oder relativ zur Ablage, muss nach
     dem Auflösen von Symlinks in der Ablage liegen. Braucht `library`.
   - `export {file, target}` — wie `--export` (Pfade absolut). Liegt `file` in der Ablage, braucht es `library`.
   Aufnahmen und `library_read` antworten mit `structuredContent {path,width,height}`, demselben JSON als Text und,
   mit `image = yes`, dem Bild als `image/png` — höchstens 8 MiB PNG, sonst nur ein Satz, dass es zu groß ist. `output`
   gilt wie `--output` (Endung, atomar, 0600, ersetzt ein vorhandenes Ziel, kein Bibliothekseintrag) und muss absolut
   sein. Ohne Einwilligung antwortet das Werkzeug mit `isError: true` und einem englischen Satz, der den Schalter und
   die Datei nennt.
4. **Bestätigung** (`confirm = yes`): Vor der Aufnahme startet der Server sich selbst als Kindprozess
   (`schnapp --internal-confirm-capture`, wie der Halter `--internal-keep-clipboard`) mit einem kleinen Fenster:
   „Ein Agent möchte <Monitor / alle Monitore / den letzten Bereich> aufnehmen“, darunter der vom Agenten gemeldete
   Name (`clientInfo.name`, ungeprüft, maskiert, höchstens 60 Zeichen, als solcher gekennzeichnet), Knöpfe „Erlauben“
   und „Ablehnen“. Esc, Schließen und jeder Fehler heißen Ablehnen; Eingabe löst nichts aus. Erlaubt ist nur Exit 0
   des Kindes. **Frist 60 s**, dann beendet der Server das Kind und lehnt ab. Fällt der Dialog aus (kein Wayland,
   Start scheitert), scheitert die Aufnahme. Aufgenommen wird erst, wenn das Kind beendet ist, mit 300 ms Abstand,
   damit das Fenster nicht im Bild steht. Kein „immer erlauben“.
5. **Hinweis** (Punkt 8 bleibt): Nach jeder Aufnahme, vor dem Ablegen, schickt der Server eine Desktop-Benachrichtigung
   „Schnapp hat für einen Agenten aufgenommen: <Ziel>“ (übersetzt). Scheitert sie, wird das Bild verworfen und die
   Aufnahme scheitert — auch mit `confirm = yes`.
6. **Ablage:** Agenten-Aufnahmen ohne `output` landen in der Bibliothek unter
   `<Bilderordner>/schnapp/agents/<JJJJ-MM>/…png` (Index-Pfad `agents/…`), mit demselben Aufräumen; der Neuaufbau
   findet sie wie jede Aufnahme. Die Bibliotheks-Oberfläche blendet sie per Schalter ein (Karte der
   Bibliotheks-Oberfläche).
7. **Beschriften und Schwärzen über die Kommandozeile** sind aus Karte #173 herausgelöst (eigene Karte); die
   MCP-Werkzeuge dazu folgen mit ihr.
8. **Schichten:** Das Protokoll liegt im Paket `internal/mcp` (ohne Gio, ohne Wissen über Schnapp), die Werkzeuge in
   `cmd/schnapp/mcp.go` auf denselben Kernschichten wie die Kommandozeile, der Dialog in `internal/ui`.

## Nachtrag 3 (22.11.2026): Monitorgröße mit Drehung (F-415)

**Status:** Akzeptiert (Karte #180, umgesetzt von Claude Sonnet 5.5, übernommen von Claude Opus 5.5).

`--list-screens --json` (und `list_screens` über MCP) meldet die Größe, wie der Monitor sie zeigt: Bei Drehung um 90
oder 270 Grad (auch gespiegelt) sind Breite und Höhe gegenüber dem wl_output-Modus vertauscht. Der Gio-Patch liefert
`WaylandOutput.Transform`, `ui.gedrehteGroesse` tauscht (`TestGedrehteGroesse`, alle acht Werte). Die Schlüssel
bleiben gleich, nur die Werte stimmen jetzt. Gegen eine echt gedrehte Ausgabe ist das nicht gemessen: Der virtuelle
KWin bietet keine Drehung über die Kommandozeile.

## Nachtrag 4 (22.11.2026): Härtung des MCP-Servers nach der Sitzungs-QA (Karte #203, F-461 bis F-471)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #203; Befunde der Sitzungs-QA Sprint 12, Kritik A, Claude Fable 5.2).
Die Entscheidungen zu F-461 und F-462 fielen in der sichereren Richtung (Register `docs/findings.md`, 22.11.2026).
Ändert Nachtrag 2 dort, wo er abweicht.

1. **Einwilligung und Orte aus dem echten Heim (F-461).** Den Server startet der Agenten-Client, und der setzt die
   Umgebung — auch über eine `.mcp.json` in einem fremden Repo. Für `schnapp mcp` kommen deshalb `settings.ini` (alle
   Werte, `[agents]` wie `[capture]` und die Sprache) und die Orte der Bibliothek (Ablage, Index, Vorschau, letzter
   Bereich) aus dem Heimverzeichnis der Benutzerdatenbank (`os/user`), nicht aus `HOME` und nicht aus `XDG_*`: die
   Einstellungen aus `<heim>/.config/schnapp/settings.ini`, der Bilderordner aus `<heim>/.config/user-dirs.dirs`, sonst
   `<heim>/Pictures`. Weicht die Umgebung davon ab, schreibt der Server beim Start eine Warnung auf stderr und arbeitet
   mit dem echten Heim. Ist es nicht bestimmbar, gelten die Voreinstellungen (Aufnehmen und Bibliothek aus), und
   jedes Werkzeug, das einen Ort braucht, lehnt ab. Die Kommandozeile (Fassung 1) folgt weiter der Umgebung: Wer sie
   aufruft, ist der Mensch oder sein Skript.
2. **Ziele (F-462, F-465).** `output` (Aufnahme) und `target` (`export`) dürfen nicht in der Ablage und nicht im
   Vorschau-Cache liegen; geprüft wird der Ordner des Ziels nach dem Auflösen seiner Symlinks. Ein vorhandenes Ziel
   wird **nie ersetzt**: Die Datei entsteht exklusiv (harter Link der temporären Datei, wie beim Export mit
   angehängter Endung), sonst lautet die Antwort „already exists“. Eine neue Datei bekommt bei `output` die Rechte
   0600, bei `export` die der umask. Liegt `file` von `export` in der Ablage **oder** im Vorschau-Cache, braucht es
   `library = yes`. Ist der Ort der Bibliothek nicht bestimmbar, lehnt der Server Aufnahmen mit `output` und jeden
   Export ab. Das ersetzt für MCP den Satz „ersetzt ein vorhandenes Ziel“ aus Nachtrag 2 Punkt 3; `--output` und
   `--export` der Kommandozeile bleiben, wie sie sind.
3. **Monitorname (F-463).** `screen` muss ein Name aus der Monitorliste sein; ein unbekannter Name wird vor dem Dialog
   abgelehnt. Der Name im Dialog- und Hinweistext ist maskiert (keine Steuerzeichen) und gekürzt wie der gemeldete
   Name des Agenten.
4. **Dialog nur per Mausklick (F-464).** Tastatur löst nie „Erlauben“ aus: Return, Leertaste und Tab-Fokus erreichen
   den Knopf nicht (er ist kein fokussierbares Element). Esc, Schließen des Fensters und „Ablehnen“ (auch per
   Tastatur) lehnen ab. Ein Klick auf „Erlauben“ zählt nur, wenn **Drücken und Loslassen** nach der Sperre von 600 ms
   liegen; ein Drücken während der Sperre zählt auch nach ihrem Ende nicht.
5. **Kind mit eigener Frist (F-469).** Das Dialog-Kind schließt nach der Frist des Servers plus 5 s selbst und lehnt
   ab; spätestens 2 s danach beendet es sich hart. So bleibt kein Fenster stehen, wenn der Server stirbt.
6. **Abbruch (F-467, F-469).** Nach einem Signal oder EOF startet der Server keinen weiteren Aufruf aus der
   Warteschlange. Ein Werkzeug prüft den Abbruch zwischen seinen Schritten; eine Aufnahme, die nach dem Abbruch fertig
   wird, wird verworfen (weder abgelegt noch an `output` geschrieben). Ein zweites SIGTERM/SIGINT/SIGHUP beendet den
   Prozess sofort mit Exit 1 — für ein hängendes Werkzeug.
7. **Grenze der Warteschlange (F-468).** Während ein Werkzeug läuft, hält der Server höchstens 64 Nachrichten und
   8 MiB; darüber antwortet er auf eine Anfrage mit `-32600` („too many pending messages“) und verwirft sie,
   Benachrichtigungen verwirft er still. Abbrüche der laufenden Anfrage zählen nicht mit.
8. **Hinweis ist nur ein Hinweis (F-466).** „Hinweis gezeigt“ heißt: Der Benachrichtigungsdienst hat eine Kennung
   vergeben. Ob ein Mensch sie sieht, weiß der Server nicht; „Nicht stören“ kann sie verbergen. Darum sendet er sie
   mit Dringlichkeit `critical` (Byte 2); Plasma zeigt kritische Hinweise bei „Nicht stören“ nach Voreinstellung weiter
   und lässt sie stehen, bis der Mensch sie wegklickt (nicht gemessen, Plasma-Einstellung „kritische Benachrichtigungen
   zeigen“). Wer jede Aufnahme selbst sehen will, lässt `confirm = yes`.
9. **Protokoll (F-470).** Gültiges JSON, das kein Objekt ist, ergibt `-32600` (nicht `-32700`). Eine Antwort des
   Clients (Kennung mit `result` oder `error`, ohne Methode) bekommt keine Antwort. `ping` beantwortet der Server in
   beiden Formen, auch ohne `_meta` (2026-07-28 kennt `ping` nicht mehr; die Antwort ist harmlos und hilft älteren
   Clients).

## Nachtrag 5 (23.11.2026): Dialog an die aufnehmende Sitzung gebunden, Abbruch nach dem Schreiben, Texte an den Agenten (Karte #205, F-480 bis F-494)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #205; Befunde aus dem Verify-Lauf von Sprint 12, Claude Fable 5.2, und dem
Bericht zu Karte #203). F-491 fiel in der sichereren Richtung: absichern statt nur als Grenze dokumentieren.

1. **Dialog nur auf dem Compositor, der aufnimmt (F-491).** Die Umgebung des Servers setzt der Agenten-Client. Zeigte
   `WAYLAND_DISPLAY` auf einen eigenen Compositor, erschiene der Dialog dort, und ein Programm könnte „Erlauben“
   klicken, während der Sitzungsbus das echte KWin aufnehmen lässt. Vor dem Dialog prüft der Server deshalb: Der
   Prozess hinter dem Wayland-Socket (`SO_PEERCRED`; `WAYLAND_DISPLAY`, sonst `wayland-0`, relativ zu
   `XDG_RUNTIME_DIR`) ist derselbe wie der Besitzer von `org.kde.KWin` auf dem Sitzungsbus
   (`GetConnectionUnixProcessID`). Weichen sie ab oder ist `WAYLAND_SOCKET` gesetzt, wird nicht aufgenommen. *Geändert durch
   Nachtrag 6 (Elternprozess zugelassen, Verbindung vererbt).* Zeigt die Umgebung
   **beides** auf einen fremden Compositor, nimmt auch die Aufnahme von dort auf; der echte Bildschirm bleibt
   unberührt. Ohne `confirm` gibt es keinen Dialog und nichts zu binden; die Hinweise laufen über denselben Bus wie
   die Aufnahme. **Grenze:** Prozessnummern gelten im eigenen PID-Namensraum; für ein Flatpak ist die Bindung neu zu
   prüfen (vor Flathub). Wer das Kommando des Servers setzt, kann ohnehin beliebigen Code als der Mensch ausführen —
   die Prüfung schützt gegen eine fremde Umgebung, nicht gegen ein fremdes Programm.
2. **Sperre auch nach dem Fokus (F-481).** Das erste Bild ist womöglich noch nicht zu sehen, wenn es gezeichnet wird.
   Die Sperre von 600 ms beginnt deshalb mit dem ersten Bild **und** neu mit dem ersten Bild nach jedem Erhalt des
   Fokus; ein Fokusverlust macht ein laufendes Drücken ungültig. Die Verdrahtung von Ereignissen und Logik ist ohne
   Fenster getestet (`input.Router`). „Nur per Mausklick“ heißt genauer: nur per Zeiger — Gio meldet Berührung und
   Stift als dasselbe Klicken; Tasten erreichen „Erlauben“ weiter nie.
3. **Abbruch nach dem Schreiben (F-482, F-483).** Endet ein Aufruf durch Signal, `notifications/cancelled` oder EOF,
   während die Aufnahme geschrieben oder abgelegt wird, entfernt der Server die Datei, die er gerade selbst angelegt
   hat: an `output` (exklusiv neu angelegt, also seine) wie in der Bibliothek (`library.Entfernen` mit dem Stand
   direkt nach dem Ablegen, samt Vorschau und Index). Scheitert das Entfernen, nennt die Antwort die Datei. Ein
   `export`, den ein Signal nach dem vollständigen Schreiben trifft, behält die Datei und sagt das (ein Export ist
   keine Aufnahme).
4. **Zweites Signal räumt zuerst auf (F-484).** Die Signalwache führt die temporären Dateien laufender Schreibvorgänge
   in einer Liste; das zweite Signal an den Server entfernt sie (nur mit dem Vorsatz `.schnapp-export-`), bevor es den
   Prozess beendet.
5. **Sprache des Dialogs (F-485).** Der Server schickt seine Sprache (aus dem echten Heim) in der Anfrage an das Kind;
   das Kind liest keine Einstellungen aus seiner Umgebung für die Sprache. Eine unbekannte Sprache bleibt ohne Wirkung.
6. **Texte an den Agenten (F-492, F-493).** Fehlertexte an den Agenten sind immer englisch und tragen keine
   Paketfehler und kein Heimverzeichnis (die Ablehnung nennt `~/.config/schnapp/settings.ini`, Nachtrag 6); Unbekanntes
   steht nur auf stderr. Pfade, die der Agent selbst genannt hat oder die er als Ergebnis bekommt, erscheinen wie
   bisher. `export` antwortet auf jede Quelle, die sich nicht
   öffnen lässt (fehlt, nicht lesbar, kein Bild, Ordner), mit demselben Satz. Ob `file` in Ablage oder Vorschau-Cache
   liegt, prüft der Server auch dem Namen nach (ohne Auflösen), damit „library = yes“ nicht nur für vorhandene Dateien
   kommt.
7. **Ohne harte Links (F-494).** Kann das Dateisystem keine harten Links, legt `SchreibeOhneErsetzen` das Ziel mit
   `renameat2(RENAME_NOREPLACE)` an — der Kernel prüft beim Umbenennen selbst. Nur wenn auch das fehlt
   (`EINVAL`/`ENOSYS`), bleibt das bestmögliche Prüfen und Umbenennen mit kurzem Fenster.

## Nachtrag 6 (23.11.2026): Sitzungsbindung in Plasma, Grenzen nach der Sitzungs-QA (Befunde A-1 bis A-7)

**Status:** Akzeptiert (Claude Opus 5.5; Befunde aus der Sitzungs-QA vom 23.11.2026, Kritiker A, Claude Fable 5.2).

1. **Wrapper zugelassen, Verbindung vererbt (A-1, A-2).** In Plasma legt `kwin_wayland_wrapper` den Wayland-Socket an
   und startet `kwin_wayland` als Kind; `SO_PEERCRED` nennt deshalb den Wrapper, nicht den Besitzer von
   `org.kde.KWin`. Nachtrag 5 lehnte damit jede echte Plasma-Sitzung ab (gemessen; der virtuelle KWin der Probe startet
   ohne Wrapper und verdeckte es). Jetzt gilt: Der Prozess hinter dem Socket ist der Besitzer von `org.kde.KWin` **oder
   dessen Elternprozess** (gelesen aus `/proc/<pid>/stat`, Feld 4 — nur diese Statuszeile, nie Kommandozeile oder
   Umgebung). Der Server reicht die geprüfte Verbindung als Deskriptor 3 an das Kind (`WAYLAND_SOCKET=3`, libwayland
   nimmt sie statt des Pfads): Das Kind löst den Pfad nicht ein zweites Mal auf, ein Pfad mit `..`, ein Symlink oder ein
   zwischen Prüfen und Start umgehängter Socket führen es nicht woandershin. In der echten Plasma-Sitzung des
   Maintainers geprüft (23.11.2026: `pruefeSitzung` ok), im virtuellen KWin per Drehbuch `agenten-mcp.sh`.
2. **Ziel von `export` verrät Ordner und Dateien (A-3), Grenze.** Die Prüfung von `target` (Ordner fehlt, Datei
   vorhanden) läuft vor dem Lesen der Quelle und antwortet unterschiedlich. Bewusst so gelassen: Ein Agent, der den
   Server starten darf, führt ohnehin Programme als der Mensch aus und sieht dessen Dateien; die Antworten sind für das
   Arbeiten mit `export` nötig. Die Quelle (`file`) verrät nichts (Nachtrag 5 Punkt 6).
3. **Zweites Signal (A-6), Grenze.** Das zweite Signal beendet sofort; geräumt werden nur die temporären Dateien des
   Schreibens (`.schnapp-export-*`). Trifft es das Ablegen in der Bibliothek oder das Verwerfen danach, kann eine
   Aufnahme in `agents/` liegen bleiben; sie wird wie jede Aufnahme aufgeräumt. Der Satz „discarded even if …“ in
   `docs/agents.md` gilt für das erste Signal, `notifications/cancelled` und EOF.
4. **Verwerfen nur der eigenen Datei (A-5).** Den Stand der abgelegten Datei hält der Server direkt nach dem Ablegen
   fest; `library.Entfernen` löscht nur eine Datei mit genau diesem Stand.
5. **Erster Klick bei unfokussiertem Dialog (A-7), offen.** Bekommt der Dialog den Fokus erst durch den Klick auf
   „Allow“, beginnt die Sperre neu und der Klick zählt nicht; der Mensch klickt ein zweites Mal. Prüfpunkt im Echtlauf
   F-490.

## Nachtrag 7 (23.11.2026): Voriger Hinweis wird vor der nächsten Aufnahme geschlossen (Karte #208, F-490, F-506)

**Status:** Akzeptiert (Claude Opus 5.5). Grundlage: Echtlauf in Andis Plasma-Sitzung am 23.11.2026
(`scripts/echtlauf-mcp-hinweise.sh`).

- **F-490 bestätigt:** Mit `confirm = no` stand der Hinweis (Dringlichkeit critical) der ersten Aufnahme im Bild der
  zweiten, fünf Sekunden später. **Entscheidung:** Der Server merkt sich die Kennung seines letzten Hinweises
  (`notify.Zeige` liefert sie) und nimmt ihn vor der nächsten Aufnahme mit `CloseNotification` zurück
  (`notify.Schliesse`), dann wartet er 600 ms (`hinweisWeg`), bis Plasma ihn ausgeblendet hat; mit Dialog geschieht
  das schon vor dem Dialog. Die Dringlichkeit bleibt critical (Nachtrag 4 Punkt 8): Der Hinweis steht, bis die nächste
  Aufnahme kommt oder der Nutzer ihn schließt, und die nächste Aufnahme zeigt ihren eigenen. Scheitert das Schließen,
  wird trotzdem aufgenommen (verborgen wird nichts, der alte Hinweis steht dann im Bild). Abgelehnte oder verworfene
  Aufnahmen hinterlassen keinen Hinweis und nichts zu schließen.
- **Grenze:** Die Kennung gilt je Serverprozess. Ein zweiter `schnapp mcp` (anderer Agent, neue Sitzung) kennt den
  Hinweis des ersten nicht; ein noch stehender Hinweis eines anderen Prozesses kann in dessen erstem Bild stehen.
  Hinweise anderer Programme sind ohnehin Teil des Bildschirms.
- **F-506 widerlegt:** Bei Dialog ohne Fokus hat ein einziger Klick auf „Erlauben“ die Aufnahme erlaubt (Echtlauf,
  Teil B); die Fokus-Neusperre verschluckt den Klick nicht. Keine Änderung.
- Die Benachrichtigungs-Attrappe der Probeläufe (`scripts/fake-notifications`) kennt `CloseNotification` und
  protokolliert es.

## Nachtrag 8 (23.11.2026): Voriger Hinweis verschwindet nie ersatzlos (Sitzungs-QA, Kritiker B, B-1, B-4, B-5)

**Status:** Akzeptiert (Claude Opus 5.5). Berichtigt Nachtrag 7.

- **Lücke in Nachtrag 7 (B-1):** Der Server schloss den vorigen Hinweis vor dem Dialog. Jeder spätere Fehlerweg
  (Abbruch während des Wartens, abgelehnter Dialog, gescheiterte Aufnahme) endete ohne neuen Hinweis; ein Agent hätte
  den Hinweis seiner Aufnahme mit einem sofortigen Fehlversuch verschwinden lassen können. **Entscheidung:** Geschlossen
  wird erst nach dem Dialog, direkt vor der Aufnahme. Endet der Aufruf danach ohne eigenen Hinweis, zeigt der Server
  den vorigen erneut (gleicher Text, neue Kennung, `hinweisZurueck` mit eigener Frist von 5 s, weil der Aufruf oft
  gerade abgebrochen wurde). Ein abgelehnter Dialog schließt nichts.
- **Berichtigung (B-5):** Der Satz „Abgelehnte oder verworfene Aufnahmen hinterlassen keinen Hinweis“ in Nachtrag 7
  gilt nur für Ablehnung und Abbruch **vor** dem Hinweis. Eine Aufnahme, die nach dem Hinweis verworfen wird (Rendern,
  Abbruch vor dem Ablegen, Schreibfehler), hat ihren Hinweis schon gezeigt; er bleibt stehen und wird vor der
  nächsten Aufnahme geschlossen — so bleibt jeder Aufnahmeversuch sichtbar.
- **Absender (B-4):** `notify` nutzt die gemeinsame Bus-Verbindung des Prozesses (`dbus.SessionBus()`) statt je Aufruf
  einer neuen; `CloseNotification` kommt so vom selben Absender wie `Notify`, auch bei Diensten, die das Schließen
  fremder Hinweise ablehnen.
- **Noch offen:** Der Echtlauf vom 23.11. lief vor dem Fix. Ob Plasma den kritischen Hinweis binnen 600 ms ausblendet,
  ist im echten Plasma nachzumessen (`scripts/echtlauf-mcp-hinweise.sh A`); bis dahin bleibt F-490 offen.
- Das Echtlauf-Skript stellt die Einstellungen nach jedem Teil zurück, schreibt sie über eine Nachbardatei und bricht
  ab, wenn ein früherer Lauf ohne Abschluss liegt (Marke `laeuft`; Sitzungs-QA B-2, B-3).

## Nachtrag 9 (23.11.2026): Grenzen des Wiederzeigens (Verify der Sitzungs-QA, N-1, N-3)

**Status:** Akzeptiert (Claude Opus 5.5). Präzisiert Nachtrag 8: „nie ersatzlos“ gilt für jeden Weg über das
MCP-Protokoll, nicht bei hartem Prozessende.

- **Schreibfehler (N-1):** Scheitert das Schreiben einer Überlauf-Ablehnung, bricht der Server das laufende Werkzeug ab
  und wartet auf sein Ende, bevor er sich beendet; das Wiederzeigen läuft also noch.
- **Grenze:** Endet der Prozess hart (SIGKILL, zweites Signal, Absturz) zwischen Schließen und neuem Hinweis, bleibt
  der vorige Hinweis weg. Das setzt Kontrolle über den Prozess voraus, nicht nur über die MCP-Nachrichten; dagegen
  hilft nur `confirm = yes`.
- **Schließen gescheitert (N-3):** Antwortet der Dienst mit einem Fehler, steht der alte Hinweis noch; er wird nicht
  zusätzlich gezeigt. Scheitert danach auch die Aufnahme, bleibt er zum Schließen vorgemerkt; gelingt sie, bleibt er
  neben dem neuen stehen (zu viel Hinweis, hingenommen). Endet der Aufruf dagegen durch Abbruch oder Frist, ist
  unbekannt, ob der Dienst schon geschlossen hat: Dann gilt der Hinweis als geschlossen und wird bei einem Fehlschlag
  wieder gezeigt (Verify 2, Funde 2 und 3).
- **Hingenommen:** Hat der Nutzer den Hinweis selbst weggeklickt, erscheint er nach einem späteren Fehlversuch erneut,
  und ein mitten im Zeigen abgebrochener neuer Hinweis kann neben dem alten stehen. Beides geht in die vorsichtige
  Richtung (zu viel statt zu wenig Hinweis).
- Das Echtlauf-Skript bricht bei einer `settings.ini` als Symlink ab, stellt nur zurück, was es selbst gesetzt hat,
  und sichert eine zwischendurch geänderte Fassung im Laufordner (N-5, N-6).
