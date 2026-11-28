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

## Nachtrag 2 (28.11.2026, Karte #396; ADR-0046 Nachtrag 1 Punkt 9, F-941)
1. **Verzögerung im Menü:** Datei → „New capture“ (Strg+N) nimmt wie bisher mit `[capture] delay` auf. Darunter stehen
   drei Einträge „New capture in 3 s“, „… 5 s“, „… 10 s“ (ohne Kürzel, grau wie „New capture“). Die Menüzeile kennt
   keine Untermenüs; drei Einträge bleiben per Tastatur und Maus gleich erreichbar, und die Stufen sind die der
   Einstellungstafel (Punkt 2).
2. **Übergabe an den neuen Prozess:** Der Editor startet das eigene Programm mit `--internal-new-capture --delay=SEK`.
   `cmd/schnapp` nimmt nach dem Schließen der geerbten Deskriptoren genau diese zweite Form an und reicht sie an die
   gewöhnliche Prüfung der Befehlszeile weiter (`--delay` geht vor `[capture] delay`, Punkt 2); jedes andere zweite
   Argument bleibt ein gewöhnlicher Aufruf (kein Schließen).
3. **„Abbrechen“ am Hinweis (löst Nachtrag 1 Punkt 3 ab):** Der Countdown-Hinweis trägt die Aktion `cancel` mit der
   Beschriftung „Cancel“. Der Countdown abonniert vor dem ersten Hinweis das Signal `ActionInvoked` des
   Benachrichtigungsdienstes; kommt es für einen seiner Hinweise mit `cancel`, schließt er den Hinweis und die Aufnahme
   endet wie ein Abbruch im Overlay (Exit 3, nichts abgelegt, kein Editor). Gelingt das Abonnement nicht (kein Dienst,
   Frist), wartet der Countdown wie bisher; Strg+C im Terminal bleibt. Das gilt für jeden Countdown, auch vom
   Tastenkürzel oder aus dem Terminal.
4. **Nach der Kritik der Sitzung (28.11.2026, Fable):** Das Abonnement lebt bis zum Ende des Countdowns; die Frist von
   800 ms gilt nur für das Anmelden beim Bus (`notify.Aktionen(ctx, aufbau)`). Die erste Fassung band die Lebensdauer
   an die Frist, und kein Klick kam an (F-1052). Die Regel nimmt nur Signale des Inhabers von
   `org.freedesktop.Notifications` an (`WithMatchSender`), ein anderes Programm bricht nichts ab (F-1054). Geprüft auf
   einem eigenen, leeren Bus (`internal/notify/abo_bus_test.go`, startet `dbus-daemon`).
