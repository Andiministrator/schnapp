# ADR-0018: Einstellungen — Ort, Format, Vorrang

## Status
Akzeptiert (Andi, 19.11.2026: „passt“; Karte #49)

## Kontext
Mit ADR-0017 Nachtrag 3 bekommt Schnapp seine erste Einstellung: was „Bildschirm aufnehmen“ ohne Monitornamen tut.
Weitere stehen schon fest oder sind absehbar: das automatische Aufräumen der Bibliothek (an/aus, Frist in Tagen,
ADR-0016, heute nur als Voreinstellung im Code `bibliothek.Voreinstellung`), die Sprache der Oberfläche (Karte
„Übersetzung ausbauen“), später Autor für Metadaten, Farben, Werkzeug-Voreinstellungen. Bevor die erste davon gebaut
wird, müssen Ort, Format und Fehlerverhalten feststehen, denn eine veröffentlichte Einstellungsdatei lässt sich nur
mit Übergangsregeln wieder ändern.

Randbedingungen: nur Wayland, KDE zuerst, später andere Desktops (ADR-0007); kein cgo und Fremdpakete nur mit
spürbarem Nutzen (ADR-0011); Pfade klein `schnapp` (ADR-0016); die Kommandozeile ist englisch (ADR-0017 Nachtrag 2).

## Entscheidung
- **Ort:** `$XDG_CONFIG_HOME/schnapp/settings.ini`, ohne gesetztes `XDG_CONFIG_HOME` also
  `~/.config/schnapp/settings.ini`. Ordner 0700, Datei 0600 wie bei der Bibliothek (ADR-0016) — die Datei verrät
  Gewohnheiten und später womöglich einen Autorennamen.
- **Abgrenzung:** Einstellungen sind, was der Nutzer bewusst wählt. Was Schnapp sich selbst merkt (letzter Bereich,
  Fenstergrößen), liegt unter `$XDG_STATE_HOME/schnapp/` (`~/.local/state/schnapp/`, ADR-0017); Index und Vorschau
  bleiben unter `~/.local/share/schnapp/` und `~/.cache/schnapp/` (ADR-0016).
- **Format:** einfache INI-Datei, von Hand lesbar und editierbar, mit eigenem kleinen Leser (Standardbibliothek, kein
  Fremdpaket): Abschnitte `[name]`, Zeilen `schlüssel = wert`, Kommentare mit `#` oder `;` am Zeilenanfang, UTF-8.
  Schlüssel und Werte englisch und klein, weil Datei und Kommandozeile öffentlich dokumentiert werden. Beispiel:

  ```ini
  [capture]
  # pick = Monitor per Klick im Overlay, pointer = Monitor unter dem Mauszeiger sofort, all = alle Monitore
  screen = pick

  [library]
  cleanup = true
  cleanup-days = 90
  ```

- **Fehlende Datei oder fehlender Schlüssel:** Voreinstellung. Schnapp legt die Datei nicht von selbst an.
- **Unbekannter Schlüssel oder ungültiger Wert:** Schnapp startet trotzdem, nimmt für diesen Schlüssel die
  Voreinstellung und gibt eine Warnung auf stderr aus (Datei, Zeile, Schlüssel). Eine kaputte Einstellungsdatei darf
  die Aufnahme nie verhindern, denn die Aufnahme ist das, was der Nutzer per Taste gerade will.
- **Vorrang:** Kommandozeile vor Einstellungsdatei vor Voreinstellung im Code. `--screen=NAME` nimmt immer diesen
  Monitor; `--screen` ohne Wert folgt `[capture] screen`.
- **Schreiben:** nur ein künftiger Einstellungsdialog schreibt, atomar (temporäre Datei im selben Ordner, dann
  umbenennen). Er ändert nur die Zeilen der Schlüssel, die er setzt, und lässt Kommentare, Reihenfolge und unbekannte
  Schlüssel stehen, damit Handänderungen nicht verloren gehen.
- **Ein Paket** (`einstellungen`) liest die Datei und liefert typisierte Werte; die übrigen Pakete bekommen die
  Werte übergeben und lesen die Datei nie selbst. Tests nur mit eigenem `XDG_CONFIG_HOME` unter `t.TempDir()`.

## Erwogene Alternativen
- **TOML/YAML mit Fremdpaket:** mächtiger, aber für eine Handvoll flacher Schlüssel kein spürbarer Nutzen (ADR-0011).
- **JSON (Standardbibliothek):** keine Kommentare, beim Bearbeiten von Hand fehleranfällig.
- **KConfig (`kwriteconfig6`) oder GSettings/dconf:** an einen Desktop gebunden bzw. mit cgo oder Schema-Installation;
  widerspricht der austauschbaren Desktop-Schicht (ADR-0007).

## Konsequenzen
- Neue Einstellungen werden in diesem ADR als Nachtrag mit Schlüssel, Werten und Voreinstellung aufgeführt; die
  Format-Spezifikation (E-15) bekommt einen Abschnitt zur Datei.
- Erste Schlüssel: `[capture] screen` = `pick` (Voreinstellung) | `pointer` | `all` (ADR-0017 Nachtrag 3);
  `[library] cleanup` = `true` (Voreinstellung) | `false`, `[library] cleanup-days` = 1…36500, Voreinstellung 90
  (ADR-0016, Grenzen wie `bibliothek.Einstellung`).
- Die Sprache (`[interface] language`) folgt mit ihrer Karte; bis dahin gilt die Systemsprache (ADR-0014).
- Umsetzung: Paket `einstellungen` mit Leser, Tests (fehlende Datei, Kommentare, unbekannter Schlüssel, ungültiger
  Wert, Rechte), angebunden in `cmd/schnapp` — eigene Karte; der Dialog kommt mit dem Editor.

## Nachtrag 1 (19.11.2026): `[capture] open-editor` (ADR-0020)

`[capture] open-editor` = `true` (Voreinstellung) | `false`: Editor nach einer interaktiven Aufnahme öffnen. Für einen
einzelnen Aufruf schaltet `--no-editor` ihn ab (ADR-0017 Nachtrag 4).

## Nachtrag 2 (22.11.2026): `[general] language` (ADR-0014 Nachtrag 2)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #28).

`[general] language` = `auto` (Voreinstellung) | Sprachcode `ll`, `lll`, `ll_CC` oder `lll_CC` (etwa `de`, `en`,
`pt_BR`): Sprache der Oberfläche. `auto` folgt der Umgebung wie gettext (ADR-0014 Nachtrag 1). Gibt es die gewählte
Sprache nicht, gilt ebenfalls die Umgebung, ohne Warnung (die Datei kennt die eingebauten Sprachen nicht); ein Wert in
anderer Form ist ungültig und wird gewarnt. `main` liest die Datei vor dem Laden der Sprache nur für diesen Schlüssel
und gibt dabei keine Warnungen aus; die Warnungen kommen wie bisher übersetzt bei Aufnahme und Editor. Neuer Abschnitt
`[general]` für Einstellungen, die das ganze Programm betreffen.

## Nachtrag 3 (22.11.2026): `[capture] seam` (Karte #96, ADR-0023 Nachtrag 6)

**Status:** Akzeptiert (Andi 22.11.2026: Form einstellbar, eine als Voreinstellung; umgesetzt von Claude Opus 5.5).

`[capture] seam` = `tear` (Voreinstellung) | `wave` | `zigzag` legt die Form der Stauch-Kante neuer Dokumente fest:
einer Aufnahme und eines einfachen PNG/JPEG, das Schnapp öffnet, auch einer beschädigten oder neueren Arbeitsdatei, die
als Bild geöffnet wird. Eine Arbeitsdatei behält ihre gespeicherte Form (`s:kante`). Die Werte gelten genau so
geschrieben (Groß-/Kleinschreibung zählt); ein anderer Wert warnt wie jeder ungültige Wert und lässt den bisherigen
Stand. Gesetzt wird die Form ohne Rückgängig-Schritt (`document.VorgabeKante`); sie ist keine Änderung des Nutzers.
`--export` liest die Einstellungen nicht; das ist ohne Wirkung, weil ein frisch geöffnetes Bild keine Naht hat.

## Nachtrag 4 (22.11.2026): Abschnitt `[agents]` (Karte #173, ADR-0030 Nachtrag 2)

**Status:** Akzeptiert (Claude Opus 5.5; Entscheidung Andi vom 22.11.2026 an Karte #173).

Neuer Abschnitt `[agents]` für den MCP-Server, Werte `yes`/`no` (auch `true`/`false`):
`capture = no`, `library = no`, `confirm = yes`, `image = yes`. Ein ungültiger Wert lässt die Voreinstellung stehen
(mit Warnung) — eine Einwilligung entsteht nie aus einem Tippfehler. Der Server liest die Datei bei jedem
Werkzeugaufruf neu und schreibt sie nie. Bedeutung im Einzelnen: ADR-0030 Nachtrag 2 Punkt 2.
*Ergänzt 23.11.2026 (Karte #202, ADR-0030 Nachtrag 10 Punkt 5):* `edit = no` — Agenten dürfen Dateien beschriften und
schwärzen (Werkzeug `annotate`).

## Nachtrag 5 (24.11.2026): Schreiben umgesetzt, zweite Datei `styles.ini`

**Status:** Akzeptiert (Claude Opus 5.5, Sitzungs-QA 24.11., F-635).

Der Satz „nur ein künftiger Einstellungsdialog schreibt“ ist mit ADR-0034 eingelöst (`settings.Schreibe`, nur die
geänderten Schlüssel, atomar). Seither gilt: Jeder neue Schlüssel bekommt im selben Commit eine Zeile in der
Einstellungstafel (ADR-0034 Punkt 2, ein Test vergleicht). Im selben Ordner liegt `styles.ini` (ADR-0037 Punkt 6): eine
Datei des Programms mit den eigenen Schnellstilen, die nicht in die Tafel gehört. `schnapp mcp` schreibt keine der
beiden Dateien; der Reiter „Agents“ der Tafel schreibt die Datei, die der Server liest (ADR-0034, F-625).

## Nachtrag 6 (24.11.2026): `[library] bar` (ADR-0036 Punkt 2, Karte #174)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #174 Teil 1).

Neuer Schlüssel `bar` im Abschnitt `[library]`: die Leiste der letzten Aufnahmen unter dem Bild (ADR-0036 Punkt 1).
Werte `yes`/`no` (auch `true`/`false`), Voreinstellung **`yes`**; ein ungültiger Wert warnt und lässt die Leiste an.
`settings.Einstellungen.Leiste`. Geschrieben wird `yes`/`no` (`settings.Werte`), von der Einstellungstafel (Reiter
Library, ADR-0034 Nachtrag 2) und vom Menüschalter Ansicht → „Recent captures“ (Strg+Umschalt+R, ADR-0026 Nachtrag 9).
Gelesen wird der Wert beim Öffnen des Editors (`internal/ui/leiste_unten.go`, `ladeLeiste`), wie die Tafel die Datei
frisch liest; `schnapp mcp` liest ihn nicht.
