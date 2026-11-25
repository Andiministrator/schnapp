// SPDX-License-Identifier: GPL-3.0-or-later

package rescue

// F-770, F-771 (ADR-0029 Nachtrag 12): Die Marke <SHA>.um ist eine flock-Sperre, die UmDatei vor der Sperre der
// Ausgangsdatei nimmt und nach ihr freigibt. Eine liegen gebliebene Marke hält niemanden auf, Aufraeumen entfernt sie.
// Mutanten: scripts/mutanten/rescue.json (N11 bis N13).

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
)

// Ein Fenster, das genau dann öffnet, wenn UmDatei die Sperre eben genommen hat (vor Lies und vor dem Schreiben),
// wartet auf die schon gehaltene Marke und wendet die nachgezogene Kopie danach an, statt fremd zu werden (F-770).
func TestFensterZwischenSperreUndMarkeWartet(t *testing.T) {
	pfad, ort, _ := aufnahmeMitKopie(t)
	alt := sperrHaken
	t.Cleanup(func() { sperrHaken = alt })
	var einmal atomic.Bool
	ergebnis := make(chan Wiederherstellung, 1)
	sperrHaken = func(name string) {
		if !strings.HasSuffix(name, sperrEndung) || !einmal.CompareAndSwap(false, true) {
			return
		}
		// Das öffnende Fenster wie cmd/schnapp: warten, Kennung, Sicherung, Wiederherstellen.
		go func() {
			WarteUmDatei(ort, pfad, 5*time.Second)
			k, err := KennungVon(pfad)
			if err != nil {
				t.Error(err)
			}
			s := NeueSicherung(ort, k)
			defer s.Ende()
			neu, _ := document.Neu(bild(64, 64))
			ergebnis <- s.Wiederherstellen(neu, nil)
		}()
		time.Sleep(200 * time.Millisecond) // das Fenster läuft, während UmDatei zwischen Sperre und Lies steht
	}
	err := UmDatei(ort, pfad, func(unv func() bool) error {
		unv()
		ueberschreibe(t, pfad, "bild mit schlagwort b, länger")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !einmal.Load() {
		t.Fatal("der Haken lief nicht")
	}
	w := <-ergebnis
	if w.Fremd || !w.Angewendet {
		t.Errorf("Fenster zwischen Sperre und Marke: fremd=%v angewendet=%v grund=%v", w.Fremd, w.Angewendet, w.Grund)
	}
}

// Während UmDatei läuft, ist die Marke gehalten; danach ist sie weg (F-770).
func TestUmDateiHaeltMarkeUndRaeumtSie(t *testing.T) {
	pfad, ort, _ := aufnahmeMitKopie(t)
	var waehrend bool
	err := UmDatei(ort, pfad, func(unv func() bool) error {
		waehrend = UmDateiLaeuft(ort, pfad)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !waehrend {
		t.Error("UmDateiLaeuft meldet während UmDatei nichts")
	}
	if _, err := os.Lstat(filepath.Join(ort, umMarkeName(pfad))); !os.IsNotExist(err) {
		t.Errorf("Marke nach UmDatei: %v", err)
	}
}

// Eine liegen gebliebene, junge Marke und ein Fenster, das die Datei offen hat (hält die Sperre): Ein zweites Öffnen
// wartet nicht, denn kein UmDatei hält die Marke (F-771).
func TestRestMarkeUndOffenesFensterWartetNicht(t *testing.T) {
	pfad, k := ausgang(t, "bild")
	ort := t.TempDir()
	s := NeueSicherung(ort, k) // Fenster 1 hält die Sperre
	defer s.Ende()
	if err := os.WriteFile(filepath.Join(ort, umMarkeName(pfad)), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	WarteUmDatei(ort, pfad, 1500*time.Millisecond)
	if d := time.Since(t0); d > 500*time.Millisecond {
		t.Errorf("wartet %v auf ein UmDatei, das nicht läuft", d)
	}
}

// Eine Marke mit Änderungszeit in der Zukunft (verstellte Uhr) hält ebenfalls nicht auf (F-771).
func TestMarkeInDerZukunftWartetNicht(t *testing.T) {
	pfad, k := ausgang(t, "bild")
	ort := t.TempDir()
	s := NeueSicherung(ort, k)
	defer s.Ende()
	marke := filepath.Join(ort, umMarkeName(pfad))
	if err := os.WriteFile(marke, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	zukunft := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(marke, zukunft, zukunft); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	WarteUmDatei(ort, pfad, 1500*time.Millisecond)
	if d := time.Since(t0); d > 500*time.Millisecond {
		t.Errorf("Marke aus der Zukunft hält %v auf", d)
	}
}

// Aufraeumen kennt die Marke (eigenerName) und entfernt eine freie, alte oder aus der Zukunft stammende; eine junge
// oder eine gehaltene bleibt (F-771).
func TestAufraeumenRaeumtFreieMarke(t *testing.T) {
	ort := t.TempDir()
	jetzt := time.Now()
	anlegen := func(pfad string, zeit time.Time) string {
		t.Helper()
		m := filepath.Join(ort, umMarkeName(pfad))
		if err := os.WriteFile(m, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(m, zeit, zeit); err != nil {
			t.Fatal(err)
		}
		return m
	}
	alt := anlegen("/a/alt.png", jetzt.Add(-400*24*time.Hour))
	minute := anlegen("/a/minute.png", jetzt.Add(-2*time.Minute))
	zukunft := anlegen("/a/zukunft.png", jetzt.Add(48*time.Hour))
	jung := anlegen("/a/jung.png", jetzt)
	gehalten := anlegen("/a/gehalten.png", jetzt.Add(-400*24*time.Hour))
	sp, err := sperreAuf(ort, "/a/gehalten.png", umMarkeName("/a/gehalten.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer sp.frei()
	if !eigenerName(filepath.Base(alt)) {
		t.Errorf("eigenerName kennt %s nicht", filepath.Base(alt))
	}
	n, err := Aufraeumen(ort, jetzt)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{alt, minute, zukunft} {
		if _, err := os.Lstat(m); !os.IsNotExist(err) {
			t.Errorf("freie Marke %s liegt nach Aufraeumen noch (%v)", filepath.Base(m), err)
		}
	}
	for _, m := range []string{jung, gehalten} {
		if _, err := os.Lstat(m); err != nil {
			t.Errorf("Marke %s entfernt: %v", filepath.Base(m), err)
		}
	}
	if n != 3 {
		t.Errorf("entfernt %d, erwartet 3", n)
	}
}
