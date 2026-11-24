# ADR-0015: Gepatchtes Gio als Kopie im Repo (`extern/gio`)

## Status
Akzeptiert (Claude Opus 5.5, 19.11.2026, Karte #16 in Sprint 2; Andi hatte Sprint 2 freigegeben, ohne diese Weiche
vorzugeben). Leicht umkehrbar: eine `replace`-Zeile je Modul und ein Ordner. Setzt die Konsequenz aus ADR-0003
Nachtrag 2 um („vor der ersten Fassung festlegen, wie der Patch ins Repo und in die CI kommt“).

## Kontext
Schnapp braucht Gio mit einem Patch (Monitorwahl, gebrochene Skalierung; ADR-0003 Nachtrag 2), bis Gio beides selbst
kann. Im Prototyp lag die gepatchte Kopie gitignoriert unter `tmp/gio-gepatcht`, und `prototyp/bauen.sh` schrieb die
Ersetzung in eine erzeugte Modul-Datei unter `tmp/`. Damit bauten nur das Skript und nur dieser Rechner. Ein frischer
Klon, `go build ./...`, `go test ./...` und die CI kannten den Patch nicht.

Möglichkeiten:

1. **Patch + Bauskript** (wie im Prototyp): kleines Repo, aber `go build` geht nur nach einem Vorlauf; `replace` auf
   einen gitignorierten Pfad bricht fremde Klone.
2. **Fork auf GitHub** (`replace gioui.org => github.com/…/gio vX`): sauber für Go, aber ein zweites öffentliches Repo
   zu pflegen, Veröffentlichung unter dem Konto des Maintainers, und jede Patch-Änderung braucht dort einen Tag.
3. **Gepatchte Kopie im Repo** als eigenes Modul (`extern/gio`) mit `replace gioui.org => ../extern/gio`: `go build`
   geht in jedem Klon ohne Vorlauf, und der Gio-Quelltext kommt aus dem Repo statt aus dem Netz (Gios eigene
   Abhängigkeiten wie `gioui.org/shader` lädt Go weiter aus dem Modulcache bzw. Netz); Flatpak-Bau braucht keinen
   Patch-Schritt. Kosten: 3 MB und rund 300 Dateien fremder Code im Repo.

## Entscheidung

Möglichkeit 3.

- `extern/gio/` ist Gio in der Version aus `extern/gio-patch.sh` (`GIO_VERSION`, derzeit v0.10.3) mit
  angewendetem Patch. Ein eigenes Go-Modul (`module gioui.org`); `go build ./...` im Wurzelmodul greift deshalb nicht
  hinein.
- `extern/gio-ausgabe.patch` bleibt die maßgebliche Beschreibung der Änderung (für Prüfung, Gio-Updates und die
  Einreichung bei Gio, Karte #17).
- `extern/gio-patch.sh` hält beides zusammen: `pruefen` (unverändertes Gio aus dem Modulcache + Patch = `extern/gio`,
  läuft in der CI), `erzeugen` (Patch nach einer Änderung in `extern/gio` neu schreiben), `anlegen` (Kopie neu aufbauen,
  etwa nach einem Gio-Update).
- Jedes Modul, das Gio nutzt, trägt `replace gioui.org => <relativer Pfad>/extern/gio` in seinem `go.mod` — heute
  `prototyp/`, später das Wurzelmodul (#10). Die `require`-Zeile nennt dieselbe Version wie `GIO_VERSION`.
- Programme bauen mit `-trimpath` und den Tags `novulkan,nox11` (ADR-0003 Nachtrag 2, ADR-0007). Ohne `-trimpath`
  stehen die lokalen Pfade des Bau-Rechners im Programm (Kernprinzip „keine Spuren nach außen“).
- Die CI (`.github/workflows/test.yml`) prüft den Patch (Job `gio-patch`) und baut und testet den Prototyp gegen
  `extern/gio` mit den cgo-Bibliotheken des Runners (Job `prototyp`).

## Konsequenzen

- Änderungen am Patch werden in `extern/gio` gemacht und mit `extern/gio-patch.sh erzeugen` in den Patch übertragen;
  beides gehört in denselben Commit, sonst schlägt die CI fehl.
- Gio-Update: `GIO_VERSION` und die `require`-Zeilen anheben, alte Kopie nach `tmp/papierkorb/` verschieben,
  `extern/gio-patch.sh anlegen` — schlägt das Anwenden fehl, den Patch nachziehen. Dabei prüfen, ob Gio die
  Fähigkeiten inzwischen selbst hat (ADR-0003 Nachtrag 2).
- Lizenz: `extern/gio/LICENSE` (Unlicense OR MIT) bleibt unverändert im Ordner; erzeugter Protokoll-Code trägt seine
  MIT-Kopfzeilen. GPL-3.0-verträglich (ADR-0009).
- Gofmt- und Vet-Läufe des Projekts gelten nicht für `extern/gio` (fremder Code im Stil von Gio).
- Der Prototyp braucht `tmp/gio-gepatcht` und `tmp/prototyp-gepatcht.mod` nicht mehr.

## Nachtrag 1 (19.11.2026): Mauszeiger bei wechselnder Skala (F-78)

Der Patch umfasst zusätzlich das Neuladen des Cursor-Themas bei Skalenwechsel samt Hotspot in der Skala des Themas
und Puffer-Skala 1 für nicht teilbare Bildgrößen (Kritik B-1, B-2, B-13 der Sitzungs-QA). Einzelheiten:
`third_party/README.md`, `knowledge/topics/gio-wayland.md`. Pfade dieses ADR lauten seit ADR-0019 `third_party/…`.

## Nachtrag 2 (20.11.2026): Bilder in die Wayland-Zwischenablage (Karte #39)

Gio v0.10.3 bietet beim Schreiben in die Wayland-Zwischenablage nur Text-MIME-Typen an (`clipboardMimeTypes` in
`app/os_wayland.go`); `clipboard.WriteCmd.Type` wird dort nicht ausgewertet. Andi braucht das fertige Bild in der
Zwischenablage (Echtlauf 20.11.2026).

Entscheidung (Claude Opus 5.5, im Rahmen von Karte #39, die einen Patch ausdrücklich vorsah): Der Patch wertet
`WriteCmd.Type` aus. Ist er ein Bild-Typ (`image/…`), bietet die `wl_data_source` genau diesen Typ an und liefert die
Bytes bei `send`; Text bleibt unverändert. Kein externes Programm (`wl-copy` wäre ein Laufzeit-Fremdpaket, ADR-0011),
kein X11-Weg (ADR-0007). Schnapp legt das gerenderte Bild als PNG ohne Bearbeitungsdaten ab (keine `snAP`/`snAO`-Blöcke,
keine Metadaten, Kernprinzip 3); Geschwärztes ist darin wie in jedem Export endgültig (ADR-0008).

Ergänzung (F-176, F-177): Bei `send` schreibt der Patch die Bytes in einer Schleife (`writeAllFD`) ganz in die Pipe —
ein PNG ist größer als ihr Puffer (64 KiB): `EINTR` wiederholen, `EAGAIN` per `poll` abwarten (nach 10 s ohne
Fortschritt aufgeben), `EPIPE` und andere Fehler beenden still. Weil Empfänger meist eine blockierende Pipe übergeben, schaltet
der Patch den Deskriptor vorher nicht blockierend (F-228; betrifft nur die Dateibeschreibung des Schreib-Endes, der
Leser bleibt blockierend) — sonst griffe die Frist nie. Fehlen dem Compositor Sitz oder Datengerät, meldet der
Patch das über `app.ClipboardWriteError` (upstream hat `clipboard.WriteCmd` keinen Rückgabewert); der Editor zeigt dann
einen Fehler statt „Copied to clipboard“ und schreibt ihn ins Debug-Log.

Konsequenz: Unter Wayland gehört der Inhalt der Zwischenablage dem Fenster, das ihn anbietet. Schließt der Editor, bleibt
das Bild nur erhalten, wenn ein Zwischenablage-Verwalter (unter KDE Klipper) es übernommen hat. Ein Weg, der das Schließen
überdauert (data-control-Protokoll oder ein kurzlebiger Hilfsprozess), ist eine spätere Entscheidung. **Stand 20.11.2026:** Klipper übernimmt das Bild bei Andi nicht; Andi will, dass es das Schließen
überdauert (E-19). **Umgesetzt in ADR-0024:** Strg+C übergibt das Bild an einen Halter-Prozess (data-control); der Weg
über Gio bleibt als Rückfall, wenn der Compositor data-control nicht anbietet.

## Nachtrag 3 (22.11.2026): Rückmeldung bei abgelöster Zwischenablage-Quelle (F-328)

Der Patch ergänzt `app.ClipboardCancelled func()` in `app/os_wayland.go`: Gio ruft es in `gio_onDataSourceCancelled`
auf, wenn die eigene aktuelle `wl_data_source` abgelöst wird (ein anderes Programm hat die Zwischenablage übernommen),
gleiche Art wie `ClipboardWriteError`; aus Gios Wayland-Schleife, vor dem Fenster zu setzen, das kopiert (Schnapp setzt ihn in `Editor()` vor dessen Fenster; die Overlay-Fenster davor kopieren nie, darum ist der Satz „before any window opens“ im Gio-Kommentar für Schnapp gleichwertig). Verwendung: ADR-0024
Nachtrag 8.

## Nachtrag 4 (24.11.2026): Mischen im sRGB-Raum (F-520, Karte #215)

Der Patch stellt die Mischung von Gio von linear auf sRGB um (Farben nicht linearisiert, RGBA8-Texturen, kein sRGB-Ausgabepuffer, keine sRGB-EGL-Fläche), damit der Editor halbdurchsichtige Beschriftungen wie Renderer und Export zeigt. Nur Linux/OpenGL; Begründung, Messung und Widerlegungsbedingung: ADR-0020 Nachtrag 14.
