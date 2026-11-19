// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"github.com/Andiministrator/schnapp/sprache"
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
)

// Aufruf ist das ausgewertete Ergebnis der Kommandozeile.
type Aufruf struct {
	Modus Modus
	// Bildschirm ist der Name des gewählten Monitors bei --screen=NAME; leer bei --screen ohne Wert
	// (dann gilt das Verhalten laut Einstellung, vorerst fest „Monitor per Klick wählen“).
	Bildschirm string
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
	}
	return "--region"
}

// Fehler ist ein Fehler in der Kommandozeile. Er trägt schon den übersetzten Text.
type Fehler struct{ Text string }

func (f *Fehler) Error() string { return f.Text }

// Auswerten liest die Argumente (ohne Programmnamen). Ohne Argument gilt --region. Es gibt genau einen Modus;
// zwei verschiedene oder unbekannte Optionen, Werte an Optionen ohne Wert und Positionsargumente sind Fehler.
// Nur --screen nimmt einen Wert (--screen=NAME); die Form „--screen NAME“ gibt es nicht, damit
// „--screen“ allein eindeutig „Monitor laut Einstellung“ meint. Die Monitorliste gibt --list-screens aus.
func Auswerten(args []string) (Aufruf, error) {
	var (
		a       Aufruf
		gesehen bool
	)
	for _, arg := range args {
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
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return Aufruf{}, &Fehler{fmt.Sprintf(sprache.T("Unknown option: %s"), name)}
			}
			return Aufruf{}, &Fehler{fmt.Sprintf(sprache.T("Unexpected argument: %s"), arg)}
		}
		if hatWert && m != ModusBildschirm {
			return Aufruf{}, &Fehler{fmt.Sprintf(sprache.T("Option %s takes no value."), m.Option())}
		}
		if hatWert && wert == "" {
			return Aufruf{}, &Fehler{sprache.T("Option --screen needs a monitor name after the equals sign.")}
		}
		if gesehen && !(a.Modus == m && m != ModusBildschirm) {
			return Aufruf{}, &Fehler{fmt.Sprintf(sprache.T("Options %s and %s cannot be combined."), a.Modus.Option(), m.Option())}
		}
		a = Aufruf{Modus: m, Bildschirm: wert}
		gesehen = true
	}
	return a, nil
}
