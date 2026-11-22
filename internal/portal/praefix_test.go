// SPDX-License-Identifier: GPL-3.0-or-later

package portal

import (
	"context"
	"errors"
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

// abonnierFehler lässt Abonniere und Close scheitern; sonst die Attrappe.
type abonnierFehler struct {
	*attrappe
	roh error
}

func (a abonnierFehler) Abonniere(dbus.ObjectPath) (<-chan Antwort, func(), error) {
	return nil, nil, a.roh
}
func (a abonnierFehler) Close() error { return a.roh }

// F-379: Rohe D-Bus- und Kontextfehler verlassen Speichern, Close und URIZuPfad mit Präfix; die Ursache bleibt über
// errors.Is erreichbar.
func TestFehlertextPraefixPaketgrenze(t *testing.T) {
	roh := errors.New("Bus kaputt")
	pruefe := func(name string, err error) {
		t.Helper()
		einmalPraefix(t, name, err)
		if !errors.Is(err, roh) {
			t.Errorf("%s: Ursache verloren: %v", name, err)
		}
	}
	// Abonnieren scheitert.
	d := &Dialog{t: abonnierFehler{&attrappe{}, roh}, token: func() string { return "tok" }}
	_, err := d.Speichern(context.Background(), Anfrage{})
	pruefe("Speichern Abonniere", err)
	pruefe("Close", d.Close())
	// SaveFile scheitert (roher Fehler der Attrappe).
	_, err = neuerDialog(&attrappe{fehler: roh}).Speichern(mitFrist(t), anfrage())
	pruefe("Speichern SaveFile", err)
	// Antwort trägt einen rohen Fehler.
	_, err = neuerDialog(&attrappe{antwort: &Antwort{Err: roh}}).Speichern(mitFrist(t), anfrage())
	pruefe("Speichern Antwort.Err", err)
	// Kontext endet vor der Antwort.
	ctx, abbruch := context.WithCancel(context.Background())
	abbruch()
	_, err = neuerDialog(&attrappe{}).Speichern(ctx, anfrage())
	einmalPraefix(t, "Speichern ctx", err)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Speichern ctx: Ursache verloren: %v", err)
	}
	// Sentinel bleibt erkennbar, ohne doppeltes Präfix.
	_, err = neuerDialog(&attrappe{antwort: &Antwort{Code: antwortAbgebrochen}}).Speichern(mitFrist(t), anfrage())
	einmalPraefix(t, "abgebrochen", err)
	if !errors.Is(err, ErrAbgebrochen) {
		t.Errorf("ErrAbgebrochen verloren: %v", err)
	}
	// URIZuPfad: nicht parsbare Adresse und Nicht-Datei.
	_, err = URIZuPfad("file://%zz/x")
	einmalPraefix(t, "URIZuPfad parse", err)
	_, err = URIZuPfad("https://x/y")
	einmalPraefix(t, "URIZuPfad Schema", err)
}
