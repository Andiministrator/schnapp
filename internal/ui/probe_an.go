// SPDX-License-Identifier: GPL-3.0-or-later

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

// probeLeisteAus: Im Drehbuch-Lauf (scripts/probe.sh setzt SCHNAPP_PROBE_LEISTE_AUS=1) startet die Leiste der letzten
// Aufnahmen ausgeschaltet, und ihr Schalter wird nicht in settings.ini gemerkt (ADR-0036 Punkt 2, Lehre aus F-539):
// Drehbücher mit festen Leinwandkoordinaten bleiben gültig, und ein Umschalten kippt den nächsten Lauf nicht.
var probeLeisteAus = os.Getenv("SCHNAPP_PROBE_LEISTE_AUS") != ""
