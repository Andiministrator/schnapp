# ADR-0011: Go-Zusatzpakete nur mit spürbarem Nutzen

## Status
Akzeptiert (Andi, 18.11.2026; übernommen am 19.11.2026)

## Kontext
Jedes Fremdpaket bringt Pflege, Lizenzfragen und Angriffsfläche. Manches (PNG-Chunks, Bézier-Mathematik) ist schnell
selbst gebaut, anderes (Schriftrasterung, PDF, D-Bus) nicht.

## Entscheidung
Go-Zusatzpakete sind erlaubt, wenn sie **spürbar Arbeit oder Fehler sparen**; sonst nachbauen. Jedes neue Paket wird
im Commit begründet (wofür, Lizenz, Alternative).

## Konsequenzen
- Erwartete Kandidaten: `gioui.org` (ADR-0003), `golang.org/x/image` (Schrift), ein D-Bus-Paket
  (`github.com/godbus/dbus`) für KWin/Portal, ggf. ein PDF-Paket. Lizenz jeweils GPL-3.0-verträglich (ADR-0009).
- `cgo` nur mit eigener ADR (Flatpak, Kreuzbau).
