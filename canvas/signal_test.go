// SPDX-License-Identifier: GPL-3.0-or-later

package canvas

import (
	"errors"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Die Tests schicken keine echten Signale: Die Wache bekommt einen eigenen, ungepufferten Kanal. Ein Senden darauf
// gelingt nur, wenn die Wache noch annimmt — genau das, was im Programm signal.Notify braucht, damit nicht die
// Voreinstellung (Prozess beenden) greift.

type meldung struct {
	s      os.Signal
	erstes bool
}

type wachTest struct {
	ein     chan os.Signal
	stopps  atomic.Int32
	meldung chan meldung
	wache   *Signalwache
}

func neueWache(t *testing.T) *wachTest {
	t.Helper()
	w := &wachTest{ein: make(chan os.Signal), meldung: make(chan meldung, 16)}
	w.wache = NeueSignalwache(w.ein, func() { w.stopps.Add(1) }, func(s os.Signal, erstes bool) {
		w.meldung <- meldung{s, erstes}
	})
	t.Cleanup(w.wache.Ende)
	return w
}

// senden schickt ein Signal; nimmt die Wache es nicht binnen zwei Sekunden an, hätte im Programm die Voreinstellung
// gegriffen.
func (w *wachTest) senden(t *testing.T, s os.Signal) {
	t.Helper()
	select {
	case w.ein <- s:
	case <-time.After(2 * time.Second):
		t.Fatalf("Signal %v nicht angenommen: Die Wache fängt nicht mehr ab (Prozess wäre beendet)", s)
	}
}

// naechste wartet auf die Meldung der Wach-Goroutine zum zuletzt gesendeten Signal.
func (w *wachTest) naechste(t *testing.T) meldung {
	t.Helper()
	select {
	case m := <-w.meldung:
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("keine Meldung der Wache")
		return meldung{}
	}
}

// F-152: Erstes Signal stößt Speichern an; weitere Signale während des Speicherns werden angenommen, ohne das
// Abfangen aufzuheben; nach dem Speichern endet alles geordnet, erst dann werden die Signale freigegeben.
func TestSignalWaehrendSpeichernAbgefangen(t *testing.T) {
	s, _, _ := sitzung(t)
	verschiebe(t, s, 10)
	w := neueWache(t)
	if w.wache.Angekommen() {
		t.Fatal("Angekommen vor dem ersten Signal")
	}

	w.senden(t, syscall.SIGTERM)
	if m := w.naechste(t); m.s != syscall.SIGTERM || !m.erstes {
		t.Fatalf("erste Meldung %+v (Soll SIGTERM, erstes)", m)
	}
	if !w.wache.Angekommen() {
		t.Fatal("nach SIGTERM nicht angekommen")
	}

	// Die Ereignisschleife sieht die Marke und speichert; währenddessen kommen SIGHUP (logind) und Strg+C.
	aufrufe := 0
	err := s.SchliessenNachSignal(func() error {
		aufrufe++
		w.senden(t, syscall.SIGHUP)
		w.senden(t, syscall.SIGINT)
		if n := w.stopps.Load(); n != 0 {
			t.Errorf("während des Speicherns %d× freigegeben (Soll 0)", n)
		}
		for _, soll := range []os.Signal{syscall.SIGHUP, syscall.SIGINT} {
			if m := w.naechste(t); m.s != soll || m.erstes {
				t.Errorf("Meldung %+v (Soll %v, nicht erstes)", m, soll)
			}
		}
		return nil
	})
	if err != nil || aufrufe != 1 {
		t.Fatalf("Speichern nach Signal: %v, %d Aufrufe (Soll nil, 1)", err, aufrufe)
	}
	if s.Geaendert() || s.Ungesichert() {
		t.Errorf("nach Speichern: geändert %v, ungesichert %v (Soll false, false)", s.Geaendert(), s.Ungesichert())
	}
	// Noch ein Signal zwischen Speichern und Rückkehr des Editors: weiter abgefangen.
	w.senden(t, syscall.SIGTERM)
	if m := w.naechste(t); m.erstes {
		t.Errorf("späteres Signal als erstes gemeldet: %+v", m)
	}
	if n := w.stopps.Load(); n != 0 {
		t.Fatalf("vor Ende %d× freigegeben (Soll 0)", n)
	}

	w.wache.Ende()
	w.wache.Ende()
	if n := w.stopps.Load(); n != 1 {
		t.Errorf("nach Ende %d× freigegeben (Soll genau 1)", n)
	}
	select {
	case m := <-w.meldung:
		t.Errorf("überzählige Meldung %+v", m)
	default:
	}
}

// Ende ohne Signal gibt frei und meldet nichts.
func TestSignalwacheEndeOhneSignal(t *testing.T) {
	w := neueWache(t)
	w.wache.Ende()
	if w.wache.Angekommen() || w.stopps.Load() != 1 || len(w.meldung) != 0 {
		t.Errorf("angekommen %v, freigegeben %d, Meldungen %d (Soll false, 1, 0)",
			w.wache.Angekommen(), w.stopps.Load(), len(w.meldung))
	}
}

// Signal nach gescheitertem Speichern beim Schließen (Fenster offen mit Fehlermeldung): Das Signal versucht das
// Speichern noch einmal — das Verwerfen bestätigt nur eine zweite Schließen-Anfrage, kein Signal — und gibt das Fenster
// in jedem Fall frei (der Editor schließt danach ohne Rückfrage, ADR-0020 Nachtrag 2).
func TestSignalNachGescheitertemSpeichern(t *testing.T) {
	kaputt := errors.New("Platte voll")
	for _, fall := range []struct {
		name    string
		zweites error
	}{
		{"gelingt jetzt", nil},
		{"scheitert wieder", kaputt},
	} {
		t.Run(fall.name, func(t *testing.T) {
			s, _, _ := sitzung(t)
			verschiebe(t, s, 10)
			if z, err := s.Schliessen(func() (bool, error) { return false, kaputt }); z != SchlussFehler || err == nil {
				t.Fatalf("Schließen: %v, %v (Soll SchlussFehler)", z, err)
			}
			aufrufe := 0
			err := s.SchliessenNachSignal(func() error { aufrufe++; return fall.zweites })
			if aufrufe != 1 {
				t.Fatalf("Speichern nach Signal: %d Aufrufe (Soll 1, trotz vorher gescheitertem Speichern)", aufrufe)
			}
			if !errors.Is(err, fall.zweites) || (fall.zweites == nil) != (err == nil) {
				t.Errorf("Fehler %v (Soll %v)", err, fall.zweites)
			}
			// Kein weiterer Versuch beim Zerstören: Gespeichert oder (nach zweitem Scheitern) verworfen.
			if s.Ungesichert() {
				t.Error("nach Signal noch ungesichert (Zerstören speicherte ein drittes Mal)")
			}
			if (fall.zweites == nil) == s.Geaendert() {
				t.Errorf("geändert %v nach Ausgang %v", s.Geaendert(), fall.zweites)
			}
		})
	}
}

// Ohne Änderung speichert ein Signal nicht (F-109: keine neu geschriebene Datei ohne Grund).
func TestSignalOhneAenderungSpeichertNicht(t *testing.T) {
	s, _, _ := sitzung(t)
	aufrufe := 0
	if err := s.SchliessenNachSignal(func() error { aufrufe++; return nil }); err != nil || aufrufe != 0 {
		t.Errorf("%v, %d Aufrufe (Soll nil, 0)", err, aufrufe)
	}
}
