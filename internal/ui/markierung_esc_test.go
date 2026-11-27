// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Esc und die Markierung der Leiste (ADR-0036 Zusatz Karte #361 Punkte 3 und 7), nachgeschärft nach dem Verify zu
// 5bc7352: Esc wirkt auch während des Speicherns vor einem Papierkorb-Wechsel (N-1), eine unsichtbare Markierung
// verbraucht kein Esc (N-2), eine Texteingabe geht vor (N-3).

import (
	"image"
	"image/color"
	"os"
	"testing"

	"gioui.org/io/key"

	"github.com/Andiministrator/schnapp/internal/canvas"
	"github.com/Andiministrator/schnapp/internal/document"
)

func dateiDa(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// aendereUndHalteSpeichern ändert die offene Aufnahme und lässt das Speichern am Tor warten.
func (l *leistenLauf) aendereUndHalteSpeichern() chan struct{} {
	l.t.Helper()
	tor := make(chan struct{})
	l.e.speichern = func(*document.Dokument) (Gespeichert, error) { <-tor; return Gespeichert{}, nil }
	if _, err := l.e.s.D.Hinzufuegen(document.Text{Ort: document.Pt(5, 15), Inhalt: "neu", Schriftgrad: 12,
		Farbe: color.NRGBA{A: 255}}); err != nil {
		l.t.Fatal(err)
	}
	return tor
}

// N-1: Esc während des Speicherns (Fokus an der Vorschau und auf der Leinwand) hebt die Markierung auf; nach dem
// Wechsel geht nur die bisher offene in den Papierkorb.
func TestLeisteEscWaehrendSpeichern(t *testing.T) {
	for _, leinwand := range []bool{false, true} {
		l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
		k := &l.e.unten
		tor := l.aendereUndHalteSpeichern()
		l.klickMit(l.kachel(1), key.ModShortcut)
		l.klickMit(l.kachel(2), key.ModShortcut)
		if leinwand {
			l.e.papierkorbMarkiert(true)
			l.fokusiere(l.e)
		} else {
			l.fokusiere(&k.knoepfe[1])
			l.taste(key.NameDeleteForward, 0)
		}
		l.warte(func() bool { return l.e.s.SpeichernLaeuft() })
		l.taste(key.NameEscape, 0)
		n := k.l.Markierung().Anzahl()
		close(tor)
		l.ruhig()
		if n != 0 || !dateiDa(l.pfade[2]) {
			t.Errorf("Leinwand %v: Esc während des Speicherns: markiert %d, weitere noch da %v", leinwand, n, dateiDa(l.pfade[2]))
		}
	}
}

// N-2: Ist die Leiste aus, verbraucht ihre Markierung kein Esc; Ausschalten leert sie.
func TestEscBeiAusgeschalteterLeiste(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	k.an = false // ohne Menü: die Markierung bleibt stehen, nur die Leiste ist unsichtbar
	l.setle()
	l.e.s.SetzeWerkzeug(canvas.WerkzeugPfeil)
	l.e.s.Abwaehlen() // Esc in Stufen: ohne Auswahl wechselt es zurück zu Auswählen
	l.fokusiere(l.e)
	l.taste(key.NameEscape, 0)
	if l.e.s.Werkzeug == canvas.WerkzeugPfeil {
		t.Error("Esc bei unsichtbarer Leiste erreichte die Leinwand nicht")
	}
	k.an = true
	l.setle()
	l.taste("R", key.ModShortcut|key.ModShift) // Ansicht → Recent captures: aus
	if k.an || k.l.Markierung().Anzahl() != 0 {
		t.Errorf("Ausschalten: an %v, markiert %d", k.an, k.l.Markierung().Anzahl())
	}
}

// N-3: Eine Texteingabe geht vor: Esc verwirft zuerst sie, die Markierung bleibt; erst das nächste Esc leert sie.
func TestLeisteEscTexteingabeGehtVor(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	l.klickMit(l.kachel(2), key.ModShortcut)
	l.klickMit(l.kachel(3), key.ModShortcut)
	l.e.s.SetzeWerkzeug(canvas.WerkzeugText)
	p := l.e.s.Abb.ZuFenster(document.Pt(5, 10))
	l.e.s.Druecken(p, canvas.TastePrimaer)
	l.e.s.Loslassen(p)
	l.fokusiere(l.e)
	if !l.e.s.TextLaeuft() {
		t.Fatal("keine Texteingabe")
	}
	l.taste(key.NameEscape, 0)
	if l.e.s.TextLaeuft() || k.l.Markierung().Anzahl() != 2 {
		t.Fatalf("erstes Esc: Text läuft %v, markiert %d", l.e.s.TextLaeuft(), k.l.Markierung().Anzahl())
	}
	l.taste(key.NameEscape, 0)
	if k.l.Markierung().Anzahl() != 0 {
		t.Errorf("zweites Esc: markiert %d", k.l.Markierung().Anzahl())
	}
}
