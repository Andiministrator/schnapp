# ADR-0017: Globale Tastenkürzel über die Desktop-Datei

## Status
Akzeptiert (Andi, 19.11.2026: E-10 Empfehlung, Taste Variante „a“ = Druck).

## Kontext
„Schnell zur Hand“ (Kernprinzip 4) heißt: ein Tastendruck, sofort das Overlay. Unter KDE gibt es drei Wege:
KGlobalAccel über D-Bus (Schnapp müsste laufen), das Portal `GlobalShortcuts` (ebenfalls laufender Prozess, dafür
auch GNOME), oder Aktionen in der `.desktop`-Datei, die KDE mit Kürzeln verbindet. Der Prototyp startet in rund
130 ms bis zum ersten Frame — ein neuer Prozess je Tastendruck ist schnell genug.

## Entscheidung
- **Erste Fassung:** `schnapp.desktop` mit Aktionen, jede startet Schnapp mit Argument:
  `Bereich` (`schnapp --bereich`), `Bildschirm`, `Fenster`, `Letzten Bereich wiederholen`.
- **Voreinstellung Druck** für „Bereich“ (Andi: Schnapp ist das Haupt-Werkzeug); Spectacle belegt Druck
  standardmäßig — dort wird das Kürzel abgeschaltet (Hinweis in README/Installation). Weitere Kürzel nur als
  Vorschlag, alles in den Systemeinstellungen änderbar.
- Ob KDE Voreinstellungen aus der `.desktop`-Datei übernimmt (`X-KDE-Shortcuts` je Aktion, vermutlich wie bei
  Spectacle), wird vor der Umsetzung am Spectacle-Quelltext belegt; sonst richtet der Nutzer das Kürzel einmal ein.
- **Später:** Portal `GlobalShortcuts` für andere Desktops (ADR-0007).

## Konsequenzen
- Kein Hintergrunddienst, kein eigener Kürzel-Code in der Ersten Fassung.
- Die `.desktop`-Datei trägt zugleich die Berechtigungszeile für Plasma ≤ 6.7 (ADR-0007 Nachtrag 3) und die
  Übersetzungen `Name[de]=…` (ADR-0014).
- „Letzten Bereich wiederholen“ braucht den zuletzt gewählten Bereich auf Platte (`~/.local/state/schnapp/`).

## Nachtrag 1 (19.11.2026): Voreinstellung Druck belegt, Umsetzung der Desktop-Datei (Karte #30)

- **Belegt:** Spectacle trägt seine Voreinstellungen als `X-KDE-Shortcuts` in der `.desktop`-Datei, im Hauptteil
  (`X-KDE-Shortcuts=Print,Meta+Shift+S`) und je Aktion (etwa `[Desktop Action FullScreenScreenShot]` mit
  `X-KDE-Shortcuts=Shift+Print`). Quelle: `desktop/org.kde.spectacle.desktop.cmake` im Spectacle-Quelltext
  (https://invent.kde.org/plasma/spectacle/-/raw/master/desktop/org.kde.spectacle.desktop.cmake, gelesen am 19.11.2026);
  `src/ShortcutActions.cpp` benennt die globalen Kürzel ausdrücklich „to match the jumplist actions in our .desktop
  file“. Dass KDE (nicht Spectacle) die Zeile auswertet, folgt daraus, dass Spectacles Code keine Voreinstellung
  setzt; den Quelltext von KGlobalAccel haben wir nicht gelesen. Ein Test auf einer echten Plasma-Sitzung steht aus
  (Andi, bei der ersten Installation).
- **Umgesetzt:** `data/schnapp.desktop.in` mit den Aktionen `Bereich`, `Bildschirm`, `Fenster`, `LetztenBereich`;
  `X-KDE-Shortcuts=Print` nur an `Bereich`. `@EXEC@` ist der absolute Pfad der Programmdatei (KWin bindet die
  Aufnahme-Berechtigung daran, ADR-0007 Nachtrag 3). Spectacle belegt `Print` ebenfalls: gleichzeitig installiert,
  meldet KDE einen Konflikt, der Nutzer entscheidet in den Systemeinstellungen (Hinweis in README/Installation, sobald
  es ein Installationsskript gibt).
- **Vorläufig (überholt durch Nachtrag 3):** Die Aktion `Bildschirm` ruft `schnapp --screen` ohne Wert; das gibt
  die Monitorliste aus. Mit der Umsetzung der Aufnahme wählt der Aufruf den Monitor selbst (Karte im Board).

## Nachtrag 2 (19.11.2026): Optionen und Aktionen englisch

**Status:** Akzeptiert (Andi, 19.11.2026: „englisch“). Schnapp ist öffentlich und englisch-zuerst (ADR-0014); die
Kommandozeile ist Teil der Oberfläche und nach einer Veröffentlichung nur unter Bruch von Kürzeln und Skripten zu
ändern. Darum: `--region` (Voreinstellung), `--screen[=NAME]`, `--window`, `--repeat`, `--version`, `--help`; die
Desktop-Aktionen heißen `Region`, `Screen`, `Window`, `Repeat` (ihre angezeigten Namen bleiben übersetzt). Die
deutschen Namen oben (`--bereich` usw.) gelten nicht mehr.

## Nachtrag 3 (19.11.2026): `--screen` ohne Wert und die Einstellung dazu (F-58)

**Status:** Akzeptiert (Andi, 19.11.2026: „dann folge ich deinem Vorschlag“).

**Stand 19.11.2026 (nach #47, #52, #34):** umgesetzt. `--list-screens` listet; `--screen` ohne Wert folgt
`[capture] screen` in der Einstellungsdatei (ADR-0018): `pick` öffnet das Overlay im Monitor-Modus, `pointer` nimmt
den aktiven Monitor sofort, `all` meldet noch „nicht umgesetzt“ (*umgesetzt seit Nachtrag 7*). Enter im Overlay nimmt den Monitor **unter dem
Zeiger** (nicht den mit dem Tastaturfokus), Esc oder Rechtsklick bricht ab (Exit 3). `--repeat` wiederholt den
zuletzt im Overlay oder mit `--screen=NAME` gewählten Bereich; `pointer` und `--window` merken keinen (KWin nennt
dabei keinen Monitornamen bzw. es ist kein Bereich).

Die Desktop-Aktion `Screen` („Bildschirm aufnehmen“, per Menü oder frei belegbarer Taste; `Print` bleibt an `Region`)
ruft `schnapp --screen` ohne Wert. Was dieser Aufruf tut, legt eine **Einstellung** fest:

| Wert | Verhalten |
|---|---|
| **Monitor per Klick wählen** (Voreinstellung) | Overlay wie bei `--region`, aber im Monitor-Modus: der Monitor unter dem Mauszeiger ist hervorgehoben, ein Klick nimmt ihn ganz, Ziehen wählt doch einen Bereich, Esc bricht ab |
| Monitor unter dem Mauszeiger sofort | ohne Overlay; KWin `CaptureActiveScreen` (Wayland gibt Programmen keine globale Zeigerposition). KWins „aktiver Bildschirm“ folgt in der Voreinstellung dem Mauszeiger, nicht dem Fenster mit Fokus (gemessen 19.11.2026, F-72, `knowledge/topics/kwin-screenshot.md`) |
| Alle Monitore | ein Bild über alle Monitore; je Bildschirm aufnehmen und zusammensetzen, nicht `CaptureWorkspace` (rechnet bei gemischter Skala hoch, `knowledge/topics/kwin-screenshot.md`) |

- `--screen=NAME` nimmt immer genau diesen Monitor, unabhängig von der Einstellung.
- Die Monitorliste gibt künftig `--list-screens` aus; `--screen` allein listet nicht mehr (Nachtrag 1, „Vorläufig“,
  ist damit überholt).
- Die Einstellung ist die erste in Schnapp. Ort und Format der Einstellungsdatei legt ADR-0018 fest (umgesetzt mit
  Karte #52).
- Umsetzung: Kommandozeile (`--list-screens`, `--screen` ohne Liste) mit dem Rest von F-58; Monitor-Modus mit dem
  Overlay.

## Nachtrag 4 (19.11.2026): Editor öffnen — `--no-editor` und `schnapp DATEI` (ADR-0020)

- Nach `--region`, `--screen`, `--window` und `--repeat` öffnet der Editor, nachdem der Pfad auf stdout steht, außer
  `[capture] open-editor = false` (ADR-0018 Nachtrag 1) oder `--no-editor`. `--no-editor` lässt sich mit diesen vier
  Modi kombinieren, mit keinem anderen.
- `schnapp DATEI` (genau ein Positionsargument, ohne weitere Option) öffnet die Datei im Editor; ein
  Argument, das mit `-` beginnt, bleibt eine Option. Den Pfad gibt Schnapp dabei nicht aus.

## Nachtrag 5 (19.11.2026): Tasten im Editor (#35, Kritik B-8)

Strg+Z rückgängig; Strg+Umschalt+Z und Strg+Y wiederholen; Entf und Rücktaste löschen die Auswahl; Esc bricht einen
Zug ab, sonst hebt es die Auswahl auf; Strg+S speichern; Strg+0 einpassen; Strg+1 100 %; Strg+W schließen. Maus: Rad
zoomt zum Zeiger, Ziehen auf freier Fläche oder mit der mittleren Taste verschiebt die Ansicht. Während eines Zugs
sind Rückgängig, Löschen, Speichern und Zoomen gesperrt (F-18, Kritik A-6). Nutzer-Doku: README.md/README.de.md.

*Ergänzt durch ADR-0021: Werkzeugkürzel V/A/R/T/N, Esc in Stufen, Tab und Enter bei Texteingabe (Nachtrag 1).*

## Nachtrag 6 (20.11.2026): `--ignore-limit` öffnet Dateien über dem Rechenbudget (F-146)

**Status:** Akzeptiert (Andi, 20.11.2026: Umgehungsweg ja).

- `schnapp --ignore-limit DATEI` (Reihenfolge beliebig) öffnet die Datei auch dann, wenn `render.Pruefe` das Rastern
  wegen des Rechenbudgets ablehnen würde (ADR-0020 Nachtrag 1 und 3). Anlass: Eigene Arbeit, die im Editor über das
  Budget gewachsen ist, wird beim Speichern nur gewarnt (F-138) und ließe sich sonst nie wieder öffnen.
- Der Schalter überspringt **nur** diese Ablehnung. Formatprüfung und Grenzen beim Lesen (Bildgröße, `MaxObjekte`,
  `MaxStreifen`, `MaxKacheln`) gelten weiter; eine Arbeitsdatei darüber wird wie bisher nur als Bild gelesen.
- Ohne Datei ist der Schalter ein Aufruffehler (Exit 2), auch bei Aufnahmen, `--help` und `--list-screens`: still
  wirkungslos wäre irreführend. Mit `--no-editor` gilt weiter „Datei nicht kombinierbar“. `--ignore-limit=WERT` ist
  unbekannt.
- Die Ablehnung beim Öffnen nennt den Schalter („… Mit --ignore-limit lässt sie sich trotzdem öffnen.“); `--help`
  führt ihn auf. Aufnahmen öffnen den Editor nie mit dem Schalter. Ein sehr aufwendiges Dokument kann das Rendern
  im Editor stark verlangsamen; das nimmt der Schalter bewusst in Kauf.

## Nachtrag 7 (21.11.2026): „Öffnen mit“, EXIF-Ausrichtung, `screen = all` (F-113, F-119, F-327)

**Status:** Akzeptiert (Claude, Andi informiert). Umgesetzt in Sprint 10.

- **„Öffnen mit“ (F-113):** Die Desktop-Datei (`data/schnapp.desktop.in`) trägt im Haupt-`Exec` ein `%f` und
  `MimeType=image/png;image/jpeg;`. Dateimanager bieten Schnapp damit für PNG und JPEG an; die gewählte Datei geht an
  `schnapp DATEI` (Nachtrag 4). Die
  Aktionen (`--region`, `--screen`, `--window`, `--repeat`) bleiben ohne Datei.
- **EXIF-Ausrichtung (F-113):** Ein JPEG mit EXIF-Ausrichtung 2 bis 8 wird beim Öffnen so gedreht bzw. gespiegelt,
  wie Bildbetrachter es zeigen (eigener kleiner Leser für Tag 0x0112, ADR-0011: kein Fremdpaket). Die Angabe selbst
  wird **nicht** übernommen (sie fiele mit allen übrigen Metadaten ohnehin weg, Kernprinzip 3); die Arbeitsdatei
  speichert das gedrehte Bild. Neben der Quelle entsteht genau eine Vollbild-Kopie (F-323).
- **`[capture] screen = all` (F-119):** `--screen` nimmt alle Monitore als **ein** Bild über KWin `CaptureWorkspace`
  auf (ScreenShot2, gleicher Pixeltransport wie die übrigen Aufnahmen, `native-resolution`): KWin setzt nach der
  Anordnung der Monitore zusammen, bei gemischter Skalierung gilt die größte, kleinere Monitore werden hochgerechnet.
  Das Bild wird als Ganzes abgelegt, `--repeat` merkt dafür keinen Bereich. Geprüft im virtuellen KWin: zwei Monitore
  1600 × 1000 ergeben 3200 × 1000.
- *Ergänzt (F-345, Karte #153):* `Alle` fragt bei ScreenShot2 die Version ab; unter Version 3 (kein `CaptureWorkspace`)
  meldet es „This KWin version cannot capture all screens at once. Use a single screen instead.“ statt des rohen
  D-Bus-Fehlers (nur gegen die Attrappe geprüft).

## Nachtrag 8 (22.11.2026): Speichergrenze beim Öffnen eines JPEG (F-361)

**Status:** Akzeptiert (Claude, Andi informiert). Umgesetzt in Sprint 11.

- Die Pixelgrenze (`document.MaxPixel`) allein begrenzt den Speicher nicht: Go `image/jpeg` hält bei einem
  progressiven JPEG je Komponente die Koeffizienten (4 Byte je Pixel), eine präparierte Datei von 1,5 MB forderte so
  4864 MiB an (gemessen, `knowledge/messungen/f-361.md`).
- `Oeffne` schätzt den Bedarf vorab aus dem SOF-Segment (Ausgabeebenen, Koeffizienten bei progressiv, CMYK-Bild,
  NRGBA des Dokuments) und lehnt über **2 GiB** mit „The image is too large.“ ab. Baseline-JPEG in YCbCr oder Grau
  bleiben bis zur Pixelgrenze erlaubt; progressiv 4:4:4 geht bis etwa 113 Megapixel, 4:2:0 bis etwa 186, CMYK bis etwa
  179 (vorher 268). Fotos aus Kameras und Telefonen liegen weit darunter.
- Kein Fremdpaket (ADR-0011): eigener kleiner Leser für die Marker bis zum ersten SOF; unbrauchbare Köpfe überlässt er
  dem Dekoder, der sie ablehnt.
