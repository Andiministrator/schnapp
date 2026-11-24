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
#   4. Repo mit Tag ohne v-Schema (sprint-ende): zählt nicht, Version 0.0.0-dev+<Kurzhash> (F-364).
#   5. schnapp-ocr (Karte #242): wird mit derselben Version aus ocr/ gebaut und neben schnapp installiert; --uninstall
#      entfernt alle vier Dateien.
#   6. Gescheiterter Bau (F-648): scheitert nur schnapp-ocr, wird schnapp installiert, ein vorhandenes schnapp-ocr bleibt,
#      Warnung und Exit 0; scheitert schnapp, Exit ungleich 0 und nichts ersetzt. In keinem Fall bleibt eine *.neu liegen.
#   7. Pfad mit @EXEC@ (F-716): Abbruch vor dem Bau, nichts angelegt.
#   8. Abbruch beim Schreiben der Desktop-Datei (F-729): keine schnapp.desktop.neu bleibt liegen.
#
# Desktop-Werkzeuge (update-desktop-database, gtk-update-icon-cache, kbuildsycoca6) laufen im Test nur als Attrappen
# aus scripts/install-test-stub/ (F-367): Der Ordner steht im PATH vor /usr/bin, und Attrappen mit gleichem Namen
# verdecken dort die echten Programme. /usr/bin bleibt im PATH für cp, git und Co. Fall 2a prüft im Protokoll,
# dass die Attrappen liefen und nur Pfade im Sandkasten erhielten.
#
# Der Sandkasten liegt unter tmp/ des Projekts (gitignoriert) und bleibt nach dem Lauf liegen; es wird nichts gelöscht.
set -eu

wurzel="$(cd "$(dirname "$0")/.." && pwd -P)"
stub="$wurzel/scripts/install-test-stub"
for n in go update-desktop-database gtk-update-icon-cache kbuildsycoca6; do
	[ -x "$stub/$n" ] || { echo "install-test: Attrappe fehlt oder ist nicht ausführbar: $stub/$n" >&2; exit 1; }
done
mkdir -p -- "$wurzel/tmp"
sand="$(mktemp -d "$wurzel/tmp/install-test.XXXXXX")"
echo "Sandkasten: $sand"

fehler=0
fail() { echo "FEHLER: $*" >&2; fehler=1; }

# Kopie der für install.sh nötigen Dateien nach $1.
kopie() {
	mkdir -p -- "$1/data" "$1/brand/logo/svg" "$1/cmd/schnapp" "$1/ocr"
	cp -- "${INSTALL_SH:-$wurzel/install.sh}" "$1/install.sh"
	cp -- "$wurzel/data/schnapp.desktop.in" "$1/data/schnapp.desktop.in"
	cp -- "$wurzel/brand/logo/svg/schnapp-symbol-dunkel.svg" "$1/brand/logo/svg/schnapp-symbol-dunkel.svg"
	chmod 0755 "$1/install.sh"
}

# install.sh in $1 mit eigenem HOME $2 ausführen; Ausgabe in $2/aus.txt, Rückgabe der Exit-Code. $3 (optional): Ordner,
# in dem die go-Attrappe scheitert (STUB_FEHL_IN).
lauf() {
	mkdir -p -- "$2/home"
	: > "$2/go.log"
	rc=0
	env -i HOME="$2/home" PATH="$stub:/usr/bin:/bin" STUB_LOG="$2/go.log" STUB_FEHL_IN="${3:-}" \
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
# Die Attrappen liefen (nur sie schreiben ins Protokoll), und ihre Pfade liegen im Sandkasten.
grep -q '^update-desktop-database ' "$f2/go.log" || fail "Fall 2a: Attrappe update-desktop-database nicht aufgerufen"
grep -q '^kbuildsycoca6' "$f2/go.log" || fail "Fall 2a: Attrappe kbuildsycoca6 nicht aufgerufen"
grep -E '^(update-desktop-database|gtk-update-icon-cache) ' "$f2/go.log" | grep -v -F -- " $f2/" \
	&& fail "Fall 2a: Desktop-Werkzeug mit Pfad außerhalb des Sandkastens: $(cat "$f2/go.log")" || true

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

# --- Fall 4: Tag ohne v-Schema (F-364) ---------------------------------------
f4="$sand/f4"
kopie "$f4/repo"
git_ein "$f4/repo" init -q
git_ein "$f4/repo" add -A
git_ein "$f4/repo" commit -q -m start
git_ein "$f4/repo" tag sprint-ende
kurz4="$(git_ein "$f4/repo" rev-parse --short HEAD)"
rc=0
lauf "$f4/repo" "$f4" || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 4: install.sh scheiterte: $(cat "$f4/aus.txt")"
grep -qF -- "-X main.version=0.0.0-dev+$kurz4 " "$f4/go.log" || fail "Fall 4: erwartet 0.0.0-dev+$kurz4 (Tag sprint-ende zählt nicht), go-Aufruf: $(cat "$f4/go.log")"

# --- Fall 5: schnapp-ocr und --uninstall (Karte #242) ------------------------
grep -qF -- "-X main.version=1.2.3 -o $f2/home/.local/bin/schnapp-ocr.neu ." "$f2/go.log" \
	|| fail "Fall 5: schnapp-ocr nicht mit Version gebaut, go-Aufruf: $(cat "$f2/go.log")"
[ -x "$f2/home/.local/bin/schnapp-ocr" ] || fail "Fall 5: schnapp-ocr nicht installiert"
[ ! -e "$f2/home/.local/bin/schnapp-ocr.neu" ] || fail "Fall 5: Nebendatei schnapp-ocr.neu liegen geblieben"
rc=0
env -i HOME="$f2/home" PATH="$stub:/usr/bin:/bin" STUB_LOG="$f2/go.log" bash "$f2/repo/install.sh" --uninstall \
	> "$f2/aus-un.txt" 2>&1 || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 5: --uninstall scheiterte: $(cat "$f2/aus-un.txt")"
for d in bin/schnapp bin/schnapp-ocr share/applications/schnapp.desktop share/icons/hicolor/scalable/apps/schnapp.svg; do
	[ ! -e "$f2/home/.local/$d" ] || fail "Fall 5: --uninstall ließ $d liegen"
done

# --- Fall 6: gescheiterter Bau (F-648) ---------------------------------------
f6="$sand/f6"
kopie "$f6/repo"
bin6="$f6/home/.local/bin"
mkdir -p -- "$bin6"
printf 'alt\n' > "$bin6/schnapp"
printf 'alt-ocr\n' > "$bin6/schnapp-ocr"
rc=0
lauf "$f6/repo" "$f6" ocr || rc=$?
[ "$rc" -eq 0 ] || fail "Fall 6a: gescheiterter OCR-Bau bricht die Installation ab (Exit $rc): $(cat "$f6/aus.txt")"
grep -q 'Warnung: schnapp-ocr ließ sich nicht bauen' "$f6/aus.txt" || fail "Fall 6a: Warnung fehlt: $(cat "$f6/aus.txt")"
grep -qx 'alt' "$bin6/schnapp" && fail "Fall 6a: schnapp nicht neu installiert"
grep -qx 'alt-ocr' "$bin6/schnapp-ocr" || fail "Fall 6a: vorhandenes schnapp-ocr ersetzt oder entfernt"
[ -f "$f6/home/.local/share/applications/schnapp.desktop" ] || fail "Fall 6a: Desktop-Datei fehlt"
for d in schnapp.neu schnapp-ocr.neu; do
	[ ! -e "$bin6/$d" ] || fail "Fall 6a: Nebendatei $d liegen geblieben"
done
printf 'alt\n' > "$bin6/schnapp"
rc=0
lauf "$f6/repo" "$f6" repo || rc=$?
[ "$rc" -ne 0 ] || fail "Fall 6b: gescheiterter Bau von schnapp endet mit Exit 0"
grep -qx 'alt' "$bin6/schnapp" || fail "Fall 6b: schnapp trotz gescheitertem Bau ersetzt"
grep -qx 'alt-ocr' "$bin6/schnapp-ocr" || fail "Fall 6b: schnapp-ocr ersetzt"
for d in schnapp.neu schnapp-ocr.neu; do
	[ ! -e "$bin6/$d" ] || fail "Fall 6b: Nebendatei $d liegen geblieben"
done

# --- Fall 7: Pfad mit @EXEC@ (F-716) ------------------------------------------
# Vor dem Bau abgelehnt: nichts gebaut, nichts installiert, keine Nebendatei (auch nicht schnapp.desktop.neu).
f7="$sand/f7"
kopie "$f7/repo"
mkdir -p -- "$f7/home"
: > "$f7/go.log"
rc=0
env -i HOME="$f7/home" PREFIX="$f7/home/pre@EXEC@fix" PATH="$stub:/usr/bin:/bin" STUB_LOG="$f7/go.log" STUB_FEHL_IN="" \
	GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null \
	bash "$f7/repo/install.sh" > "$f7/aus.txt" 2>&1 || rc=$?
[ "$rc" -ne 0 ] || fail "Fall 7: Pfad mit @EXEC@ angenommen"
grep -q '@EXEC@ wird nicht unterstützt' "$f7/aus.txt" || fail "Fall 7: Meldung fehlt: $(cat "$f7/aus.txt")"
grep -q -- '-o ' "$f7/go.log" && fail "Fall 7: es wurde gebaut: $(cat "$f7/go.log")"
[ -z "$(find "$f7/home" -type f)" ] || fail "Fall 7: Dateien angelegt: $(find "$f7/home" -type f)"

# --- Fall 8: Abbruch beim Schreiben der Desktop-Datei (F-729) -----------------
# Eine chmod-Attrappe scheitert an schnapp.desktop.neu (wie volle Platte oder Signal zwischen Anlegen und Umbenennen):
# install.sh endet mit Fehler, die Falle räumt die Nebendatei weg.
f8="$sand/f8"
kopie "$f8/repo"
mkdir -p -- "$f8/home" "$f8/chmod"
printf '%s\n' '#!/bin/sh' 'for a in "$@"; do case "$a" in *.desktop.neu) echo "chmod-Attrappe: scheitert (gewollt)" >&2; exit 1 ;; esac; done' 'exec /usr/bin/chmod "$@"' > "$f8/chmod/chmod"
/usr/bin/chmod 0755 "$f8/chmod/chmod"
: > "$f8/go.log"
rc=0
env -i HOME="$f8/home" PATH="$f8/chmod:$stub:/usr/bin:/bin" STUB_LOG="$f8/go.log" STUB_FEHL_IN="" \
	GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null \
	bash "$f8/repo/install.sh" > "$f8/aus.txt" 2>&1 || rc=$?
[ "$rc" -ne 0 ] || fail "Fall 8: Abbruch beim Schreiben der Desktop-Datei endet mit Exit 0"
grep -q 'chmod-Attrappe: scheitert' "$f8/aus.txt" || fail "Fall 8: Attrappe lief nicht: $(cat "$f8/aus.txt")"
[ ! -e "$f8/home/.local/share/applications/schnapp.desktop.neu" ] || fail "Fall 8: schnapp.desktop.neu liegen geblieben"

if [ "$fehler" -eq 0 ]; then
	echo "install-test: alle Fälle bestanden"
else
	echo "install-test: FEHLGESCHLAGEN" >&2
	exit 1
fi
