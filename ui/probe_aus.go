//go:build !probe

package ui

// probeAutoWahl ist nur im Probe-Build an (probe_an.go); hier fällt der Zweig im Overlay beim Übersetzen weg.
const probeAutoWahl = false

// ProbeBuild meldet den Probe-Build (probe_an.go); schnapp --version hängt dann „+probe“ an (F-179).
const ProbeBuild = probeAutoWahl
