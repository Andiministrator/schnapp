// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"
)

func TestZugZeitenMittelUndMaximum(t *testing.T) {
	var z zugZeiten
	z.bild(time.Second) // außerhalb eines Zugs: nichts
	if _, _, _, ok := z.beende(); ok {
		t.Fatal("beende ohne Zug meldet ok")
	}
	z.beginne()
	for _, ms := range []int{10, 30, 20} {
		z.bild(time.Duration(ms) * time.Millisecond)
	}
	n, mittel, laengste, ok := z.beende()
	if !ok || n != 3 || mittel != 20*time.Millisecond || laengste != 30*time.Millisecond {
		t.Fatalf("n=%d mittel=%v längste=%v ok=%v", n, mittel, laengste, ok)
	}
	if _, _, _, ok := z.beende(); ok {
		t.Error("zweites beende meldet ok")
	}
}

func TestZugZeitenOhneBilderUndNeuerZug(t *testing.T) {
	var z zugZeiten
	z.beginne()
	n, mittel, laengste, ok := z.beende()
	if !ok || n != 0 || mittel != 0 || laengste != 0 {
		t.Fatalf("leerer Zug: n=%d mittel=%v längste=%v ok=%v", n, mittel, laengste, ok)
	}
	z.beginne()
	z.bild(5 * time.Millisecond)
	z.beginne() // neuer Zug verwirft den alten Stand
	z.bild(7 * time.Millisecond)
	n, mittel, laengste, _ = z.beende()
	if n != 1 || mittel != 7*time.Millisecond || laengste != 7*time.Millisecond {
		t.Fatalf("n=%d mittel=%v längste=%v", n, mittel, laengste)
	}
}
