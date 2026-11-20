#!/bin/bash
# Probelauf in einem virtuellen KWin: baut Schnapp mit dem Tag „probe“ (Overlay wählt ohne Eingabe den ersten Monitor)
# und startet es in kwin_wayland --virtual auf einem privaten D-Bus (dbus-run-session). Aufgenommen wird nur der leere
# virtuelle Bildschirm, nie der echte Desktop. KWin und Schnapp laufen beide mit HOME, XDG-Ordnern und Laufzeitordner
# unter tmp/probe/ — die echte Konfiguration wird weder gelesen noch beschrieben. Die Tastenbelegung ist fest US
# (XKB_DEFAULT_LAYOUT=us), damit die evdev-Codes der Fernbedienung unabhängig vom Rechner dieselben Tasten treffen.
#
#   scripts/probe.sh [--monitore N] [--skala S] [-- SCHNAPP-ARGUMENTE]   (Voreinstellung: --no-editor)
#   scripts/probe.sh --innen SKRIPT
#     führt SKRIPT (bash) als Sitzung des virtuellen KWin aus statt Schnapp; Ausgabe nach tmp/probe/ergebnis.txt.
#     Das Programm liegt als tmp/probe/schnapp bereit.
#   scripts/probe.sh --drehbuch DATEI [--bild BILD] [--breite W] [--hoehe H] [--skala S] [--monitore N] [--zeit SEK]
#     Fernbedienung (Karte #95): startet den Editor (mit --bild BILD die Datei, sonst eine Aufnahme mit --region, die im
#     Probe-Build den ersten Monitor nimmt) im Vollbild des virtuellen KWin, wartet auf die Zeile „[editor] Fenstergröße“
#     im Debug-Log, spielt dann das Drehbuch mit tmp/probe/fernbedienung ab (Zeiger-/Tasteneingaben über
#     org_kde_kwin_fake_input, nur im virtuellen KWin) und beendet Schnapp danach (nur die eigene PID, SIGTERM, nach 5 s
#     SIGKILL; vorher Startzeit und Programmname wie beim Start). Fotos als PNG in tmp/probe/fotos/<Zeitstempel>/ (je
#     Lauf ein neuer Ordner, nichts wird gelöscht). Größe des virtuellen Bildschirms: --breite/--hoehe (Voreinstellung
#     1600x1000), die Zeitgrenze des Laufs --zeit SEK (Voreinstellung 90, höchstens 600).
#     Drehbuch, eine Zeile je Befehl, # = Kommentar (Beispiel: scripts/drehbuecher/beispiel-schwaerzen.txt):
#       warte MS | zeiger X Y | klick [X Y] | doppelklick X Y | rechtsklick [X Y] | mittelklick [X Y]
#       ziehe X1 Y1 X2 Y2 [SCHRITTE] | ziehe-mitte … | taste KOMBI (B, Esc, Strg+Z, Strg+Umschalt+E, Entf, …)
#       schreibe TEXT | rad X Y SCHRITTE (negativ = hoch) | foto NAME [BILDSCHIRM]
#     Koordinaten sind logische Pixel des virtuellen Bildschirms (Fenster im Vollbild: gleich Fensterkoordinaten).
#     foto nimmt den Bildschirm unter dem Zeiger auf, bei mehreren Monitoren BILDSCHIRM (Name) angeben.
#   scripts/probe.sh --zwischenablage BILD.png
#     prüft den Halter der Zwischenablage (ADR-0024): setzt BILD als image/png, liest es mit wl-paste zurück (nur aus
#     der Zwischenablage des virtuellen KWin), ersetzt sie mit wl-copy und prüft, dass der Halter endet.
#
# Ergebnis: tmp/probe/ergebnis.txt (stdout und stderr getrennt, Exit-Code, in diesem Lauf abgelegte Dateien). Harte
# Zeitgrenze 90 s, bei --drehbuch mit --zeit verlängerbar (danach SIGKILL nach weiteren 5 s). Exit: 0 Lauf durch, 1 Fehler, 124 Zeitgrenze.
set -u
cd "$(dirname "$0")/.." || exit 1
monitore=1
skala=1
ablage=""
innen=""
drehbuch=""
bild=""
breite=1600
hoehe=1000
zeit=90
while [ $# -gt 0 ]; do
	case "$1" in
	--monitore) monitore="$2"; shift 2 ;;
	--skala) skala="$2"; shift 2 ;;
	--innen) innen="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--drehbuch) drehbuch="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--bild) bild="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--breite) breite="$2"; shift 2 ;;
	--hoehe) hoehe="$2"; shift 2 ;;
	--zeit) zeit="$2"; shift 2 ;;
	--zwischenablage) ablage="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--) shift; break ;;
	*) echo "unbekannt: $1" >&2; exit 2 ;;
	esac
done
[ $# -gt 0 ] || set -- --no-editor
case "$breite$hoehe$zeit" in *[!0-9]* | "") echo "probe.sh: --breite, --hoehe und --zeit sind ganze Zahlen" >&2; exit 2 ;; esac
[ "$zeit" -ge 1 ] && [ "$zeit" -le 600 ] || { echo "probe.sh: --zeit liegt zwischen 1 und 600" >&2; exit 2; }

probe="$PWD/tmp/probe"

# Identität eines Prozesses ohne Befehlszeile: Startzeit (Feld 22 aus /proc/PID/stat) und Programmname (comm).
# Ausgabe „STARTZEIT NAME“; leer und Rückgabe 1, wenn nicht lesbar. Das Feld comm steht in Klammern und kann Leerzeichen
# enthalten, deshalb werden die Felder erst hinter der letzten Klammer gezählt.
identitaet() {
	local stat rest name
	stat="$(cat "/proc/$1/stat" 2> /dev/null)" || return 1
	name="$(cat "/proc/$1/comm" 2> /dev/null)" || return 1
	rest="${stat##*) }"
	[ "$rest" != "$stat" ] || return 1
	set -- $rest
	[ -n "${20:-}" ] || return 1
	printf '%s %s\n' "${20}" "$name"
}

# Drehbuch-Lauf (innerer Teil, Sitzung des virtuellen KWin): Schnapp im Hintergrund, Fernbedienung im Vordergrund.
drehbuch_innen() {
	local skript fotos bildpfad pid id0 id1 fb
	skript="$(cat "$probe/drehbuch.txt")"
	fotos="$(cat "$probe/fotos.txt")"
	bildpfad="$(cat "$probe/bild.txt")"
	mkdir -p "$HOME/Bilder" "$fotos"
	printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
	: > "$probe/stdout.txt"
	: > "$probe/stderr.txt"
	if [ -n "$bildpfad" ]; then
		SCHNAPP_DEBUG=1 "$probe/schnapp" "$bildpfad" > "$probe/stdout.txt" 2> "$probe/stderr.txt" &
	else
		SCHNAPP_DEBUG=1 "$probe/schnapp" --region > "$probe/stdout.txt" 2> "$probe/stderr.txt" &
	fi
	pid=$!
	# Fenster steht, wenn der Editor die erste Größe meldet (Zeile „[editor] Fenstergröße“); danach 1 s Ruhe, weil
	# das Vollbild erst nach dem ersten Bild greift und die Größe dann noch einmal wechseln kann.
	for _ in $(seq 1 300); do
		grep -q 'Fenstergröße' "$probe/stderr.txt" && break
		kill -0 "$pid" 2> /dev/null || break
		sleep 0.1
	done
	sleep 1
	# Identität jetzt festhalten (nach dem exec ist der Programmname stabil); ist sie unlesbar, greift das Skript nie ein.
	id0="$(identitaet "$pid")" || id0=""
	{
		echo "== Drehbuch $(basename "$skript"), Schnapp-PID $pid, Identität beim Start: ${id0:-unlesbar}"
		FOTOS="$fotos" "$probe/fernbedienung" "$skript" 2>&1
		fb=$?
		echo "fernbedienung exit $fb"
		for _ in $(seq 1 100); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
		if kill -0 "$pid" 2> /dev/null; then
			id1="$(identitaet "$pid")" || id1=""
			if [ -n "$id0" ] && [ "$id0" = "$id1" ]; then
				echo "schnapp läuft nach dem Drehbuch noch: SIGTERM an die eigene PID $pid"
				kill -TERM "$pid" 2> /dev/null
				for _ in $(seq 1 50); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
				if kill -0 "$pid" 2> /dev/null; then
					id1="$(identitaet "$pid")" || id1=""
					if [ -n "$id0" ] && [ "$id0" = "$id1" ]; then
						echo "schnapp reagiert nicht auf SIGTERM: SIGKILL an PID $pid"
						kill -KILL "$pid" 2> /dev/null
					else
						echo "probe.sh: Identität der PID $pid hat sich geändert — kein SIGKILL"
					fi
				fi
			else
				echo "probe.sh: Identität der PID $pid nicht bestätigt (vorher '${id0:-unlesbar}', jetzt '${id1:-unlesbar}') — nicht eingegriffen"
			fi
		else
			echo "schnapp hat sich selbst beendet"
		fi
		wait "$pid"
		echo "schnapp exit $?"
		echo "-- stdout von schnapp:"; cat "$probe/stdout.txt"
		echo "-- stderr von schnapp:"; cat "$probe/stderr.txt"
		echo "-- Fotos in $fotos:"; find "$fotos" -name '*.png' | sort
		echo "-- in diesem Lauf abgelegt:"; find "$HOME/Bilder" -name '*.png' -newer "$probe/start" | sort
	} > "$probe/ergebnis.txt" 2>&1
}

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
	if [ -s "$probe/innen.txt" ]; then
		bash "$(cat "$probe/innen.txt")" > "$probe/ergebnis.txt" 2>&1
		echo "exit $?" >> "$probe/ergebnis.txt"
		exit 0
	fi
	if [ -s "$probe/drehbuch.txt" ]; then
		drehbuch_innen
		exit 0
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
printf '%s' "$innen" > "$probe/innen.txt"
printf '%s' "$drehbuch" > "$probe/drehbuch.txt"
printf '%s' "$bild" > "$probe/bild.txt"
if [ -n "$drehbuch" ]; then
	fotos="$probe/fotos/$(date +%Y%m%d-%H%M%S)"
	printf '%s' "$fotos" > "$probe/fotos.txt"
	GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o "$probe/fernbedienung" ./cmd/schnapp-fernbedienung || exit 1
fi
env -u WAYLAND_DISPLAY -u DISPLAY PROBE_INNEN=1 PROBE_AUSSENBUS="${DBUS_SESSION_BUS_ADDRESS:-}" \
	HOME="$probe/home" XDG_CONFIG_HOME="$probe/home/.config" XDG_DATA_HOME="$probe/home/.local/share" \
	XDG_CACHE_HOME="$probe/home/.cache" XDG_RUNTIME_DIR="$probe/run" \
	XKB_DEFAULT_LAYOUT=us XKB_DEFAULT_VARIANT= XKB_DEFAULT_MODEL= XKB_DEFAULT_OPTIONS= \
	timeout -k 5 "$zeit" dbus-run-session -- \
	kwin_wayland --virtual --socket schnapp-probe --width "$breite" --height "$hoehe" --output-count "$monitore" \
	--scale "$skala" --no-lockscreen --no-global-shortcuts \
	--exit-with-session "$PWD/scripts/probe.sh" > "$probe/kwin.log" 2>&1
code=$?
if [ ! -s "$probe/ergebnis.txt" ]; then
	echo "probe.sh: kein Ergebnis (KWin-Exit $code, siehe tmp/probe/kwin.log)" >&2
	[ "$code" = 0 ] && code=1
	exit "$code"
fi
[ "$code" = 124 ] && echo "Zeitgrenze erreicht ($zeit s), KWin beendet" >&2
cat "$probe/ergebnis.txt"
grep -q '^probe.sh:' "$probe/ergebnis.txt" && exit 1
exit "$code"
