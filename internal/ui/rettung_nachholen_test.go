// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Tests zu F-473 (Gegenproben der Lage-Meldung), F-474 (Lage-Meldung ohne Fehler hinter einer haltenden Meldung, Wecken
// im Takt) und F-477 (Speicher- und Kopierfehler ohne Paketpräfix), im Editor ohne Fenster.

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/Andiministrator/schnapp/internal/canvas"
)

// F-474: Eine Lage-Meldung ohne Fehler, die hinter einer haltenden Meldung nicht erscheinen kann, geht nicht verloren:
// Der Zähler bleibt unverbraucht, und sie erscheint, sobald die haltende Meldung weicht.
func TestStatusleisteLageOhneFehlerWirdNachgeholt(t *testing.T) {
	e := testEditor(t)
	aus := fehlerAbfangen(t)
	r := &lageRettung{}
	e.rettung = r
	e.rettungStart()
	e.meldeHaltend(haltVerwerfen, "Saving failed", true)
	r.lText, r.lNr = "wieder geschützt", 1
	e.rettungPruefen()
	if e.meldung.Text != "Saving failed" {
		t.Fatalf("Meldung %q, die haltende Meldung muss stehen bleiben", e.meldung.Text)
	}
	if e.rett.lageNr != 0 {
		t.Errorf("lageNr %d: Der Zähler ist verbraucht, die Meldung wurde nie gezeigt", e.rett.lageNr)
	}
	if aus.Len() != 0 {
		t.Errorf("Fehlerausgabe %q: Eine Meldung ohne Fehler gehört nicht dorthin", aus.String())
	}
	e.rettungPruefen() // noch haltend: weiter warten
	if e.meldung.Text != "Saving failed" {
		t.Fatalf("Meldung %q nach zweiter Prüfung", e.meldung.Text)
	}
	e.melde("Gespeichert", false) // die haltende Meldung weicht
	e.rettungPruefen()
	if e.meldung.Text != "wieder geschützt" {
		t.Errorf("Meldung %q, erwartet die nachgeholte Lage-Meldung", e.meldung.Text)
	}
	if e.rett.lageNr != 1 {
		t.Errorf("lageNr %d, erwartet 1 nach der Anzeige", e.rett.lageNr)
	}
}

// F-473: Die Lage wird auch dann geprüft, wenn das Schließen schon freigegeben ist oder ein Speichern läuft; die Kette der
// Rettung wechselt die Sperre nach dem Speichern.
func TestStatusleisteLageAuchBeiErzwungenemSchliessen(t *testing.T) {
	e := testEditor(t)
	r := &lageRettung{}
	e.rettung = r
	e.rettungStart()
	e.erzwungen = true
	r.lText, r.lNr = "fremd: another window", 1
	e.rettungPruefen()
	if e.meldung.Text != "fremd: another window" {
		t.Errorf("Meldung %q, erwartet die Lage-Meldung auch bei erzwungenem Schließen", e.meldung.Text)
	}
}

// zaehlWecker liefert einen Wecker, der die Weckrufe zählt und von selbst nicht wiederholt.
func zaehlWecker(t *testing.T) (*canvas.Wecker, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	w := canvas.NeuerWecker(func() { n.Add(1) }, time.Hour, time.Hour)
	t.Cleanup(w.Ende)
	return w, &n
}

func warteAuf(t *testing.T, frist time.Duration, bedingung func() bool) bool {
	t.Helper()
	ende := time.Now().Add(frist)
	for time.Now().Before(ende) {
		if bedingung() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return bedingung()
}

// F-473: Nach einem gelungenen Speichern weckt der Editor nach rettungNachweckZeit (Vorgabe 300 ms), damit die Lage der
// Sperre sichtbar wird. Mit den Vorgabewerten, nicht mit verkürzten.
func TestNachweckenNachSpeichernWeckt(t *testing.T) {
	e := testEditor(t)
	e.rettung = &rettungAttrappe{}
	w, n := zaehlWecker(t)
	e.wecker = w
	e.rettungGespeichert()
	if n.Load() != 0 {
		t.Fatalf("%d Weckrufe sofort, erwartet erst nach dem Takt", n.Load())
	}
	if !warteAuf(t, 2*time.Second, func() bool { return n.Load() >= 1 }) {
		t.Errorf("kein Weckruf nach dem Speichern")
	}
}

// F-474: Braucht die Kette länger als einen Takt, wiederholt sich das Wecken bis zur Grenze und endet dann.
func TestNachweckenWiederholtSichBisZurGrenze(t *testing.T) {
	zeit, grenze := rettungNachweckZeit, rettungNachweckGrenze
	rettungNachweckZeit, rettungNachweckGrenze = 20*time.Millisecond, 200*time.Millisecond
	t.Cleanup(func() { rettungNachweckZeit, rettungNachweckGrenze = zeit, grenze })
	e := testEditor(t)
	e.rettung = &rettungAttrappe{}
	w, n := zaehlWecker(t)
	e.wecker = w
	e.rettungGespeichert()
	if !warteAuf(t, 2*time.Second, func() bool { return n.Load() >= 3 }) {
		t.Fatalf("%d Weckrufe, erwartet mindestens 3", n.Load())
	}
	time.Sleep(400 * time.Millisecond) // über die Grenze, der letzte Takt ist durch
	a := n.Load()
	time.Sleep(100 * time.Millisecond)
	if b := n.Load(); b != a {
		t.Errorf("Weckrufe nach der Grenze: %d, dann %d, erwartet Ende", a, b)
	}
}

// F-477: Speicherfehler und Kopierfehler erscheinen in der Statusleiste und auf der Fehlerausgabe ohne Paketpräfix.
func TestSpeicherUndKopierFehlerOhnePaketpraefix(t *testing.T) {
	t.Run("speichern", func(t *testing.T) {
		e := testEditor(t)
		e.speichernMelden(speicherErgebnis{err: errors.New("library: fileformat: Platte voll")}, canvas.Ende{}, false, false)
		if e.meldung.Text != "Could not save: Platte voll" {
			t.Errorf("Meldung %q", e.meldung.Text)
		}
	})
	t.Run("speichern, Schließen", func(t *testing.T) {
		e := testEditor(t)
		e.speichernMelden(speicherErgebnis{err: errors.New("library: Platte voll")}, canvas.Ende{Schluss: canvas.SchlussFehler}, false, false)
		if !strings.HasPrefix(e.meldung.Text, "Could not save: Platte voll.") {
			t.Errorf("Meldung %q", e.meldung.Text)
		}
	})
	t.Run("kopieren", func(t *testing.T) {
		e, _ := haltStart(t)
		aus := fehlerAbfangen(t)
		e.schliessenNachKopie = true // Fehlerausgabe wird nur bei wartendem Schließen beschrieben
		e.kopierAntwort <- kopierErgebnis{err: errors.New("document: canvas: kaputt")}
		e.kopierenAbholen(layout.Context{})
		if e.meldung.Text != "Could not copy: kaputt" {
			t.Errorf("Meldung %q", e.meldung.Text)
		}
		if !strings.Contains(aus.String(), "Could not copy: kaputt") || strings.Contains(aus.String(), "document:") {
			t.Errorf("Fehlerausgabe %q", aus.String())
		}
	})
}
