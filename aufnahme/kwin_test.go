// SPDX-License-Identifier: GPL-3.0-or-later

package aufnahme

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

// attrappe ersetzt D-Bus: keine echte Aufnahme.
type attrappe struct {
	version3, version2 bool // welche Schnittstelle "existiert"
	versionAufrufe     []string
	antwort            map[string]dbus.Variant
	daten              []byte
	fehler             error // Fehler von capture
	letzte             schnittstelle
	letzteMethode      string
	letzteArgs         []any
	letzteOpt          map[string]dbus.Variant
	geschlossen        bool
	quelleGeschlossen  bool
}

func (a *attrappe) version(_ context.Context, s schnittstelle) (uint32, error) {
	a.versionAufrufe = append(a.versionAufrufe, s.name)
	if (s == schnittstelle3 && a.version3) || (s == schnittstelle2 && a.version2) {
		return 5, nil
	}
	return 0, errors.New("org.freedesktop.DBus.Error.UnknownObject")
}

func (a *attrappe) capture(_ context.Context, s schnittstelle, methode string, args []any, opt map[string]dbus.Variant) (map[string]dbus.Variant, datenquelle, error) {
	a.letzte, a.letzteMethode, a.letzteArgs, a.letzteOpt = s, methode, args, opt
	if a.fehler != nil {
		return nil, nil, a.fehler
	}
	return a.antwort, &bytesQuelle{a: a, d: a.daten}, nil
}

func (a *attrappe) Close() error { a.geschlossen = true; return nil }

type bytesQuelle struct {
	a *attrappe
	d []byte
}

func (b *bytesQuelle) lies(n uint64) ([]byte, error) {
	if uint64(len(b.d)) < n {
		return nil, fmt.Errorf("nur %d Bytes", len(b.d))
	}
	return b.d[:n], nil
}
func (b *bytesQuelle) Close() error { b.a.quelleGeschlossen = true; return nil }

// gueltigeAntwort beschreibt ein 2×2-Bild im Format RGBX8888 (16), Zeilenlänge 8.
func gueltigeAntwort() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"type":   dbus.MakeVariant("raw"),
		"width":  dbus.MakeVariant(uint32(2)),
		"height": dbus.MakeVariant(uint32(2)),
		"stride": dbus.MakeVariant(uint32(8)),
		"format": dbus.MakeVariant(uint32(16)),
		"scale":  dbus.MakeVariant(1.6),
		"screen": dbus.MakeVariant("DP-2"),
	}
}

func gueltigeDaten() []byte {
	return []byte{1, 2, 3, 0, 4, 5, 6, 0, 7, 8, 9, 0, 10, 11, 12, 0}
}

func neu(a *attrappe) *KWin { return &KWin{t: a} }

func TestBildschirm(t *testing.T) {
	a := &attrappe{version3: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	k := neu(a)
	b, err := k.Bildschirm(context.Background(), "DP-1", Optionen{Mauszeiger: true})
	if err != nil {
		t.Fatal(err)
	}
	soll := []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255}
	if string(b.Pixel.Pix) != string(soll) || b.Skala != 1.6 || b.Bildschirm != "DP-2" {
		t.Errorf("Bild %v, Skala %v, Bildschirm %q", b.Pixel.Pix, b.Skala, b.Bildschirm)
	}
	if a.letzteMethode != "CaptureScreen" || len(a.letzteArgs) != 1 || a.letzteArgs[0] != "DP-1" {
		t.Errorf("Aufruf %s %v", a.letzteMethode, a.letzteArgs)
	}
	for schluessel, soll := range map[string]bool{"native-resolution": true, "include-cursor": true, "hide-caller-windows": true} {
		if v, _ := a.letzteOpt[schluessel].Value().(bool); v != soll {
			t.Errorf("Option %s = %v, Soll %v", schluessel, v, soll)
		}
	}
	if !a.quelleGeschlossen {
		t.Error("Datenquelle nicht geschlossen")
	}
	// Ohne Mauszeiger.
	if _, err := k.Bildschirm(context.Background(), "DP-1", Optionen{}); err != nil {
		t.Fatal(err)
	}
	if v, _ := a.letzteOpt["include-cursor"].Value().(bool); v {
		t.Error("include-cursor trotz Optionen{} gesetzt")
	}
}

func TestFensterInteraktiv(t *testing.T) {
	a := &attrappe{version2: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	delete(a.antwort, "screen")
	b, err := neu(a).Fenster(context.Background(), Optionen{})
	if err != nil {
		t.Fatal(err)
	}
	if a.letzteMethode != "CaptureInteractive" || len(a.letzteArgs) != 1 || a.letzteArgs[0] != uint32(0) {
		t.Errorf("Aufruf %s %#v (kind muss uint32(0) sein)", a.letzteMethode, a.letzteArgs)
	}
	if b.Bildschirm != "" {
		t.Errorf("Bildschirm %q ohne Angabe", b.Bildschirm)
	}
}

func TestWahlSchnittstelle(t *testing.T) {
	// Beide vorhanden: ScreenShot3.
	a := &attrappe{version3: true, version2: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	k := neu(a)
	if _, err := k.Bildschirm(context.Background(), "x", Optionen{}); err != nil {
		t.Fatal(err)
	}
	if a.letzte != schnittstelle3 || !a.letzte.memfd {
		t.Errorf("gewählt %v, Soll ScreenShot3", a.letzte.name)
	}
	// Die Wahl gilt für alle weiteren Aufrufe, ohne erneute Abfrage.
	n := len(a.versionAufrufe)
	k.Bildschirm(context.Background(), "x", Optionen{})
	if len(a.versionAufrufe) != n {
		t.Error("Version wurde erneut abgefragt")
	}
	// Nur ScreenShot2: Rückfall nach der Abfrage von ScreenShot3.
	a = &attrappe{version2: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()}
	if _, err := neu(a).Bildschirm(context.Background(), "x", Optionen{}); err != nil {
		t.Fatal(err)
	}
	if a.letzte != schnittstelle2 || a.letzte.memfd {
		t.Errorf("gewählt %v, Soll ScreenShot2", a.letzte.name)
	}
	if got := strings.Join(a.versionAufrufe, ","); got != schnittstelle3.name+","+schnittstelle2.name {
		t.Errorf("Abfragefolge %s", got)
	}
	// Keine von beiden.
	a = &attrappe{}
	_, err := neu(a).Bildschirm(context.Background(), "x", Optionen{})
	if !errors.Is(err, ErrKeineSchnittstelle) {
		t.Errorf("Fehler %v, Soll ErrKeineSchnittstelle", err)
	}
	if err == nil || err.Error() == "" {
		t.Error("leere Meldung")
	}
}

func TestNichtBerechtigt(t *testing.T) {
	de := dbus.Error{Name: "org.kde.KWin.ScreenShot2.Error.NoAuthorized", Body: []any{"The process is not authorized to take a screenshot"}}
	a := &attrappe{version2: true, fehler: de}
	_, err := neu(a).Bildschirm(context.Background(), "x", Optionen{})
	if !errors.Is(err, ErrNichtBerechtigt) {
		t.Fatalf("Fehler %v, Soll ErrNichtBerechtigt", err)
	}
	if errors.Is(err, ErrAbgebrochen) {
		t.Error("fälschlich auch ErrAbgebrochen")
	}
	if !strings.Contains(err.Error(), "Schnapp") {
		t.Errorf("Meldung %q", err)
	}
	var ursache dbus.Error
	if !errors.As(err, &ursache) || ursache.Name != de.Name {
		t.Error("technische Ursache nicht erreichbar")
	}
	// Auch in einem Umschlag erkennbar.
	a.fehler = fmt.Errorf("Umschlag: %w", de)
	if _, err := neu(a).Bildschirm(context.Background(), "x", Optionen{}); !errors.Is(err, ErrNichtBerechtigt) {
		t.Errorf("umhüllt: %v", err)
	}
}

func TestAbgebrochenUndSonstigeFehler(t *testing.T) {
	a := &attrappe{version3: true, fehler: dbus.Error{Name: "org.kde.KWin.ScreenShot2.Error.Cancelled"}}
	_, err := neu(a).Fenster(context.Background(), Optionen{})
	if !errors.Is(err, ErrAbgebrochen) {
		t.Errorf("Fehler %v, Soll ErrAbgebrochen", err)
	}
	a.fehler = dbus.Error{Name: "org.kde.KWin.ScreenShot2.Error.InvalidArea"}
	_, err = neu(a).Fenster(context.Background(), Optionen{})
	if err == nil || errors.Is(err, ErrAbgebrochen) || errors.Is(err, ErrNichtBerechtigt) {
		t.Errorf("sonstiger Fehler falsch eingeordnet: %v", err)
	}
	if !strings.Contains(err.Error(), "CaptureInteractive") {
		t.Errorf("Methode fehlt in der Meldung: %v", err)
	}
}

func TestAntwortAuswertung(t *testing.T) {
	ohne := func(schluessel string) map[string]dbus.Variant {
		m := gueltigeAntwort()
		delete(m, schluessel)
		return m
	}
	mit := func(schluessel string, wert any) map[string]dbus.Variant {
		m := gueltigeAntwort()
		m[schluessel] = dbus.MakeVariant(wert)
		return m
	}
	faelle := []struct {
		name    string
		antwort map[string]dbus.Variant
	}{
		{"type fehlt", ohne("type")},
		{"type ist nicht raw", mit("type", "png")},
		{"type hat falschen Typ", mit("type", uint32(1))},
		{"width fehlt", ohne("width")},
		{"height fehlt", ohne("height")},
		{"stride fehlt", ohne("stride")},
		{"format fehlt", ohne("format")},
		{"width als String", mit("width", "2")},
		{"width als float", mit("width", 2.0)},
		{"height negativ", mit("height", int32(-2))},
		{"format negativ", mit("format", int32(-16))},
		{"format riesig", mit("format", uint64(1<<40+16))},
		{"unbekanntes Format", mit("format", uint32(99))},
		{"width 1<<62", mit("width", uint64(1<<62))},
		{"stride kleiner als Breite×4", mit("stride", uint32(4))},
	}
	for _, f := range faelle {
		a := &attrappe{version3: true, antwort: f.antwort, daten: gueltigeDaten()}
		if _, err := neu(a).Bildschirm(context.Background(), "x", Optionen{}); err == nil {
			t.Errorf("%s: kein Fehler", f.name)
		}
	}
	// Zu wenig Daten.
	a := &attrappe{version3: true, antwort: gueltigeAntwort(), daten: gueltigeDaten()[:10]}
	if _, err := neu(a).Bildschirm(context.Background(), "x", Optionen{}); err == nil {
		t.Error("zu wenig Daten ohne Fehler")
	}
	// Unsinnige oder fehlende Skala ergibt 1; ein angefragter Name bleibt ohne "screen" erhalten.
	m := ohne("scale")
	delete(m, "screen")
	a = &attrappe{version3: true, antwort: m, daten: gueltigeDaten()}
	b, err := neu(a).Bildschirm(context.Background(), "DP-9", Optionen{})
	if err != nil || b.Skala != 1 || b.Bildschirm != "DP-9" {
		t.Errorf("ohne scale/screen: %+v, %v", b, err)
	}
	for _, s := range []float64{0, -1, 1e9} {
		if g := skalaAus(s); g != 1 {
			t.Errorf("skalaAus(%v) = %v", s, g)
		}
	}
}

func TestClose(t *testing.T) {
	a := &attrappe{}
	if err := neu(a).Close(); err != nil || !a.geschlossen {
		t.Errorf("Close: %v, geschlossen=%v", err, a.geschlossen)
	}
}
