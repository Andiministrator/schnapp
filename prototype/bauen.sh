#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-or-later
# Baut den Prototyp. Das gepatchte Gio liegt im Repo unter third_party/gio (ADR-0015); go.mod ersetzt gioui.org
# dorthin, ein gewöhnliches `go build` genügt also. Das Skript setzt nur Build-Tags, -trimpath und das Ziel.
#
# Aufruf: prototype/bauen.sh [Zieldatei]   (Vorgabe: ~/.local/bin/schnapp-prototype)
set -euo pipefail
hier="$(cd "$(dirname "$0")" && pwd)"
ziel="$(realpath -m -- "${1:-$HOME/.local/bin/schnapp-prototype}")"

cd "$hier"
# novulkan: Vulkan-Header nicht nötig (OpenGL ES über EGL); nox11: nur Wayland (ADR-0007).
# -trimpath: keine lokalen Pfade im Programm (Kernprinzip „keine Spuren nach außen“).
go build -trimpath -tags novulkan,nox11 -o "$ziel" .
echo "gebaut: $ziel"
