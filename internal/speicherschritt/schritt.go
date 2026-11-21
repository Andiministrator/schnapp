// SPDX-License-Identifier: GPL-3.0-or-later

// Package speicherschritt meldet die Schritte des Speicherns (F-297): Zeitmarken zum Messen. Mit gesetztem
// SCHNAPP_DEBUG schreibt Marke eine Zeile `[speichern] <Schritt>: +<Dauer seit der vorigen Marke> (gesamt <Dauer>)`
// auf stderr; ein Test oder Benchmark hängt sich mit Haken ein. Ohne beides tut Marke nichts. Das Paket ändert kein
// Verhalten des Speicherns und trägt keine Fortschrittsmeldung für die Oberfläche (das bleibt Karte #119, F-298).
package speicherschritt

import (
	"log"
	"os"
	"sync"
	"time"
)

var (
	mu    sync.Mutex
	start time.Time
	vorig time.Time
	// Haken, falls gesetzt, bekommt jede Marke mit dem Namen des Schritts (nur für Tests und Benchmarks; vor dem
	// Lauf setzen, nicht nebenläufig ändern).
	Haken func(schritt string)
)

// Marke meldet, dass der Schritt schritt gerade fertig ist. Die Marke "start" setzt die Uhr zurück.
func Marke(schritt string) {
	debug := os.Getenv("SCHNAPP_DEBUG") != ""
	if !debug && Haken == nil {
		return
	}
	jetzt := time.Now()
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
