#!/bin/bash
# Probe ADR-0030 Fassung 1 (Karte #43): --list-screens --json und Aufnahmen mit --output/--json im virtuellen KWin.
# Aufruf: scripts/probe.sh --innen scripts/drehbuecher/agenten-kommandozeile.sh   (nie direkt starten)
# Erwartet: Monitorliste als eine JSON-Zeile; --screen=NAME --output=…png und --screen --output=…svg legen die Datei an,
# melden Pfad und Größe als JSON und lassen die Bibliothek leer; --output ohne Gleichheitszeichen endet mit Exit 2.
# Das Skript prüft das selbst (BESTANDEN/DURCHGEFALLEN). Es startet nur Schnapp-Aufrufe, die von selbst enden.
set -u
wurzel="$PWD" # probe.sh läuft in der Wurzel des Repos
[ -x "$wurzel/scripts/probe.sh" ] || { echo "probe.sh: nicht in der Wurzel des Repos ($wurzel)"; exit 1; }
probe="$wurzel/tmp/probe"
[ "${PROBE_INNEN:-}" = 1 ] || { echo "probe.sh: nur über scripts/probe.sh --innen aufrufen"; exit 1; }
for ordner in "${HOME:-}" "${XDG_CONFIG_HOME:-}" "${XDG_STATE_HOME:-}"; do
	case "$ordner" in
	"$probe/"*) ;;
	*) echo "probe.sh: Ordner außerhalb von tmp/probe ($ordner), Abbruch"; exit 1 ;;
	esac
done
mkdir -p "$HOME/Bilder" "$HOME/aus"
printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
s="$probe/schnapp"
ok=1
fail() { echo "DURCHGEFALLEN: $*"; ok=0; }

liste="$("$s" --list-screens --json)" || fail "--list-screens --json: Exit $?"
echo "Monitorliste: $liste"
name="$(printf '%s' "$liste" | sed -n 's/^{"screens":\[{"name":"\([^"]*\)".*/\1/p')"
[ -n "$name" ] || fail "kein Monitorname in $liste"

vorher="$(find "$HOME/Bilder" -type f | wc -l)"
ein="$("$s" --screen="$name" --output="$HOME/aus/bild.png" --json)" || fail "--screen=NAME --output: Exit $?"
echo "Aufnahme: $ein"
case "$ein" in
"{\"path\":\"$HOME/aus/bild.png\",\"width\":"*) ;;
*) fail "JSON der Aufnahme: $ein" ;;
esac
file "$HOME/aus/bild.png" | grep -q 'PNG image data' || fail "bild.png ist kein PNG"

zwei="$("$s" --screen --output="$HOME/aus/bild.svg" --json)" || fail "--screen --output=svg: Exit $?"
echo "Aufnahme SVG: $zwei"
head -c 5 "$HOME/aus/bild.svg" | grep -q '<?xml' || fail "bild.svg ist kein SVG"

nachher="$(find "$HOME/Bilder" -type f | wc -l)"
[ "$vorher" = "$nachher" ] || fail "Bibliothek verändert: $vorher → $nachher Dateien"

"$s" --output "$HOME/aus/x.png" > /dev/null 2>&1
code=$?
[ "$code" = 2 ] || fail "--output ohne = : Exit $code, Soll 2"

[ "$ok" = 1 ] && echo "BESTANDEN"
