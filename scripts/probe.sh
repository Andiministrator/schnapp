#!/bin/bash
# Probelauf in einem virtuellen KWin: baut Schnapp mit dem Tag „probe“ (Overlay wählt ohne Eingabe den ersten Monitor)
# und startet es in kwin_wayland --virtual auf einem privaten D-Bus (dbus-run-session). Aufgenommen wird nur der leere
# virtuelle Bildschirm, nie der echte Desktop. KWin und Schnapp laufen beide mit HOME, XDG-Ordnern und Laufzeitordner
# unter tmp/probe/ — die echte Konfiguration wird weder gelesen noch beschrieben.
#
#   scripts/probe.sh [--monitore N] [--skala S] [-- SCHNAPP-ARGUMENTE]   (Voreinstellung: --no-editor)
#   scripts/probe.sh --zwischenablage BILD.png
#     prüft den Halter der Zwischenablage (ADR-0024): setzt BILD als image/png, liest es mit wl-paste zurück (nur aus
#     der Zwischenablage des virtuellen KWin), ersetzt sie mit wl-copy und prüft, dass der Halter endet.
#
# Ergebnis: tmp/probe/ergebnis.txt (stdout und stderr getrennt, Exit-Code, in diesem Lauf abgelegte Dateien). Harte
# Zeitgrenze 90 s (danach SIGKILL nach weiteren 5 s). Exit: 0 Lauf durch, 1 Fehler, 124 Zeitgrenze.
set -u
cd "$(dirname "$0")/.." || exit 1
monitore=1
skala=1
ablage=""
while [ $# -gt 0 ]; do
	case "$1" in
	--monitore) monitore="$2"; shift 2 ;;
	--skala) skala="$2"; shift 2 ;;
	--zwischenablage) ablage="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--) shift; break ;;
	*) echo "unbekannt: $1" >&2; exit 2 ;;
	esac
done
[ $# -gt 0 ] || set -- --no-editor

probe="$PWD/tmp/probe"

if [ "${PROBE_INNEN:-}" = 1 ]; then
	# Innerer Teil: läuft als Sitzung des virtuellen KWin. Schutz gegen einen Direktaufruf auf dem echten Sitzungsbus:
	# Der Bus muss ein anderer sein als der des äußeren Aufrufs, und der Socket des virtuellen KWin muss existieren.
	if [ -z "${PROBE_AUSSENBUS+x}" ] || [ "${XDG_RUNTIME_DIR:-}" != "$probe/run" ] ||
		[ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ] || [ "${DBUS_SESSION_BUS_ADDRESS}" = "${PROBE_AUSSENBUS}" ]; then
		echo "probe.sh: innerer Teil nicht auf einem privaten Bus — Abbruch" >&2
		exit 1
	fi
	export WAYLAND_DISPLAY=schnapp-probe
	for _ in $(seq 1 50); do [ -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ] && break; sleep 0.1; done
	if [ ! -S "$XDG_RUNTIME_DIR/$WAYLAND_DISPLAY" ]; then
		echo "probe.sh: Socket des virtuellen KWin fehlt — Abbruch" > "$probe/ergebnis.txt"
		exit 1
	fi
	if [ -s "$probe/ablage.txt" ]; then
		bild="$(cat "$probe/ablage.txt")"
		{
			echo "== Halter der Zwischenablage mit $(basename "$bild") ($(stat -c %s "$bild") Byte)"
			# fd 3 als Pipe (der Halter meldet nur über eine Pipe, ADR-0024)
			setsid "$probe/schnapp" --internal-keep-clipboard < "$bild" 3> >(cat > "$probe/meldung.txt") > /dev/null 2>&1 &
			halter=$!
			for _ in $(seq 1 50); do grep -q . "$probe/meldung.txt" && break; sleep 0.1; done
			echo "meldung: $(cat "$probe/meldung.txt")"
			grep -q '^ok ' "$probe/meldung.txt" || echo "probe.sh: Halter meldet kein ok"
			echo "typen: $(timeout 10 wl-paste --list-types | tr '\n' ' ')"
			timeout 10 wl-paste --type image/png > "$probe/eingefuegt.png"
			cmp -s "$bild" "$probe/eingefuegt.png" && echo "eingefuegt: gleich" || echo "probe.sh: eingefuegt VERSCHIEDEN ($(stat -c %s "$probe/eingefuegt.png") Byte)"
			kill -0 "$halter" 2> /dev/null && echo "halter: läuft" || echo "halter: beendet"
			timeout 10 wl-copy "anderer Inhalt"
			for _ in $(seq 1 50); do kill -0 "$halter" 2> /dev/null || break; sleep 0.1; done
			kill -0 "$halter" 2> /dev/null && echo "probe.sh: nach wl-copy läuft der Halter NOCH" || echo "nach wl-copy: halter beendet"
			echo "jetzt: $(timeout 10 wl-paste --no-newline)"
		} > "$probe/ergebnis.txt" 2>&1
		exit 0
	fi
	mapfile -t argumente < "$probe/argumente.txt"
	set -- "${argumente[@]}"
	mkdir -p "$HOME/Bilder"
	printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
	{
		echo "== schnapp $*"
		SCHNAPP_DEBUG=1 "$probe/schnapp" "$@" > "$probe/stdout.txt" 2> "$probe/stderr.txt"
		echo "exit $?"
		echo "-- stdout:"; cat "$probe/stdout.txt"
		echo "-- stderr:"; cat "$probe/stderr.txt"
		echo "-- in diesem Lauf abgelegt:"; find "$HOME/Bilder" -name '*.png' -newer "$probe/start" | sort
	} > "$probe/ergebnis.txt" 2>&1
	exit 0
fi

mkdir -p "$probe/home/.config" "$probe/home/.local/share" "$probe/home/.cache" "$probe/run"
chmod 0700 "$probe/run"
# Ergebnisse des vorigen Laufs überschreiben, damit nie ein altes Ergebnis als neues erscheint.
: > "$probe/ergebnis.txt"
: > "$probe/stdout.txt"
: > "$probe/stderr.txt"
touch "$probe/start"
GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o "$probe/schnapp" ./cmd/schnapp || exit 1
printf '%s\n' "$@" > "$probe/argumente.txt"
printf '%s' "$ablage" > "$probe/ablage.txt"
env -u WAYLAND_DISPLAY -u DISPLAY PROBE_INNEN=1 PROBE_AUSSENBUS="${DBUS_SESSION_BUS_ADDRESS:-}" \
	HOME="$probe/home" XDG_CONFIG_HOME="$probe/home/.config" XDG_DATA_HOME="$probe/home/.local/share" \
	XDG_CACHE_HOME="$probe/home/.cache" XDG_RUNTIME_DIR="$probe/run" \
	timeout -k 5 90 dbus-run-session -- \
	kwin_wayland --virtual --socket schnapp-probe --width 1600 --height 1000 --output-count "$monitore" \
	--scale "$skala" --no-lockscreen --no-global-shortcuts \
	--exit-with-session "$PWD/scripts/probe.sh" > "$probe/kwin.log" 2>&1
code=$?
if [ ! -s "$probe/ergebnis.txt" ]; then
	echo "probe.sh: kein Ergebnis (KWin-Exit $code, siehe tmp/probe/kwin.log)" >&2
	[ "$code" = 0 ] && code=1
	exit "$code"
fi
[ "$code" = 124 ] && echo "Zeitgrenze erreicht (90 s), KWin beendet" >&2
cat "$probe/ergebnis.txt"
grep -q '^probe.sh:' "$probe/ergebnis.txt" && exit 1
exit "$code"
