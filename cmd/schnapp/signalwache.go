// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Signalwache beim Schreiben der Ausgabe von --output und --export (ADR-0030 Nachtrag 1, F-396, F-441 bis F-443).

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Andiministrator/schnapp/internal/export"
	"github.com/Andiministrator/schnapp/internal/i18n"
)

// Nähte für Tests: Anmelden und Abmelden der Signale, Beenden des Prozesses (zweites Signal) und das Schreiben der
// Bytes selbst (blockierendes oder verzögertes Schreiben nachbauen, statt einen fremden Prozess zu beobachten).
var (
	meldeAn       = signal.Notify
	meldeAb       = signal.Stop
	beendeProzess = os.Exit
	// wacheFehler liefert stderr der Signalwache von --output/--export (Meldung beim zweiten Signal), zur Zeit der Meldung.
	wacheFehler = func() io.Writer { return os.Stderr }
	schreibNaht = func(w io.Writer, inhalt []byte, abgebrochen func() bool) error {
		_, err := w.Write(inhalt)
		return err
	}
)

// errAbgebrochen: SIGTERM, SIGINT oder SIGHUP kam, bevor das erste Byte geschrieben wurde (F-396). Das Ziel ist
// unberührt, es bleibt keine temporäre Datei.
var errAbgebrochen = errors.New("schnapp: Schreiben durch Signal abgebrochen")

// errNachSchreiben: Das Signal kam während oder nach dem Schreiben; das Schreiben selbst gelang, das Ziel ist
// vollständig ersetzt (F-441). Der Aufruf endet trotzdem mit Exit 1 (ADR-0030 Nachtrag 1), meldet aber den wahren Stand.
var errNachSchreiben = errors.New("schnapp: Signal kam nach dem Schreiben, Ziel vollständig")

// signaleDerWache sind die Signale, die die Wache fängt.
var signaleDerWache = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP}

const tempVorsatz = ".schnapp-export-"

// mitSignalwache fängt SIGTERM, SIGINT und SIGHUP, solange f läuft (nur das kurze Schreiben der Ausgabe von --output und
// --export, nie das Rendern und nie der Editor). Ein Signal tötet den Prozess dann nicht mitten im Schreiben und lässt
// keine `.schnapp-export-*` neben dem Ziel liegen: f fragt abgebrochen() vor dem Schreiben, ein Fehler räumt die
// temporäre Datei ab. Kam ein Signal vor dem ersten Byte, endet f mit errAbgebrochen; kam es danach, endet
// mitSignalwache mit errNachSchreiben, auch wenn f gelang (die Datei ist dann vollständig, F-441).
// Signale, die der Prozess schon ignoriert (etwa SIGHUP unter nohup), meldet die Wache nicht an (F-443): Notify höbe
// das Ignorieren auf. Hängt das Schreiben (blockierendes Dateisystem), beendet ein zweites Signal den Prozess nach dem
// Entfernen der temporären Datei mit Exit 1 (F-442); f meldet deren Namen über merkeTemp. Entfernt wird nur eine Datei
// mit dem Vorsatz `.schnapp-export-`, nie etwas anderes. Der Editor-Weg fängt seine Signale selbst
// (internal/ui/editor.go) und ruft schreibeBild nie auf.
func mitSignalwache(f func(abgebrochen func() bool, merkeTemp func(string)) error) error {
	var kamen atomic.Int32 // gezählte Signale
	var temp atomic.Pointer[string]
	var direkt atomic.Bool // Rückfall: das Ziel selbst wird beschrieben, es gibt keine temporäre Datei (F-458)
	var fangen []os.Signal
	for _, s := range signaleDerWache {
		if !signal.Ignored(s) {
			fangen = append(fangen, s)
		}
	}
	kanal := make(chan os.Signal, 4)
	if len(fangen) > 0 { // Notify ohne Signale fängt alle
		meldeAn(kanal, fangen...)
	}
	// nimm zählt die bisher eingetroffenen Signale; nur hier wird aus dem Kanal gelesen, damit kein Signal zwischen
	// zwei Lesern verloren geht oder vom falschen gezählt wird.
	nimm := func() {
		for {
			select {
			case <-kanal:
				kamen.Add(1)
			default:
				return
			}
		}
	}
	abgebrochen := func() bool { nimm(); return kamen.Load() > 0 }
	fertig := make(chan struct{})
	ende := make(chan struct{})
	go func() { // wartet auf das zweite Signal, während das Schreiben hängt (F-442)
		defer close(ende)
		takt := time.NewTicker(10 * time.Millisecond)
		defer takt.Stop()
		for {
			select {
			case <-fertig:
				return
			case <-takt.C:
				if nimm(); kamen.Load() >= 2 {
					meldeKurz(wacheFehler(), zweitesSignalMeldung(direkt.Load(), temp.Load()))
					beendeProzess(exitFehler)
					return
				}
			}
		}
	}()
	var gemerkt *string
	err := f(abgebrochen, func(name string) {
		if name == "" { // kein Dateiname: der Rückfall schreibt das Ziel direkt (F-458)
			direkt.Store(true)
			return
		}
		temp.Store(&name)
		if gemerkt != nil {
			laufendeTemps.Delete(gemerkt)
		}
		gemerkt = &name
		laufendeTemps.Store(gemerkt, name)
	})
	if gemerkt != nil {
		laufendeTemps.Delete(gemerkt)
	}
	meldeAb(kanal)
	close(fertig)
	<-ende
	// Ein Signal, das knapp vor dem Abmelden eintraf, zählt noch.
	if abgebrochen() && err == nil {
		return errNachSchreiben
	}
	return err
}

// meldungFrist: So lange wartet ein Beenden höchstens auf seine Meldung auf stderr (Sitzungs-QA QS-1). Variable für Tests.
var meldungFrist = 200 * time.Millisecond

// meldeKurz schreibt text als Zeile nach w, wartet darauf aber höchstens meldungFrist: Hängt stderr (eine volle Pipe,
// die niemand liest), hält die Meldung ein Beenden nicht auf (QS-1). Das Schreiben läuft in einer eigenen Goroutine;
// kehrt es nicht zurück, bleibt sie liegen — der Prozess endet ohnehin.
func meldeKurz(w io.Writer, text string) {
	fertig := make(chan struct{})
	go func() {
		defer close(fertig)
		fmt.Fprintln(w, text)
	}()
	frist := time.NewTimer(meldungFrist)
	defer frist.Stop()
	select {
	case <-fertig:
	case <-frist.C:
	}
}

// zweitesSignalMeldung räumt, soweit möglich, auf und liefert den wahren Stand für die Meldung beim zweiten Signal
// (F-458). Im Rückfall (direkt) beschreibt der hängende Schreiber das Ziel selbst: Die Wache fasst es nicht an, ein
// zweiter Schreiber auf dieselbe Datei risse sie womöglich nur weiter entzwei; sie meldet ehrlich, dass es unvollständig
// sein kann. Ohne Namen der temporären Datei hat das Schreiben noch nicht begonnen.
func zweitesSignalMeldung(direkt bool, temp *string) string {
	switch {
	case direkt:
		return i18n.T("Interrupted again while writing directly to the target; it may be incomplete.")
	case temp == nil:
		return i18n.T("Interrupted again before writing started; the target was not changed.")
	}
	err := entferneTemp(*temp)
	switch {
	case err == nil:
		return i18n.T("Interrupted again while writing; the temporary file was removed.")
	case errors.Is(err, os.ErrNotExist):
		// Umbenannt oder schon weg: das Ziel ist vollständig ersetzt.
		return i18n.T("Interrupted again; the file was already complete.")
	}
	return i18n.T("Interrupted again while writing; the temporary file could not be removed.")
}

// laufendeTemps hält die temporären Dateien der gerade laufenden Schreibvorgänge unter der Signalwache. Ein anderer
// Signal-Empfänger, der den Prozess beendet (zweites Signal im MCP-Server), räumt sie vorher ab (F-484): Sonst überholte
// er die Wache, die nur im 10-ms-Takt nachsieht, und die Datei mit Bildschirminhalt bliebe liegen.
var laufendeTemps sync.Map // *string → string

// raeumeLaufendeTempsAuf entfernt die temporären Dateien aller laufenden Schreibvorgänge (nur mit dem Vorsatz der
// Exportdatei, entferneTemp).
func raeumeLaufendeTempsAuf() {
	laufendeTemps.Range(func(_, name any) bool {
		if err := entferneTemp(name.(string)); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, i18n.T("Interrupted again while writing; the temporary file could not be removed."))
		}
		return true
	})
}

// errKeinTemp: der Name trägt den Vorsatz der Exportdatei nicht; entferneTemp fasst ihn nicht an.
var errKeinTemp = errors.New("schnapp: keine temporäre Exportdatei")

// entferneTemp entfernt die temporäre Datei name, aber nur, wenn sie den Vorsatz der
// Exportdatei trägt (der Name stammt aus internal/export).
func entferneTemp(name string) error {
	if !strings.HasPrefix(filepath.Base(name), tempVorsatz) {
		return errKeinTemp
	}
	return os.Remove(name)
}

// schreibeInhalt schreibt inhalt atomar nach abs, unter der Signalwache (F-396). vorSchreiben (nur Tests) läuft
// innerhalb der Wache, bevor das Schreiben beginnt.
func schreibeInhalt(abs string, inhalt []byte, privat bool, vorSchreiben func(abgebrochen func() bool)) error {
	schreiber := export.SchreibeAtomar
	if privat {
		schreiber = export.SchreibeAtomarPrivat
	}
	return schreibeInhaltMit(abs, inhalt, schreiber, vorSchreiben)
}

// schreibeInhaltMit ist schreibeInhalt mit frei gewähltem Schreiber aus internal/export (MCP: ohne Ersetzen,
// ADR-0030 Nachtrag 4 Punkt 2).
func schreibeInhaltMit(abs string, inhalt []byte, schreiber func(string, func(io.Writer) error) error,
	vorSchreiben func(abgebrochen func() bool)) error {
	return mitSignalwache(func(abgebrochen func() bool, merkeTemp func(string)) error {
		if vorSchreiben != nil {
			vorSchreiben(abgebrochen)
		}
		schreibe := func(w io.Writer) error {
			if abgebrochen() {
				return errAbgebrochen
			}
			if n, ok := w.(interface{ Name() string }); ok {
				merkeTemp(n.Name())
			} else {
				merkeTemp("") // Rückfall: Puffer statt temporärer Datei, danach schreibt export das Ziel direkt
			}
			return schreibNaht(w, inhalt, abgebrochen)
		}
		return schreiber(abs, schreibe)
	})
}

// meldeSchreibFehler schreibt die Meldung zu einem gescheiterten Schreiben von ziel auf fehler. Kam das Signal erst
// nach dem Schreiben (F-441), steht dort der wahre Stand: Die Datei liegt vollständig vor.
func meldeSchreibFehler(fehler io.Writer, ziel string, err error) {
	if errors.Is(err, errNachSchreiben) {
		fmt.Fprintf(fehler, i18n.T("Interrupted by a signal after %s was written completely.")+"\n", ziel)
		return
	}
	if errors.Is(err, errAbgebrochen) {
		// F-452: Das Signal kam vor dem ersten Byte, das Ziel ist unberührt.
		fmt.Fprintf(fehler, i18n.T("Interrupted before anything was written to %s; it is untouched.")+"\n", ziel)
		return
	}
	fmt.Fprintf(fehler, i18n.T("Could not write %s: %s")+"\n", ziel, schreibFehler(err))
}
