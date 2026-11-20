// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"sync/atomic"
	"testing"
	"time"
)

// TestWeckerWiederholtBisQuittiert (F-231): Ein verworfenes erstes Wecken (Gio-Invalidate verpufft) ist nicht das Ende;
// der Wecker wiederholt, bis die Schleife quittiert, und hört danach auf.
func TestWeckerWiederholtBisQuittiert(t *testing.T) {
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, 5*time.Millisecond, time.Minute)
	defer w.Ende()
	w.Anfordern()
	deadline := time.Now().Add(2 * time.Second)
	for n.Load() < 4 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if n.Load() < 4 {
		t.Fatalf("nur %d Weckrufe, erwartet Wiederholungen", n.Load())
	}
	w.Quittieren()
	time.Sleep(40 * time.Millisecond) // ein laufender Takt darf noch auslaufen
	vor := n.Load()
	time.Sleep(60 * time.Millisecond)
	if n.Load() != vor {
		t.Errorf("Weckrufe nach Quittieren: %d -> %d", vor, n.Load())
	}
	if w.Offen() {
		t.Error("Offen nach Quittieren")
	}
}

// TestWeckerAnforderungNachQuittieren: Eine Anforderung, die nach dem Quittieren eintrifft, wird nicht verschluckt.
func TestWeckerAnforderungNachQuittieren(t *testing.T) {
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, 5*time.Millisecond, time.Minute)
	defer w.Ende()
	w.Anfordern()
	w.Quittieren()
	w.Anfordern() // die Antwort kam nach dem Ereignisanfang
	if !w.Offen() {
		t.Fatal("zweite Anforderung nicht offen")
	}
	vor := n.Load()
	time.Sleep(80 * time.Millisecond)
	if n.Load() <= vor {
		t.Error("keine Wiederholung nach der Anforderung nach Quittieren")
	}
}

// TestWeckerObergrenze: Ohne Quittieren hört der Wecker nach der Obergrenze auf (Wächterregel: harte Zeitgrenze).
func TestWeckerObergrenze(t *testing.T) {
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, 5*time.Millisecond, 40*time.Millisecond)
	defer w.Ende()
	w.Anfordern()
	time.Sleep(150 * time.Millisecond)
	vor := n.Load()
	time.Sleep(60 * time.Millisecond)
	if n.Load() != vor {
		t.Errorf("Wecker lief über die Obergrenze: %d -> %d", vor, n.Load())
	}
}

// TestWeckerEnde: Nach Ende weckt Anfordern nicht mehr.
func TestWeckerEnde(t *testing.T) {
	var n atomic.Int32
	w := NeuerWecker(func() { n.Add(1) }, 5*time.Millisecond, time.Minute)
	w.Ende()
	w.Anfordern()
	if n.Load() != 0 {
		t.Errorf("%d Weckrufe nach Ende", n.Load())
	}
}
