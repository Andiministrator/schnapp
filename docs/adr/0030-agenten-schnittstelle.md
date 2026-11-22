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

