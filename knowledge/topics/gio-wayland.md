# Gio unter Wayland (KDE): Grenzen, Patch, Messen

Stand: 2026-11-19 — aus dem Prototyp (`prototyp/`, ADR-0003 Nachtrag 2), Gio v0.10.3, Plasma 6.8. Quelle ist der
Gio-Quelltext im Go-Modulcache (`go list -m -f '{{.Dir}}' gioui.org`) und eigene Messung.

## Was Gio unter Wayland nicht kann (v0.10.3)

- **Kein Monitor wählbar:** `app.Fullscreen` ruft `xdg_toplevel_set_fullscreen(…, nil)` auf — der Compositor
  entscheidet. `wl_output` wird nur in Version 2 gebunden, also ohne Namen. Ein Overlay je Monitor ist so nicht
  machbar.
- **Keine gebrochene Skalierung:** Gio rechnet mit ganzzahliger Skala (`w.scale int`, rund 25 Stellen in
  `app/os_wayland.go`). Bei Skala 1,6 rendert es in 2, KWin rechnet herunter → unscharf, nicht pixelgenau.
- **Jedes Fenster hat eine eigene Wayland-Verbindung** (`newWLWindow` → `newWLDisplay`); Ausgaben vor dem ersten
  Fenster aufzulisten heißt: eine kurze eigene Verbindung öffnen.

## Der Patch (`extern/gio-ausgabe.patch`, angewendet in `extern/gio/`)

- `app.FullscreenOn(name)` (Feld `Config.Output`) und `app.WaylandOutputs()`; `wl_output` in Version `min(v, 4)`,
  Ereignisse `name`/`description` (braucht wayland-client ≥ 1.20).
- `wp_fractional_scale_v1` + `wp_viewporter`: Puffer-Skala 1, Puffer in echter Pixelgröße
  (`round(logisch × Skala)`), Viewport-Ziel = logische Größe, gesetzt bei jedem `draw`. Zeiger, Metrik und Größen
  über `pxScale()`. Fehlen die Protokolle, bleibt alles wie vorher.
- Protokoll-Code mit `wayland-scanner` aus wayland-protocols 1.45 (XML aus dem öffentlichen Repo, das Paket
  `wayland-protocols` war nicht installiert). Upstream erzeugt Gio solchen Code per `go:generate`.
- `WaylandOutputs()` liefert nur benannte Ausgaben, stabil nach Name sortiert (`sortedOutputs`, Test
  `os_wayland_test.go`). `FullscreenOn` mit unbekanntem Namen schreibt `app: Wayland output "X" not found, the
  compositor chooses the output` ins Log und überlässt die Wahl dem Compositor; ein leerer Name bleibt ohne Meldung
  (F-33, 19.11.2026).
- Umfang (Stand 19.11.2026, nach F-33): von Hand 5 Dateien +203/−26 Zeilen (einschließlich 30 Zeilen Test), erzeugt
  4 Dateien +811 Zeilen — zählen mit `git apply --numstat extern/gio-ausgabe.patch`.
- Ablage (ADR-0015, seit 19.11.2026): gepatchte Kopie als Modul `extern/gio/` im Repo, `replace gioui.org =>
  ../extern/gio` im `go.mod` jedes Nutzers; `extern/gio-patch.sh pruefen|erzeugen|anlegen` hält Kopie und Patch
  zusammen, die CI prüft beides. Vorher (Sprint 1) lag die Kopie gitignoriert unter `tmp/` — frische Klone und die CI
  bauten nicht.
- **Falle `-o` nach `cd`:** ein Bauskript, das erst in sein Verzeichnis wechselt, muss den Zielpfad vorher absolut
  machen (`realpath -m`), sonst landet das Programm unter `prototyp/tmp/…` und man prüft die alte Datei.
- **`-trimpath`** gehört in jeden Bau: ohne stehen Modulcache- und Projektpfade des Rechners im Programm.

## Bauen und Abhängigkeiten

- Gio braucht unter Linux **cgo**: libwayland-client, libwayland-egl, libwayland-cursor, libEGL, libxkbcommon.
- Build-Tags: `novulkan` (sonst werden `vulkan/vulkan.h`-Header verlangt), `nox11` (nur Wayland, ADR-0007; spart die
  X11-Bibliotheken).
- `ldd` auf die Binärdatei zeigt, was wirklich gelinkt ist.

## Eingabe-API (v0.10)

- Ereignisse holen mit `gtx.Event(filter…)` in jedem Frame; registriert wird mit `event.Op(gtx.Ops, tag)` unter einem
  `clip`-Bereich — Ereignisse kommen aus der Registrierung des **vorigen** Frames.
- Tasten: `key.Filter{Name: key.NameEscape}`, Kürzel `key.Filter{Name: "Z", Required: key.ModShortcut}` (Strg unter
  Linux). Buchstaben heißen groß („Z“). Ohne `Focus` greift der Filter fensterweit.
- Zeiger: `pointer.Filter{Target, Kinds, ScrollY: pointer.ScrollRange{…}}` — ohne ScrollRange kommt kein Mausrad an.
  Positionen in Pixeln. Ein Mausrad-Rasterschritt ergab im Handtest 2 genau 25 % Zoom bei Faktor 1,0015 je Einheit,
  also ≈ 150 Einheiten je Raster (1,0015¹⁵⁰ ≈ 1,25) — Maus und Treiber-abhängig.
- Bild: `paint.NewImageOp` einmal anlegen und wiederverwenden (Textur bleibt auf der GPU); `Filter = FilterNearest`
  beim Hineinzoomen.

## Messen

- Gio zeichnet im Takt der Wayland-Frame-Callbacks: Die Bildrate im Dauerlauf entspricht der Bildwiederholrate des
  Monitors (Messmodus `-messen`: 73,5 fps).
- **Bildrate beim Ziehen von Hand täuscht:** Gio zeichnet nur bei Ereignissen, Mauspausen zählen als lange Frames.
  Aussagekräftig ist der **Stau**: Zieh-Ereignisse je Frame. 1–2 sind normales Zusammenfassen (Handtest 3: 289
  Ereignisse in 3,07 s ≈ 94 je Sekunde, schneller als der Bildtakt); einzelne Frames mit 3–4 Ereignissen heißen: ein
  Frame kam einen Takt zu spät — bei ≤ 10 % der Frames nicht sichtbar. Dauerhaft mehr wäre Ruckeln.
  Quelle der Zahlen: Ausgaben der Handtests 1–3 vom 19.11.2026 (Protokolle nicht versioniert, ADR-0003 Nachtrag 2).
- Startzeit ab `main` bis zum ersten übergebenen Frame; das Laden vor `main` und die Darstellung durch KWin fehlen.

## Fallen

- Overlay-Testläufe öffnen Vollbildfenster auf allen Monitoren — vorher ankündigen.
- Der erste Frame kommt oft noch in der Skala, die Gio vorab aus allen Ausgaben schätzt (ungepatcht: die größte der von
  allen Ausgaben gemeldeten Ganzzahl-Skalen — hier 2, weil 1,6 als 2 gemeldet wird; Handtest 1: 7680×3200 px für einen 3840×1600-Monitor mit Skala 1); erst die Konfiguration danach
  stimmt. Abbildungen deshalb bei jeder Größenänderung neu berechnen.
- **Mehrere Fenster, je eine Goroutine** (Overlay, Karte #34, erster echter Lauf 19.11.2026):
  - `text.Shaper` ist nicht nebenläufig nutzbar. Ein gemeinsames `material.Theme` für alle Fenster stürzt im
    Text-Setzer ab (`bidi … slice bounds out of range`). Je Fenster ein eigenes Thema samt Shaper.
  - `Window.Perform(system.ActionClose)` aus einer **fremden** Goroutine läuft in v0.10.3 in einen Datenwettlauf
    (`processEvent` gegen `nextEvent`, `go build -race`), danach kam kein `DestroyEvent` mehr — das Programm hing.
    Unter Wayland führt Gio `Perform` aber im aufrufenden Goroutine aus (`window.Run` ruft `f()` direkt): Jedes Fenster schließt sich deshalb **selbst** in seiner eigenen Ereignisschleife (Flag setzen, `Invalidate`, im nächsten `FrameEvent` `Perform(ActionClose)`) — wettlauffrei, gemessen mit `-race` (F-77 behoben).
  - Echte Läufe mit `go build -race` bauen und fahren: Der Detector fand den Wettlauf im ersten Versuch.
  - **Zeigerbild versetzt bei gemischter Skala (F-78, im Patch behoben 19.11.2026):** Ungepatcht lädt Gio das
    Cursor-Thema einmal in der vorab geschätzten Skala (größte Monitorskala) und teilt den Hotspot später durch die
    aktuelle. Der Patch merkt sich die Skala des Themas, lädt es bei Skalenwechsel neu (`reloadCursorTheme`, auch
    sofort aus `updateOutputs`) und teilt den Hotspot durch die Skala des Themas. Das Overlay blendet den Zeiger aus
    (`pointer.CursorNone`) und zeichnet ein eigenes Fadenkreuz; nach `pointer.Leave` neu zeichnen lassen
    (`op.InvalidateCmd`), sonst bleibt es am Rand des alten Monitors stehen.
  - **Kein `pointer.Leave` beim Verlassen des Fensters:** `gio_onPointerLeave` schickt nur bei Compositor-Zügen ein
    `Cancel`, sonst nichts. Wer wissen muss, dass der Zeiger auf ein anderes Fenster gewechselt ist, koordiniert das
    selbst (Overlay: gemeinsamer „aktiver Monitor“, das alte Fenster per `Window.Invalidate` neu zeichnen lassen —
    `Invalidate` ist aus jeder Goroutine erlaubt, `Perform` nicht).
  - **`Invalidate` verpufft während eines Frames:** `Window.Invalidate` tut nichts, solange das Fenster gerade ein
    Ereignis bearbeitet (`mayInvalidate` false, `app/window.go`). Wer im eigenen Frame etwas auslöst, das einen
    weiteren Frame braucht, fordert ihn mit `gtx.Execute(op.InvalidateCmd{})` an (Overlay: Schließen des
    auslösenden Fensters, F-79, Kritiker Sitzungs-QA Sprint 4).
  - **Tasten kommen im Fenster mit dem Tastaturfokus an,** nicht in dem unter dem Zeiger. Bei mehreren Vollbild-
    Fenstern bezieht sich Enter deshalb auf den gemeinsamen „aktiven Monitor“ (F-80); Rechtsklick bricht ab, falls
    kein Fenster den Fokus hat (F-81).
  - Diagnose bei echten Läufen: `SCHNAPP_DEBUG=1` schreibt Fenstergröße, PxPerDp, Abbildung, Druck/Loslassen je
    Monitor auf stderr (ohne Monitornamen). Gemessen 19.11.2026: auf allen drei Monitoren Fenster = Bild, Zoom 1,
    auch bei Skala 1,6 (PxPerDp 1,600).
  - Gemessen 19.11.2026 (ohne `-race`): drei Monitore aufnehmen 71 ms, Bereich speichern (Arbeitsdatei, Vorschau,
    Index) 56 ms. Mit `-race` 1,2 s bzw. 1,8 s — Zeiten nie mit dem Race-Build messen.
