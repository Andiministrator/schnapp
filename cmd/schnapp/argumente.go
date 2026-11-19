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
	ModusBereich     Modus = iota // Voreinstellung ohne Argument (ADR-0017)
	ModusBildschirm               // --bildschirm[=NAME]
	ModusFenster                  // --fenster
	ModusWiederholen              // --wiederholen
	ModusVersion                  // --version
	ModusHilfe                    // --help
)

// Aufruf ist das ausgewertete Ergebnis der Kommandozeile.
type Aufruf struct {
	Modus Modus
	// Bildschirm ist der Name des gewählten Monitors bei --bildschirm=NAME; leer bei --bildschirm ohne Wert
	// (dann: Monitorliste ausgeben).
	Bildschirm string
}

// Option nennt die Bezeichnung des Modus, wie sie auf der Kommandozeile steht.
func (m Modus) Option() string {
	switch m {
	case ModusBildschirm:
		return "--bildschirm"
	case ModusFenster:
		return "--fenster"
	case ModusWiederholen:
		return "--wiederholen"
	case ModusVersion:
		return "--version"
	case ModusHilfe:
		return "--help"
	}
	return "--bereich"
}

// Fehler ist ein Fehler in der Kommandozeile. Er trägt schon den übersetzten Text.
type Fehler struct{ Text string }

func (f *Fehler) Error() string { return f.Text }

// Auswerten liest die Argumente (ohne Programmnamen). Ohne Argument gilt --bereich. Es gibt genau einen Modus;
// zwei verschiedene oder unbekannte Optionen, Werte an Optionen ohne Wert und Positionsargumente sind Fehler.
// Nur --bildschirm nimmt einen Wert (--bildschirm=NAME); die Form „--bildschirm NAME“ gibt es nicht, damit
// „--bildschirm“ allein eindeutig die Monitorliste meint.
func Auswerten(args []string) (Aufruf, error) {
	var (
		a       Aufruf
		gesehen bool
	)
	for _, arg := range args {
		name, wert, hatWert := strings.Cut(arg, "=")
		var m Modus
		switch name {
		case "--bereich":
			m = ModusBereich
		case "--bildschirm":
			m = ModusBildschirm
		case "--fenster":
			m = ModusFenster
		case "--wiederholen":
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
			return Aufruf{}, &Fehler{sprache.T("Option --bildschirm needs a monitor name after the equals sign.")}
		}
		if gesehen && !(a.Modus == m && m != ModusBildschirm) {
			return Aufruf{}, &Fehler{fmt.Sprintf(sprache.T("Options %s and %s cannot be combined."), a.Modus.Option(), m.Option())}
		}
		a = Aufruf{Modus: m, Bildschirm: wert}
		gesehen = true
	}
	return a, nil
}
