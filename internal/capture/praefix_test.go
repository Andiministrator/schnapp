// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

// einmalPraefix prüft die Fehlertext-Regel (ADR-0019 Nachtrag 2, F-339): der Text beginnt mit "capture: " und das
// Präfix kommt genau einmal vor (keine doppelte Hülle).
func einmalPraefix(t *testing.T, name string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: kein Fehler", name)
		return
	}
	s := err.Error()
	if !strings.HasPrefix(s, "capture: ") || strings.Count(s, "capture:") != 1 {
		t.Errorf("%s: Fehlertext %q ohne genau ein Präfix \"capture: \"", name, s)
	}
}

func TestFehlertextPraefix(t *testing.T) {
	// Fehlerwerte der Schnittstelle (nicht die übersetzte Nutzermeldung von nutzerFehler).
	for name, e := range map[string]error{"ErrNichtBerechtigt": ErrNichtBerechtigt, "ErrAbgebrochen": ErrAbgebrochen, "ErrKeineSchnittstelle": ErrKeineSchnittstelle} {
		einmalPraefix(t, name, e)
	}
	// Fehlerwege der Aufnahme: Antwort auswerten, Größen, zu wenig Daten, D-Bus-Fehler.
	mit := func(schluessel string, wert any) map[string]dbus.Variant {
		m := gueltigeAntwort()
		m[schluessel] = dbus.MakeVariant(wert)
		return m
	}
	ohne := gueltigeAntwort()
	delete(ohne, "width")
	faelle := map[string]map[string]dbus.Variant{
		"type":            mit("type", "png"),
		"Schlüssel fehlt": ohne,
		"Wert ungültig":   mit("width", "2"),
		"Format":          mit("format", uint32(99)),
		"Formatgrenze":    mit("format", uint64(1<<40+16)),
		"Bildgröße":       mit("width", uint64(1<<62)),
		"Zeilenlänge":     mit("stride", uint32(4)),
	}
	for name, antwort := range faelle {
		a := &attrappe{version3: true, antwort: antwort, daten: gueltigeDaten()}
		_, err := neu(a).Bildschirm(context.Background(), "x", Optionen{})
		einmalPraefix(t, name, err)
	}
	// Quelle liefert zu wenig: die innere Meldung trägt kein eigenes Präfix, die Hülle genau eines.
	a := &attrappe{version3: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()[:10]}
	_, err := neu(a).Bildschirm(context.Background(), "x", Optionen{})
	einmalPraefix(t, "zu wenig Daten", err)
	// Sonstiger D-Bus-Fehler und unbekannte Schnittstelle.
	a = &attrappe{version3: true, fehler: dbus.Error{Name: "org.kde.KWin.ScreenShot3.Error.InvalidArea"}}
	_, err = neu(a).Bildschirm(context.Background(), "x", Optionen{})
	einmalPraefix(t, "D-Bus-Fehler", err)
	a = &attrappe{version3: true, fehler: errors.New("Pipe kaputt")}
	_, err = neu(a).Bildschirm(context.Background(), "x", Optionen{})
	einmalPraefix(t, "anderer Fehler", err)
	// Pixelumwandlung mit zu wenig Daten (direkt, ohne Quelle).
	_, err = rohZuNRGBA(make([]byte, 3), 2, 2, 8, 16)
	einmalPraefix(t, "rohZuNRGBA", err)
	_, err = rohZuNRGBA(make([]byte, 16), 2, 2, 8, 99)
	einmalPraefix(t, "rohZuNRGBA Format", err)
}
