// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"strings"
	"testing"
	"time"
)

// F-759: Ein Klick auf eine Vorschau während eines laufenden Kopierens wechselt nicht und sagt in der Fußzeile warum.
func TestWechselWaehrendKopierenMeldetEsInDerFusszeile(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.wechselAnfrage(l.pfade[0])
	if m := e.meldung; m.Text == "" || !strings.Contains(m.Text, "Copying is still running") || m.Fehler {
		t.Errorf("Meldung nach dem Klick: %+v", m)
	}
	if a := l.aufrufe(); len(a) != 0 || e.wechsel.ziel != "" {
		t.Errorf("gewechselt während des Kopierens: %v", a)
	}
}

// F-759: Eine haltende Meldung (etwa eine Verwerfen-Warnung) verdrängt der Hinweis nicht.
func TestWechselWaehrendKopierenVerdraengtHaltendeMeldungNicht(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	e := l.e
	e.s.Gespeichert()
	e.meldeHaltend(haltVerwerfen, "halte mich", true)
	e.kopierLaeuft, e.kopierStart, e.kopierStand = true, time.Now(), e.s.D.Stand()
	e.wechselAnfrage(l.pfade[0])
	if e.meldung.Text != "halte mich" {
		t.Errorf("haltende Meldung verdrängt: %q", e.meldung.Text)
	}
}
