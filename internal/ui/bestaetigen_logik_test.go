// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"

	"gioui.org/io/key"
)

// TestFreigabeKlickNachSperre: Drücken und Loslassen nach der Sperre erlauben; alles andere nicht (F-464).
func TestFreigabeKlickNachSperre(t *testing.T) {
	t0 := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	nach := t0.Add(bestaetigenSperre + time.Millisecond)
	in := t0.Add(bestaetigenSperre / 2)
	fall := map[string]struct {
		druck, klick time.Time
		abbruch      bool
		ohneDruck    bool
		want         bool
	}{
		"beides nach der Sperre":           {druck: nach, klick: nach.Add(50 * time.Millisecond), want: true},
		"Drücken in der Sperre":            {druck: in, klick: nach, want: false},
		"beides in der Sperre":             {druck: in, klick: in.Add(time.Millisecond), want: false},
		"Loslassen ohne Drücken":           {ohneDruck: true, klick: nach, want: false},
		"abgebrochen":                      {druck: nach, abbruch: true, klick: nach.Add(time.Millisecond), want: false},
		"Drücken genau am Ende der Sperre": {druck: t0.Add(bestaetigenSperre), klick: nach, want: true},
	}
	for name, c := range fall {
		var f freigabe
		f.bild(t0)
		f.bild(nach) // weitere Bilder verschieben die Sperre nicht
		if !c.ohneDruck {
			f.druck(c.druck)
		}
		if c.abbruch {
			f.abbruch()
		}
		if got := f.klick(c.klick); got != c.want {
			t.Errorf("%s: erlaubt %v, erwartet %v", name, got, c.want)
		}
	}
}

// TestFreigabeVorErstemBild: Ohne gezeigtes Bild ist alles gesperrt.
func TestFreigabeVorErstemBild(t *testing.T) {
	var f freigabe
	jetzt := time.Now()
	f.druck(jetzt)
	if f.klick(jetzt.Add(time.Hour)) {
		t.Error("Klick vor dem ersten Bild erlaubt")
	}
}

// TestFreigabeEinKlickJeDruck: Ein zweites Loslassen ohne neues Drücken zählt nicht.
func TestFreigabeEinKlickJeDruck(t *testing.T) {
	t0 := time.Now()
	var f freigabe
	f.bild(t0)
	nach := t0.Add(time.Second)
	f.druck(nach)
	if !f.klick(nach) {
		t.Fatal("gültiger Klick abgelehnt")
	}
	if f.klick(nach) {
		t.Error("Loslassen ohne neues Drücken erlaubt")
	}
}

// TestFreigabeTastaturErlaubtNie: Keine Taste erlaubt; Esc lehnt ab, Return, Leertaste und Tab tun nichts.
func TestFreigabeTastaturErlaubtNie(t *testing.T) {
	var f freigabe
	f.bild(time.Now().Add(-time.Hour))
	for _, k := range []key.Name{key.NameReturn, key.NameEnter, key.NameSpace, key.NameTab, "A"} {
		if f.taste(k) {
			t.Errorf("%q lehnt ab", k)
		}
		if f.druckGueltig {
			t.Errorf("%q machte einen Klick gültig", k)
		}
	}
	if !f.taste(key.NameEscape) {
		t.Error("Esc lehnt nicht ab")
	}
}
