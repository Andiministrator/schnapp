// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"os"
	"sync"
	"sync/atomic"
)

// Signalwache nimmt die Signale an, die den Editor geordnet beenden (SIGTERM, SIGINT, SIGHUP; ADR-0020 Nachtrag 2), und
// hält sie bis zum Ende fest (F-152): Das erste Signal setzt die Marke und weckt das Fenster, jedes weitere wird nur
// angenommen. Solange die Wache läuft, gilt für diese Signale nie die Voreinstellung (Prozess beenden) — ein zweites
// Strg+C oder SIGHUP nach SIGTERM beim Abmelden bricht ein laufendes Speichern nicht ab. Erst Ende gibt die Signale
// frei. Die Wache kennt weder Gio noch os/signal: Der Kanal und das Freigeben (signal.Stop) kommen vom Aufrufer, im
// Test ein eigener Kanal ohne echte Signale.
type Signalwache struct {
	angekommen atomic.Bool
	stoppen    func()
	fertig     chan struct{}
	beendet    chan struct{}
	einmal     sync.Once
}

// NeueSignalwache startet die Wache auf ein. melden läuft in der Wach-Goroutine, je Signal einmal, erstes sagt, ob es
// das erste war (dann weckt der Aufrufer das Fenster; gespeichert wird in dessen Ereignisschleife, F-77). stoppen gibt
// die Signale frei und läuft genau einmal, in Ende.
func NeueSignalwache(ein <-chan os.Signal, stoppen func(), melden func(s os.Signal, erstes bool)) *Signalwache {
	w := &Signalwache{stoppen: stoppen, fertig: make(chan struct{}), beendet: make(chan struct{})}
	go func() {
		defer close(w.beendet)
		for {
			select {
			case s := <-ein:
				erstes := !w.angekommen.Swap(true)
				melden(s, erstes)
			case <-w.fertig:
				return
			}
		}
	}()
	return w
}

// Angekommen meldet, ob ein Signal kam (die Ereignisschleife fragt nach jedem Ereignis).
func (w *Signalwache) Angekommen() bool { return w.angekommen.Load() }

// Ende gibt die Signale frei und beendet die Wache; danach gilt wieder die Voreinstellung. Mehrfacher Aufruf schadet
// nicht.
func (w *Signalwache) Ende() {
	w.einmal.Do(func() {
		w.stoppen()
		close(w.fertig)
		<-w.beendet
	})
}
