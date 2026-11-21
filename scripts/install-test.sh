#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Test für install.sh (F-324): läuft ohne echtes HOME, ohne echtes go (Attrappe in scripts/install-test-stub/).
#
#   scripts/install-test.sh               prüft ./install.sh
#   INSTALL_SH=/pfad/install.sh scripts/install-test.sh   prüft eine andere Fassung (Gegenprobe)
#
# Fälle:
#   1. Vorlage ohne exakte Zeile [Desktop Entry]: Abbruch mit Fehler, nichts angelegt, nicht gebaut (F-122).
#   2. Repo ohne Tag: Version 0.0.0-dev+<Kurzhash>, mit .dirty bei geändertem Baum; mit Tag die Tag-Version (F-113).
#   3. Ordner in fremdem Repo (mit Tag), selbst ohne Repo: der fremde Tag zählt nicht, Version 0.0.0-dev (F-113).
#
# Der Sandkasten liegt unter tmp/ des Projekts (gitignoriert) und bleibt nach dem Lauf liegen; es wird nichts gelöscht.
set -eu

wurzel="$(cd "$(dirname "$0")/.." && pwd -P)"
stub="$wurzel/scripts/install-test-stub"
[ -x "$stub/go" ] || { echo "install-test: Attrappe fehlt oder ist nicht ausführbar: $stub/go" >&2; exit 1; }
mkdir -p -- "$wurzel/tmp"
sand="$(mktemp -d "$wurzel/tmp/install-test.XXXXXX")"
echo "Sandkasten: $sand"

fehler=0
fail() { echo "FEHLER: $*" >&2; fehler=1; }

# Kopie der für install.sh nötigen Dateien nach $1.
kopie() {
	mkdir -p -- "$1/data" "$1/brand/logo/svg" "$1/cmd/schnapp"
	cp -- "${INSTALL_SH:-$wurzel/install.sh}" "$1/install.sh"
	cp -- "$wurzel/data/schnapp.desktop.in" "$1/data/schnapp.desktop.in"
	cp -- "$wurzel/brand/logo/svg/schnapp-symbol-dunkel.svg" "$1/brand/logo/svg/schnapp-symbol-dunkel.svg"
	chmod 0755 "$1/install.sh"
}

# install.sh in $1 mit eigenem HOME $2 ausführen; Ausgabe in $2/aus.txt, Rückgabe der Exit-Code.
lauf() {
	mkdir -p -- "$2/home"
	: > "$2/go.log"
	rc=0
	env -i HOME="$2/home" PATH="$stub:/usr/bin:/bin" STUB_LOG="$2/go.log" \
		GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null \
		bash "$1/install.sh" > "$2/aus.txt" 2>&1 || rc=$?
	return "$rc"
}

git_ein() { git -c user.name=test -c user.email=test@invalid -C "$1" "${@:2}"; }

# --- Fall 1: Vorlage ohne [Desktop Entry] ----------------------------------
f1="$sand/f1"
kopie "$f1/repo"
printf '[Desktop Entry] \nName=Test\nExec=@EXEC@\n' > "$f1/repo/data/schnapp.desktop.in"
rc=0
lauf "$f1/repo" "$f1" || rc=$?
[ "$rc" -ne 0 ] || fail "Fall 1: install.sh hätte mit Fehler abbrechen müssen"
grep -q 'keine Zeile \[Desktop Entry\]' "$f1/aus.txt" || fail "Fall 1: Fehlermeldung fehlt: $(cat "$f1/aus.txt")"
[ ! -e "$f1/home/.local" ] || fail "Fall 1: es wurde etwas angelegt"
[ ! -s "$f1/go.log" ] || fail "Fall 1: es wurde gebaut"

# --- Fall 2: Repo ohne Tag --------------------------------------------------
f2="$sand/f2"
kopie "$f2/repo"
git_ein "$f2/repo" init -q
git_ein "$f2/repo" add -A
git_ein "$f2/repo" commit -q -m start
kurz="$(git_ein "$f2/repo" rev-parse --short HEAD)"
rc=0
lauf "$f2/repo" "$f2" || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 2a: install.sh scheiterte: $(cat "$f2/aus.txt")"
grep -qF -- "-X main.version=0.0.0-dev+$kurz " "$f2/go.log" || fail "Fall 2a: erwartet 0.0.0-dev+$kurz, go-Aufruf: $(cat "$f2/go.log")"
grep -qF -- "0.0.0-dev+$kurz.dirty" "$f2/go.log" && fail "Fall 2a: sauberer Baum darf nicht .dirty sein"
[ -x "$f2/home/.local/bin/schnapp" ] || fail "Fall 2a: Programm nicht installiert"
grep -q "^Exec=\"$f2/home/.local/bin/schnapp\"" "$f2/home/.local/share/applications/schnapp.desktop" \
	|| fail "Fall 2a: Exec= in der Desktop-Datei falsch: $(cat "$f2/home/.local/share/applications/schnapp.desktop" 2>&1 || true)"

printf '# geändert\n' >> "$f2/repo/data/schnapp.desktop.in"
rc=0
lauf "$f2/repo" "$f2" || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 2b: install.sh scheiterte: $(cat "$f2/aus.txt")"
grep -qF -- "-X main.version=0.0.0-dev+$kurz.dirty" "$f2/go.log" || fail "Fall 2b: erwartet 0.0.0-dev+$kurz.dirty, go-Aufruf: $(cat "$f2/go.log")"

git_ein "$f2/repo" checkout -q -- data/schnapp.desktop.in
git_ein "$f2/repo" tag -a v1.2.3 -m t
rc=0
lauf "$f2/repo" "$f2" || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 2c: install.sh scheiterte: $(cat "$f2/aus.txt")"
grep -qF -- "-X main.version=1.2.3 " "$f2/go.log" || fail "Fall 2c: erwartet 1.2.3, go-Aufruf: $(cat "$f2/go.log")"

# --- Fall 3: Ordner im fremden Repo mit Tag, selbst ohne Repo --------------
f3="$sand/f3"
mkdir -p -- "$f3"
git_ein "$f3" init -q
printf 'x\n' > "$f3/x"
git_ein "$f3" add x
git_ein "$f3" commit -q -m fremd
git_ein "$f3" tag -a v9.9.9 -m t
kopie "$f3/unterordner"
rc=0
lauf "$f3/unterordner" "$f3" || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 3: install.sh scheiterte: $(cat "$f3/aus.txt")"
grep -qF -- "-X main.version=0.0.0-dev " "$f3/go.log" || fail "Fall 3: erwartet 0.0.0-dev (fremder Tag zählt nicht), go-Aufruf: $(cat "$f3/go.log")"

if [ "$fehler" -eq 0 ]; then
	echo "install-test: alle Fälle bestanden"
else
	echo "install-test: FEHLGESCHLAGEN" >&2
	exit 1
fi
