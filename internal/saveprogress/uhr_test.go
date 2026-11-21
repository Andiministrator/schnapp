// SPDX-License-Identifier: GPL-3.0-or-later

package saveprogress

import (
	"testing"
	"time"
)

// SetzeHaken liefert den bisherigen Haken zurück; Marke ruft den gesetzten auf (F-318).
func TestSetzeHaken(t *testing.T) {
	var gesehen []string
	alt := SetzeHaken(func(s string) { gesehen = append(gesehen, s) })
	t.Cleanup(func() { SetzeHaken(alt) })
	if alt != nil {
		t.Fatal("ein Haken war schon gesetzt")
	}
	Marke("a")
	zweiter := SetzeHaken(nil)
	if zweiter == nil {
		t.Fatal("SetzeHaken gibt den bisherigen nicht zurück")
	}
	Marke("b") // ohne Haken
	if len(gesehen) != 1 || gesehen[0] != "a" {
		t.Errorf("Haken sah %v", gesehen)
	}
}

// Mit Ersatzuhr folgt Zuletzt der Ersatzuhr, und das Zurückstellen vergisst den Zeitpunkt aus der anderen Uhr (F-317).
func TestSetzeUhr(t *testing.T) {
	jetzt := Basis().Add(time.Hour)
	alt := SetzeUhr(func() time.Time { return jetzt })
	t.Cleanup(func() { SetzeUhr(alt) })
	Puls()
	if z := Zuletzt(); !z.Equal(jetzt) {
		t.Errorf("Zuletzt %v, erwartet %v", z, jetzt)
	}
	jetzt = jetzt.Add(5 * time.Second)
	Puls()
	if z := Zuletzt(); !z.Equal(jetzt) || !Jetzt().Equal(jetzt) {
		t.Errorf("Zuletzt %v, Jetzt %v, erwartet %v", z, Jetzt(), jetzt)
	}
	SetzeUhr(nil)
	if !Zuletzt().IsZero() {
		t.Error("Zuletzt aus der Ersatzuhr überlebt das Zurückstellen")
	}
}
