// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"sync"
	"time"
)

// Wecker weckt die Ereignisschleife eines Fensters, bis sie sich meldet (F-231). Ein einzelnes Window.Invalidate
// garantiert das nicht: Gio verwirft es, solange seit der letzten leeren Ereignisschlange schon eines angenommen wurde
// (app.Window.mayInvalidate). Trifft eine Antwort (Dateidialog) oder ein Signal ein, während ein angestoßenes Bild schon
// hinter seinem Abholen ist, bliebe sie bis zum nächsten Ereignis liegen. Der Wecker wiederholt das Wecken im Takt, bis
// die Schleife Quittieren aufruft — am Anfang jedes Ereignisses; dann sind Antwort und Marke, die vor Anfordern
// abgelegt wurden, in der Schleife sichtbar. Er kennt kein Gio: Das Wecken kommt als Funktion vom Aufrufer.
//
// Er tut nichts außer wecken und beendet sich von selbst: bei Quittieren, bei Ende und spätestens nach der Obergrenze.
type Wecker struct {
	wecke func()
	takt  time.Duration
	grenz time.Duration

	mu      sync.Mutex
	offen   bool // angefordert, noch nicht quittiert
	laeuft  bool // Wiederholungs-Goroutine aktiv
	beendet bool
	gen     uint64 // zählt Anforderungen; eine neue verlängert die Obergrenze
}

// NeuerWecker liefert einen Wecker, der wecke je takt aufruft, höchstens grenze lang je Anforderung.
func NeuerWecker(wecke func(), takt, grenze time.Duration) *Wecker {
	return &Wecker{wecke: wecke, takt: takt, grenz: grenze}
}

// Anfordern weckt sofort und wiederholt das Wecken, bis Quittieren oder Ende kommt. Aus jeder Goroutine erlaubt. Die
// Daten, auf die die Schleife warten soll, müssen vorher abgelegt sein.
func (w *Wecker) Anfordern() {
	w.mu.Lock()
	if w.beendet {
		w.mu.Unlock()
		return
	}
	w.offen = true
	w.gen++
	start := !w.laeuft
	w.laeuft = true
	w.mu.Unlock()
	w.wecke()
	if start {
		go w.wiederhole()
	}
}

// Quittieren meldet, dass die Schleife ein Ereignis bearbeitet: Eine vorher abgelegte Antwort ist ab jetzt in der
// Schleife zu sehen. Eine Anforderung nach diesem Aufruf bleibt offen.
func (w *Wecker) Quittieren() {
	w.mu.Lock()
	w.offen = false
	w.mu.Unlock()
}

// Offen meldet, ob eine Anforderung noch nicht quittiert ist.
func (w *Wecker) Offen() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.offen
}

// Ende beendet das Wiederholen; danach tut Anfordern nichts mehr.
func (w *Wecker) Ende() {
	w.mu.Lock()
	w.beendet = true
	w.offen = false
	w.mu.Unlock()
}

func (w *Wecker) wiederhole() {
	t := time.NewTicker(w.takt)
	defer t.Stop()
	w.mu.Lock()
	gen := w.gen
	w.mu.Unlock()
	ablauf := time.Now().Add(w.grenz)
	for range t.C {
		w.mu.Lock()
		if w.beendet || !w.offen {
			w.laeuft = false
			w.mu.Unlock()
			return
		}
		if w.gen != gen { // neue Anforderung: Obergrenze neu zählen
			gen = w.gen
			ablauf = time.Now().Add(w.grenz)
		}
		if time.Now().After(ablauf) {
			w.offen = false
			w.laeuft = false
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
		w.wecke()
	}
}
