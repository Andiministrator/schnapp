#!/bin/bash
# Echtlauf für F-490 und F-506 (Karte #208) in der eigenen Plasma-Sitzung; der virtuelle KWin hat keinen echten
# Benachrichtigungsdienst und keinen echten Fokuswechsel durch den Nutzer.
#
#   scripts/echtlauf-mcp-hinweise.sh        (aus der Wurzel des Repos, im eigenen Terminal)
#
# Was es tut:
#   - baut Schnapp mit -race nach tmp/echtlauf-208/<Zeit>/;
#   - sichert ~/.config/schnapp/settings.ini dorthin und setzt für die Dauer des Laufs [agents] capture = yes,
#     image = no (die übrigen Abschnitte bleiben); beim Ende (auch bei Strg+C) kommt die alte Datei zurück, gab es
#     keine, wird die Laufdatei in den Laufordner verschoben (nichts wird gelöscht);
#   - Teil A (F-490), confirm = no: zwei Aufnahmen aller Bildschirme im Abstand von 5 s nach a1.png und a2.png im
#     Laufordner. Frage: Steht der Hinweis der ersten Aufnahme in a2.png?
#   - Teil B (F-506), confirm = yes: eine Aufnahme mit Dialog. Erst neben den Dialog klicken (er verliert den Fokus),
#     dann EINMAL auf „Erlauben“. Frage: Hat der eine Klick erlaubt (b.png da), oder brauchte es einen zweiten?
# Die Bilder zeigen den echten Bildschirm: Sie bleiben unter tmp/ (nicht versioniert) und gehören nie ins Repo.
set -u
[ -x scripts/probe.sh ] || { echo "Bitte aus der Wurzel des Repos starten."; exit 1; }
lauf="tmp/echtlauf-208/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$lauf"
echo "== Bauen (mit -race) nach $lauf/schnapp …"
GOFLAGS= go build -race -trimpath -tags novulkan,nox11 -o "$lauf/schnapp" ./cmd/schnapp || exit 1

ini="$HOME/.config/schnapp/settings.ini"
hatte=0
if [ -f "$ini" ]; then
	cp -p "$ini" "$lauf/settings.ini.vorher" || exit 1
	hatte=1
fi
zurueck() {
	if [ "$hatte" = 1 ]; then
		cp -p "$lauf/settings.ini.vorher" "$ini" && echo "== Einstellungen zurückgestellt ($ini)."
	elif [ -f "$ini" ]; then
		mv "$ini" "$lauf/settings.ini.vom-lauf" && echo "== Laufeinstellung aus $ini in den Laufordner verschoben."
	fi
}
trap zurueck EXIT
trap 'exit 130' INT TERM # auch bei Strg+C über EXIT zurückstellen
mkdir -p "$(dirname "$ini")"

# setze CONFIRM: alte Datei ohne [agents]-Abschnitt, dann der Abschnitt für den Lauf.
setze() {
	{
		[ "$hatte" = 1 ] && awk '/^\[/{drin = ($0 ~ /^\[agents\]/)} !drin' "$lauf/settings.ini.vorher"
		printf '\n[agents]\ncapture = yes\nlibrary = no\nconfirm = %s\nimage = no\n' "$1"
	} > "$ini"
}

M='"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"Echtlauf-208","version":"1"}}'
ruf() { printf '{"jsonrpc":"2.0","id":%s,"method":"tools/call","params":{"name":"capture_screen","arguments":{"all":true,"output":"%s"},%s}}\n' "$1" "$2" "$M"; }
# warte DATEI ID SEKUNDEN: bis die Antwort mit dieser Kennung da ist (höchstens SEKUNDEN).
warte() { local i; for ((i = 0; i < $3 * 4; i++)); do grep -q -F "\"id\":$2," "$1" 2> /dev/null && return 0; sleep 0.25; done; return 1; }
urteil() { local z; z="$(grep -F "\"id\":$2," "$1" | head -1)"; case "$z" in *'"isError":false'*) echo "erlaubt/aufgenommen" ;; '') echo "keine Antwort" ;; *) echo "abgelehnt: ${z:0:200}" ;; esac; }

echo
echo "== Teil A (F-490): zwei Aufnahmen ohne Dialog, Abstand 5 s. Nichts anklicken, Hinweise nicht wegklicken."
read -r -p "   Enter startet …" _
setze no
{
	ruf 1 "$PWD/$lauf/a1.png"
	warte "$lauf/a.out" 1 30
	sleep 5
	ruf 2 "$PWD/$lauf/a2.png"
	warte "$lauf/a.out" 2 30
} | "$lauf/schnapp" mcp > "$lauf/a.out" 2> "$lauf/a.err"
echo "   Aufnahme 1: $(urteil "$lauf/a.out" 1)"
echo "   Aufnahme 2: $(urteil "$lauf/a.out" 2)"
echo "   Bitte ansehen: $lauf/a2.png — steht der Hinweis der ersten Aufnahme darin?"

echo
echo "== Teil B (F-506): eine Aufnahme mit Dialog."
echo "   Wenn der Dialog erscheint: zuerst einmal NEBEN den Dialog klicken (z. B. in dieses Terminal),"
echo "   dann nach einer Sekunde genau EINMAL auf „Erlauben“. Kein zweiter Klick."
read -r -p "   Enter startet …" _
setze yes
{
	ruf 3 "$PWD/$lauf/b.png"
	warte "$lauf/b.out" 3 70
} | "$lauf/schnapp" mcp > "$lauf/b.out" 2> "$lauf/b.err"
echo "   Ergebnis: $(urteil "$lauf/b.out" 3)"
[ -f "$lauf/b.png" ] && echo "   b.png liegt vor." || echo "   b.png fehlt."
echo
echo "== Fertig. Laufordner: $lauf (Bilder zeigen deinen Bildschirm, bleiben unter tmp/)."
