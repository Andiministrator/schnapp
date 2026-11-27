# ADR-0042: Aufnahme mit Verzögerung

## Status
Akzeptiert (Claude Opus 5.5, 27.11.2026, Karte #338 in Sprint 14; Vorschlag Claude, von Andi am 25.11.2026 übernommen;
Andi 27.11.: „arbeite selbstständig“). Ergänzt ADR-0017 (Aufruf) und ADR-0018 (Einstellungen), Anforderung A-11.

## Kontext
Aufgeklappte Menüs, Tooltips und Hover-Zustände schließen sich, sobald man das Tastenkürzel drückt oder ins Overlay
klickt. Snagit, Spectacle und Flameshot bieten deshalb eine Wartezeit vor der Aufnahme.

## Entscheidung
1. **Befehlszeile `--delay=SEK`** (nur mit Gleichheitszeichen, ganze Sekunden 0 bis 60) für alle Aufnahmearten
   (`--region`, `--screen`, `--window`, `--repeat`); mit einer Datei, `--list-screens`, `--help` usw. ein Fehler.
   Mehrfach nur mit demselben Wert.
2. **Einstellung `[capture] delay`** (0 bis 60, Voreinstellung 0); `--delay` geht vor, `--delay=0` hebt sie auf.
   Die Einstellungstafel (Reiter Capture) bietet „Keine“, 3, 5 und 10 Sekunden; ein anderer Wert aus der Datei bleibt
   und wird als Zahl gezeigt.
3. **Gewartet wird vor der Verbindung zu KWin**, also bevor das Bild entsteht; beim Bereich zeigt das Overlay danach
   das eingefrorene Bild wie sonst. Ein Abbruch während der Wartezeit (Strg+C) hinterlässt nichts.
4. **Countdown als Benachrichtigung** (Dringlichkeit normal): „Aufnahme in N Sekunden“, jede Sekunde ersetzt
   (`replaces_id`, `notify.Ersetze`). Der letzte Hinweis wird 600 ms vor der Aufnahme geschlossen (Plasma blendet aus,
   F-490), damit er nicht im Bild steht; die Gesamtzeit bleibt N Sekunden. Gibt es keinen Benachrichtigungsdienst, wird
   trotzdem gewartet.
5. **MCP und Editor:** Der MCP-Server kennt keine Verzögerung; „Neue Aufnahme“ im Editor (Karte #341) soll die Stufen
   später anbieten.

## Erwogene Alternativen
- **Eigenes Countdown-Fenster:** braucht ein zweites Gio-Fenster vor der Aufnahme und müsste ebenso verschwinden; die
  Benachrichtigung gibt es schon (MCP-Hinweis).
- **Gebrochene Sekunden:** keine Vorbilder, unnötig.

## Konsequenzen
`cmd/schnapp/argumente.go`, `main.go` (`plane`), `aufnehmen.go` (`anschluesse.warte`), `countdown.go`;
`internal/notify` (`Ersetze`); `internal/settings` (`Verzoegerung`, `MaxVerzoegerung`); Tafel
`internal/ui/einstellungen_logik.go`. Tests und Mutanten `scripts/mutanten/verzoegerung.json`. Echtlauf mit
Benachrichtigung in der Plasma-Sitzung offen.

## Nachtrag 1 (27.11.2026, Kritik #338, F-938, F-939, F-941)
1. **Wartezeit hängt nur an der Uhr (F-938):** Jeder Hinweis hat eine Frist von 800 ms; nach dem ersten Fehler gibt der
   Countdown die Hinweise auf und wartet nur noch. Die angezeigte Zahl ist die aufgerundete Restzeit, ein Hinweis wird
   nur ersetzt, wenn sie sich ändert. Vorher verlängerte ein hängender Dienst `--delay=3` auf 15 s.
2. **Abstand zum Schließen (F-939):** Kam das Schließen des letzten Hinweises später als 600 ms vor dem Ende (langsamer
   Dienst), wartet der Countdown 600 ms ab dem Schließen; ein sauberes Bild geht vor der genauen Gesamtzeit.
3. **Abbrechen (F-941, Entscheidung Claude):** Ein laufender Countdown lässt sich nur mit Strg+C im Terminal abbrechen;
   vom Tastenkürzel gestartet wartet er ab, beim Bereich führt Esc im Overlay heraus. Eine Aktion „Abbrechen“ am
   Hinweis (`ActionInvoked`) kommt erst mit dem Menü „Neue Aufnahme“ (#341) in Betracht; bis dahin bleibt es so.
