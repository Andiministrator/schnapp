#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Baut den Prototyp mit gepatchtem Gio (gio-ausgabe.patch: Vollbild auf gewählter Wayland-Ausgabe,
# Ausgaben auflisten). Die gepatchte Kopie liegt in tmp/gio-gepatcht (gitignoriert); go.mod bleibt unverändert,
# die Ersetzung steht nur in einer erzeugten Modul-Datei unter tmp/.
#
# Aufruf: prototyp/bauen.sh [Zieldatei]   (Vorgabe: ~/.local/bin/schnapp-prototyp)
set -euo pipefail
hier="$(cd "$(dirname "$0")" && pwd)"
wurzel="$(dirname "$hier")"
ziel="${1:-$HOME/.local/bin/schnapp-prototyp}"
kopie="$wurzel/tmp/gio-gepatcht"

cd "$hier"
gio_version="$(go list -m -f '{{.Version}}' gioui.org)"
# Die Kopie gilt nur, wenn sie aus genau diesem Patch und dieser Gio-Version entstand (Marke in der Kopie).
# Sonst — geänderter Patch, andere Version oder ein abgebrochener Lauf ohne Marke — wird sie in den
# Papierkorb unter tmp/ verschoben (nicht gelöscht) und neu angelegt.
soll="$gio_version $(sha256sum "$hier/gio-ausgabe.patch" | cut -d' ' -f1)"
marke="$kopie/.schnapp-patch-stand"
if [ -d "$kopie" ] && [ "$(cat "$marke" 2>/dev/null)" != "$soll" ]; then
	papierkorb="$wurzel/tmp/papierkorb"
	mkdir -p "$papierkorb"
	mv "$kopie" "$papierkorb/gio-gepatcht-$(date +%Y%m%d-%H%M%S)"
	echo "veraltete Gio-Kopie nach $papierkorb verschoben"
fi
if [ ! -d "$kopie" ]; then
	quelle="$(go list -m -f '{{.Dir}}' gioui.org)"
	mkdir -p "$wurzel/tmp"
	cp -r "$quelle" "$kopie"
	chmod -R u+w "$kopie"
	(cd "$kopie" && patch -p1 --quiet < "$hier/gio-ausgabe.patch")
	for probe in 'func FullscreenOn' 'func WaylandOutputs' 'wp_fractional_scale_v1'; do
		grep -rq "$probe" "$kopie/app" || { echo "Patch unvollständig in $kopie: $probe fehlt" >&2; exit 1; }
	done
	printf '%s\n' "$soll" > "$marke"
fi

modfile="$wurzel/tmp/prototyp-gepatcht.mod"
cp go.mod "$modfile"
cp go.sum "${modfile%.mod}.sum"
printf '\nreplace gioui.org %s => %s\n' "$gio_version" "$kopie" >> "$modfile"

# novulkan: Vulkan-Header nicht nötig (OpenGL ES über EGL); nox11: nur Wayland (ADR-0007).
go build -modfile="$modfile" -tags novulkan,nox11 -o "$ziel" .
echo "gebaut: $ziel"
