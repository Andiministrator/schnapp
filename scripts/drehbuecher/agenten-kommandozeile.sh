#!/bin/bash
# Probe ADR-0030 Fassung 1 (Karte #43, Kritik KB-2): Schnittstelle für Skripte und Agenten im virtuellen KWin.
# Aufruf: scripts/probe.sh --zeit 120 --innen scripts/drehbuecher/agenten-kommandozeile.sh   (nie direkt starten)
# Erwartet:
#   - --list-screens --json: Monitorliste als eine JSON-Zeile;
#   - --screen=NAME --output=…png und --screen --output=…svg (mit eigener settings.ini: [capture] screen = pointer)
#     legen die Datei an, melden Pfad und Größe als JSON (beide gleich groß) und lassen die Bibliothek leer;
#   - Positivkontrolle: --screen=NAME --no-editor ohne --output legt genau eine Aufnahme in der Bibliothek ab;
#   - --export=ZIEL DATEI schreibt die Aufnahme ohne Fenster als PNG, mit JSON in derselben Größe;
#   - --export auf die Datei selbst endet mit Exit 1 und lässt sie unverändert (KA-2);
#   - --output ohne Gleichheitszeichen endet mit Exit 2.
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
mkdir -p "$HOME/Bilder" "$HOME/aus" "$XDG_CONFIG_HOME/schnapp"
printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
# Eigene Einstellungen statt eines Rests früherer Läufe (KB-2): --screen ohne Namen nimmt den Monitor unter dem Zeiger,
# ohne Klick; der Editor öffnet nach einer Aufnahme nicht (sonst bliebe die Positivkontrolle offen).
printf '[capture]\nscreen = pointer\nopen-editor = false\n' > "$XDG_CONFIG_HOME/schnapp/settings.ini"
s="$probe/schnapp"
ok=1
fail() { echo "DURCHGEFALLEN: $*"; ok=0; }
anzahl() { find "$HOME/Bilder" -type f -name '*.png' | wc -l; }
# groesse JSON → „BREITE HÖHE“ aus einer Ergebniszeile
groesse() { printf '%s' "$1" | sed -n 's/.*"width":\([0-9]*\),"height":\([0-9]*\)}$/\1 \2/p'; }

liste="$("$s" --list-screens --json)" || fail "--list-screens --json: Exit $?"
echo "Monitorliste: $liste"
name="$(printf '%s' "$liste" | sed -n 's/^{"screens":\[{"name":"\([^"]*\)".*/\1/p')"
[ -n "$name" ] || fail "kein Monitorname in $liste"

vorher="$(anzahl)"
ein="$("$s" --screen="$name" --output="$HOME/aus/bild.png" --json)" || fail "--screen=NAME --output: Exit $?"
echo "Aufnahme: $ein"
case "$ein" in
"{\"path\":\"$HOME/aus/bild.png\",\"width\":"*) ;;
*) fail "JSON der Aufnahme: $ein" ;;
esac
file "$HOME/aus/bild.png" | grep -q 'PNG image data' || fail "bild.png ist kein PNG"
g1="$(groesse "$ein")"
[ -n "$g1" ] || fail "keine Größe in $ein"

zwei="$("$s" --screen --output="$HOME/aus/bild.svg" --json)" || fail "--screen --output=svg: Exit $?"
echo "Aufnahme SVG: $zwei"
case "$zwei" in
"{\"path\":\"$HOME/aus/bild.svg\",\"width\":"*) ;;
*) fail "JSON der SVG-Aufnahme: $zwei" ;;
esac
[ "$(groesse "$zwei")" = "$g1" ] || fail "Größe der SVG-Aufnahme $(groesse "$zwei"), Soll $g1"
head -c 5 "$HOME/aus/bild.svg" | grep -q '<?xml' || fail "bild.svg ist kein SVG"

[ "$(anzahl)" = "$vorher" ] || fail "Bibliothek nach --output verändert: $vorher → $(anzahl) Dateien"

# Positivkontrolle: ohne --output landet die Aufnahme in der Bibliothek (sonst bewiese die Zählung oben nichts).
abgelegt="$("$s" --screen="$name" --no-editor)" || fail "Aufnahme in die Bibliothek: Exit $?"
echo "Bibliothek: $abgelegt"
[ "$(anzahl)" = "$((vorher + 1))" ] || fail "Bibliothek nach der Aufnahme: $(anzahl) Dateien, Soll $((vorher + 1))"
[ -f "$abgelegt" ] || fail "abgelegte Aufnahme fehlt: $abgelegt"

# --export der abgelegten Arbeitsdatei: PNG ohne Fenster, JSON mit Pfad und Größe.
drei="$("$s" --export="$HOME/aus/export.png" --json "$abgelegt")" || fail "--export: Exit $?"
echo "Export: $drei"
case "$drei" in
"{\"path\":\"$HOME/aus/export.png\",\"width\":"*) ;;
*) fail "JSON des Exports: $drei" ;;
esac
[ "$(groesse "$drei")" = "$g1" ] || fail "Größe des Exports $(groesse "$drei"), Soll $g1"
file "$HOME/aus/export.png" | grep -q 'PNG image data' || fail "export.png ist kein PNG"

# --export auf sich selbst: Exit 1, die Arbeitsdatei bleibt Byte für Byte (KA-2).
cp -- "$abgelegt" "$HOME/aus/vorher.png"
"$s" --export="$abgelegt" "$abgelegt" > /dev/null 2> "$HOME/aus/selbst.txt"
code=$?
echo "Export auf sich selbst: Exit $code, $(cat "$HOME/aus/selbst.txt")"
[ "$code" = 1 ] || fail "--export auf sich selbst: Exit $code, Soll 1"
cmp -s -- "$abgelegt" "$HOME/aus/vorher.png" || fail "--export auf sich selbst hat die Arbeitsdatei verändert"

"$s" --output "$HOME/aus/x.png" > /dev/null 2>&1
code=$?
[ "$code" = 2 ] || fail "--output ohne = : Exit $code, Soll 2"

[ "$ok" = 1 ] && echo "BESTANDEN"
