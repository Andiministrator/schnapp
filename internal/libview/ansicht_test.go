// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/i18n"
	"github.com/Andiministrator/schnapp/internal/library"
)

// ADR-0036 Punkt 3: Zeiträume ab Mitternacht in der Zone von jetzt, heute eingeschlossen.
func TestZeitraumVon(t *testing.T) {
	zone := time.FixedZone("Z", 3600)
	jetzt := time.Date(2026, 3, 5, 14, 30, 0, 0, zone)
	for _, f := range []struct {
		z    Zeitraum
		soll time.Time
	}{
		{ZeitraumHeute, time.Date(2026, 3, 5, 0, 0, 0, 0, zone)},
		{ZeitraumWoche, time.Date(2026, 2, 27, 0, 0, 0, 0, zone)},
		{ZeitraumMonat, time.Date(2026, 2, 4, 0, 0, 0, 0, zone)},
		{ZeitraumJahr, time.Date(2026, 1, 1, 0, 0, 0, 0, zone)},
		{ZeitraumAlle, time.Time{}},
	} {
		if got := f.z.Von(jetzt); !got.Equal(f.soll) {
			t.Errorf("%d: %v, Soll %v", f.z, got, f.soll)
		}
	}
	if len(Zeitraeume) != 5 || Zeitraeume[0] != ZeitraumHeute || Zeitraeume[4] != ZeitraumAlle {
		t.Errorf("Reihenfolge %v", Zeitraeume)
	}
	i18n.Setze(nil)
	namen := map[string]bool{}
	for _, z := range Zeitraeume {
		namen[z.Text()] = true
	}
	if len(namen) != 5 || !namen["All dates"] || !namen["Today"] {
		t.Errorf("Namen %v", namen)
	}
}

// Die Anfrage der Ansicht: Teilwort ohne Leerraum (F-671, nicht das genaue Schlagwort), Zeitraum, Agenten nur mit dem Schalter.
func TestAnfrage(t *testing.T) {
	jetzt := time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)
	a := Anfrage("  bug ", ZeitraumHeute, false, jetzt)
	if a.Teilwort != "bug" || a.Schlagwort != "" || !a.Von.Equal(time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)) || !a.OhneAgenten || !a.Bis.IsZero() {
		t.Errorf("%+v", a)
	}
	if a := Anfrage("", ZeitraumAlle, true, jetzt); a.OhneAgenten || !a.Von.IsZero() || a.Teilwort != "" {
		t.Errorf("mit Agenten: %+v", a)
	}
}

// ADR-0036 Punkt 4: Komma trennt, Leerraum und Leeres fallen weg, Doppeltes ohne Rücksicht auf Groß/klein auch.
func TestZerlegeSchlagworte(t *testing.T) {
	for _, f := range []struct {
		ein  string
		soll []string
	}{
		{"bug, login", []string{"bug", "login"}},
		{" bug ,, Login ,BUG, login", []string{"bug", "Login"}},
		{"", nil},
		{" , ", nil},
		{"rechnung 2026", []string{"rechnung 2026"}},
	} {
		if got := ZerlegeSchlagworte(f.ein); !reflect.DeepEqual(got, f.soll) {
			t.Errorf("%q: %q, Soll %q", f.ein, got, f.soll)
		}
	}
	if s := SchlagwortText([]string{"bug", "login"}); s != "bug, login" {
		t.Errorf("Text %q", s)
	}
	if got := ZerlegeSchlagworte(SchlagwortText([]string{"a b", "c"})); !reflect.DeepEqual(got, []string{"a b", "c"}) {
		t.Errorf("hin und zurück: %q", got)
	}
}

func TestTippeUndZurueck(t *testing.T) {
	if s := Tippe("ab", "c\nd\x7f"); s != "abcd" {
		t.Errorf("ohne Steuerzeichen: %q", s)
	}
	lang := Tippe("", strings.Repeat("ä", MaxEingabe+10))
	if n := len([]rune(lang)); n != MaxEingabe {
		t.Errorf("Grenze: %d", n)
	}
	if s := Tippe(lang, "x"); s != lang {
		t.Error("über der Grenze kommt nichts dazu")
	}
	if s := Zurueck("aä"); s != "a" {
		t.Errorf("Rücktaste: %q", s)
	}
	if s := Zurueck(""); s != "" {
		t.Errorf("leer: %q", s)
	}
}

// Texte unter der Kachel und Überschrift des Monats (englisch ohne Katalog).
func TestKachelTexte(t *testing.T) {
	i18n.Setze(nil)
	jetzt := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	if s := KachelZeit(time.Date(2026, 11, 24, 10, 59, 0, 0, time.UTC), jetzt); s != "Nov 24 10:59" {
		t.Errorf("dieses Jahr: %q", s)
	}
	if s := KachelZeit(time.Date(2025, 10, 3, 9, 5, 0, 0, time.UTC), jetzt); s != "Oct 3, 2025 09:05" {
		t.Errorf("anderes Jahr: %q", s)
	}
	// Gerechnet in der Zone von jetzt.
	if s := KachelZeit(time.Date(2026, 11, 24, 23, 30, 0, 0, time.UTC), jetzt.In(time.FixedZone("O", 3600))); s != "Nov 25 00:30" {
		t.Errorf("Zone: %q", s)
	}
	e := library.Eintrag{Breite: 1200, Hoehe: 800}
	if s := KachelAngaben(e); s != "1200 × 800" {
		t.Errorf("ohne Schlagworte: %q", s)
	}
	e.Schlagworte = []string{"bug", "login"}
	if s := KachelAngaben(e); s != "1200 × 800 · bug, login" {
		t.Errorf("mit Schlagworten: %q", s)
	}
	if s := MonatTitel(Gruppe{Jahr: 2026, Monat: time.November, Von: 2, Bis: 15}); s != "November 2026  ·  13 captures" {
		t.Errorf("Überschrift %q", s)
	}
	if s := MonatTitel(Gruppe{Jahr: 2025, Monat: time.January, Von: 0, Bis: 1}); s != "January 2025  ·  1 capture" {
		t.Errorf("Einzahl %q", s)
	}
	for m := time.January; m <= time.December; m++ {
		if monatName(m) != m.String() {
			t.Errorf("Monat %v: %q", m, monatName(m))
		}
	}
}
