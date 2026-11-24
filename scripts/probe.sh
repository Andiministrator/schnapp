#!/bin/bash
# Probelauf in einem virtuellen KWin: baut Schnapp mit dem Tag „probe“ (Overlay wählt ohne Eingabe den ersten Monitor)
# und startet es in kwin_wayland --virtual auf einem privaten D-Bus (dbus-run-session). Aufgenommen wird nur der leere
# virtuelle Bildschirm, nie der echte Desktop. KWin und Schnapp laufen beide mit HOME, XDG-Ordnern und Laufzeitordner
# unter tmp/probe/ (auch XDG_STATE_HOME) — die echte Konfiguration wird weder gelesen noch beschrieben. Die Tastenbelegung ist fest US
# (XKB_DEFAULT_LAYOUT=us), damit die evdev-Codes der Fernbedienung unabhängig vom Rechner dieselben Tasten treffen.
#
#   scripts/probe.sh [--monitore N] [--skala S] [-- SCHNAPP-ARGUMENTE]   (Voreinstellung: --no-editor)
#   scripts/probe.sh --innen SKRIPT
#     führt SKRIPT (bash) als Sitzung des virtuellen KWin aus statt Schnapp; Ausgabe nach tmp/probe/ergebnis.txt.
#     Das Programm liegt als tmp/probe/schnapp bereit.
#   scripts/probe.sh --drehbuch DATEI [--bild BILD] [--breite W] [--hoehe H] [--skala S] [--monitore N] [--zeit SEK] [--ende SEK]
#     Fernbedienung (Karte #95): startet den Editor (mit --bild BILD die Datei, sonst eine Aufnahme mit --region, die im
#     Probe-Build den ersten Monitor nimmt) im Vollbild des virtuellen KWin, wartet auf die Zeile „[editor] Fenstergröße“
#     im Debug-Log, spielt dann das Drehbuch mit tmp/probe/fernbedienung ab (Zeiger-/Tasteneingaben über
#     org_kde_kwin_fake_input, nur im virtuellen KWin) und beendet Schnapp danach (nur die eigene PID, SIGTERM, nach
#     --ende SEK (Voreinstellung 5, 1 bis 300) SIGKILL; vorher Startzeit und Programmname wie beim Start, der Name muss
#     „schnapp“ sein; ist die Identität nicht prüfbar, greift das Skript nicht ein und meldet es im Ergebnis). Die Frist
#     gibt Schnapp Zeit zum Speichern im Probe-Build (langsam, mit -race, F-296). Fotos als PNG in tmp/probe/fotos/<Zeitstempel>/ (je
#     Lauf ein neuer Ordner, nichts wird gelöscht). Größe des virtuellen Bildschirms: --breite/--hoehe (Voreinstellung
#     1600x1000), die Zeitgrenze des Laufs --zeit SEK (Voreinstellung 90, höchstens 600). Die äußere Grenze eines
#     Drehbuch-Laufs ist --zeit + --ende + 10 s, damit eine lange Frist nicht von ihr abgeschnitten wird. --ende wirkt nur
#     bei --drehbuch.
#     Drehbuch, eine Zeile je Befehl, # = Kommentar (Beispiel: scripts/drehbuecher/beispiel-schwaerzen.txt):
#       warte MS | zeiger X Y | klick [X Y] | doppelklick X Y | rechtsklick [X Y] | mittelklick [X Y]
#       ziehe X1 Y1 X2 Y2 [SCHRITTE] | ziehe-mitte … | taste KOMBI (B, Esc, Strg+Z, Strg+Umschalt+E, Entf, …)
#       schreibe TEXT | rad X Y SCHRITTE (negativ = hoch) | foto NAME [BILDSCHIRM]
#       Halten (F-599): druecke[-rechts|-mitte] [X Y] | bewege X Y [SCHRITTE] | loslassen | taste-halten KOMBI | taste-loslassen KOMBI
#       (Zug mit Fotos mittendrin: scripts/drehbuecher/einrasten-196-halten.txt; am Ende wird alles Gehaltene gelöst)
#       Buchführung (F-614): klick/doppelklick/ziehe verweigern sich mit Fehler (Exit 1), solange DIESELBE Maustaste von
#       druecke gehalten wird, und taste/schreibe, solange eine Taste der Kombination von taste-halten gehalten wird — sie
#       würden sie sonst still lösen. Erst loslassen/taste-loslassen; andere Tasten (z. B. rechtsklick bei gehaltener linker)
#       gehen. Nach einem Fehler oder am Ende wird alles Gehaltene trotzdem gelöst.
#     Koordinaten sind logische Pixel des virtuellen Bildschirms (Fenster im Vollbild: gleich Fensterkoordinaten).
#     foto nimmt den Bildschirm unter dem Zeiger auf, bei mehreren Monitoren BILDSCHIRM (Name) angeben.
#   scripts/probe.sh --zwischenablage BILD.png   (--innen, --drehbuch und --zwischenablage schließen sich aus)
#     prüft den Halter der Zwischenablage (ADR-0024): setzt BILD als image/png, liest es mit wl-paste zurück (nur aus
#     der Zwischenablage des virtuellen KWin), ersetzt sie mit wl-copy und prüft, dass der Halter endet.
#     Der Halter wird nur beobachtet, nie beendet: Startzeit und Programmname (comm) der PID werden beim Start festgehalten
#     und bei jeder Beobachtung verglichen; weicht die Identität ab oder ist sie nicht prüfbar, steht „unklar“ im Ergebnis.
#
# Ergebnis: tmp/probe/ergebnis.txt (stdout und stderr getrennt, Exit-Code, in diesem Lauf abgelegte Dateien). Harte
# Zeitgrenze 90 s, bei --drehbuch mit --zeit verlängerbar und um --ende + 10 s erweitert (danach SIGKILL nach weiteren 5 s).
# Exit: 0 Lauf durch, 1 Fehler, 124 Zeitgrenze.
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
ende=5
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
	--ende) ende="$2"; shift 2 ;;
	--zwischenablage) ablage="$(realpath -e -- "$2")" || exit 2; shift 2 ;;
	--) shift; break ;;
	*) echo "unbekannt: $1" >&2; exit 2 ;;
	esac
done
[ $# -gt 0 ] || set -- --no-editor
# --innen, --drehbuch und --zwischenablage sind verschiedene Betriebsarten und schließen sich gegenseitig aus.
arten=0
for a in "$innen" "$drehbuch" "$ablage"; do [ -z "$a" ] || arten=$((arten + 1)); done
[ "$arten" -le 1 ] || { echo "probe.sh: --innen, --drehbuch und --zwischenablage schließen sich gegenseitig aus" >&2; exit 2; }
case "$breite$hoehe$zeit$monitore" in *[!0-9]* | "") echo "probe.sh: --breite, --hoehe, --zeit und --monitore sind ganze Zahlen" >&2; exit 2 ;; esac
case "$skala" in *[!0-9.]* | "" | . | .* | *. | *.*.*) echo "probe.sh: --skala ist eine Zahl (etwa 1 oder 1.5)" >&2; exit 2 ;; esac
[ "$monitore" -ge 1 ] && [ "$breite" -ge 1 ] && [ "$hoehe" -ge 1 ] || { echo "probe.sh: --monitore, --breite und --hoehe sind mindestens 1" >&2; exit 2; }
[ "$zeit" -ge 1 ] && [ "$zeit" -le 600 ] || { echo "probe.sh: --zeit liegt zwischen 1 und 600" >&2; exit 2; }
case "$ende" in *[!0-9]* | "") echo "probe.sh: --ende ist eine ganze Zahl (1 bis 300)" >&2; exit 2 ;; esac
[ "$ende" -ge 1 ] && [ "$ende" -le 300 ] || { echo "probe.sh: --ende liegt zwischen 1 und 300" >&2; exit 2; }
# Dezimal lesen: bash rechnet in $(( )) eine führende Null oktal („08“ ist ungültig, „010“ ist 8; Kritik Sprint 9 K9-1).
breite=$((10#$breite)) hoehe=$((10#$hoehe)) zeit=$((10#$zeit)) monitore=$((10#$monitore)) ende=$((10#$ende))

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

# Wahr, wenn die PID noch dieselbe Instanz ist wie beim Start: gleiche Startzeit und gleicher Name wie in $2 (Ausgabe von
# identitaet beim Start). Leeres $2, unlesbar oder abweichend heißt: nein.
gleiche_identitaet() {
	local jetzt
	[ -n "${2:-}" ] || return 1
	jetzt="$(identitaet "$1")" || return 1
	[ "$jetzt" = "$2" ]
}

# Wahr, wenn die PID noch dieselbe Schnapp-Instanz ist wie beim Start (id0, gesetzt in drehbuch_innen) und der Name
# wörtlich „schnapp“ ist. Unlesbar oder abweichend heißt: nicht eingreifen.
ist_schnapp() {
	gleiche_identitaet "$1" "${id0:-}" && [ "${id0#* }" = schnapp ]
}

# Zustand eines beobachteten Prozesses ($1 PID, $2 Identität beim Start): „läuft“ (gleiche Startzeit und gleicher Name),
# „beendet“ (PID gibt es nicht mehr) oder „unklar“ (PID lebt, aber Identität fehlt oder weicht ab — PID wiederverwendet
# oder nicht prüfbar). Reine Beobachtung, kein Eingriff (F-208).
zustand() {
	if gleiche_identitaet "$1" "$2"; then
		echo "läuft"
	elif ! kill -0 "$1" 2> /dev/null; then
		echo "beendet"
	else
		echo "unklar"
	fi
}

# Drehbuch-Lauf (innerer Teil, Sitzung des virtuellen KWin): Schnapp im Hintergrund, Fernbedienung im Vordergrund.
drehbuch_innen() {
	local skript fotos bildpfad ende pid id0 id1 fb
	skript="$(cat "$probe/drehbuch.txt")"
	fotos="$(cat "$probe/fotos.txt")"
	bildpfad="$(cat "$probe/bild.txt")"
	ende="$(cat "$probe/ende.txt")"
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
	# Identität sofort festhalten, bevor die PID wiederverwendet sein kann: bis zu 2 s warten, bis der Prozess nach dem
	# exec „schnapp“ heißt (davor trägt er den Namen der Shell), dann Startzeit und Namen merken. Gelingt das nicht,
	# greift das Skript später nie ein.
	id0=""
	for _ in $(seq 1 20); do
		if id1="$(identitaet "$pid")" && [ "${id1#* }" = schnapp ]; then id0="$id1"; break; fi
		kill -0 "$pid" 2> /dev/null || break
		sleep 0.1
	done
	# Fenster steht, wenn der Editor die erste Größe meldet (Zeile „[editor] Fenstergröße“); danach 1 s Ruhe, weil
	# das Vollbild erst nach dem ersten Bild greift und die Größe dann noch einmal wechseln kann.
	for _ in $(seq 1 300); do
		grep -q 'Fenstergröße' "$probe/stderr.txt" && break
		kill -0 "$pid" 2> /dev/null || break
		sleep 0.1
	done
	sleep 1
	{
		echo "== Drehbuch $(basename "$skript"), Schnapp-PID $pid, Identität beim Start: ${id0:-unlesbar}"
		FOTOS="$fotos" "$probe/fernbedienung" "$skript" 2>&1
		fb=$?
		echo "fernbedienung exit $fb"
		for _ in $(seq 1 100); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
		if kill -0 "$pid" 2> /dev/null; then
			if ist_schnapp "$pid"; then
				echo "schnapp läuft nach dem Drehbuch noch: SIGTERM an die eigene PID $pid"
				kill -TERM "$pid" 2> /dev/null
				for _ in $(seq 1 $((ende * 10))); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
				if kill -0 "$pid" 2> /dev/null; then
					if ist_schnapp "$pid"; then
						echo "schnapp reagiert nach $ende s nicht auf SIGTERM: SIGKILL an PID $pid"
						kill -KILL "$pid" 2> /dev/null
						for _ in $(seq 1 20); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
					else
						echo "probe.sh: Identität der PID $pid hat sich geändert — kein SIGKILL"
					fi
				fi
			else
				echo "probe.sh: Identität nicht prüfbar, Schnapp läuft weiter (beim Start '${id0:-unlesbar}', jetzt '$(identitaet "$pid" || echo unlesbar)') — nicht eingegriffen"
			fi
		else
			echo "schnapp hat sich selbst beendet"
		fi
		# Nur auf ein beendetes Schnapp warten: ein weiterlaufendes würde wait bis zur äußeren Zeitgrenze blockieren
		# (der äußere timeout beendet KWin samt Sitzung).
		if kill -0 "$pid" 2> /dev/null; then
			echo "probe.sh: schnapp läuft noch — kein Exit-Code; die äußere Zeitgrenze beendet KWin samt Sitzung"
		else
			wait "$pid"
			echo "schnapp exit $?"
		fi
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
			# Identität sofort festhalten (wie bei --drehbuch): setsid ersetzt sich per exec durch Schnapp, bis dahin trägt die
			# PID einen anderen Namen; bis zu 2 s warten, bis der Name „schnapp“ ist. Gelingt das nicht, meldet der Lauf
			# „unklar“ statt „läuft“/„beendet“ (F-208).
			idh=""
			for _ in $(seq 1 20); do
				if idx="$(identitaet "$halter")" && [ "${idx#* }" = schnapp ]; then idh="$idx"; break; fi
				kill -0 "$halter" 2> /dev/null || break
				sleep 0.1
			done
			echo "halter: PID $halter, Identität beim Start: ${idh:-unlesbar}"
			for _ in $(seq 1 50); do grep -q . "$probe/meldung.txt" && break; sleep 0.1; done
			echo "meldung: $(cat "$probe/meldung.txt")"
			grep -q '^ok ' "$probe/meldung.txt" || echo "probe.sh: Halter meldet kein ok"
			echo "typen: $(timeout 10 wl-paste --list-types | tr '\n' ' ')"
			timeout 10 wl-paste --type image/png > "$probe/eingefuegt.png"
			cmp -s "$bild" "$probe/eingefuegt.png" && echo "eingefuegt: gleich" || echo "probe.sh: eingefuegt VERSCHIEDEN ($(stat -c %s "$probe/eingefuegt.png") Byte)"
			z="$(zustand "$halter" "$idh")"
			echo "halter: $z"
			[ "$z" = unklar ] && echo "probe.sh: Identität des Halters weicht ab oder ist nicht prüfbar"
			timeout 10 wl-copy "anderer Inhalt"
			for _ in $(seq 1 50); do [ "$(zustand "$halter" "$idh")" = läuft ] || break; sleep 0.1; done
			case "$(zustand "$halter" "$idh")" in
			läuft) echo "probe.sh: nach wl-copy läuft der Halter NOCH" ;;
			beendet) echo "nach wl-copy: halter beendet" ;;
			*) echo "probe.sh: nach wl-copy ist der Zustand des Halters unklar (Identität nicht prüfbar oder PID wiederverwendet)" ;;
			esac
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

mkdir -p "$probe/home/.config" "$probe/home/.local/share" "$probe/home/.local/state" "$probe/home/.cache" "$probe/run"
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
printf '%s' "$ende" > "$probe/ende.txt"
if [ -n "$drehbuch" ]; then
	fotos="$probe/fotos/$(date +%Y%m%d-%H%M%S)"
	printf '%s' "$fotos" > "$probe/fotos.txt"
	GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o "$probe/fernbedienung" ./cmd/schnapp-fernbedienung || exit 1
fi
# Alle Ordner, in die Schnapp schreibt (settings: XDG_CONFIG_HOME; Bibliothek: XDG_DATA_HOME, XDG_CACHE_HOME; letzter Bereich
# und Tagesmarke: XDG_STATE_HOME; Bilderordner aus $XDG_CONFIG_HOME/user-dirs.dirs), liegen unter tmp/probe/home. Ein
# geerbter WAYLAND_SOCKET (fertig verbundener Socket der echten Sitzung) und ein gesetztes XDG_PICTURES_DIR werden entfernt
# (F-230).
# Äußere Grenze: bei --drehbuch --zeit + --ende + 10 s, sonst --zeit. Die Frist nach SIGTERM liegt im Lauf und darf nicht
# von der Grenze abgeschnitten werden; die Zeit für Start und Drehbuch bleibt --zeit.
grenze="$zeit"
[ -z "$drehbuch" ] || grenze=$((zeit + ende + 10))
env -u WAYLAND_DISPLAY -u WAYLAND_SOCKET -u DISPLAY -u XDG_PICTURES_DIR PROBE_INNEN=1 PROBE_AUSSENBUS="${DBUS_SESSION_BUS_ADDRESS:-}" \
	HOME="$probe/home" XDG_CONFIG_HOME="$probe/home/.config" XDG_DATA_HOME="$probe/home/.local/share" \
	XDG_STATE_HOME="$probe/home/.local/state" XDG_CACHE_HOME="$probe/home/.cache" XDG_RUNTIME_DIR="$probe/run" \
	XKB_DEFAULT_LAYOUT=us XKB_DEFAULT_VARIANT= XKB_DEFAULT_MODEL= XKB_DEFAULT_OPTIONS= \
	timeout -k 5 "$grenze" dbus-run-session -- \
	kwin_wayland --virtual --socket schnapp-probe --width "$breite" --height "$hoehe" --output-count "$monitore" \
	--scale "$skala" --no-lockscreen --no-global-shortcuts \
	--exit-with-session "$PWD/scripts/probe.sh" > "$probe/kwin.log" 2>&1
code=$?
if [ ! -s "$probe/ergebnis.txt" ]; then
	echo "probe.sh: kein Ergebnis (KWin-Exit $code, siehe tmp/probe/kwin.log)" >&2
	[ "$code" = 0 ] && code=1
	exit "$code"
fi
[ "$code" = 124 ] && echo "Zeitgrenze erreicht ($grenze s), KWin beendet" >&2
cat "$probe/ergebnis.txt"
grep -q '^probe.sh:' "$probe/ergebnis.txt" && exit 1
exit "$code"
