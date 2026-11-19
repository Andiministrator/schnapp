#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Hält die gepatchte Gio-Kopie extern/gio und den Patch extern/gio-ausgabe.patch zusammen (ADR-0015).
#
#   extern/gio-patch.sh pruefen    unverändertes Gio aus dem Modulcache + Patch == extern/gio? (CI)
#   extern/gio-patch.sh erzeugen   Patch aus extern/gio neu schreiben, nachdem dort geändert wurde
#   extern/gio-patch.sh anlegen    extern/gio aus unverändertem Gio + Patch neu aufbauen (Gio-Update)
#
# Gio-Version: GIO_VERSION unten; muss zur require-Zeile der Module passen, die gioui.org ersetzen.
set -euo pipefail

GIO_VERSION="v0.10.3"

hier="$(cd "$(dirname "$0")" && pwd)"
wurzel="$(dirname "$hier")"
kopie="$hier/gio"
patchdatei="$hier/gio-ausgabe.patch"

befehl="${1:-}"
case "$befehl" in
pruefen | erzeugen | anlegen) ;;
*)
	echo "Aufruf: $0 pruefen|erzeugen|anlegen" >&2
	exit 2
	;;
esac

# Unverändertes Gio aus dem Modulcache holen (lädt es bei Bedarf herunter; Prüfsumme über sum.golang.org).
# Im Wurzelverzeichnis des Systems ausführen (kein go.mod), sonst könnte ein replace auf extern/gio greifen.
quelle="$(go -C / mod download -json "gioui.org@$GIO_VERSION" | sed -n 's/^\t"Dir": "\(.*\)",$/\1/p')"
[ -n "$quelle" ] && [ -f "$quelle/go.mod" ] || {
	echo "Gio $GIO_VERSION nicht im Modulcache gefunden" >&2
	exit 1
}

# Arbeitsverzeichnis unter tmp/ im Projekt; wird am Ende entfernt (nur dieses, geprüft gegen das Muster).
mkdir -p "$wurzel/tmp"
lauf="$(mktemp -d "$wurzel/tmp/gio-patch.XXXXXX")"
aufraeumen() {
	[ -n "$lauf" ] || return 0
	case "$lauf" in
	*/tmp/gio-patch.??????) rm -rf --one-file-system -- "$lauf" ;;
	*) echo "Arbeitsverzeichnis nicht entfernt (unerwarteter Pfad): $lauf" >&2 ;;
	esac
}
trap aufraeumen EXIT

# Unveränderte Kopie als git-Arbeitsbaum: Grundlage für Patch anwenden und Patch erzeugen.
basis="$lauf/basis"
cp -r "$quelle" "$basis"
chmod -R u+w "$basis"
git -C "$basis" init -q
git -C "$basis" -c user.name=schnapp -c user.email=schnapp@invalid add -A
git -C "$basis" -c user.name=schnapp -c user.email=schnapp@invalid -c commit.gpgsign=false -c core.hooksPath=/dev/null commit -q --no-verify -m basis

case "$befehl" in
pruefen)
	git -C "$basis" apply --whitespace=nowarn "$patchdatei"
	# diff -r vergleicht keine Dateirechte; Ausführbits deshalb getrennt als Liste vergleichen.
	ausfuehrbar() { (cd "$1" && find . -path ./.git -prune -o -type f -perm -u+x -print | sort); }
	if diff -r --exclude=.git "$basis" "$kopie" >"$lauf/abweichung" &&
		diff <(ausfuehrbar "$basis") <(ausfuehrbar "$kopie") >>"$lauf/abweichung"; then
		echo "extern/gio entspricht Gio $GIO_VERSION + extern/gio-ausgabe.patch"
	else
		echo "extern/gio weicht von Gio $GIO_VERSION + Patch ab:" >&2
		cat "$lauf/abweichung" >&2
		echo "Nach Änderungen in extern/gio: $0 erzeugen" >&2
		exit 1
	fi
	;;
erzeugen)
	# extern/gio kopieren und die git-Geschichte der Basis hineinlegen: der Diff ist dann genau der Patch.
	neu="$lauf/neu"
	cp -r "$kopie" "$neu"
	mv "$basis/.git" "$neu/.git"
	git -C "$neu" add -A
	# Feste Diff-Form unabhängig von der git-Konfiguration des Nutzers (Präfixe, Farbe, externe Diff-Programme,
	# Umbenennungen); erst in tmp/ schreiben, dann ersetzen — bricht git ab, bleibt der alte Patch stehen.
	git -C "$neu" -c diff.noprefix=false -c diff.mnemonicPrefix=false -c color.ui=never -c core.quotePath=false \
		diff --cached --binary --no-ext-diff --no-color --no-renames --src-prefix=a/ --dst-prefix=b/ >"$lauf/neu.patch"
	git apply --whitespace=nowarn --numstat "$lauf/neu.patch"
	mv "$lauf/neu.patch" "$patchdatei"
	echo "geschrieben: extern/gio-ausgabe.patch"
	;;
anlegen)
	[ ! -e "$kopie" ] || {
		echo "$kopie besteht schon — vorher nach tmp/papierkorb/ verschieben" >&2
		exit 1
	}
	git -C "$basis" apply --whitespace=nowarn "$patchdatei"
	mkdir -p "$kopie"
	cp -r "$basis/." "$kopie/"
	# Das .git der Basis gehört nicht in die Kopie (verschieben statt löschen).
	mkdir -p "$wurzel/tmp/papierkorb"
	mv "$kopie/.git" "$wurzel/tmp/papierkorb/gio-basis-git-$(date +%Y%m%d-%H%M%S)"
	echo "angelegt: extern/gio (Gio $GIO_VERSION + Patch)"
	;;
esac
