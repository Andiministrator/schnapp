#!/bin/bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Probe ADR-0030 Fassung 2 (Karte #173): MCP-Server `schnapp mcp` im virtuellen KWin, mit echter Aufnahme über KWin und
# echtem D-Bus-Aufruf der Benachrichtigung (an eine Attrappe auf dem privaten Bus des Laufs).
# Vorher bauen (außerhalb, im Repo):
#   GOFLAGS= go build -race -trimpath -tags novulkan,nox11 -o tmp/probe/fake-notifications ./scripts/fake-notifications
#   GOFLAGS= go build -race -trimpath -tags novulkan,nox11,probe -o tmp/probe/fernbedienung ./cmd/schnapp-fernbedienung
# Aufruf: scripts/probe.sh --zeit 240 --innen scripts/drehbuecher/agenten-mcp.sh   (nie direkt starten)
# Erwartet (das Skript prüft es selbst, OK/FEHL):
#   A neue Form (2026-07-28): server/discover, tools/list mit sechs Werkzeugen, list_screens, capture_screen ohne Ziel
#     (Bibliothek, agents/), capture_screen all mit output (Datei 0600, nicht in der Bibliothek), library_search;
#     Benachrichtigung je Aufnahme, vor der zweiten wird die erste geschlossen (F-490); danach in derselben Sitzung capture = no → abgelehnt (Widerruf wirkt sofort).
#   B library_read und export auf die Aufnahme aus A; Altform mit initialize.
#   C ohne Benachrichtigungsdienst: Aufnahme verworfen, Bibliothek unverändert.
#   D mit confirm = yes: Dialog erscheint (Foto), Esc lehnt ab, nichts aufgenommen.
#   E mit confirm = yes: Klick auf „Erlauben“ (Koordinaten aus PROBE_ERLAUBEN="X Y", sonst übersprungen) nimmt auf;
#     bei 1600x1000 und deutscher Oberfläche: PROBE_ERLAUBEN="981 577" (Foto dialog.png aus D zeigt die Lage).
#   F mit confirm = yes (ADR-0030 Nachtrag 4 Punkt 4, F-464): Tab, Tab, Return erlaubt nie — Return trifft höchstens
#     „Ablehnen“; danach schließt Esc einen noch offenen Dialog. Antwort „did not allow“, nichts aufgenommen, kein Hinweis.
#   G mit confirm = yes und PROBE_ERLAUBEN (F-481): Klick auf „Erlauben“, sobald das Kind sein erstes Bild meldet
#     (SCHNAPP_DEBUG) — er fällt in die Sperre und erlaubt nicht; danach Esc. Meldet das Kind kein Drücken in der
#     Sperre, ist der Lauf nicht aussagekräftig (FEHL).
# Seit ADR-0030 Nachtrag 5 (F-491) erscheint der Dialog nur, wenn der Wayland-Socket und org.kde.KWin auf dem Bus zum
# selben Prozess gehören; im virtuellen KWin ist das so (sonst lautet die Antwort in D bis G „different compositors“).
# Der Probe-Build nimmt für den MCP-Server HOME als Heim (probe_an.go); probe.sh setzt es unter tmp/probe.
# Es startet nur Prozesse, die von selbst enden (Attrappe mit eigener Zeitgrenze, Fernbedienung, schnapp mcp mit EOF).
set -u
wurzel="$PWD"
[ -x "$wurzel/scripts/probe.sh" ] || { echo "probe.sh: nicht in der Wurzel des Repos ($wurzel)"; exit 1; }
probe="$wurzel/tmp/probe"
[ "${PROBE_INNEN:-}" = 1 ] || { echo "probe.sh: nur über scripts/probe.sh --innen aufrufen"; exit 1; }
for ordner in "${HOME:-}" "${XDG_CONFIG_HOME:-}" "${XDG_STATE_HOME:-}"; do
	case "$ordner" in
	"$probe/"*) ;;
	*) echo "probe.sh: Ordner außerhalb von tmp/probe ($ordner), Abbruch"; exit 1 ;;
	esac
done
[ -x "$probe/fake-notifications" ] || { echo "probe.sh: tmp/probe/fake-notifications fehlt (siehe Kopf)"; exit 1; }
# Je Lauf ein eigener Ausgabeordner: output und target ersetzen nie eine vorhandene Datei (ADR-0030 Nachtrag 4 Punkt 2).
aus="$HOME/aus/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$HOME/Bilder" "$aus" "$XDG_CONFIG_HOME/schnapp" "$probe/mcp"
printf 'XDG_PICTURES_DIR="$HOME/Bilder"\n' > "$XDG_CONFIG_HOME/user-dirs.dirs"
ini="$XDG_CONFIG_HOME/schnapp/settings.ini"
s="$probe/schnapp"
fehler=0
pruefe() { if [ "$2" = "$3" ]; then echo "OK   $1 ($2)"; else echo "FEHL $1: $2, erwartet $3"; fehler=1; fi; }
enthaelt() { if grep -q -F -- "$3" "$2"; then echo "OK   $1"; else echo "FEHL $1: »$3« fehlt in $2"; fehler=1; fi; }
fehlt() { if grep -q -F -- "$3" "$2"; then echo "FEHL $1: »$3« in $2"; fehler=1; else echo "OK   $1"; fi; }
aufnahmen() { find "$HOME/Bilder/schnapp" -type f -name '*.png' 2> /dev/null | wc -l; }
M='"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"Probe-Agent","version":"1"}}'
ruf() { printf '{"jsonrpc":"2.0","id":%s,"method":"tools/call","params":{"name":"%s","arguments":%s,%s}}\n' "$1" "$2" "$3" "$M"; }
antwort() { grep -F "\"id\":$2," "$1" | head -1; } # Antwortzeile zur Kennung
# urteil NAME DATEI ID FEHLER(true|false) [TEXT]: Antwort hat isError wie verlangt und enthält TEXT
urteil() {
	local z
	z="$(antwort "$2" "$3")"
	if printf '%s' "$z" | grep -q -F "\"isError\":$4" && { [ -z "${5:-}" ] || printf '%s' "$z" | grep -q -F -- "$5"; }; then
		echo "OK   $1"
	else
		echo "FEHL $1: ${z:0:300}"; fehler=1
	fi
}

notify_log="$probe/mcp/notify.txt"
: > "$notify_log"
"$probe/fake-notifications" "$notify_log" 45 &
for _ in $(seq 1 50); do grep -q bereit "$notify_log" && break; sleep 0.1; done
enthaelt "Attrappe bereit" "$notify_log" "bereit"

echo "== A: neue Form"
printf '[capture]\nopen-editor = false\n[agents]\ncapture = yes\nlibrary = yes\nconfirm = no\nimage = yes\n' > "$ini"
vorher="$(aufnahmen)"
{
	printf '{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{%s}}\n' "$M"
	printf '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{%s}}\n' "$M"
	ruf 3 list_screens '{}'
	ruf 4 capture_screen '{}'
	ruf 5 capture_screen "{\"all\":true,\"output\":\"$aus/mcp.png\"}"
	ruf 6 library_search '{"limit":5}'
	sleep 25
	printf '[capture]\nopen-editor = false\n[agents]\ncapture = no\n' > "$ini"
	ruf 7 capture_screen '{}'
	sleep 3
} | SCHNAPP_DEBUG=1 "$s" mcp > "$probe/mcp/a.out" 2> "$probe/mcp/a.err"
pruefe "A Exit" "$?" 0
pruefe "A Antwortzeilen" "$(wc -l < "$probe/mcp/a.out")" 7
enthaelt "A discover" "$probe/mcp/a.out" '"supportedVersions":["2026-07-28"'
pruefe "A sechs Werkzeuge" "$(antwort "$probe/mcp/a.out" 2 | grep -o '"inputSchema"' | wc -l)" 6
antwort "$probe/mcp/a.out" 3 | grep -q '\\"screens\\":\[{\\"name\\"' && echo "OK   A list_screens" || { echo "FEHL A list_screens: $(antwort "$probe/mcp/a.out" 3)"; fehler=1; }
a4="$(antwort "$probe/mcp/a.out" 4)"
case "$a4" in *'"isError":false'*'"type":"image"'* | *'"type":"image"'*'"isError":false'*) echo "OK   A capture_screen mit Bild" ;; *) echo "FEHL A capture_screen: ${a4:0:400}"; fehler=1 ;; esac
pfad="$(printf '%s' "$a4" | grep -o '"structuredContent":{"path":"[^"]*"' | sed 's/.*"path":"//; s/"$//')"
case "$pfad" in "$HOME/Bilder/schnapp/agents/"*.png) echo "OK   A Ablage unter agents/ ($pfad)" ;; *) echo "FEHL A Pfad: $pfad"; fehler=1 ;; esac
pruefe "A Rechte der Ablage" "$(stat -c %a "$pfad" 2> /dev/null)" 600
pruefe "A Rechte von output" "$(stat -c %a "$aus/mcp.png" 2> /dev/null)" 600
file "$aus/mcp.png" | grep -q 'PNG image data' && echo "OK   A output ist PNG" || { echo "FEHL A output kein PNG"; fehler=1; }
pruefe "A genau eine neue Aufnahme in der Bibliothek" "$(($(aufnahmen) - vorher))" 1
enthaelt "A library_search nennt die Aufnahme" "$probe/mcp/a.out" "$pfad"
urteil "A Widerruf wirkt sofort" "$probe/mcp/a.out" 7 true 'capture = yes'
pruefe "A zwei Benachrichtigungen" "$(grep -c '^notify' "$notify_log")" 2
enthaelt "A Benachrichtigung von Schnapp" "$notify_log" 'notify app="Schnapp"'
# F-490 (ADR-0030 Nachtrag 7): vor der zweiten Aufnahme schließt der Server den ersten Hinweis über den Bus.
enthaelt "A voriger Hinweis geschlossen" "$notify_log" "close id=1"
fehlt "A stdout nur Protokoll" "$probe/mcp/a.out" "schnapp mcp:"

echo "== B: library_read, export, Altform"
printf '[capture]\nopen-editor = false\n[agents]\nlibrary = yes\nimage = no\n' > "$ini"
{
	ruf 1 library_read "{\"path\":\"$pfad\"}"
	ruf 2 export "{\"file\":\"$pfad\",\"target\":\"$aus/export.jpg\"}"
	ruf 3 library_read "{\"path\":\"$aus/mcp.png\"}"
	printf '{"jsonrpc":"2.0","id":4,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"Alt","version":"1"}}}\n'
	printf '{"jsonrpc":"2.0","method":"notifications/initialized"}\n'
	printf '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_screens","arguments":{}}}\n'
	sleep 15
} | "$s" mcp > "$probe/mcp/b.out" 2> "$probe/mcp/b.err"
pruefe "B Exit" "$?" 0
b1="$(antwort "$probe/mcp/b.out" 1)"
case "$b1" in *'"isError":false'*) echo "OK   B library_read" ;; *) echo "FEHL B library_read: ${b1:0:300}"; fehler=1 ;; esac
case "$b1" in *'"type":"image"'*) echo "FEHL B image = no lieferte ein Bild"; fehler=1 ;; *) echo "OK   B ohne Bild bei image = no" ;; esac
file "$aus/export.jpg" | grep -q 'JPEG image data' && echo "OK   B export JPEG" || { echo "FEHL B export"; fehler=1; }
case "$(antwort "$probe/mcp/b.out" 3)" in *'"isError":true'*) echo "OK   B Datei außerhalb der Bibliothek abgelehnt" ;; *) echo "FEHL B library_read außerhalb"; fehler=1 ;; esac
enthaelt "B Altform initialize" "$probe/mcp/b.out" '"protocolVersion":"2025-06-18"'
case "$(antwort "$probe/mcp/b.out" 5)" in *'"isError":false'*) echo "OK   B Altform tools/call" ;; *) echo "FEHL B Altform tools/call"; fehler=1 ;; esac

echo "== C: ohne Benachrichtigungsdienst"
wait # die Attrappe endet von selbst nach ihrer Zeitgrenze
printf '[capture]\nopen-editor = false\n[agents]\ncapture = yes\nconfirm = no\n' > "$ini"
vorher="$(aufnahmen)"
{ ruf 1 capture_screen '{}'; sleep 15; } | "$s" mcp > "$probe/mcp/c.out" 2> "$probe/mcp/c.err"
urteil "C verworfen" "$probe/mcp/c.out" 1 true discarded
pruefe "C Bibliothek unverändert" "$(($(aufnahmen) - vorher))" 0

if [ -x "$probe/fernbedienung" ]; then
	echo "== D: Dialog, Esc lehnt ab"
	: > "$notify_log"
	"$probe/fake-notifications" "$notify_log" 60 &
	for _ in $(seq 1 50); do grep -q bereit "$notify_log" && break; sleep 0.1; done
	printf '[capture]\nopen-editor = false\n[agents]\ncapture = yes\nconfirm = yes\n' > "$ini"
	fotos="$probe/fotos/mcp-$(date +%Y%m%d-%H%M%S)"
	mkdir -p "$fotos"
	printf 'warte 4000\nfoto dialog\ntaste Esc\nwarte 500\n' > "$probe/mcp/d.txt"
	vorher="$(aufnahmen)"
	{ ruf 1 capture_screen '{}'; sleep 25; } | SCHNAPP_DEBUG=1 "$s" mcp > "$probe/mcp/d.out" 2> "$probe/mcp/d.err" &
	mcp_pid=$!
	FOTOS="$fotos" "$probe/fernbedienung" "$probe/mcp/d.txt" 2>&1 | tail -3
	wait "$mcp_pid"
	urteil "D Esc lehnt ab" "$probe/mcp/d.out" 1 true 'did not allow'
	pruefe "D Bibliothek unverändert" "$(($(aufnahmen) - vorher))" 0
	echo "Foto: $fotos"
	if [ -n "${PROBE_ERLAUBEN:-}" ]; then
		echo "== E: Klick auf Allow bei ${PROBE_ERLAUBEN}"
		printf 'warte 4000\nfoto vor-klick\nklick %s\nwarte 2000\n' "$PROBE_ERLAUBEN" > "$probe/mcp/e.txt"
		{ ruf 1 capture_screen '{}'; sleep 25; } | SCHNAPP_DEBUG=1 "$s" mcp > "$probe/mcp/e.out" 2> "$probe/mcp/e.err" &
		mcp_pid=$!
		FOTOS="$fotos" "$probe/fernbedienung" "$probe/mcp/e.txt" 2>&1 | tail -3
		wait "$mcp_pid"
		urteil "E Erlauben nimmt auf" "$probe/mcp/e.out" 1 false
		pruefe "E Benachrichtigung" "$(grep -c '^notify' "$notify_log")" 1
		pruefe "E eine neue Aufnahme" "$(($(aufnahmen) - vorher))" 1
	fi
	echo "== F: Tab, Tab, Return erlaubt nie"
	printf 'warte 4000\ntaste Tab\nwarte 200\ntaste Tab\nwarte 200\nfoto tastatur-fokus\ntaste Return\nwarte 1000\ntaste Esc\nwarte 500\n' > "$probe/mcp/f.txt"
	vorher="$(aufnahmen)"
	hinweise_vorher="$(grep -c '^notify' "$notify_log")"
	{ ruf 1 capture_screen '{}'; sleep 25; } | SCHNAPP_DEBUG=1 "$s" mcp > "$probe/mcp/f.out" 2> "$probe/mcp/f.err" &
	mcp_pid=$!
	FOTOS="$fotos" "$probe/fernbedienung" "$probe/mcp/f.txt" 2>&1 | tail -3
	wait "$mcp_pid"
	urteil "F Tastatur erlaubt nicht" "$probe/mcp/f.out" 1 true 'did not allow'
	pruefe "F Bibliothek unverändert" "$(($(aufnahmen) - vorher))" 0
	pruefe "F kein Hinweis" "$(grep -c '^notify' "$notify_log")" "$hinweise_vorher"
	if [ -n "${PROBE_ERLAUBEN:-}" ]; then
		echo "== G: Klick auf Allow in der Sperre erlaubt nicht (F-481)"
		vorher="$(aufnahmen)"
		hinweise_vorher="$(grep -c '^notify' "$notify_log")"
		printf 'klick %s\nwarte 1500\nfoto nach-sperrklick\ntaste Esc\nwarte 500\n' "$PROBE_ERLAUBEN" > "$probe/mcp/g.txt"
		{ ruf 1 capture_screen '{}'; sleep 25; } | SCHNAPP_DEBUG=1 "$s" mcp > "$probe/mcp/g.out" 2> "$probe/mcp/g.err" &
		mcp_pid=$!
		# Erst klicken, wenn das Kind sein erstes Bild meldet: Der Klick soll in die 600 ms der Sperre fallen.
		for _ in $(seq 1 200); do grep -q -F '[bestaetigen] erstes Bild' "$probe/mcp/g.err" && break; sleep 0.05; done
		FOTOS="$fotos" "$probe/fernbedienung" "$probe/mcp/g.txt" 2>&1 | tail -3
		wait "$mcp_pid"
		if grep -q -F '[bestaetigen] Drücken gesperrt=true' "$probe/mcp/g.err"; then
			urteil "G Klick in der Sperre erlaubt nicht" "$probe/mcp/g.out" 1 true 'did not allow'
			pruefe "G Bibliothek unverändert" "$(($(aufnahmen) - vorher))" 0
			pruefe "G kein Hinweis" "$(grep -c '^notify' "$notify_log")" "$hinweise_vorher"
		else
			# Kam der Klick erst nach der Sperre an, sagt der Lauf nichts aus: nicht als bestanden zählen.
			echo "FEHL G nicht aussagekräftig: kein Drücken in der Sperre gemeldet (Fernbedienung zu spät?)"
			grep -F '[bestaetigen]' "$probe/mcp/g.err" | head -5
			fehler=1
		fi
	fi
	wait
fi
echo "-- stderr A (Auszug):"; head -20 "$probe/mcp/a.err"
[ "$fehler" = 0 ] && echo "BESTANDEN" || echo "DURCHGEFALLEN"
