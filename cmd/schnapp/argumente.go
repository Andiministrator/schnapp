// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Modus ist, was der Aufruf verlangt.
type Modus int

const (
	ModusBereich      Modus = iota // Voreinstellung ohne Argument (ADR-0017)
	ModusBildschirm                // --screen[=NAME]; ohne Namen gilt die Einstellung (ADR-0017 Nachtrag 3)
	ModusMonitorliste              // --list-screens
	ModusFenster                   // --window
	ModusWiederholen               // --repeat
	ModusVersion                   // --version
	ModusHilfe                     // --help
	ModusOeffnen                   // schnapp DATEI: Datei im Editor öffnen (ADR-0020)
	ModusExport                    // --export=ZIEL DATEI: Datei ohne Fenster exportieren (ADR-0030 Punkt 3)
)

// Aufruf ist das ausgewertete Ergebnis der Kommandozeile.
type Aufruf struct {
	Modus Modus
	// Bildschirm ist der Name des gewählten Monitors bei --screen=NAME; leer bei --screen ohne Wert
	// (dann gilt das Verhalten laut Einstellung `[capture] screen`: Monitor per Klick, unter dem Zeiger oder alle,
	// ADR-0018).
	Bildschirm string
	// OhneEditor ist --no-editor: nach der Aufnahme keinen Editor öffnen (ADR-0017 Nachtrag 4).
	OhneEditor bool
	// Datei ist bei ModusOeffnen der Pfad der Datei.
	Datei string
	// OhneGrenze ist --ignore-limit: die Datei auch über dem Rechenbudget öffnen (ADR-0017 Nachtrag 6, F-146).
	OhneGrenze bool
	// JSON ist --json: bei Erfolg eine Zeile JSON auf stdout (ADR-0030 Punkt 1).
	JSON bool
	// Ausgabe ist --output=DATEI: die Aufnahme dorthin schreiben statt in die Bibliothek (ADR-0030 Punkt 2).
	Ausgabe string
	// Ziel ist bei ModusExport das Ziel von --export=ZIEL (ADR-0030 Punkt 3).
	Ziel string
}

// Aufnahme meldet, ob der Modus eine Aufnahme ist (nur diese öffnen danach den Editor und vertragen --no-editor).
func (m Modus) Aufnahme() bool {
	return m == ModusBereich || m == ModusBildschirm || m == ModusFenster || m == ModusWiederholen
}

// Option nennt die Bezeichnung des Modus, wie sie auf der Kommandozeile steht.
func (m Modus) Option() string {
	switch m {
	case ModusBildschirm:
		return "--screen"
	case ModusMonitorliste:
		return "--list-screens"
	case ModusFenster:
		return "--window"
	case ModusWiederholen:
		return "--repeat"
	case ModusVersion:
		return "--version"
	case ModusHilfe:
		return "--help"
	case ModusOeffnen:
		return "FILE"
	case ModusExport:
		return "--export"
	}
	return "--region"
}

// Fehler ist ein Fehler in der Kommandozeile. Er trägt schon den übersetzten Text.
type Fehler struct{ Text string }

func (f *Fehler) Error() string { return f.Text }

// Auswerten liest die Argumente (ohne Programmnamen). Ohne Argument gilt --region. Es gibt genau einen Modus;
// zwei verschiedene oder unbekannte Optionen und Werte an Optionen ohne Wert sind Fehler. Genau ein
// Positionsargument ohne weitere Option öffnet die Datei im Editor; --no-editor gilt nur für Aufnahmen.
// Nur --screen nimmt einen Wert (--screen=NAME); die Form „--screen NAME“ gibt es nicht, damit
// „--screen“ allein eindeutig „Monitor laut Einstellung“ meint. Die Monitorliste gibt --list-screens aus.
func Auswerten(args []string) (Aufruf, error) {
	var (
		a          Aufruf
		gesehen    bool
		ohneEditor bool
		ohneGrenze bool
		alsJSON    bool
		ausgabe    string
		exportZiel string
	)
	for _, arg := range args {
		if arg == "--ignore-limit" {
			ohneGrenze = true
			continue
		}
		if arg == "--json" {
			alsJSON = true
			continue
		}
		// Optionen mit Dateiwert (ADR-0030 Punkt 4): nur mit Gleichheitszeichen, mehrfach nur mit demselben Wert.
		if name, wert, hatWert := strings.Cut(arg, "="); name == "--output" || name == "--export" {
			ziel := &ausgabe
			if name == "--export" {
				ziel = &exportZiel
			}
			if !hatWert || wert == "" {
				return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Option %s needs a file name after an equals sign: %s=FILE."), name, name)}
			}
			if *ziel != "" && *ziel != wert {
				return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Option %s cannot be given more than once with different files."), name)}
			}
			if _, ok := export.AusEndung(wert); !ok {
				return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Unknown file type for %s: use a name ending in .png, .jpg or .svg."), name)}
			}
			*ziel = wert
			continue
		}
		if arg == "--no-editor" {
			ohneEditor = true
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			if gesehen {
				if a.Modus == ModusOeffnen {
					return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Unexpected argument: %s"), arg)}
				}
				return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("A file cannot be combined with %s."), a.Modus.Option())}
			}
			a = Aufruf{Modus: ModusOeffnen, Datei: arg}
			gesehen = true
			continue
		}
		if gesehen && a.Modus == ModusOeffnen {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("A file cannot be combined with %s."), arg)}
		}
		name, wert, hatWert := strings.Cut(arg, "=")
		var m Modus
		switch name {
		case "--region":
			m = ModusBereich
		case "--screen":
			m = ModusBildschirm
		case "--list-screens":
			m = ModusMonitorliste
		case "--window":
			m = ModusFenster
		case "--repeat":
			m = ModusWiederholen
		case "--version", "-V":
			m = ModusVersion
		case "--help", "-h":
			m = ModusHilfe
		case "--json":
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Option %s takes no value."), "--json")}
		default:
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Unknown option: %s"), name)}
		}
		if hatWert && m != ModusBildschirm {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Option %s takes no value."), m.Option())}
		}
		if hatWert && wert == "" {
			return Aufruf{}, &Fehler{i18n.T("Option --screen needs a monitor name after the equals sign.")}
		}
		if gesehen && a.Modus == ModusBildschirm && m == ModusBildschirm && a.Bildschirm != wert {
			return Aufruf{}, &Fehler{i18n.T("Option --screen cannot be given more than once with different monitor names.")}
		}
		if gesehen && a.Modus != m {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Options %s and %s cannot be combined."), a.Modus.Option(), m.Option())}
		}
		a = Aufruf{Modus: m, Bildschirm: wert}
		gesehen = true
	}
	if ohneGrenze {
		// Ohne Datei (auch bei Aufnahme, --help usw.) ist der Schalter ein Fehler, nicht still wirkungslos.
		if a.Modus != ModusOeffnen {
			return Aufruf{}, &Fehler{i18n.T("Option --ignore-limit works only together with a file.")}
		}
		a.OhneGrenze = true
	}
	if ohneEditor {
		if a.Modus == ModusOeffnen {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("A file cannot be combined with %s."), "--no-editor")}
		}
		if !a.Modus.Aufnahme() {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Options %s and %s cannot be combined."), a.Modus.Option(), "--no-editor")}
		}
		a.OhneEditor = true
	}
	if exportZiel != "" {
		if a.Modus != ModusOeffnen {
			if !gesehen {
				return Aufruf{}, &Fehler{i18n.T("Option --export needs the file to export: --export=TARGET FILE.")}
			}
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Options %s and %s cannot be combined."), a.Modus.Option(), "--export")}
		}
		a.Modus, a.Ziel = ModusExport, exportZiel
	}
	if ausgabe != "" {
		if !a.Modus.Aufnahme() {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Options %s and %s cannot be combined."), a.Modus.Option(), "--output")}
		}
		a.Ausgabe = ausgabe
	}
	if alsJSON {
		if a.Modus == ModusHilfe || a.Modus == ModusOeffnen {
			return Aufruf{}, &Fehler{fmt.Sprintf(i18n.T("Options %s and %s cannot be combined."), a.Modus.Option(), "--json")}
		}
		a.JSON = true
	}
	return a, nil
}
