// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"
	"time"
)

// anderePfade liefert die erste Aufnahme, die nicht offen ist, und die offene (leer, wenn keine offen ist).
func anderePfade(l *leistenLauf) (andere, offen string) {
	for _, p := range l.pfade {
		if l.e.istOffen(p) {
			offen = p
		} else if andere == "" {
			andere = p
		}
	}
	return andere, offen
}

// F-774: Ein Klick auf die bereits offene Aufnahme meldet nichts und wechselt nicht, auch nicht während des Kopierens.
func TestKlickAufOffeneAufnahmeBleibtStill(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	_, offen := anderePfade(l)
	if offen == "" {
		t.Fatal("keine der Probe-Aufnahmen ist die offene")
	}
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.wechselAnfrage(offen)
	if e.meldung.Text != "" {
		t.Errorf("Klick auf die offene Aufnahme meldet %q", e.meldung.Text)
	}
	if a := l.aufrufe(); len(a) != 0 || e.wechsel.ziel != "" {
		t.Errorf("gewechselt: %v", a)
	}
}

// F-775: Bei laufender Arbeit der Ansicht sagt der gesperrte Klick es in der Fußzeile.
func TestKlickWaehrendArbeitDerAnsichtMeldet(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	andere, _ := anderePfade(l)
	e.bibl.arbeitLaeuft = true
	e.wechselAnfrage(andere)
	if m := e.meldung; !strings.Contains(m.Text, "library is still working") || m.Fehler {
		t.Errorf("Meldung nach dem Klick: %+v", m)
	}
	if a := l.aufrufe(); len(a) != 0 || e.wechsel.ziel != "" {
		t.Errorf("gewechselt: %v", a)
	}
}

// F-775: Dasselbe beim Laden einer anderen Aufnahme.
func TestKlickWaehrendLadenMeldet(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	andere, _ := anderePfade(l)
	e.wechsel.laedt = true
	e.wechselAnfrage(andere)
	if m := e.meldung; !strings.Contains(m.Text, "still loading") || m.Fehler {
		t.Errorf("Meldung nach dem Klick: %+v", m)
	}
	if e.wechsel.ziel != "" {
		t.Errorf("Ziel gesetzt: %q", e.wechsel.ziel)
	}
}

// F-775: Dasselbe während des Speicherns.
func TestKlickWaehrendSpeichernMeldet(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	andere, _ := anderePfade(l)
	if _, err := e.s.SpeichernJetzt(); err != nil || !e.s.SpeichernLaeuft() {
		t.Fatalf("Kontrolle: Speichern beginnt nicht (%v)", err)
	}
	e.wechselAnfrage(andere)
	if m := e.meldung; !strings.Contains(m.Text, "Saving is still running") || m.Fehler {
		t.Errorf("Meldung nach dem Klick: %+v", m)
	}
	if e.wechsel.ziel != "" {
		t.Errorf("Ziel gesetzt: %q", e.wechsel.ziel)
	}
}
