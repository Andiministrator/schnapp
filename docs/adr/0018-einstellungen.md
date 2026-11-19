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
