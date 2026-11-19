// SPDX-License-Identifier: GPL-3.0-or-later

// Command schnapp ist das Screenshot- und Beschriftungswerkzeug (siehe README). Dieses Gerüst wertet die Argumente aus;
// die Aufnahme-Modi folgen mit den nächsten Karten.
//
// Bauen mit den Gio-Build-Tags novulkan und nox11, siehe CONTRIBUTING.md.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Andiministrator/schnapp/oberflaeche"
	"github.com/Andiministrator/schnapp/po"
	"github.com/Andiministrator/schnapp/sprache"
)

// version wird beim Bau gesetzt: go build -ldflags "-X main.version=0.1.0".
var version = "0.0.0-dev"

// Exit-Codes.
const (
	exitOK     = 0
	exitFehler = 1 // Laufzeitfehler oder Modus noch nicht umgesetzt
	exitAufruf = 2 // Fehler in der Kommandozeile
)

func main() {
	// Scheitert das Laden der Sprache, bleibt Englisch aktiv; das Programm läuft trotzdem (sprache.Starte).
	_, _ = sprache.Starte(po.Dateien, "")
	os.Exit(laufe(os.Args[1:], os.Stdout, os.Stderr, oberflaeche.Monitore))
}

// laufe führt einen Aufruf aus und liefert den Exit-Code. Die Monitorabfrage kommt von außen, damit der Test ohne
// Wayland auskommt.
func laufe(args []string, aus, fehler io.Writer, monitore func() ([]oberflaeche.Monitor, error)) int {
	a, err := Auswerten(args)
	if err != nil {
		fmt.Fprintln(fehler, err)
		fmt.Fprintln(fehler, sprache.T("Try --help for the available options."))
		return exitAufruf
	}
	switch a.Modus {
	case ModusHilfe:
		fmt.Fprint(aus, hilfe())
		return exitOK
	case ModusVersion:
		fmt.Fprintf(aus, sprache.T("Schnapp %s")+"\n", version)
		return exitOK
	case ModusMonitorliste:
		return gibMonitoreAus(aus, fehler, monitore)
	}
	fmt.Fprintf(fehler, sprache.T("The capture mode %s is not implemented yet.")+"\n", a.Modus.Option())
	return exitFehler
}

// hilfe liefert den Hilfetext.
func hilfe() string {
	return sprache.T("Usage: schnapp [option]") + "\n\n" +
		sprache.T("Take a screenshot. Without an option, you select an area (--region).") + "\n\n" +
		"  --region       " + sprache.T("Select an area") + "\n" +
		"  --screen       " + sprache.T("Capture a monitor, as set in the settings (default: pick by click)") + "\n" +
		"  --screen=NAME  " + sprache.T("Capture the monitor NAME") + "\n" +
		"  --list-screens " + sprache.T("List the monitors") + "\n" +
		"  --window       " + sprache.T("Select a window") + "\n" +
		"  --repeat       " + sprache.T("Repeat the last area") + "\n" +
		"  -V, --version  " + sprache.T("Show the version") + "\n" +
		"  -h, --help     " + sprache.T("Show this help") + "\n"
}

// gibMonitoreAus schreibt die Monitorliste nach aus.
func gibMonitoreAus(aus, fehler io.Writer, monitore func() ([]oberflaeche.Monitor, error)) int {
	liste, err := monitore()
	if err != nil {
		fmt.Fprintf(fehler, sprache.T("Could not query the monitors: %v")+"\n", err)
		return exitFehler
	}
	if len(liste) == 0 {
		fmt.Fprintln(fehler, sprache.T("No monitors found."))
		return exitFehler
	}
	fmt.Fprintf(aus, sprache.N("%d monitor found:", "%d monitors found:", len(liste))+"\n", len(liste))
	for _, m := range liste {
		// Ohne Skala: die ganzzahlige Pufferskala (2 bei 1,6) führte in die Irre (F-57).
		fmt.Fprintf(aus, sprache.T("  %s: %d × %d pixels")+"\n", m.Name, m.Breite, m.Hoehe)
	}
	return exitOK
}
