#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
#
# Installationsskript für Schnapp (Karte #42, ADR-0017).
#
#   ./install.sh              baut und installiert
#   ./install.sh --uninstall  entfernt die drei installierten Dateien
#   ./install.sh --help       zeigt diese Hilfe
#
# Umgebung: PREFIX (Vorgabe $HOME/.local), XDG_DATA_HOME (Vorgabe $HOME/.local/share).
# Installiert werden drei Dateien:
#   $PREFIX/bin/schnapp
#   $XDG_DATA_HOME/applications/schnapp.desktop
#   $XDG_DATA_HOME/icons/hicolor/scalable/apps/schnapp.svg
# Dazu aktualisieren vorhandene Desktop-Werkzeuge ihre Caches (Menü-Datenbank; den Symbol-Cache nur, wenn es ihn
# schon gibt). --uninstall entfernt nur die drei Dateien.
#
# Löschregel (Projekt): rm nur mit nicht leerem Ziel, das genau Präfix + wörtlich ausgeschriebenes Suffix ist. Das
# Präfix selbst kommt aus PREFIX/XDG_DATA_HOME und ist damit Sache des Aufrufers — wer PREFIX=/usr als root setzt,
# entfernt /usr/bin/schnapp. Mehr als diese drei fest benannten Dateien kann das Skript nicht löschen.
set -eu

case "${1:-}" in -h | --help) ;; *) [ "$#" -le 1 ] || { echo "install.sh: zu viele Argumente" >&2; exit 2; } ;; esac

hilfe() {
	cat <<'EOF'
Usage: ./install.sh [--uninstall | --help]

  (no argument)  build schnapp and install it for the current user
  --uninstall    remove the three installed files (program, desktop file, icon)
  --help         show this help

Environment:
  PREFIX         install prefix for bin/ (default: $HOME/.local)
  XDG_DATA_HOME  data directory (default: $HOME/.local/share)
EOF
}

case "${1:-}" in
	-h | --help) hilfe; exit 0 ;;
esac

# --- Ziele ------------------------------------------------------------------
[ -n "${HOME:-}" ] || { echo "install.sh: HOME ist nicht gesetzt" >&2; exit 1; }
PREFIX="${PREFIX:-$HOME/.local}"
DATEN="${XDG_DATA_HOME:-$HOME/.local/share}"
case "$PREFIX" in /*) ;; *) echo "install.sh: PREFIX muss ein absoluter Pfad sein: $PREFIX" >&2; exit 1 ;; esac
case "$DATEN" in /*) ;; *) echo "install.sh: XDG_DATA_HOME muss ein absoluter Pfad sein: $DATEN" >&2; exit 1 ;; esac

PROGRAMM="$PREFIX/bin/schnapp"
DESKTOP="$DATEN/applications/schnapp.desktop"
SYMBOL="$DATEN/icons/hicolor/scalable/apps/schnapp.svg"

wurzel="$(cd "$(dirname "$0")" && pwd)"

# Entfernt genau eine Datei. Prüft vor rm: nicht leer, beginnt mit dem
# wörtlich übergebenen Präfix (PREFIX bzw. XDG_DATA_HOME) plus ausgeschriebenem Suffix.
# Nie Verzeichnisse, nie Globs.
entferne() {
	ziel="$1" praefix="$2" suffix="$3"
	[ -n "$ziel" ] && [ -n "$praefix" ] && [ -n "$suffix" ] || exit 1
	case "$praefix" in /?*) ;; *) echo "install.sh: Präfix verworfen: $praefix" >&2; exit 1 ;; esac
	[ "$ziel" = "$praefix$suffix" ] || { echo "install.sh: Ziel verworfen: $ziel" >&2; exit 1; }
	case "$ziel" in *'*'* | *'?'* | *'['*) echo "install.sh: Ziel verworfen: $ziel" >&2; exit 1 ;; esac
	if [ -f "$ziel" ] || [ -L "$ziel" ]; then
		rm -f -- "$ziel"
		echo "entfernt: $ziel"
	else
		echo "nicht vorhanden: $ziel"
	fi
}

# Hilfsprogramme für Menü und Symbolcache; Fehler sind nur Warnungen.
cache_aktualisieren() {
	if command -v update-desktop-database >/dev/null 2>&1; then
		update-desktop-database "$DATEN/applications" >/dev/null 2>&1 \
			|| echo "Warnung: update-desktop-database fehlgeschlagen" >&2
	fi
	# Nur einen vorhandenen Symbol-Cache auffrischen: einen neuen anzulegen, ließe GTK später Symbole anderer
	# Programme übersehen, solange der Cache nicht neu gebaut wird (Kritik B-3).
	if command -v gtk-update-icon-cache >/dev/null 2>&1 && [ -f "$DATEN/icons/hicolor/icon-theme.cache" ]; then
		gtk-update-icon-cache -f -t "$DATEN/icons/hicolor" >/dev/null 2>&1 \
			|| echo "Warnung: gtk-update-icon-cache fehlgeschlagen" >&2
	fi
	if command -v kbuildsycoca6 >/dev/null 2>&1; then
		kbuildsycoca6 >/dev/null 2>&1 || echo "Warnung: kbuildsycoca6 fehlgeschlagen" >&2
	fi
}

deinstallieren() {
	entferne "$PROGRAMM" "$PREFIX" "/bin/schnapp"
	entferne "$DESKTOP" "$DATEN" "/applications/schnapp.desktop"
	entferne "$SYMBOL" "$DATEN" "/icons/hicolor/scalable/apps/schnapp.svg"
	cache_aktualisieren
}

installieren() {
	vorlage="$wurzel/data/schnapp.desktop.in"
	quelle_symbol="$wurzel/brand/logo/svg/schnapp-symbol-dunkel.svg"
	[ -f "$vorlage" ] || { echo "install.sh: Vorlage fehlt: $vorlage" >&2; exit 1; }
	[ -f "$quelle_symbol" ] || { echo "install.sh: Symbol fehlt: $quelle_symbol" >&2; exit 1; }
	# Ohne exakte Zeile [Desktop Entry] (kein Leerzeichen am Ende, kein CRLF) bliebe die Desktop-Datei leer, bei
	# Erfolgsmeldung (F-122). Deshalb schon hier prüfen, bevor gebaut oder etwas installiert wird.
	grep -qxF '[Desktop Entry]' "$vorlage" \
		|| { echo "install.sh: Vorlage hat keine Zeile [Desktop Entry] (exakt, ohne Leerzeichen am Ende oder CRLF): $vorlage" >&2; exit 1; }
	command -v go >/dev/null 2>&1 || { echo "install.sh: go wurde nicht gefunden" >&2; exit 1; }

	# Der Pfad landet in Exec=. Nach der Desktop-Entry-Spezifikation müssten ", `, $ und \ maskiert werden;
	# das unterstützen wir nicht. % wird zu %% verdoppelt (Feldcode).
	case "$PROGRAMM" in
		*'"'* | *'`'* | *'$'* | *'\'*)
			echo "install.sh: Pfad mit \" \` \$ oder \\ wird nicht unterstützt: $PROGRAMM" >&2; exit 1 ;;
	esac
	case "$PROGRAMM" in *$'\n'*) echo "install.sh: Pfad mit Zeilenumbruch" >&2; exit 1 ;; esac

	# Version aus git, ohne führendes v (ADR-0006). Nur ein Tag dieses Repos zählt: Liegt der Ordner in einem fremden
	# Repo (Tarball im Home-Repo), gilt dessen Stand nicht. Nur v-Tags zählen (F-364), ein Tag wie sprint-ende
	# würde sonst als Programmversion landen. Ohne erreichbaren Tag (flacher Klon, Tarball) entsteht eine
	# gültige SemVer-Fassung: 0.0.0-dev+<Kurzhash>[.dirty], ohne git 0.0.0-dev (F-113).
	version=""
	if command -v git >/dev/null 2>&1 \
		&& [ "$(git -C "$wurzel" rev-parse --show-toplevel 2>/dev/null || true)" = "$(cd "$wurzel" && pwd -P)" ]; then
		version="$(git -C "$wurzel" describe --tags --match 'v[0-9]*' --dirty 2>/dev/null || true)"
		if [ -z "$version" ]; then
			kurz="$(git -C "$wurzel" rev-parse --short HEAD 2>/dev/null || true)"
			if [ -n "$kurz" ]; then
				version="0.0.0-dev+$kurz"
				[ -z "$(git -C "$wurzel" status --porcelain --untracked-files=no 2>/dev/null || true)" ] || version="$version.dirty"
			fi
		fi
	fi
	version="${version#v}"
	[ -n "$version" ] || version="0.0.0-dev"

	mkdir -p -- "$PREFIX/bin" "$DATEN/applications" "$DATEN/icons/hicolor/scalable/apps"

	echo "baue schnapp $version ..."
	# Erst in eine Nebendatei bauen, dann umbenennen: ein laufendes Programm lässt sich so ersetzen.
	neu="$PROGRAMM.neu"
	(cd "$wurzel" && GOFLAGS="-tags=novulkan,nox11" go build -trimpath -ldflags "-X main.version=$version" -o "$neu" ./cmd/schnapp)
	chmod 0755 "$neu"
	mv -f -- "$neu" "$PROGRAMM"
	echo "installiert: $PROGRAMM"

	# Desktop-Datei: @EXEC@ durch den absoluten Pfad ersetzen (% verdoppelt). Reine Shell-Ersetzung,
	# damit &, | und \ im Pfad nichts verfälschen. KWin entfernt die Anführungszeichen (QProcess::splitCommand),
	# siehe knowledge/topics/kwin-screenshot.md (F-76), deshalb bleiben sie in der Vorlage.
	exec_pfad="${PROGRAMM//%/%%}"
	tmp="$DESKTOP.neu"
	: > "$tmp"
	im_eintrag=0
	while IFS= read -r zeile || [ -n "$zeile" ]; do
		# Die Kommentare vor [Desktop Entry] sind Doku der Vorlage, nicht der installierten Datei (Kritik B-7).
		if [ "$im_eintrag" = 0 ]; then
			case "$zeile" in '[Desktop Entry]') im_eintrag=1 ;; *) continue ;; esac
		fi
		while :; do
			case "$zeile" in
				*@EXEC@*) zeile="${zeile%%@EXEC@*}$exec_pfad${zeile#*@EXEC@}" ;;
				*) break ;;
			esac
			# Schutz vor Endlosschleife, falls der Pfad selbst @EXEC@ enthielte
			case "$exec_pfad" in *@EXEC@*) echo "install.sh: ungültiger Pfad" >&2; exit 1 ;; esac
		done
		printf '%s\n' "$zeile" >> "$tmp"
	done < "$vorlage"
	# Zweite Sicherung: Ergebnis muss den Eintrag enthalten und nicht leer sein. Sonst die Zwischendatei weg
	# (entferne prüft Präfix und Suffix) und abbrechen; die alte Desktop-Datei bleibt unberührt.
	if [ "$im_eintrag" != 1 ] || [ ! -s "$tmp" ]; then
		entferne "$tmp" "$DATEN" "/applications/schnapp.desktop.neu"
		echo "install.sh: Desktop-Datei würde leer; bestehende Datei bleibt unverändert" >&2
		exit 1
	fi
	chmod 0644 "$tmp"
	mv -f -- "$tmp" "$DESKTOP"
	echo "installiert: $DESKTOP"

	cp -f -- "$quelle_symbol" "$SYMBOL"
	chmod 0644 "$SYMBOL"
	echo "installiert: $SYMBOL"

	cache_aktualisieren
	echo "Fertig."
	case ":$PATH:" in *":$PREFIX/bin:"*) ;; *) echo "Hinweis: $PREFIX/bin liegt nicht in PATH." ;; esac
}

case "${1:-}" in
	"") installieren ;;
	--uninstall) deinstallieren ;;
	-h | --help) hilfe ;;
	*) hilfe >&2; exit 2 ;;
esac
