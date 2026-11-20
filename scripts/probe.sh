#!/bin/bash
# Probelauf in einem virtuellen KWin: baut Schnapp mit dem Tag „probe“ (Overlay wählt ohne Eingabe den ersten Monitor)
# und startet es in kwin_wayland --virtual auf einem privaten D-Bus (dbus-run-session). Aufgenommen wird nur der leere
# virtuelle Bildschirm, nie der echte Desktop; HOME und Bibliothek liegen unter tmp/probe/.
#
#   scripts/probe.sh [--monitore N] [--skala S] [-- SCHNAPP-ARGUMENTE]   (Voreinstellung: --no-editor)
#
# Ergebnis: tmp/probe/ergebnis.txt (stdout und stderr getrennt, Exit-Code, abgelegte Dateien). Harte Zeitgrenze 90 s.
set -u
cd "$(dirname "$0")/.." || exit 1
monitore=1
skala=1
while [ $# -gt 0 ]; do
	case "$1" in
	--monitore) monitore="$2"; shift 2 ;;
	--skala) skala="$2"; shift 2 ;;
	--) shift; break ;;
	*) echo "unbekannt: $1" >&2; exit 2 ;;
	esac
done
[ $# -gt 0 ] || set -- --no-editor

if [ "${PROBE_INNEN:-}" = 1 ]; then
	# Innerer Teil: läuft als Sitzung des virtuellen KWin; die Argumente kommen über tmp/probe/argumente.txt.
	mapfile -t argumente < tmp/probe/argumente.txt
	set -- "${argumente[@]}"
	export WAYLAND_DISPLAY=schnapp-probe
	export HOME="$PWD/tmp/probe/home"
	export XDG_CONFIG_HOME="$HOME/.config" XDG_DATA_HOME="$HOME/.local/share" XDG_CACHE_HOME="$HOME/.cache"
	mkdir -p "$HOME/Bilder" "$XDG_CONFIG_HOME"
	printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
	for _ in $(seq 1 50); do [ -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ] && break; sleep 0.1; done
	aus=tmp/probe/ergebnis.txt
	{
		echo "== schnapp $*"
		SCHNAPP_DEBUG=1 tmp/probe/schnapp "$@" > tmp/probe/stdout.txt 2> tmp/probe/stderr.txt
		echo "exit $?"
		echo "-- stdout:"; cat tmp/probe/stdout.txt
		echo "-- stderr:"; cat tmp/probe/stderr.txt
		echo "-- Bibliothek:"; find "$HOME/Bilder" -name '*.png' | sort
	} > "$aus" 2>&1
	exit 0
fi

mkdir -p tmp/probe
GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o tmp/probe/schnapp ./cmd/schnapp || exit 1
printf '%s\n' "$@" > tmp/probe/argumente.txt
env -u WAYLAND_DISPLAY -u DISPLAY PROBE_INNEN=1 timeout 90 dbus-run-session -- \
	kwin_wayland --virtual --socket schnapp-probe --width 1600 --height 1000 --output-count "$monitore" \
	--scale "$skala" --no-lockscreen --no-global-shortcuts \
	--exit-with-session "$PWD/scripts/probe.sh" > tmp/probe/kwin.log 2>&1
code=$?
[ "$code" = 124 ] && echo "Zeitgrenze erreicht (90 s), KWin beendet" >&2
cat tmp/probe/ergebnis.txt
