// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/library"
)

// F-688 (KB1): Enden im Hintergrund laufen nacheinander; rettungEnden schließt erst, wenn auch das frühere fertig ist.
func TestRettungEndenNacheinander(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	tor := make(chan struct{})
	var einmal sync.Once
	oeffne := func() { einmal.Do(func() { close(tor) }) }
	t.Cleanup(oeffne)
	erst, dann := &leistenRettung{tor: tor}, &leistenRettung{}
	l.e.rettungEndeImHintergrund(erst)
	l.e.rettungEndeImHintergrund(dann)
	time.Sleep(50 * time.Millisecond)
	select {
	case <-l.e.rettungEnden:
		t.Fatal("rettungEnden geschlossen, obwohl das frühere Ende noch läuft")
	default:
	}
	oeffne()
	l.e.rettungEndenAbwarten(5 * time.Second)
	if erst.ende != 1 || dann.ende != 1 {
		t.Errorf("Enden: %d, %d", erst.ende, dann.ende)
	}
}

// F-688 (KB2): Verlässt der Wechsel eine unveränderte Aufnahme, wird ihre Rettungskopie verworfen.
func TestUebernimmVerwirftAlteRettung(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	l.e.s.Gespeichert()
	l.klick(l.kachel(0))
	l.ruhig()
	if l.alt.verworfen != 1 || l.alt.ende != 1 {
		t.Errorf("alte Rettung: verworfen %d, Ende %d", l.alt.verworfen, l.alt.ende)
	}
}

// zaehleAbgleich ersetzt gleicheAb durch einen Zähler (und gleicht weiter ab).
func zaehleAbgleich(t *testing.T) *atomic.Int32 {
	t.Helper()
	var n atomic.Int32
	alt := gleicheAb
	gleicheAb = func(b *library.Bibliothek) { n.Add(1); alt(b) }
	t.Cleanup(func() { gleicheAb = alt })
	return &n
}

// F-688 (KB3, KB5): Abgeglichen wird beim Start der Leiste und beim Öffnen der Ansicht, nicht bei jedem Neulesen
// (nach dem eigenen Speichern, bei jeder Suche): sonst kostete jedes Speichern und jeder Tastendruck einen Durchlauf
// über alle Dateien (F-638).
func TestAbgleichNurBeiAnlass(t *testing.T) {
	n := zaehleAbgleich(t)
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	if got := n.Load(); got != 1 {
		t.Fatalf("Start: %d Abgleiche", got)
	}
	l.e.leisteNachSpeichern()
	l.ruhig()
	if got := n.Load(); got != 1 {
		t.Errorf("nach dem Speichern: %d Abgleiche", got)
	}
	l.taste("L", key.ModShortcut|key.ModShift)
	l.warte(func() bool { return l.e.bibl.offen && !l.e.bibl.indexLaeuft })
	if got := n.Load(); got != 2 {
		t.Errorf("Ansicht geöffnet: %d Abgleiche", got)
	}
	l.e.bibl.suche = "x"
	l.e.bibliothekNeuLesen()
	l.warte(func() bool { return !l.e.bibl.indexLaeuft })
	if got := n.Load(); got != 2 {
		t.Errorf("nach einer Suche: %d Abgleiche", got)
	}
}

// F-688 (KB7): Die Leiste meldet je gezeichnetem Bild eine Runde (sonst verfielen weggerollte Aufträge nie, F-669).
func TestLeisteMeldetRunden(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	vorher := l.e.unten.v.Runden()
	l.bild()
	l.bild()
	if got := l.e.unten.v.Runden(); got < vorher+2 {
		t.Errorf("Runden %d → %d nach zwei Bildern", vorher, got)
	}
}
