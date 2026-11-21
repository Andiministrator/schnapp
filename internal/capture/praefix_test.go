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

// F-345: Der Abbruchfehler trägt das Präfix (errors.Is bleibt); Alle ohne CaptureWorkspace (ScreenShot2 vor Version 3)
// liefert die übersetzte Nutzermeldung statt des rohen D-Bus-Fehlers und ruft die Methode gar nicht erst auf.
func TestPraefixAbbruchUndAlleOhneWorkspace(t *testing.T) {
	ctx, abbruch := context.WithCancel(context.Background())
	abbruch()
	_, err := neu(&attrappe{}).Bildschirm(ctx, "x", Optionen{})
	einmalPraefix(t, "ctx.Err", err)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(Canceled) verloren: %v", err)
	}

	a := &attrappe{version2: true, version2Nr: 2, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	_, err = neu(a).Alle(context.Background(), Optionen{})
	if !errors.Is(err, ErrKeineGesamtaufnahme) {
		t.Fatalf("Fehler %v, erwartet ErrKeineGesamtaufnahme", err)
	}
	if a.letzteMethode != "" {
		t.Errorf("CaptureWorkspace trotzdem aufgerufen: %s", a.letzteMethode)
	}
	if strings.Contains(err.Error(), "capture:") || strings.Contains(err.Error(), "org.") {
		t.Errorf("Nutzermeldung nicht übersetzt oder roh: %q", err)
	}
	einmalPraefix(t, "Sentinel", ErrKeineGesamtaufnahme)
	// F-350: die Meldung ist die für diesen Fall, nicht die Rückfallmeldung.
	if !strings.Contains(err.Error(), "all screens") {
		t.Errorf("Meldung für ErrKeineGesamtaufnahme falsch: %q", err)
	}

	a = &attrappe{version2: true, version2Nr: 3, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	if _, err := neu(a).Alle(context.Background(), Optionen{}); err != nil {
		t.Errorf("Version 3 muss gehen: %v", err)
	}
}

// F-350: NeuKWin meldet einen Verbindungsfehler mit genau einem Präfix und hält die Ursache erreichbar.
func TestNeuKWinVerbindungsfehlerPraefix(t *testing.T) {
	alt := verbindeTransport
	defer func() { verbindeTransport = alt }()
	ursache := errors.New("kein Bus")
	verbindeTransport = func() (transport, error) { return nil, ursache }
	k, err := NeuKWin()
	if k != nil {
		t.Errorf("KWin trotz Fehler: %v", k)
	}
	einmalPraefix(t, "NeuKWin", err)
	if !errors.Is(err, ursache) {
		t.Errorf("Ursache verloren: %v", err)
	}
}
