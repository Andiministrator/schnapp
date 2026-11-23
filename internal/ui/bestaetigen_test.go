// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// dialogProbe fährt den Bestätigungsdialog ohne Fenster: input.Router statt Gio-Schleife (F-481).
type dialogProbe struct {
	t   *testing.T
	d   bestaetigenDialog
	r   input.Router
	ops op.Ops
	t0  time.Time
}

const probeBreite, probeHoehe = 480, 200

func neueDialogProbe(t *testing.T) *dialogProbe {
	return &dialogProbe{t: t, t0: time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)}
}

// bild zeichnet ein Bild zur Zeit t0 + ab.
func (p *dialogProbe) bild(ab time.Duration) {
	p.ops.Reset()
	gtx := layout.Context{
		Ops:         &p.ops,
		Source:      p.r.Source(),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(probeBreite, probeHoehe)),
		Now:         p.t0.Add(ab),
	}
	p.d.bild(gtx, bestaetigenThema(), "the active monitor", "probe", p.t0.Add(ab))
	p.r.Frame(&p.ops)
}

// erlaubenPunkt liegt im Knopf „Allow“ (rechts unten, Rand 18 dp).
var erlaubenPunkt = f32.Pt(probeBreite-18-6, probeHoehe-18-6)

func (p *dialogProbe) druecke(ab time.Duration) {
	p.r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: erlaubenPunkt},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: erlaubenPunkt})
	p.bild(ab)
}

func (p *dialogProbe) lasseLos(ab time.Duration) {
	p.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: erlaubenPunkt})
	p.bild(ab)
}

// TestBestaetigenVerdrahtung: Der Klick auf „Allow“ kommt über gesture.Click bei der Logik an — in der Sperre nie,
// danach ja; Esc lehnt ab; Return und Leertaste erlauben nie (F-481, F-464).
func TestBestaetigenVerdrahtung(t *testing.T) {
	nach := bestaetigenSperre + 50*time.Millisecond

	t.Run("Klick nach der Sperre erlaubt", func(t *testing.T) {
		p := neueDialogProbe(t)
		p.bild(0)
		p.druecke(nach)
		p.lasseLos(nach + 20*time.Millisecond)
		if !p.d.fertig || !p.d.antwort {
			t.Fatalf("Klick nach der Sperre: fertig=%v antwort=%v, erwartet erlaubt (Gegenprobe der Koordinaten)", p.d.fertig, p.d.antwort)
		}
	})
	t.Run("Klick in der Sperre erlaubt nicht", func(t *testing.T) {
		p := neueDialogProbe(t)
		p.bild(0)
		p.druecke(100 * time.Millisecond)
		p.lasseLos(150 * time.Millisecond)
		if p.d.antwort {
			t.Fatal("Klick innerhalb der Sperre hat erlaubt")
		}
	})
	t.Run("Drücken in der Sperre, Loslassen danach", func(t *testing.T) {
		p := neueDialogProbe(t)
		p.bild(0)
		p.druecke(100 * time.Millisecond)
		p.lasseLos(nach)
		if p.d.antwort {
			t.Fatal("Drücken in der Sperre zählte nach ihrem Ende")
		}
	})
	t.Run("Fokus setzt die Sperre neu", func(t *testing.T) {
		p := neueDialogProbe(t)
		p.bild(0)
		p.bild(nach) // erste Sperre vorbei
		p.d.f.fokus(true)
		p.bild(nach + 10*time.Millisecond) // Sperre beginnt neu
		p.druecke(nach + 100*time.Millisecond)
		p.lasseLos(nach + 150*time.Millisecond)
		if p.d.antwort {
			t.Fatal("Klick kurz nach dem Fokus-Erhalt hat erlaubt")
		}
		ab := nach + 10*time.Millisecond + bestaetigenSperre + 10*time.Millisecond
		p.druecke(ab)
		p.lasseLos(ab + 20*time.Millisecond)
		if !p.d.antwort {
			t.Fatal("Klick nach der neuen Sperre wurde nicht angenommen")
		}
	})
	t.Run("Tasten", func(t *testing.T) {
		p := neueDialogProbe(t)
		p.bild(0)
		p.bild(nach)
		for _, n := range []key.Name{key.NameReturn, key.NameEnter, key.NameSpace, key.NameTab} {
			p.r.Queue(key.Event{Name: n, State: key.Press}, key.Event{Name: n, State: key.Release})
			p.bild(nach + time.Millisecond)
			if p.d.antwort {
				t.Fatalf("Taste %q hat erlaubt", n)
			}
		}
		p.r.Queue(key.Event{Name: key.NameEscape, State: key.Press})
		p.bild(nach + 2*time.Millisecond)
		if !p.d.fertig || p.d.antwort {
			t.Fatalf("Esc: fertig=%v antwort=%v, erwartet abgelehnt", p.d.fertig, p.d.antwort)
		}
	})
}

// TestFreigabeFokus: Fokus-Erhalt setzt die Sperre mit dem nächsten Bild neu, Fokusverlust macht ein Drücken ungültig.
func TestFreigabeFokus(t *testing.T) {
	t0 := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	var f freigabe
	f.bild(t0)
	spaet := t0.Add(2 * bestaetigenSperre)
	f.fokus(true)
	f.bild(spaet)
	if !f.gesperrt(spaet.Add(bestaetigenSperre / 2)) {
		t.Fatal("nach Fokus-Erhalt nicht neu gesperrt")
	}
	f.fokus(true) // kein Wechsel: keine neue Sperre
	f.bild(spaet.Add(bestaetigenSperre + time.Millisecond))
	frei := spaet.Add(bestaetigenSperre + 2*time.Millisecond)
	if f.gesperrt(frei) {
		t.Fatal("Fokus ohne Wechsel hat die Sperre verlängert")
	}
	f.druck(frei)
	f.fokus(false)
	if f.klick(frei.Add(time.Millisecond)) {
		t.Fatal("Drücken vor einem Fokusverlust zählte")
	}
}
