// SPDX-License-Identifier: GPL-3.0-or-later

package ui

// Karte #312 (ADR-0036 Zusatz): Ein ✕ oben rechts in der Vorschau legt die Aufnahme in den Papierkorb — sichtbar unter
// dem Zeiger oder nach langem Druck, in der Leiste und in der vollen Ansicht; nie an der offenen Aufnahme. Entf an einer
// Vorschau der Leiste tut dasselbe.

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/widget"
)

func papierkorbDateien(t *testing.T, l *leistenLauf) int {
	t.Helper()
	papier, _ := os.ReadDir(filepath.Join(l.b.Orte.Papierkorb, "files"))
	return len(papier)
}

func TestLeisteKreuzLegtInDenPapierkorb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 4)
	k := &l.e.unten
	if !k.l.IstAktuell(1) {
		t.Fatal("Vorbedingung: Stelle 1 ist die offene Aufnahme")
	}
	// Ohne Zeiger kein ✕.
	for i, r := range k.loeschRahmen {
		if !r.Empty() {
			t.Fatalf("✕ an Stelle %d ohne Zeiger", i)
		}
	}
	// Zeiger über der offenen: kein ✕.
	l.maus(pointer.Move, l.kachel(1), 0)
	l.setle()
	if !k.loeschRahmen[1].Empty() {
		t.Error("✕ an der offenen Aufnahme")
	}
	// Zeiger über Stelle 2: ✕ oben rechts in der Vorschau.
	l.maus(pointer.Move, l.kachel(2), 0)
	l.setle()
	lr, kr := k.loeschRahmen[2], k.kacheln[2]
	if lr.Empty() || !lr.In(kr) || lr.Max.X < kr.Max.X-8 || lr.Min.Y > kr.Min.Y+8 {
		t.Fatalf("✕ %v nicht oben rechts in %v", lr, kr)
	}
	// Zeiger auf das ✕: Hinweis, dann Klick legt in den Papierkorb, die Aufnahme öffnet sich nicht.
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	if k.hinweis != loeschHinweis() {
		t.Errorf("Hinweis am ✕: %q", k.hinweis)
	}
	weg := l.pfade[2]
	l.klick(mittenPunkt(k.loeschRahmen[2]))
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(weg); !os.IsNotExist(err) {
		t.Fatalf("Datei noch da: %v", err)
	}
	if n := papierkorbDateien(t, l); n != 1 {
		t.Errorf("Papierkorb hat %d Dateien", n)
	}
	if len(l.offen) != 0 {
		t.Errorf("✕ öffnete eine Aufnahme: %v", l.offen)
	}
	l.warte(func() bool { return len(k.l.Eintraege()) == 3 })
	if !strings.Contains(l.e.meldung.Text, "trash") || l.e.meldung.Fehler {
		t.Errorf("Meldung %q", l.e.meldung.Text)
	}
}

func TestLeisteEntfLegtInDenPapierkorb(t *testing.T) {
	l := neuerLeistenLauf(t, image.Pt(1200, 800), 3)
	k := &l.e.unten
	l.fokusiere(&k.knoepfe[2])
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft && !k.indexLaeuft })
	if _, err := os.Stat(l.pfade[2]); !os.IsNotExist(err) {
		t.Fatalf("Entf an der Vorschau: Datei noch da (%v)", err)
	}
	// An der offenen Aufnahme nur ein Hinweis.
	l.warte(func() bool { return len(k.l.Eintraege()) == 2 })
	l.setle()
	l.fokusiere(&k.knoepfe[1])
	l.taste(key.NameDeleteForward, 0)
	l.warte(func() bool { return !l.e.bibl.arbeitLaeuft })
	if _, err := os.Stat(l.pfade[1]); err != nil {
		t.Errorf("offene Aufnahme verschoben: %v", err)
	}
	if !strings.Contains(l.e.meldung.Text, "open capture") {
		t.Errorf("Meldung an der offenen: %q", l.e.meldung.Text)
	}
}

func TestLangerDruck(t *testing.T) {
	t0 := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	for _, f := range []struct {
		h    []widget.Press
		lang bool
	}{
		{nil, false},
		{[]widget.Press{{Start: t0, End: t0.Add(100 * time.Millisecond)}}, false},
		{[]widget.Press{{Start: t0, End: t0.Add(langDruckZeit)}}, true},
		{[]widget.Press{{Start: t0, End: t0.Add(time.Second), Cancelled: true}}, false},
		{[]widget.Press{{Start: t0}}, false}, // läuft noch
		{[]widget.Press{{Start: t0, End: t0.Add(time.Second)}, {Start: t0, End: t0.Add(time.Millisecond)}}, false},
	} {
		if got := langerDruck(f.h); got != f.lang {
			t.Errorf("%v: %v, Soll %v", f.h, got, f.lang)
		}
	}
}

func TestAnsichtKreuzUndLangerDruck(t *testing.T) {
	l := bibLauf(t, 4)
	b := &l.e.bibl
	// Zeiger über Stelle 2: ✕; über der offenen (1) keins.
	l.maus(pointer.Move, l.bibKachel(1), 0)
	l.setle()
	if !b.loesch[1-b.von].Empty() {
		t.Error("✕ an der offenen Aufnahme")
	}
	l.maus(pointer.Move, l.bibKachel(2), 0)
	l.setle()
	lr := b.loesch[2-b.von]
	if lr.Empty() {
		t.Fatal("kein ✕ unter dem Zeiger")
	}
	l.maus(pointer.Move, mittenPunkt(lr), 0)
	l.setle()
	if h := l.e.bibliothekHinweis(); h != loeschHinweis() {
		t.Errorf("Hinweis am ✕: %q", h)
	}
	weg := l.pfade[2]
	l.bibDruck(mittenPunkt(lr), time.Second)
	l.bibRuhig()
	if _, err := os.Stat(weg); !os.IsNotExist(err) {
		t.Fatalf("✕ in der Ansicht: Datei noch da (%v)", err)
	}
	if len(b.r.Eintraege()) != 3 {
		t.Errorf("%d Einträge nach dem ✕", len(b.r.Eintraege()))
	}
	// Langer Druck (Finger: kein Zeiger darüber) zeigt das ✕ der Kachel.
	l.maus(pointer.Leave, image.Pt(0, 0), 0)
	l.setle()
	p := l.bibKachel(2)
	pos := f32.Pt(float32(p.X), float32(p.Y))
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Buttons: pointer.ButtonPrimary, Position: pos,
		Time: 10 * time.Second})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos,
		Time: 10*time.Second + langDruckZeit})
	l.setle()
	if b.langDruck != 2 || b.loesch[2-b.von].Empty() {
		t.Errorf("langer Druck: Stelle %d, ✕ %v", b.langDruck, b.loesch[2-b.von])
	}
	// Ein kurzer Druck zeigt es nicht.
	b.langDruck = -1
	l.r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, Buttons: pointer.ButtonPrimary, Position: pos,
		Time: 20 * time.Second})
	l.r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, Position: pos, Time: 20*time.Second + 100*time.Millisecond})
	l.setle()
	if b.langDruck != -1 {
		t.Errorf("kurzer Druck zeigte das ✕ (Stelle %d)", b.langDruck)
	}
}
