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
