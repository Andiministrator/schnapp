// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"
	"time"
)

// F-243: Eine neue Anforderung, während der Wecker noch wiederholt, zählt die Obergrenze neu. Sonst endete das
// Wecken für die zweite Antwort mit der Frist der ersten.
func TestWeckerNeueAnforderungVerlaengertGrenze(t *testing.T) {
	const takt, grenze = 5 * time.Millisecond, 200 * time.Millisecond
	w := NeuerWecker(func() {}, takt, grenze)
	t.Cleanup(w.Ende)
	w.Anfordern()
	time.Sleep(140 * time.Millisecond)
	w.Anfordern() // läuft schon: dieselbe Goroutine, neue Frist bis etwa 340 ms
	time.Sleep(120 * time.Millisecond)
	// Bei etwa 260 ms: Die erste Frist (200 ms) ist vorbei, die zweite nicht.
	if !w.Offen() {
		t.Fatal("Wecker endete mit der Frist der ersten Anforderung")
	}
	time.Sleep(200 * time.Millisecond)
	// Bei etwa 460 ms: auch die zweite Frist ist vorbei.
	if w.Offen() {
		t.Error("Wecker läuft über die verlängerte Obergrenze hinaus")
	}
}
