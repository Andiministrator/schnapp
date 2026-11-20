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

	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/library"
	"github.com/Andiministrator/schnapp/po"
	"github.com/Andiministrator/schnapp/settings"
	"github.com/Andiministrator/schnapp/ui"
	"github.com/Andiministrator/schnapp/wlclip"
)

// version wird beim Bau gesetzt: go build -ldflags "-X main.version=0.1.0".
var version = "0.0.0-dev"

// versionsText ist die Version für --version; der Probe-Build (Tag probe, ui.ProbeBuild) hängt „+probe“ an, damit er
// nicht für ein ausgeliefertes Programm gehalten wird (F-179).
func versionsText() string {
	if ui.ProbeBuild {
		return version + "+probe"
	}
	return version
}

// Exit-Codes.
const (
	exitOK     = 0
	exitFehler = 1 // Laufzeitfehler oder Modus noch nicht umgesetzt
	exitAufruf = 2 // Fehler in der Kommandozeile
	// exitAbgebrochen = 3 (aufnehmen.go): vom Nutzer abgebrochen
)

func main() {
	// Halter der Zwischenablage (ADR-0024): vom Editor gestartet, ohne Gio, ohne Einstellungen, ohne Sprache.
	if len(os.Args) == 2 && os.Args[1] == wlclip.HelferArgument {
		os.Exit(wlclip.HelferLauf())
	}
	// Scheitert das Laden der Sprache, bleibt Englisch aktiv; das Programm läuft trotzdem (i18n.Starte).
	_, _ = i18n.Starte(po.Dateien, "")
	// Gio braucht die Hauptschleife auf dem Haupt-Goroutine, sobald das Overlay ein Fenster öffnet.
	ui.Starte(func() int {
		return laufe(os.Args[1:], os.Stdout, os.Stderr, ui.Monitore, settings.Lade, echteAnschluesse().Ausfuehren)
	})
}

// Ladefunktion lädt die Einstellungen und liefert die Warnungen dazu (settings.Lade); sie kommt von außen, damit
// der Test ohne echte Datei auskommt.
type Ladefunktion func() (settings.Einstellungen, []string)

// laufe führt einen Aufruf aus und liefert den Exit-Code. Die Monitorabfrage und das Laden der Einstellungen kommen
// von außen, damit der Test ohne Wayland und ohne echte Datei auskommt. --version, --help und --list-screens lesen
// die Einstellungsdatei nicht (und warnen nicht über sie); nur die Aufnahme-Modi laden sie, geben jede Warnung als
// eine Zeile auf fehler aus und machen weiter (ADR-0018).
func laufe(args []string, aus, fehler io.Writer, monitore func() ([]ui.Monitor, error), lade Ladefunktion, ausfuehren Ausfuehrung) int {
	a, err := Auswerten(args)
	if err != nil {
		fmt.Fprintln(fehler, err)
		fmt.Fprintln(fehler, i18n.T("Try --help for the available options."))
		return exitAufruf
	}
	switch a.Modus {
	case ModusHilfe:
		fmt.Fprint(aus, hilfe())
		return exitOK
	case ModusVersion:
		fmt.Fprintf(aus, i18n.T("Schnapp %s")+"\n", versionsText())
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
	// Aufraeumen ist [library] cleanup/cleanup-days; es läuft nach dem Speichern, höchstens einmal am Tag
	// (ADR-0016 Nachtrag 2).
	Aufraeumen library.Einstellung
	// Editor: nach der Aufnahme den Editor öffnen ([capture] open-editor, außer --no-editor; ADR-0020).
	Editor bool
	// Datei: bei ModusOeffnen die zu öffnende Datei.
	Datei string
	// OhneGrenze: bei ModusOeffnen das Rechenbudget nicht prüfen (--ignore-limit, ADR-0017 Nachtrag 6).
	OhneGrenze bool
}

// Bildschirmziel sagt, welcher Monitor aufgenommen wird.
type Bildschirmziel struct {
	Name string                  // --screen=NAME; leer = laut Einstellung
	Wahl settings.Bildschirmwahl // nur bei leerem Namen maßgeblich
}

// plane löst den Aufruf mit den Einstellungen auf. Vorrang: Kommandozeile vor Datei vor Voreinstellung
// (ADR-0018): --screen=NAME ignoriert [capture] screen, --screen ohne Wert folgt ihr.
func plane(a Aufruf, e settings.Einstellungen) Auftrag {
	t := Auftrag{Modus: a.Modus, Aufraeumen: e.Bibliothek, Editor: a.Modus.Aufnahme() && e.Editor && !a.OhneEditor,
		Datei: a.Datei, OhneGrenze: a.OhneGrenze}
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
	return i18n.T("Usage: schnapp [option] | schnapp [--ignore-limit] FILE") + "\n\n" +
		i18n.T("Take a screenshot. Without an option, you select an area (--region).") + "\n\n" +
		"  --region       " + i18n.T("Select an area") + "\n" +
		"  --screen       " + i18n.T("Capture a monitor, as set in the settings (default: pick by click)") + "\n" +
		"  --screen=NAME  " + i18n.T("Capture the monitor NAME") + "\n" +
		"  --list-screens " + i18n.T("List the monitors") + "\n" +
		"  --window       " + i18n.T("Select a window") + "\n" +
		"  --repeat       " + i18n.T("Repeat the last area") + "\n" +
		"  --no-editor    " + i18n.T("Do not open the editor after capturing") + "\n" +
		"  FILE           " + i18n.T("Open FILE in the editor") + "\n" +
		"  --ignore-limit " + i18n.T("Open FILE even if it is too costly to render") + "\n" +
		"  -V, --version  " + i18n.T("Show the version") + "\n" +
		"  -h, --help     " + i18n.T("Show this help") + "\n"
}

// gibMonitoreAus schreibt die Monitorliste nach aus.
func gibMonitoreAus(aus, fehler io.Writer, monitore func() ([]ui.Monitor, error)) int {
	liste, err := monitore()
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Could not query the monitors: %v")+"\n", err)
		return exitFehler
	}
	if len(liste) == 0 {
		fmt.Fprintln(fehler, i18n.T("No monitors found."))
		return exitFehler
	}
	fmt.Fprintf(aus, i18n.N("%d monitor found:", "%d monitors found:", len(liste))+"\n", len(liste))
	for _, m := range liste {
		// Ohne Skala: die ganzzahlige Pufferskala (2 bei 1,6) führte in die Irre (F-57).
		fmt.Fprintf(aus, i18n.T("  %s: %d × %d pixels")+"\n", m.Name, m.Breite, m.Hoehe)
	}
	return exitOK
}
