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

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/po"
	"github.com/Andiministrator/schnapp/internal/settings"
	"github.com/Andiministrator/schnapp/internal/ui"
	"github.com/Andiministrator/schnapp/internal/wlclip"
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
	if code, ok := halter(os.Args[1:], os.Stderr, wlclip.HelferLauf); ok {
		os.Exit(code)
	}
	starteSprache(settings.Lade)
	// Gio braucht die Hauptschleife auf dem Haupt-Goroutine, sobald das Overlay ein Fenster öffnet.
	ui.Starte(func() int {
		// MCP-Server und sein Bestätigungsdialog (ADR-0030 Nachtrag 2): nur genau so aufgerufen, ohne weitere
		// Argumente; der Dialog braucht die Gio-Hauptschleife, der Server die Monitorabfrage.
		switch args := os.Args[1:]; {
		case len(args) == 1 && args[0] == mcpArgument:
			return laufeMcp(os.Stdin, os.Stdout, os.Stderr, echteMcpAnschluesse(os.Stderr))
		case len(args) == 1 && args[0] == bestaetigenArgument:
			return laufeBestaetigung(os.Stdin, os.Stderr)
		}
		return laufe(os.Args[1:], os.Stdout, os.Stderr, ui.Monitore, settings.Lade, echteAnschluesse().Ausfuehren)
	})
}

// starteSprache setzt die Sprache der Oberfläche: aus [general] language, sonst aus der Umgebung (ADR-0014 Nachtrag
// 2). Die Warnungen über die Einstellungsdatei gibt erst laufe aus (übersetzt, nur bei Aufnahmen); hier zählt nur die
// Sprache. Scheitert das Laden der Sprache, bleibt Englisch aktiv; das Programm läuft trotzdem (i18n.Starte). lade
// kommt von außen, damit der Test ohne echte Datei auskommt (KB-11).
func starteSprache(lade Ladefunktion) {
	einst, _ := lade()
	_, _ = i18n.Starte(po.Dateien, einst.Sprache)
}

// halter führt den Halter der Zwischenablage aus (ADR-0024), wenn args genau das Helfer-Argument ist: vom Editor
// gestartet, ohne Gio, ohne Einstellungen, ohne Sprache. ok ist false bei jedem anderen Aufruf. Der Probe-Build läuft
// nur im virtuellen KWin (F-229); lauf kommt von außen, damit der Test den Schutz ohne echte Zwischenablage prüft
// (F-244).
func halter(args []string, fehler io.Writer, lauf func() int) (code int, ok bool) {
	if len(args) != 1 || args[0] != wlclip.HelferArgument {
		return 0, false
	}
	if err := probeGuard(); err != nil {
		fmt.Fprintln(fehler, "schnapp (Probe-Build):", err)
		return exitFehler, true
	}
	return lauf(), true
}

// Ladefunktion lädt die Einstellungen und liefert die Warnungen dazu (settings.Lade); sie kommt von außen, damit
// der Test ohne echte Datei auskommt.
type Ladefunktion func() (settings.Einstellungen, []string)

// laufe führt einen Aufruf aus und liefert den Exit-Code. Die Monitorabfrage und das Laden der Einstellungen kommen
// von außen, damit der Test ohne Wayland und ohne echte Datei auskommt. --version, --help, --list-screens und
// --export warnen nicht über die Einstellungsdatei (main liest aus ihr vorher nur die Sprache); nur die Aufnahme-Modi
// laden sie hier, geben jede Warnung als eine Zeile auf fehler aus und machen weiter (ADR-0018).
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
		if a.JSON {
			return gibJSONAus(aus, fehler, struct {
				Version string `json:"version"`
			}{versionsText()})
		}
		fmt.Fprintf(aus, i18n.T("Schnapp %s")+"\n", versionsText())
		return exitOK
	case ModusMonitorliste:
		return gibMonitoreAus(aus, fehler, monitore, a.JSON)
	case ModusExport:
		return exportiere(a, aus, fehler)
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
	// Ausgabe: --output=DATEI, die Aufnahme dorthin statt in die Bibliothek, ohne Editor (ADR-0030 Punkt 2).
	Ausgabe string
	// JSON: --json, das Ergebnis als eine Zeile JSON (ADR-0030 Punkt 1).
	JSON bool
	// Kante: [capture] seam, die Form der Stauch-Kante für neue Dokumente — die Aufnahme und ein einfaches PNG/JPEG,
	// das ModusOeffnen öffnet (ADR-0023 Nachtrag 6). Eine Arbeitsdatei behält ihre eigene.
	Kante document.Kante
}

// Bildschirmziel sagt, welcher Monitor aufgenommen wird.
type Bildschirmziel struct {
	Name string                  // --screen=NAME; leer = laut Einstellung
	Wahl settings.Bildschirmwahl // nur bei leerem Namen maßgeblich
}

// plane löst den Aufruf mit den Einstellungen auf. Vorrang: Kommandozeile vor Datei vor Voreinstellung
// (ADR-0018): --screen=NAME ignoriert [capture] screen, --screen ohne Wert folgt ihr.
func plane(a Aufruf, e settings.Einstellungen) Auftrag {
	t := Auftrag{Modus: a.Modus, Aufraeumen: e.Bibliothek,
		Editor: a.Modus.Aufnahme() && e.Editor && !a.OhneEditor && a.Ausgabe == "",
		Datei:  a.Datei, OhneGrenze: a.OhneGrenze, Ausgabe: a.Ausgabe, JSON: a.JSON, Kante: e.Kante}
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
	return i18n.T("Usage: schnapp [option] [--json] [--output=FILE] | schnapp [--ignore-limit] [--export=TARGET [--json]] FILE") + "\n\n" +
		i18n.T("Take a screenshot. Without an option, you select an area (--region).") + "\n\n" +
		"  --region       " + i18n.T("Select an area") + "\n" +
		"  --screen       " + i18n.T("Capture a monitor, as set in the settings (default: pick by click)") + "\n" +
		"  --screen=NAME  " + i18n.T("Capture the monitor NAME") + "\n" +
		"  --list-screens " + i18n.T("List the monitors") + "\n" +
		"  --window       " + i18n.T("Select a window") + "\n" +
		"  --repeat       " + i18n.T("Repeat the last area") + "\n" +
		"  --no-editor    " + i18n.T("Do not open the editor after capturing") + "\n" +
		"  --output=FILE  " + i18n.T("Write the screenshot to FILE (.png, .jpg, .svg) instead of the library") + "\n" +
		"  --json         " + i18n.T("Print the result as one line of JSON") + "\n" +
		"  FILE           " + i18n.T("Open FILE in the editor") + "\n" +
		"  --export=TARGET FILE " + i18n.T("Export FILE to TARGET (.png, .jpg, .svg) without a window") + "\n" +
		"  --ignore-limit " + i18n.T("Open FILE even if it is too costly to render") + "\n" +
		"  mcp            " + i18n.T("Run as MCP server for AI agents (standard input and output only)") + "\n" +
		"  -V, --version  " + i18n.T("Show the version") + "\n" +
		"  -h, --help     " + i18n.T("Show this help") + "\n"
}

// gibMonitoreAus schreibt die Monitorliste nach aus, mit alsJSON als eine Zeile JSON (F-98, ADR-0030 Punkt 1).
func gibMonitoreAus(aus, fehler io.Writer, monitore func() ([]ui.Monitor, error), alsJSON bool) int {
	liste, err := monitore()
	if err != nil {
		fmt.Fprintf(fehler, i18n.T("Could not query the monitors: %v")+"\n", err)
		return exitFehler
	}
	if len(liste) == 0 {
		fmt.Fprintln(fehler, i18n.T("No monitors found."))
		return exitFehler
	}
	if alsJSON {
		type bildschirm struct {
			Name   string `json:"name"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		}
		j := struct {
			Screens []bildschirm `json:"screens"`
		}{Screens: []bildschirm{}}
		for _, m := range liste {
			j.Screens = append(j.Screens, bildschirm{m.Name, m.Breite, m.Hoehe})
		}
		return gibJSONAus(aus, fehler, j)
	}
	fmt.Fprintf(aus, i18n.N("%d monitor found:", "%d monitors found:", len(liste))+"\n", len(liste))
	for _, m := range liste {
		// Ohne Skala: die ganzzahlige Pufferskala (2 bei 1,6) führte in die Irre (F-57).
		fmt.Fprintf(aus, i18n.T("  %s: %d × %d pixels")+"\n", m.Name, m.Breite, m.Hoehe)
	}
	return exitOK
}
