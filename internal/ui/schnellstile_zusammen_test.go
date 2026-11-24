// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/settings"
)

func fremderKasten(breite float64) settings.Stil {
	return settings.Stil{Art: settings.ArtKasten, Farbe: color.NRGBA{R: 9, G: 9, B: 9, A: 255}, Breite: breite}
}

func fremdHinzu(t *testing.T, s settings.Stil) {
	t.Helper()
	if _, _, err := settings.AendereStile(func(alt []settings.Stil) []settings.Stil { return settings.StilHinzu(alt, s) }); err != nil {
		t.Fatal(err)
	}
}

// F-627: Ein zweites Fenster legt zwischendurch einen Stil an; „+“ hier lässt ihn stehen, und die Kacheln zeigen danach
// beide. Ein hier entfernter Stil kommt nicht zurück, der des anderen Fensters bleibt.
func TestSchnellstileZweiFensterZusammenfuehren(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.SetzeBreite(6)
	l.setle()
	l.klick(mitte(k.plusRahmen)) // eigener Stil A
	fremdHinzu(t, fremderKasten(11))
	l.e.s.SetzeBreite(7)
	l.setle()
	l.klick(mitte(k.plusRahmen)) // eigener Stil C
	pfad, _ := settings.StilePfad()
	if daten, _ := os.ReadFile(pfad); strings.Count(string(daten), "[box]") != 3 {
		t.Errorf("styles.ini soll drei Stile haben (A, fremd, C): %q", daten)
	}
	if n := len(l.e.s.EigeneSchnellstile()); n != 3 {
		t.Errorf("%d eigene Stile in der Sitzung, Soll 3 (Stand der Datei)", n)
	}
	// A entfernen (die erste eigene Kachel); der fremde bleibt.
	l.setle()
	l.rechtsklick(l.kachel(7))
	roh, warn := settings.LadeStile()
	if len(warn) != 0 || len(roh) != 2 || roh[0].Breite != 11 || roh[1].Breite != 7 {
		t.Errorf("nach dem Entfernen: %+v %q", roh, warn)
	}
}

// F-629 b: Eine styles.ini mit Verworfenem wird nicht stillschweigend ersetzt: Sicherung, Meldung; gibt es die
// Sicherung schon, bleibt die Datei, und die Fußzeile zeigt den Fehler.
func TestSchnellstileUnlesbareDateiGesichert(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	pfad, _ := settings.StilePfad()
	kaputt := "[box]\ncolor = #010203\nwidth = 5\n\n[box]\nwidth = nonsense\ncolor = #010203\n"
	if err := os.MkdirAll(pfad[:strings.LastIndex(pfad, "/")], 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pfad, []byte(kaputt), 0o600); err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeBreite(6)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	if l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "styles.ini.bak") {
		t.Errorf("Meldung %q (Fehler %v), Soll der Hinweis auf styles.ini.bak", l.e.meldung.Text, l.e.meldung.Fehler)
	}
	if b, _ := os.ReadFile(pfad + ".bak"); string(b) != kaputt {
		t.Errorf("Sicherung: %q", b)
	}
	// Wieder kaputt, Sicherung da: verweigert.
	if err := os.WriteFile(pfad, []byte(kaputt), 0o600); err != nil {
		t.Fatal(err)
	}
	l.e.s.SetzeBreite(8.5)
	l.setle()
	l.klick(mitte(k.plusRahmen))
	if !l.e.meldung.Fehler || !strings.Contains(l.e.meldung.Text, "styles.ini.bak") {
		t.Errorf("Meldung %q (Fehler %v), Soll ein Fehler zur vorhandenen Sicherung", l.e.meldung.Text, l.e.meldung.Fehler)
	}
	if b, _ := os.ReadFile(pfad); string(b) != kaputt {
		t.Errorf("Datei wurde trotz Sicherung ersetzt: %q", b)
	}
}

// F-628: „+“ ist bei einem unspeicherbaren Stand grau und sagt warum.
func TestSchnellstilePlusGrauBeiUnbrauchbaremStand(t *testing.T) {
	l, _ := schnellLauf(t)
	k := &l.e.schnell
	l.e.s.Stil.Breite = 0
	l.setle()
	if k.plusAn {
		t.Errorf("„+“ wirkt bei Strich 0")
	}
	if l.e.s.PruefeSchnellstilHinzu(canvas.SchnellKasten) != canvas.HinzuUnbrauchbar || !strings.Contains(l.e.schnellPlusText(), "cannot be saved") {
		t.Errorf("Hinweis %q", l.e.schnellPlusText())
	}
}
