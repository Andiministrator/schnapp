// SPDX-License-Identifier: GPL-3.0-or-later

// Command schnapp ist das Screenshot- und Beschriftungswerkzeug (siehe README). Es wertet die Argumente aus, nimmt
// auf (Overlay für Bereich und Monitor, aufnehmen.go) und legt die Aufnahme in der Bibliothek ab.
//
// Bauen mit den Gio-Build-Tags novulkan und nox11, siehe CONTRIBUTING.md.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Andiministrator/schnapp/einstellungen"
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
	// exitAbgebrochen = 3 (aufnehmen.go): vom Nutzer abgebrochen
)

func main() {
	// Scheitert das Laden der Sprache, bleibt Englisch aktiv; das Programm läuft trotzdem (sprache.Starte).
	_, _ = sprache.Starte(po.Dateien, "")
	// Gio braucht die Hauptschleife auf dem Haupt-Goroutine, sobald das Overlay ein Fenster öffnet.
	oberflaeche.Starte(func() int {
		return laufe(os.Args[1:], os.Stdout, os.Stderr, oberflaeche.Monitore, einstellungen.Lade, echteAnschluesse().Ausfuehren)
	})
}

// Ladefunktion lädt die Einstellungen und liefert die Warnungen dazu (einstellungen.Lade); sie kommt von außen, damit
// der Test ohne echte Datei auskommt.
type Ladefunktion func() (einstellungen.Einstellungen, []string)

// laufe führt einen Aufruf aus und liefert den Exit-Code. Die Monitorabfrage und das Laden der Einstellungen kommen
// von außen, damit der Test ohne Wayland und ohne echte Datei auskommt. --version, --help und --list-screens lesen
// die Einstellungsdatei nicht (und warnen nicht über sie); nur die Aufnahme-Modi laden sie, geben jede Warnung als
// eine Zeile auf fehler aus und machen weiter (ADR-0018).
func laufe(args []string, aus, fehler io.Writer, monitore func() ([]oberflaeche.Monitor, error), lade Ladefunktion, ausfuehren Ausfuehrung) int {
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
	einst, warnungen := lade()
	for _, w := range warnungen {
		fmt.Fprintln(fehler, w)
	}
	return ausfuehren(plane(a, einst), aus, fehler)
}

// Auftrag ist der Aufruf samt aufgelöstem Bildschirmziel: das, was ein Aufnahme-Modus später ausführt.
type Auftrag struct {
	Modus Modus
	// Ziel gilt nur bei ModusBildschirm: Name != "" ist ein bestimmter Monitor (--screen=NAME), sonst entscheidet
	// Wahl (Einstellung [capture] screen).
	Ziel Bildschirmziel
}

// Bildschirmziel sagt, welcher Monitor aufgenommen wird.
type Bildschirmziel struct {
	Name string                       // --screen=NAME; leer = laut Einstellung
	Wahl einstellungen.Bildschirmwahl // nur bei leerem Namen maßgeblich
}

// plane löst den Aufruf mit den Einstellungen auf. Vorrang: Kommandozeile vor Datei vor Voreinstellung
// (ADR-0018): --screen=NAME ignoriert [capture] screen, --screen ohne Wert folgt ihr.
func plane(a Aufruf, e einstellungen.Einstellungen) Auftrag {
	t := Auftrag{Modus: a.Modus}
	if a.Modus == ModusBildschirm {
		if a.Bildschirm != "" {
			t.Ziel = Bildschirmziel{Name: a.Bildschirm}
		} else {
			t.Ziel = Bildschirmziel{Wahl: e.Bildschirm}
		}
	}
	return t
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
