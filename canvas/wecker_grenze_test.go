// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"testing"
	"time"
)

// F-243: Eine neue Anforderung, während der Wecker noch wiederholt, zählt die Obergrenze neu. Sonst endete das
// Wecken für die zweite Antwort mit der Frist der ersten.
func TestWeckerNeueAnforderungVerlaengertGrenze(t *testing.T) {
	const takt, grenze = 5 * time.Millisecond, time.Second
	w := NeuerWecker(func() {}, takt, grenze)
	t.Cleanup(w.Ende)
	w.Anfordern()
	time.Sleep(700 * time.Millisecond)
	w.Anfordern() // läuft schon: dieselbe Goroutine, neue Frist bis etwa 1,7 s
	time.Sleep(600 * time.Millisecond)
	// Bei etwa 1,3 s: Die erste Frist (1 s) ist vorbei, die zweite nicht. Je rund 300 ms Luft, auch unter -race auf
	// belasteter Maschine (F-259).
	if !w.Offen() {
		t.Fatal("Wecker endete mit der Frist der ersten Anforderung")
	}
	time.Sleep(time.Second)
	// Bei etwa 2,3 s: auch die zweite Frist ist vorbei.
	if w.Offen() {
		t.Error("Wecker läuft über die verlängerte Obergrenze hinaus")
	}
}
