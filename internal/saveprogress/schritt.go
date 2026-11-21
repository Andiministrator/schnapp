// SPDX-License-Identifier: GPL-3.0-or-later

// Package saveprogress meldet den Fortschritt des Speicherns. Zwei Arten von Meldungen:
//
//   - Marke(schritt) zum Abschluss eines benannten Schritts (F-297): Mit gesetztem SCHNAPP_DEBUG schreibt sie eine
//     Zeile `[speichern] <Schritt>: +<Dauer seit der vorigen Marke> (gesamt <Dauer>)` auf stderr; ein Test oder
//     Benchmark hängt sich mit Haken ein.
//   - Puls() aus langen Schleifen (Rastern, PNG-Kodierung über Schreiber, Lesen über Leser, Verkleinern): billig, ohne
//     Ausgabe.
//
// Beide setzen den Zeitpunkt des letzten Fortschritts (Zuletzt). Daran erkennt der Editor ein hängendes Speichern: Es
// gilt erst als hängend, wenn eine Frist lang kein Fortschritt kam (F-298, ADR-0020 Nachtrag 9). Der Zeitpunkt gilt
// für den ganzen Prozess; auch ein Kopieren oder Exportieren, das nebenher rendert, setzt ihn. Das verzögert das
// Erkennen eines hängenden Speicherns höchstens um die Dauer dieser Arbeit und ist hingenommen.
package saveprogress

import (
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var (
	mu    sync.Mutex
	start time.Time
	vorig time.Time
	// Haken, falls gesetzt, bekommt jede Marke mit dem Namen des Schritts (nur für Tests und Benchmarks; vor dem
	// Lauf setzen, nicht nebenläufig ändern).
	Haken func(schritt string)

	// basis trägt den monotonen Anteil der Uhr (time.Now beim Laden des Pakets); zuletzt zählt Nanosekunden seit basis
	// plus 1, damit 0 „noch keiner“ bleibt. So springt Zuletzt nicht mit der Wanduhr (Aufwachen, NTP; Verify F-298 B-5).
	basis   = time.Now()
	zuletzt atomic.Int64
	pulse   atomic.Uint64 // Anzahl der Pulse (nicht der Marken), für Tests
)

func merke() { zuletzt.Store(int64(time.Since(basis)) + 1) }

// Marke meldet, dass der Schritt schritt gerade fertig ist. Die Marke "start" setzt die Uhr der Debug-Ausgabe zurück.
func Marke(schritt string) {
	merke()
	jetzt := time.Now()
	debug := os.Getenv("SCHNAPP_DEBUG") != ""
	if !debug && Haken == nil {
		return
	}
	mu.Lock()
	if schritt == "start" || start.IsZero() {
		start, vorig = jetzt, jetzt
	}
	seit, gesamt := jetzt.Sub(vorig), jetzt.Sub(start)
	vorig = jetzt
	mu.Unlock()
	if debug {
		log.Printf("[speichern] %s: +%s (gesamt %s)", schritt, seit.Round(time.Millisecond), gesamt.Round(time.Millisecond))
	}
	if Haken != nil {
		Haken(schritt)
	}
}

// Puls meldet Fortschritt ohne Schrittnamen und ohne Ausgabe (aus langen Schleifen).
func Puls() {
	merke()
	pulse.Add(1)
}

// Pulse zählt die bisherigen Pulse im Prozess (Tests prüfen damit, dass ein Weg Fortschritt meldet).
func Pulse() uint64 { return pulse.Load() }

// Zuletzt liefert den Zeitpunkt des letzten Fortschritts im Prozess, mit monotonem Anteil (Vergleiche und time.Since
// folgen keinem Sprung der Wanduhr); der Nullwert, solange es keinen gab.
func Zuletzt() time.Time {
	n := zuletzt.Load()
	if n == 0 {
		return time.Time{}
	}
	return basis.Add(time.Duration(n - 1))
}

// Schreiber reicht jedes Write an w weiter und meldet dabei Fortschritt (PNG-Kodierung schreibt in Stücken).
func Schreiber(w io.Writer) io.Writer { return pulsSchreiber{w} }

// Leser reicht jedes Read an r weiter und meldet dabei Fortschritt (Dekodieren der alten Datei).
func Leser(r io.Reader) io.Reader { return pulsLeser{r} }

type pulsSchreiber struct{ w io.Writer }

func (p pulsSchreiber) Write(b []byte) (int, error) {
	Puls()
	return p.w.Write(b)
}

type pulsLeser struct{ r io.Reader }

func (p pulsLeser) Read(b []byte) (int, error) {
	Puls()
	return p.r.Read(b)
}
