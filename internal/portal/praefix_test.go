// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-339): Präfix "portal: " genau einmal am Anfang.
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	s := err.Error()
	if !strings.HasPrefix(s, "portal: ") || strings.Count(s, "portal:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"portal: \"", name, s)
	}
}

func TestFehlertextPraefix(t *testing.T) {
	for name, e := range map[string]error{
		"ErrAbgebrochen": ErrAbgebrochen, "ErrKeinPortal": ErrKeinPortal, "ErrVerbindung": ErrVerbindung,
		"ErrAntwort": ErrAntwort, "ErrOhneDatei": ErrOhneDatei, "ErrKeineDatei": ErrKeineDatei,
		"errKeinPortal":     errKeinPortal(fmt.Errorf("Bus weg")),
		"Antwort mit Code":  fmt.Errorf("%w (Antwort %d)", ErrAntwort, 2),
		"ErrOhneDatei+Ursa": fmt.Errorf("%w: %v", ErrOhneDatei, "x"),
	} {
		einmalPraefix(t, name, e)
	}
	// Echte Wege durch auswerten.
	_, err := auswerten(Antwort{Code: 2}, Anfrage{})
	einmalPraefix(t, "auswerten Code 2", err)
	_, err = auswerten(Antwort{Code: antwortOK, Ergebnisse: map[string]dbus.Variant{"uris": dbus.MakeVariant(1)}}, Anfrage{})
	einmalPraefix(t, "auswerten uris falsch", err)
	_, err = auswerten(Antwort{Code: antwortOK}, Anfrage{})
	einmalPraefix(t, "auswerten ohne Datei", err)
	_, err = auswerten(Antwort{Code: antwortAbgebrochen}, Anfrage{})
	einmalPraefix(t, "auswerten abgebrochen", err)
}
