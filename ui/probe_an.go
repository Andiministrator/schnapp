//go:build probe

package ui

// probeAutoWahl: Probe-Build für automatische Läufe in einem virtuellen KWin (go build -tags probe). Das Overlay wählt
// dann ohne Eingabe den ganzen ersten Monitor. Nie in ausgelieferten Programmen (install.sh und CI bauen ohne Tag).
const probeAutoWahl = true
