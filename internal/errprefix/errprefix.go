// SPDX-License-Identifier: GPL-3.0-or-later

// Package errprefix setzt die Fehlertext-Regel aus ADR-0019 Nachtrag 2 um: Was ein Paket unter internal/ über eine
// exportierte Funktion verlässt, beginnt mit dem Paketnamen und einem Doppelpunkt. Fremde Fehler (os, image/png, io,
// andere Pakete) werden mit %w gehüllt, damit errors.Is und errors.As weiter durchgreifen.
package errprefix

import (
	"errors"
	"fmt"
	"strings"
)

// Wrap liefert err mit dem Präfix "<paket>: ". Beginnt der Text schon so, bleibt err unverändert (kein doppeltes
// Präfix). Ein errors.Join wird nicht als Ganzes gehüllt, sondern Zweig für Zweig, damit es ein Join bleibt: Wer den
// Baum später durchgeht (etwa die Ausgabe der Speicherfehler im Editor), findet weiter die einzelnen Fehler.
func Wrap(paket string, err error) error {
	if err == nil {
		return nil
	}
	if fmt.Sprintf("%T", err) == "*errors.joinError" {
		if j, ok := err.(interface{ Unwrap() []error }); ok {
			zweige := j.Unwrap()
			neu := make([]error, len(zweige))
			for i, z := range zweige {
				neu[i] = Wrap(paket, z)
			}
			return errors.Join(neu...)
		}
	}
	if strings.HasPrefix(err.Error(), paket+": ") {
		return err
	}
	return fmt.Errorf("%s: %w", paket, err)
}

// Defer ist für `defer errprefix.Defer("paket", &fehler)` am Anfang einer exportierten Funktion mit benanntem
// Fehlerergebnis: Es hüllt den zurückgegebenen Fehler beim Verlassen mit Wrap.
func Defer(paket string, fehler *error) {
	*fehler = Wrap(paket, *fehler)
}
