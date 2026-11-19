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
if [ ! -d "$kopie" ]; then
	quelle="$(go list -m -f '{{.Dir}}' gioui.org)"
	mkdir -p "$wurzel/tmp"
	cp -r "$quelle" "$kopie"
	chmod -R u+w "$kopie"
	(cd "$kopie" && patch -p1 --quiet < "$hier/gio-ausgabe.patch")
fi
grep -q 'func FullscreenOn' "$kopie/app/window.go" || { echo "Patch fehlt in $kopie" >&2; exit 1; }

modfile="$wurzel/tmp/prototyp-gepatcht.mod"
cp go.mod "$modfile"
cp go.sum "${modfile%.mod}.sum"
printf '\nreplace gioui.org %s => %s\n' "$gio_version" "$kopie" >> "$modfile"

# novulkan: Vulkan-Header nicht nötig (OpenGL ES über EGL); nox11: nur Wayland (ADR-0007).
go build -modfile="$modfile" -tags novulkan,nox11 -o "$ziel" .
echo "gebaut: $ziel"
