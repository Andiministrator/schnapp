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

## Der Patch (`prototyp/gio-ausgabe.patch`)

- `app.FullscreenOn(name)` (Feld `Config.Output`) und `app.WaylandOutputs()`; `wl_output` in Version `min(v, 4)`,
  Ereignisse `name`/`description` (braucht wayland-client ≥ 1.20).
- `wp_fractional_scale_v1` + `wp_viewporter`: Puffer-Skala 1, Puffer in echter Pixelgröße
  (`round(logisch × Skala)`), Viewport-Ziel = logische Größe, gesetzt bei jedem `draw`. Zeiger, Metrik und Größen
  über `pxScale()`. Fehlen die Protokolle, bleibt alles wie vorher.
- Protokoll-Code mit `wayland-scanner` aus wayland-protocols 1.45 (XML aus dem öffentlichen Repo, das Paket
  `wayland-protocols` war nicht installiert). Upstream erzeugt Gio solchen Code per `go:generate`.
- Umfang: von Hand +153/−26 Zeilen, erzeugt 811 Zeilen — zählen mit `git apply --numstat prototyp/gio-ausgabe.patch`.
- Bauen: `prototyp/bauen.sh` (gepatchte Kopie unter `tmp/gio-gepatcht` mit Marke aus Version und Prüfsumme,
  `replace` nur in einer erzeugten Modul-Datei unter `tmp/`; `go.mod` bleibt unberührt).

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
