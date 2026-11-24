// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !probe

package ui

// probeAutoWahl ist nur im Probe-Build an (probe_an.go); hier fällt der Zweig im Overlay beim Übersetzen weg.
const probeAutoWahl = false

// ProbeBuild meldet den Probe-Build (probe_an.go); schnapp --version hängt dann „+probe“ an (F-179).
const ProbeBuild = probeAutoWahl

// probeOhneFolgebildBeiZug gibt es nur im Probe-Build (probe_an.go); hier fällt der Zweig beim Übersetzen weg.
const probeOhneFolgebildBeiZug = false

// probeLeisteAus gibt es nur im Probe-Build (probe_an.go); hier fällt der Zweig beim Übersetzen weg.
const probeLeisteAus = false
