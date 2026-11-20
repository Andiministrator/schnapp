// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !probe

package main

// probeSchutz ist im ausgelieferten Programm leer; die Prüfung gilt nur im Probe-Build (probe_an.go, F-229).
func probeSchutz() error { return nil }
