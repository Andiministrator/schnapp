#!/bin/bash
# Rückfallprobe F-144 (ADR-0029): Rettungskopie nach SIGKILL. Aufruf (die Fernbedienung baut --innen nicht selbst):
#   GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o tmp/probe/fernbedienung ./cmd/schnapp-fernbedienung
#   scripts/probe.sh --zeit 120 --innen scripts/drehbuecher/rueckfall-rettung-kill9.sh
# Erwartet: Kopie liegt vor und nach SIGKILL, Foto 3 zeigt das Rechteck und die Meldung „Restored unsaved changes …“,
# nach SIGTERM (Speichern) ist der Rettungsordner leer. Läuft als Sitzung des virtuellen KWin. Greift nur in die
# eigene, hier gestartete Schnapp-PID ein, nach Prüfung von Startzeit und Programmname (wie probe.sh).
set -u
wurzel="$PWD" # probe.sh läuft in der Wurzel des Repos
[ -x "$wurzel/scripts/probe.sh" ] || { echo "probe.sh: nicht in der Wurzel des Repos ($wurzel)"; exit 1; }
probe="$wurzel/tmp/probe"
lauf="$wurzel/scripts/drehbuecher"
fotos="$probe/fotos/kill9-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$HOME/Bilder" "$fotos"
printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
bild="$HOME/probe-kill9.png"
cp -- "$wurzel/scripts/drehbuecher/testbild-1200x800.png" "$bild"
rettung="$XDG_STATE_HOME/schnapp/rescue"

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

# starte NR: Schnapp mit dem Bild starten, Identität merken, auf das Fenster warten. Setzt pid und id0.
starte() {
	SCHNAPP_DEBUG=1 "$probe/schnapp" "$bild" > "$probe/stdout-$1.txt" 2> "$probe/stderr-$1.txt" &
	pid=$!
	id0=""
	for _ in $(seq 1 20); do
		if id1="$(identitaet "$pid")" && [ "${id1#* }" = schnapp ]; then id0="$id1"; break; fi
		sleep 0.1
	done
	for _ in $(seq 1 300); do
		grep -q 'Fenstergröße' "$probe/stderr-$1.txt" && break
		kill -0 "$pid" 2> /dev/null || break
		sleep 0.1
	done
	sleep 1
	echo "== Lauf $1: PID $pid, Identität ${id0:-unlesbar}"
}

# beende SIGNAL: nur die eigene PID, nur bei gleicher Identität.
beende() {
	local jetzt
	jetzt="$(identitaet "$pid")" || { echo "PID $pid schon weg"; return; }
	if [ -n "$id0" ] && [ "$jetzt" = "$id0" ] && [ "${id0#* }" = schnapp ]; then
		kill "-$1" "$pid"
		for _ in $(seq 1 100); do kill -0 "$pid" 2> /dev/null || break; sleep 0.1; done
		wait "$pid" 2> /dev/null
		echo "SIG$1 an $pid, Exit $?"
	else
		echo "Identität weicht ab oder unlesbar ($jetzt / $id0): kein Eingriff"
	fi
}

starte 1
FOTOS="$fotos" "$probe/fernbedienung" "$lauf/rueckfall-rettung-kill9-1.txt" 2>&1
echo "fernbedienung exit $?"
echo "-- Rettungsordner vor SIGKILL:"
ls -la "$rettung" 2>&1
beende KILL
echo "-- Rettungsordner nach SIGKILL:"
ls -la "$rettung" 2>&1

starte 2
FOTOS="$fotos" "$probe/fernbedienung" "$lauf/rueckfall-rettung-kill9-2.txt" 2>&1
echo "fernbedienung exit $?"
beende TERM
echo "-- Rettungsordner nach Lauf 2 (Speichern beim SIGTERM verwirft die Kopie):"
ls -la "$rettung" 2>&1
echo "-- stderr Lauf 1 (rescue/editor):"
grep -E 'rescue|Rett|Signal' "$probe/stderr-1.txt" | head -20
echo "-- stderr Lauf 2:"
grep -E 'rescue|Rett|Restored|Speichern' "$probe/stderr-2.txt" | head -20
echo "-- Fotos: $fotos"
ls "$fotos"
