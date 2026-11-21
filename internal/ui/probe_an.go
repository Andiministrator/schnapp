//go:build probe

package ui

import "os"

// probeAutoWahl: Probe-Build für automatische Läufe in einem virtuellen KWin (go build -tags probe). Das Overlay wählt
// dann ohne Eingabe den ganzen ersten Monitor. Nie in ausgelieferten Programmen (install.sh und CI bauen ohne Tag).
const probeAutoWahl = true

// ProbeBuild meldet den Probe-Build; schnapp --version hängt dann „+probe“ an (F-179).
const ProbeBuild = probeAutoWahl

// probeOhneFolgebildBeiZug schaltet im Probe-Build das Folgebild je Zeigerzug ab (SCHNAPP_PROBE_OHNE_FOLGEBILD=1), um
// es für die Messung F-235 zu vergleichen; im Normalbau gibt es die Weiche nicht.
var probeOhneFolgebildBeiZug = os.Getenv("SCHNAPP_PROBE_OHNE_FOLGEBILD") != ""
